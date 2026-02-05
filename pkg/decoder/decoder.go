// Package decoder implements an AAC decoder pipeline.
//
// This is a direct port of the AAC decoder module from AAC.js by Devon Govett
// (LGPL v3).
package decoder

import (
	"fmt"

	"github.com/skrashevich/go-aac/pkg/adts"
	"github.com/skrashevich/go-aac/pkg/cce"
	"github.com/skrashevich/go-aac/pkg/cpe"
	"github.com/skrashevich/go-aac/pkg/filterbank"
	"github.com/skrashevich/go-aac/pkg/ics"
	"github.com/skrashevich/go-aac/pkg/tables"
)

const (
	aotAACMain  = 1
	aotAACLC    = 2
	aotAACLTP   = 4
	aotERAACLD  = 23
	aotEscape   = 31
	aotERAACELD = 39
)

const (
	channelConfigNone               = 0
	channelConfigMono               = 1
	channelConfigStereo             = 2
	channelConfigStereoPlusCenter   = 3
	channelConfigStereoPlusRearMono = 4
	channelConfigFive               = 5
	channelConfigFivePlusOne        = 6
	channelConfigSevenPlusOne       = 8
)

const (
	sceElement = 0
	cpeElement = 1
	cceElement = 2
	lfeElement = 3
	dseElement = 4
	pceElement = 5
	filElement = 6
	endElement = 7
)

// ELDExtension types
const (
	eldExtTerm          = 0
	eldExtSAOC          = 1
	eldExtLDSAC         = 2
	eldExtDownscaleInfo = 3
)

// Config contains the AAC decoder configuration.
type Config struct {
	Profile                int
	SampleIndex            int
	SampleRate             int
	ChanConfig             int
	FrameLength            int
	SectionDataResilience  bool
	ScalefactorResilience  bool
	SpectralDataResilience bool

	// ELD-specific configuration
	IsELD           bool
	IsLD            bool
	FrameLengthFlag bool // false = 512 samples, true = 480 samples
	ELDSBRPresent   bool
	ELDSBRRate      bool
	ELDSBRCrc       bool
}

// Decoder is a high-level AAC decoder.
type Decoder struct {
	Config     Config
	FilterBank *filterbank.FilterBank
	CCEs       []*cce.Element
	Data       [][]float32
	SBRPresent bool
}

// New creates a new AAC decoder.
func New() *Decoder {
	return &Decoder{}
}

// SetASC parses an MPEG-4 AudioSpecificConfig (2 bytes) and initializes the decoder.
func (d *Decoder) SetASC(data []byte) error {
	stream := NewBitstream(data)
	config := Config{}

	config.Profile = int(stream.ReadBits(5))
	if config.Profile == aotEscape {
		config.Profile = 32 + int(stream.ReadBits(6))
	}

	config.SampleIndex = int(stream.ReadBits(4))
	if config.SampleIndex == 0x0f {
		config.SampleRate = int(stream.ReadBits(24))
		for i := 0; i < len(tables.SampleRates); i++ {
			if int(tables.SampleRates[i]) == config.SampleRate {
				config.SampleIndex = i
				break
			}
		}
	} else {
		if config.SampleIndex < 0 || config.SampleIndex >= len(tables.SampleRates) {
			return fmt.Errorf("decoder: invalid sample index %d", config.SampleIndex)
		}
		config.SampleRate = int(tables.SampleRates[config.SampleIndex])
	}

	config.ChanConfig = int(stream.ReadBits(4))

	switch config.Profile {
	case aotAACMain, aotAACLC, aotAACLTP:
		if stream.ReadBits(1) != 0 {
			return fmt.Errorf("decoder: frameLengthFlag not supported")
		}
		config.FrameLength = 1024

		if stream.ReadBits(1) != 0 {
			_ = stream.ReadBits(14)
		}

		if stream.ReadBits(1) != 0 {
			if config.Profile > 16 {
				config.SectionDataResilience = stream.ReadBits(1) != 0
				config.ScalefactorResilience = stream.ReadBits(1) != 0
				config.SpectralDataResilience = stream.ReadBits(1) != 0
			}
			_ = stream.ReadBits(1)
		}

		if config.ChanConfig == channelConfigNone {
			_ = stream.ReadBits(4)
			return fmt.Errorf("decoder: PCE unimplemented")
		}

	case aotERAACELD:
		config.IsELD = true
		if err := d.parseELDSpecificConfig(stream, &config); err != nil {
			return err
		}

	case aotERAACLD:
		config.IsLD = true
		config.FrameLengthFlag = stream.ReadBits(1) != 0
		if config.FrameLengthFlag {
			config.FrameLength = 480
		} else {
			config.FrameLength = 512
		}

		// dependsOnCoreCoder
		if stream.ReadBits(1) != 0 {
			_ = stream.ReadBits(14) // coreCoderDelay
		}
		// extensionFlag
		if stream.ReadBits(1) != 0 {
			config.SectionDataResilience = stream.ReadBits(1) != 0
			config.ScalefactorResilience = stream.ReadBits(1) != 0
			config.SpectralDataResilience = stream.ReadBits(1) != 0
			_ = stream.ReadBits(1) // extensionFlag3
		}

		if config.ChanConfig == channelConfigNone {
			return fmt.Errorf("decoder: PCE unimplemented for AAC-LD")
		}

	default:
		return fmt.Errorf("decoder: AAC profile %d not supported", config.Profile)
	}

	if err := stream.Error(); err != nil {
		return err
	}

	var fb *filterbank.FilterBank
	var err error
	if config.IsELD || config.IsLD {
		fb, err = filterbank.NewELD(config.FrameLength, config.ChanConfig)
	} else {
		fb, err = filterbank.New(false, config.ChanConfig)
	}
	if err != nil {
		return err
	}

	d.Config = config
	d.FilterBank = fb
	return nil
}

// parseELDSpecificConfig parses the ELD-specific configuration from the bitstream.
// Reference: ISO/IEC 14496-3:2009, Section 4.4.2.1
func (d *Decoder) parseELDSpecificConfig(stream *Bitstream, config *Config) error {
	config.FrameLengthFlag = stream.ReadBits(1) != 0
	if config.FrameLengthFlag {
		config.FrameLength = 480
	} else {
		config.FrameLength = 512
	}

	// aacSectionDataResilienceFlag
	config.SectionDataResilience = stream.ReadBits(1) != 0
	// aacScalefactorDataResilienceFlag
	config.ScalefactorResilience = stream.ReadBits(1) != 0
	// aacSpectralDataResilienceFlag
	config.SpectralDataResilience = stream.ReadBits(1) != 0

	// ldSbrPresentFlag
	config.ELDSBRPresent = stream.ReadBits(1) != 0
	if config.ELDSBRPresent {
		config.ELDSBRRate = stream.ReadBits(1) != 0
		config.ELDSBRCrc = stream.ReadBits(1) != 0

		// Skip ld_sbr_header (we don't implement SBR yet, but need to parse it)
		// For now, this is a simplified parse that skips the SBR header
	}

	// Parse ELD extensions
	for {
		eldExtType := int(stream.ReadBits(4))
		if eldExtType == eldExtTerm {
			break
		}

		eldExtLen := int(stream.ReadBits(4))
		if eldExtLen == 15 {
			eldExtLen += int(stream.ReadBits(8))
		}
		if eldExtLen == 15+255 {
			eldExtLen = int(stream.ReadBits(16))
		}

		// Skip extension data
		stream.Advance(eldExtLen * 8)
	}

	if config.ChanConfig == channelConfigNone {
		return fmt.Errorf("decoder: PCE unimplemented for AAC-ELD")
	}

	return stream.Error()
}

// DecodeFrame decodes a single AAC frame and returns interleaved PCM samples.
func (d *Decoder) DecodeFrame(data []byte) ([]float32, error) {
	stream := NewBitstream(data)
	if stream.PeekBits(12) == 0xfff {
		header, err := adts.ReadHeaderFromBytes(data)
		if err != nil {
			return nil, err
		}
		if d.Config.SampleIndex == 0 && d.Config.SampleRate == 0 {
			if err := d.setConfigFromADTS(header); err != nil {
				return nil, err
			}
		}

		headerBits := 56
		if !header.ProtectionAbsent {
			headerBits = 72
		}
		stream.Advance(headerBits)
	}

	if d.FilterBank == nil || d.Config.FrameLength == 0 {
		return nil, fmt.Errorf("decoder: config not initialized")
	}

	d.CCEs = nil
	elements := make([]frameElement, 0, 8)
	config := d.Config
	frameLength := config.FrameLength

	if config.IsELD || config.IsLD {
		// ER (Error Resilient) syntax: fixed structure based on channel config,
		// no element ID/type loop
		var err error
		elements, err = d.decodeERFrame(stream)
		if err != nil {
			return nil, err
		}
	} else {
		// GA (General Audio) syntax: element type loop
		for elementType := int(stream.ReadBits(3)); elementType != endElement; elementType = int(stream.ReadBits(3)) {
			id := int(stream.ReadBits(4))
			switch elementType {
			case sceElement, lfeElement:
				icsStream, err := ics.New(d.icsConfig())
				if err != nil {
					return nil, err
				}
				if err := icsStream.Decode(stream, d.icsConfig(), false); err != nil {
					return nil, err
				}
				elements = append(elements, frameElement{kind: elemSCE, id: id, sce: icsStream})
			case cpeElement:
				cpeElem, err := cpe.New(d.icsConfig())
				if err != nil {
					return nil, err
				}
				if err := cpeElem.Decode(stream, d.icsConfig()); err != nil {
					return nil, err
				}
				elements = append(elements, frameElement{kind: elemCPE, id: id, cpe: cpeElem})
			case cceElement:
				cceElem, err := cce.New(d.icsConfig())
				if err != nil {
					return nil, err
				}
				if err := cceElem.Decode(stream, d.icsConfig()); err != nil {
					return nil, err
				}
				d.CCEs = append(d.CCEs, cceElem)
			case dseElement:
				align := stream.ReadBits(1)
				count := int(stream.ReadBits(8))
				if count == 255 {
					count += int(stream.ReadBits(8))
				}
				if align != 0 {
					stream.Align()
				}
				stream.Advance(count * 8)
			case pceElement:
				return nil, fmt.Errorf("decoder: PCE_ELEMENT not implemented")
			case filElement:
				if id == 15 {
					id += int(stream.ReadBits(8)) - 1
				}
				stream.Advance(id * 8)
			default:
				return nil, fmt.Errorf("decoder: unknown element type %d", elementType)
			}
		}
	}

	stream.Align()
	if err := d.process(elements); err != nil {
		return nil, err
	}
	if err := stream.Error(); err != nil {
		return nil, err
	}

	channels := len(d.Data)
	output := make([]float32, frameLength*channels)
	idx := 0
	for k := 0; k < frameLength; k++ {
		for i := 0; i < channels; i++ {
			output[idx] = d.Data[i][k] / 32768.0
			idx++
		}
	}
	return output, nil
}

type elementKind int

const (
	elemSCE elementKind = iota
	elemCPE
)

type frameElement struct {
	kind elementKind
	id   int
	sce  *ics.ICStream
	cpe  *cpe.Element
}

func (d *Decoder) process(elements []frameElement) error {
	channels := d.Config.ChanConfig

	length := d.Config.FrameLength
	d.Data = make([][]float32, channels)
	for i := 0; i < channels; i++ {
		d.Data[i] = make([]float32, length)
	}

	channel := 0
	for i := 0; i < len(elements) && channel < channels; i++ {
		e := elements[i]
		switch e.kind {
		case elemSCE:
			count, err := d.processSingle(e.id, e.sce, channel)
			if err != nil {
				return err
			}
			channel += count
		case elemCPE:
			if err := d.processPair(e.id, e.cpe, channel); err != nil {
				return err
			}
			channel += 2
		default:
			return fmt.Errorf("decoder: unknown element kind")
		}
	}

	return nil
}

func (d *Decoder) processSingle(id int, element *ics.ICStream, channel int) (int, error) {
	profile := d.Config.Profile
	info := element.Info
	data := element.Data
	isELD := d.Config.IsELD || d.Config.IsLD

	if profile == aotAACMain {
		return 0, fmt.Errorf("decoder: main prediction unimplemented")
	}
	if profile == aotAACLTP {
		return 0, fmt.Errorf("decoder: LTP prediction unimplemented")
	}

	if !isELD {
		if err := d.applyChannelCoupling(id, false, cce.BeforeTNS, data, nil); err != nil {
			return 0, err
		}
	}

	if element.TnsPresent {
		element.ApplyTNS(data, false)
	}

	if !isELD {
		if err := d.applyChannelCoupling(id, false, cce.AfterTNS, data, nil); err != nil {
			return 0, err
		}
	}

	if isELD {
		if err := d.FilterBank.ProcessELD(data, d.Data[channel], channel); err != nil {
			return 0, err
		}
	} else {
		if err := d.FilterBank.Process(windowInfo(info), data, d.Data[channel], channel); err != nil {
			return 0, err
		}
	}

	if profile == aotAACLTP {
		return 0, fmt.Errorf("decoder: LTP prediction unimplemented")
	}

	if !isELD {
		if err := d.applyChannelCoupling(id, false, cce.AfterIMDCT, d.Data[channel], nil); err != nil {
			return 0, err
		}
	}

	if element.GainPresent {
		return 0, fmt.Errorf("decoder: gain control not implemented")
	}
	if d.SBRPresent {
		return 0, fmt.Errorf("decoder: SBR not implemented")
	}

	return 1, nil
}

func (d *Decoder) processPair(id int, element *cpe.Element, channel int) error {
	profile := d.Config.Profile
	left := element.Left
	right := element.Right
	lInfo := left.Info
	rInfo := right.Info
	lData := left.Data
	rData := right.Data
	isELD := d.Config.IsELD || d.Config.IsLD

	if element.CommonWindow && element.MaskPresent {
		d.processMS(element, lData, rData)
	}

	if profile == aotAACMain {
		return fmt.Errorf("decoder: main prediction unimplemented")
	}

	d.processIS(element, lData, rData)

	if profile == aotAACLTP {
		return fmt.Errorf("decoder: LTP prediction unimplemented")
	}

	if !isELD {
		if err := d.applyChannelCoupling(id, true, cce.BeforeTNS, lData, rData); err != nil {
			return err
		}
	}

	if left.TnsPresent {
		left.ApplyTNS(lData, false)
	}
	if right.TnsPresent {
		right.ApplyTNS(rData, false)
	}

	if !isELD {
		if err := d.applyChannelCoupling(id, true, cce.AfterTNS, lData, rData); err != nil {
			return err
		}
	}

	if isELD {
		if err := d.FilterBank.ProcessELD(lData, d.Data[channel], channel); err != nil {
			return err
		}
		if err := d.FilterBank.ProcessELD(rData, d.Data[channel+1], channel+1); err != nil {
			return err
		}
	} else {
		if err := d.FilterBank.Process(windowInfo(lInfo), lData, d.Data[channel], channel); err != nil {
			return err
		}
		if err := d.FilterBank.Process(windowInfo(rInfo), rData, d.Data[channel+1], channel+1); err != nil {
			return err
		}
	}

	if profile == aotAACLTP {
		return fmt.Errorf("decoder: LTP prediction unimplemented")
	}

	if !isELD {
		if err := d.applyChannelCoupling(id, true, cce.AfterIMDCT, d.Data[channel], d.Data[channel+1]); err != nil {
			return err
		}
	}

	if left.GainPresent || right.GainPresent {
		return fmt.Errorf("decoder: gain control not implemented")
	}
	if d.SBRPresent {
		return fmt.Errorf("decoder: SBR not implemented")
	}

	return nil
}

func (d *Decoder) processIS(element *cpe.Element, left, right []float32) {
	icsRight := element.Right
	info := icsRight.Info
	offsets := info.SwbOffsets
	windowGroups := info.GroupCount
	maxSFB := info.MaxSFB
	bandTypes := icsRight.BandTypes
	sectEnd := icsRight.SectEnd
	scaleFactors := icsRight.ScaleFactors

	// groupStep: offset between windows in a group
	groupStep := 128
	if info.WindowSequence != ics.EightShortSequence && info.WindowCount == 1 {
		groupStep = d.Config.FrameLength
	}

	idx := 0
	groupOff := 0
	for g := 0; g < windowGroups; g++ {
		for i := 0; i < maxSFB; {
			end := sectEnd[idx]
			if bandTypes[idx] == ics.IntensityBT || bandTypes[idx] == ics.IntensityBT2 {
				for ; i < end; i, idx = i+1, idx+1 {
					c := float32(1)
					if bandTypes[idx] == ics.IntensityBT2 {
						c = -1
					}
					if element.MaskPresent {
						if element.MSUsed[idx] {
							c = -c
						}
					}
					scale := c * scaleFactors[idx]
					for w := 0; w < info.GroupLength[g]; w++ {
						off := groupOff + w*groupStep + offsets[i]
						length := offsets[i+1] - offsets[i]
						for j := 0; j < length; j++ {
							right[off+j] = left[off+j] * scale
						}
					}
				}
			} else {
				idx += end - i
				i = end
			}
		}
		groupOff += info.GroupLength[g] * groupStep
	}
}

func (d *Decoder) processMS(element *cpe.Element, left, right []float32) {
	icsLeft := element.Left
	info := icsLeft.Info
	offsets := info.SwbOffsets
	windowGroups := info.GroupCount
	maxSFB := info.MaxSFB
	sfbCBl := icsLeft.BandTypes
	sfbCBr := element.Right.BandTypes

	// groupStep: offset between windows in a group
	groupStep := 128
	if info.WindowSequence != ics.EightShortSequence && info.WindowCount == 1 {
		groupStep = d.Config.FrameLength
	}

	groupOff := 0
	idx := 0
	for g := 0; g < windowGroups; g++ {
		for i := 0; i < maxSFB; i, idx = i+1, idx+1 {
			if element.MSUsed[idx] && sfbCBl[idx] < ics.NoiseBT && sfbCBr[idx] < ics.NoiseBT {
				for w := 0; w < info.GroupLength[g]; w++ {
					off := groupOff + w*groupStep + offsets[i]
					for j := 0; j < offsets[i+1]-offsets[i]; j++ {
						t := left[off+j] - right[off+j]
						left[off+j] += right[off+j]
						right[off+j] = t
					}
				}
			}
		}
		groupOff += info.GroupLength[g] * groupStep
	}
}

func (d *Decoder) applyChannelCoupling(elementID int, isChannelPair bool, couplingPoint int, data1, data2 []float32) error {
	applyIndependent := couplingPoint == cce.AfterIMDCT
	for _, element := range d.CCEs {
		if element.CouplingPoint != couplingPoint {
			continue
		}
		index := 0
		for c := 0; c < element.CoupledCount; c++ {
			chSelect := element.ChSelect[c]
			if element.ChannelPair[c] == isChannelPair && element.IDSelect[c] == elementID {
				if chSelect != 1 {
					if applyIndependent {
						if err := element.ApplyIndependentCoupling(index, data1); err != nil {
							return err
						}
					} else {
						if err := element.ApplyDependentCoupling(index, data1); err != nil {
							return err
						}
					}
					if chSelect != 0 {
						index++
					}
				}
				if chSelect != 2 {
					if applyIndependent {
						if err := element.ApplyIndependentCoupling(index, data2); err != nil {
							return err
						}
					} else {
						if err := element.ApplyDependentCoupling(index, data2); err != nil {
							return err
						}
					}
					index++
				}
			} else {
				index += 1
				if chSelect == 3 {
					index++
				}
			}
		}
	}
	return nil
}

func (d *Decoder) setConfigFromADTS(header adts.Header) error {
	config := d.Config
	config.Profile = header.Profile
	config.SampleIndex = header.SamplingIndex
	if config.SampleIndex < 0 || config.SampleIndex >= len(tables.SampleRates) {
		return fmt.Errorf("decoder: invalid sample index %d", config.SampleIndex)
	}
	config.SampleRate = int(tables.SampleRates[config.SampleIndex])
	config.ChanConfig = header.ChannelConfig
	config.FrameLength = 1024

	filterBank, err := filterbank.New(false, config.ChanConfig)
	if err != nil {
		return err
	}
	if config.Profile != aotAACMain && config.Profile != aotAACLC && config.Profile != aotAACLTP {
		return fmt.Errorf("decoder: AAC profile %d not supported", config.Profile)
	}
	if config.ChanConfig == channelConfigNone {
		return fmt.Errorf("decoder: PCE unimplemented")
	}

	d.Config = config
	d.FilterBank = filterBank
	return nil
}

// decodeERFrame decodes an ER (Error Resilient) AAC frame for ELD/LD profiles.
// In ER syntax, the element structure is determined by the channel configuration,
// not by element type IDs in the bitstream.
func (d *Decoder) decodeERFrame(stream *Bitstream) ([]frameElement, error) {
	elements := make([]frameElement, 0, 4)
	config := d.icsConfig()

	switch d.Config.ChanConfig {
	case channelConfigMono:
		// Single Channel Element
		icsStream, err := ics.New(config)
		if err != nil {
			return nil, err
		}
		if err := icsStream.Decode(stream, config, false); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemSCE, id: 0, sce: icsStream})

	case channelConfigStereo:
		// Channel Pair Element
		cpeElem, err := cpe.NewELD(config)
		if err != nil {
			return nil, err
		}
		if err := cpeElem.Decode(stream, config); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemCPE, id: 0, cpe: cpeElem})

	case channelConfigStereoPlusCenter:
		// SCE + CPE
		sce, err := ics.New(config)
		if err != nil {
			return nil, err
		}
		if err := sce.Decode(stream, config, false); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemSCE, id: 0, sce: sce})

		cpeElem, err := cpe.NewELD(config)
		if err != nil {
			return nil, err
		}
		if err := cpeElem.Decode(stream, config); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemCPE, id: 0, cpe: cpeElem})

	case channelConfigStereoPlusRearMono:
		// SCE + CPE + SCE
		sce1, err := ics.New(config)
		if err != nil {
			return nil, err
		}
		if err := sce1.Decode(stream, config, false); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemSCE, id: 0, sce: sce1})

		cpeElem, err := cpe.NewELD(config)
		if err != nil {
			return nil, err
		}
		if err := cpeElem.Decode(stream, config); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemCPE, id: 0, cpe: cpeElem})

		sce2, err := ics.New(config)
		if err != nil {
			return nil, err
		}
		if err := sce2.Decode(stream, config, false); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemSCE, id: 1, sce: sce2})

	case channelConfigFive:
		// SCE + CPE + CPE
		sce, err := ics.New(config)
		if err != nil {
			return nil, err
		}
		if err := sce.Decode(stream, config, false); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemSCE, id: 0, sce: sce})

		cpe1, err := cpe.NewELD(config)
		if err != nil {
			return nil, err
		}
		if err := cpe1.Decode(stream, config); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemCPE, id: 0, cpe: cpe1})

		cpe2, err := cpe.NewELD(config)
		if err != nil {
			return nil, err
		}
		if err := cpe2.Decode(stream, config); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemCPE, id: 1, cpe: cpe2})

	case channelConfigFivePlusOne:
		// SCE + CPE + CPE + LFE
		sce, err := ics.New(config)
		if err != nil {
			return nil, err
		}
		if err := sce.Decode(stream, config, false); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemSCE, id: 0, sce: sce})

		cpe1, err := cpe.NewELD(config)
		if err != nil {
			return nil, err
		}
		if err := cpe1.Decode(stream, config); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemCPE, id: 0, cpe: cpe1})

		cpe2, err := cpe.NewELD(config)
		if err != nil {
			return nil, err
		}
		if err := cpe2.Decode(stream, config); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemCPE, id: 1, cpe: cpe2})

		lfe, err := ics.New(config)
		if err != nil {
			return nil, err
		}
		if err := lfe.Decode(stream, config, false); err != nil {
			return nil, err
		}
		elements = append(elements, frameElement{kind: elemSCE, id: 2, sce: lfe})

	default:
		return nil, fmt.Errorf("decoder: unsupported channel config %d for ELD", d.Config.ChanConfig)
	}

	return elements, nil
}

func (d *Decoder) icsConfig() ics.Config {
	return ics.Config{
		SampleIndex: d.Config.SampleIndex,
		FrameLength: d.Config.FrameLength,
		Profile:     d.Config.Profile,
		IsELD:       d.Config.IsELD,
		IsLD:        d.Config.IsLD,
	}
}

func windowInfo(info *ics.ICSInfo) filterbank.WindowInfo {
	return filterbank.WindowInfo{
		WindowSequence: info.WindowSequence,
		WindowShape:    info.WindowShape,
	}
}

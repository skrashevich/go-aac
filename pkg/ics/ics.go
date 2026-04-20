// Package ics implements the Individual Channel Stream decoding for AAC.
//
// This is a direct port of the ICS module from AAC.js by Devon Govett
// (LGPL v3).
package ics

import (
	"fmt"
	"math"

	"github.com/skrashevich/go-aac/pkg/huffman"
	"github.com/skrashevich/go-aac/pkg/tables"
	"github.com/skrashevich/go-aac/pkg/tns"
)

// BitReader provides bit-level access to the AAC bitstream.
type BitReader interface {
	ReadBits(n int) uint32
}

// Config contains the AAC configuration required by the ICS decoder.
type Config struct {
	SampleIndex int
	FrameLength int
	Profile     int
}

const (
	ZeroBT       = 0
	FirstPairBT  = 5
	EscBT        = 11
	NoiseBT      = 13
	IntensityBT2 = 14
	IntensityBT  = 15
)

const (
	OnlyLongSequence   = 0
	LongStartSequence  = 1
	EightShortSequence = 2
	LongStopSequence   = 3
)

const (
	maxSections         = 120
	maxWindowGroupCount = 8
)

const (
	sfDelta  = 60
	sfOffset = 200
)

// ICStream decodes the AAC Individual Channel Stream.
type ICStream struct {
	Info         *ICSInfo
	BandTypes    []int
	SectEnd      []int
	Data         []float32
	ScaleFactors []float32

	GlobalGain int

	randomState int32
	tns         *tns.TNS
	specBuf     []int

	pulsePresent bool
	pulseOffset  []int
	pulseAmp     []int

	TnsPresent  bool
	GainPresent bool
}

// New creates an ICStream decoder for the given AAC config.
func New(config Config) (*ICStream, error) {
	if config.FrameLength <= 0 {
		return nil, fmt.Errorf("ics: invalid frame length %d", config.FrameLength)
	}
	if config.SampleIndex < 0 || config.SampleIndex >= len(tables.SampleRates) {
		return nil, fmt.Errorf("ics: invalid sample index %d", config.SampleIndex)
	}
	switch config.FrameLength {
	case 1024, 512, 480:
	default:
		return nil, fmt.Errorf("ics: unsupported frame length %d", config.FrameLength)
	}

	decoder := &ICStream{
		Info:         NewInfo(),
		BandTypes:    make([]int, maxSections),
		SectEnd:      make([]int, maxSections),
		Data:         make([]float32, config.FrameLength),
		ScaleFactors: make([]float32, maxSections),
		randomState:  0x1F2E3D4C,
		specBuf:      make([]int, 4),
	}

	var err error
	decoder.tns, err = tns.New(config.SampleIndex, config.FrameLength)
	if err != nil {
		return nil, err
	}

	return decoder, nil
}

// Decode reads the ICS data from the bitstream.
func (ics *ICStream) Decode(stream BitReader, config Config, commonWindow bool) error {
	ics.GlobalGain = int(stream.ReadBits(8))

	if !commonWindow {
		if err := ics.Info.Decode(stream, config, commonWindow); err != nil {
			return err
		}
	}

	if err := ics.decodeBandTypes(stream); err != nil {
		return err
	}
	if err := ics.decodeScaleFactors(stream); err != nil {
		return err
	}

	ics.pulsePresent = stream.ReadBits(1) != 0
	if ics.pulsePresent {
		if ics.Info.WindowSequence == EightShortSequence {
			return fmt.Errorf("ics: pulse tool not allowed in eight short sequence")
		}
		if err := ics.decodePulseData(stream); err != nil {
			return err
		}
	}

	ics.TnsPresent = stream.ReadBits(1) != 0
	if ics.TnsPresent {
		if err := ics.tns.Decode(stream, ics.tnsInfo()); err != nil {
			return err
		}
	}

	ics.GainPresent = stream.ReadBits(1) != 0
	if ics.GainPresent {
		return fmt.Errorf("ics: gain control not implemented")
	}

	if err := ics.decodeSpectralData(stream); err != nil {
		return err
	}

	return nil
}

// ApplyTNS applies temporal noise shaping if present.
func (ics *ICStream) ApplyTNS(data []float32, decode bool) {
	if !ics.TnsPresent {
		return
	}
	ics.tns.Process(ics.tnsInfo(), ics.Info.MaxSFB, data, decode)
}

func (ics *ICStream) decodeBandTypes(stream BitReader) error {
	bits := 5
	if ics.Info.WindowSequence == EightShortSequence {
		bits = 3
	}

	groupCount := ics.Info.GroupCount
	maxSFB := ics.Info.MaxSFB
	idx := 0
	escape := (1 << bits) - 1

	for g := 0; g < groupCount; g++ {
		k := 0
		for k < maxSFB {
			end := k
			bandType := int(stream.ReadBits(4))
			if bandType == 12 {
				return fmt.Errorf("ics: invalid band type 12")
			}

			incr := 0
			for {
				incr = int(stream.ReadBits(bits))
				end += incr
				if incr != escape {
					break
				}
			}

			if end > maxSFB {
				return fmt.Errorf("ics: too many bands (%d > %d)", end, maxSFB)
			}

			for ; k < end; k++ {
				ics.BandTypes[idx] = bandType
				ics.SectEnd[idx] = end
				idx++
			}
		}
	}

	return nil
}

func (ics *ICStream) decodeScaleFactors(stream BitReader) error {
	groupCount := ics.Info.GroupCount
	maxSFB := ics.Info.MaxSFB
	offset := [3]int{ics.GlobalGain, ics.GlobalGain - 90, 0}
	idx := 0
	noiseFlag := true

	for g := 0; g < groupCount; g++ {
		for i := 0; i < maxSFB; {
			runEnd := ics.SectEnd[idx]
			switch ics.BandTypes[idx] {
			case ZeroBT:
				for ; i < runEnd; i, idx = i+1, idx+1 {
					ics.ScaleFactors[idx] = 0
				}
			case IntensityBT, IntensityBT2:
				for ; i < runEnd; i, idx = i+1, idx+1 {
					offset[2] += huffman.DecodeScaleFactor(stream) - sfDelta
					tmp := clampInt(offset[2], -155, 100)
					ics.ScaleFactors[idx] = tables.ScalefactorTable[-tmp+sfOffset]
				}
			case NoiseBT:
				for ; i < runEnd; i, idx = i+1, idx+1 {
					if noiseFlag {
						offset[1] += int(stream.ReadBits(9)) - 256
						noiseFlag = false
					} else {
						offset[1] += huffman.DecodeScaleFactor(stream) - sfDelta
					}
					tmp := clampInt(offset[1], -100, 155)
					ics.ScaleFactors[idx] = -tables.ScalefactorTable[tmp+sfOffset]
				}
			default:
				for ; i < runEnd; i, idx = i+1, idx+1 {
					offset[0] += huffman.DecodeScaleFactor(stream) - sfDelta
					if offset[0] > 255 {
						return fmt.Errorf("ics: scalefactor out of range: %d", offset[0])
					}
					ics.ScaleFactors[idx] = tables.ScalefactorTable[offset[0]-100+sfOffset]
				}
			}
		}
	}

	return nil
}

func (ics *ICStream) decodePulseData(stream BitReader) error {
	pulseCount := int(stream.ReadBits(2)) + 1
	pulseSWB := int(stream.ReadBits(6))

	if pulseSWB >= ics.Info.SwbCount {
		return fmt.Errorf("ics: pulse SWB out of range: %d", pulseSWB)
	}

	if len(ics.pulseOffset) != pulseCount {
		ics.pulseOffset = make([]int, pulseCount)
		ics.pulseAmp = make([]int, pulseCount)
	}

	ics.pulseOffset[0] = ics.Info.SwbOffsets[pulseSWB] + int(stream.ReadBits(5))
	ics.pulseAmp[0] = int(stream.ReadBits(4))

	if ics.pulseOffset[0] > 1023 {
		return fmt.Errorf("ics: pulse offset out of range: %d", ics.pulseOffset[0])
	}

	for i := 1; i < pulseCount; i++ {
		ics.pulseOffset[i] = int(stream.ReadBits(5)) + ics.pulseOffset[i-1]
		if ics.pulseOffset[i] > 1023 {
			return fmt.Errorf("ics: pulse offset out of range: %d", ics.pulseOffset[i])
		}
		ics.pulseAmp[i] = int(stream.ReadBits(4))
	}

	return nil
}

func (ics *ICStream) decodeSpectralData(stream BitReader) error {
	data := ics.Data
	info := ics.Info
	maxSFB := info.MaxSFB
	windowGroups := info.GroupCount
	offsets := info.SwbOffsets
	bandTypes := ics.BandTypes
	scaleFactors := ics.ScaleFactors
	buf := ics.specBuf

	groupOff := 0
	idx := 0
	for g := 0; g < windowGroups; g++ {
		groupLen := info.GroupLength[g]
		for sfb := 0; sfb < maxSFB; sfb, idx = sfb+1, idx+1 {
			hcb := bandTypes[idx]
			off := groupOff + offsets[sfb]
			width := offsets[sfb+1] - offsets[sfb]

			switch hcb {
			case ZeroBT, IntensityBT, IntensityBT2:
				for group := 0; group < groupLen; group++ {
					for i := off; i < off+width; i++ {
						data[i] = 0
					}
					off += 128
				}
			case NoiseBT:
				for group := 0; group < groupLen; group++ {
					energy := 0.0
					for k := 0; k < width; k++ {
						ics.randomState = int32(uint32(ics.randomState) * 1015568748)
						data[off+k] = float32(ics.randomState)
						v := float64(data[off+k])
						energy += v * v
					}
					scale := scaleFactors[idx] / float32(math.Sqrt(energy))
					for k := 0; k < width; k++ {
						data[off+k] *= scale
					}
					off += 128
				}
			default:
				for group := 0; group < groupLen; group++ {
					num := 4
					if hcb >= FirstPairBT {
						num = 2
					}
					for k := 0; k < width; k += num {
						if err := huffman.DecodeSpectralData(stream, hcb, buf, 0); err != nil {
							return err
						}
						for j := 0; j < num; j++ {
							v := buf[j]
							if v >= 0 {
								data[off+k+j] = tables.IQTable[v]
							} else {
								data[off+k+j] = -tables.IQTable[-v]
							}
							data[off+k+j] *= scaleFactors[idx]
						}
					}
					off += 128
				}
			}
		}
		groupOff += groupLen << 7
	}

	if ics.pulsePresent {
		return fmt.Errorf("ics: pulse data not implemented")
	}

	return nil
}

// ICSInfo contains windowing and scalefactor band information.
type ICSInfo struct {
	WindowShape      [2]int
	WindowSequence   int
	GroupLength      []int
	GroupCount       int
	MaxSFB           int
	WindowCount      int
	SwbOffsets       []int
	SwbCount         int
	PredictorPresent bool
	LtpData1Present  bool
	LtpData2Present  bool
}

// NewInfo creates an ICSInfo with default values.
func NewInfo() *ICSInfo {
	info := &ICSInfo{
		WindowSequence: OnlyLongSequence,
		GroupLength:    make([]int, maxWindowGroupCount),
	}
	info.GroupLength[0] = 1
	info.GroupCount = 1
	return info
}

// Decode reads ICSInfo from the bitstream.
func (info *ICSInfo) Decode(stream BitReader, config Config, commonWindow bool) error {
	_ = commonWindow
	_ = stream.ReadBits(1)

	info.WindowSequence = int(stream.ReadBits(2))
	info.WindowShape[0] = info.WindowShape[1]
	info.WindowShape[1] = int(stream.ReadBits(1))

	info.GroupCount = 1
	info.GroupLength[0] = 1

	if info.WindowSequence == EightShortSequence {
		if config.FrameLength != 1024 {
			return fmt.Errorf("ics: short window sequences not supported for frame length %d", config.FrameLength)
		}
		info.MaxSFB = int(stream.ReadBits(4))
		for i := 0; i < 7; i++ {
			if stream.ReadBits(1) != 0 {
				info.GroupLength[info.GroupCount-1]++
			} else {
				info.GroupCount++
				info.GroupLength[info.GroupCount-1] = 1
			}
		}

		info.WindowCount = 8
		info.SwbOffsets = toIntSlice(tables.SWBOffset128[config.SampleIndex])
		info.SwbCount = int(tables.SWBShortWindowCount[config.SampleIndex])
		info.PredictorPresent = false
	} else {
		info.MaxSFB = int(stream.ReadBits(6))
		info.WindowCount = 1
		switch config.FrameLength {
		case 1024:
			info.SwbOffsets = toIntSlice(tables.SWBOffset1024[config.SampleIndex])
			info.SwbCount = int(tables.SWBLongWindowCount[config.SampleIndex])
		case 512:
			info.SwbOffsets = toIntSlice(tables.SWBOffset512[config.SampleIndex])
			info.SwbCount = int(tables.SWBLongWindowCount512[config.SampleIndex])
		case 480:
			info.SwbOffsets = toIntSlice(tables.SWBOffset480[config.SampleIndex])
			info.SwbCount = int(tables.SWBLongWindowCount480[config.SampleIndex])
		default:
			return fmt.Errorf("ics: unsupported frame length %d", config.FrameLength)
		}
		info.PredictorPresent = stream.ReadBits(1) != 0
		if info.PredictorPresent {
			return fmt.Errorf("ics: prediction not implemented")
		}
	}

	return nil
}

func (ics *ICStream) tnsInfo() tns.Info {
	return tns.Info{
		WindowCount:    ics.Info.WindowCount,
		WindowSequence: ics.Info.WindowSequence,
		SwbCount:       ics.Info.SwbCount,
		SwbOffsets:     ics.Info.SwbOffsets,
	}
}

func toIntSlice(input []uint16) []int {
	output := make([]int, len(input))
	for i, v := range input {
		output[i] = int(v)
	}
	return output
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

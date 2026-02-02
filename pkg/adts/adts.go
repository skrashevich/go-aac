// Package adts implements parsing helpers for ADTS headers.
//
// This is a direct port of the ADTS demuxer parsing logic from AAC.js by
// Devon Govett (LGPL v3).
package adts

import (
	"fmt"

	"github.com/skrashevich/go-aac/pkg/tables"
)

// Header contains the parsed ADTS header fields needed by the decoder.
type Header struct {
	Profile          int
	SamplingIndex    int
	ChannelConfig    int
	FrameLength      int
	NumFrames        int
	ProtectionAbsent bool
}

// Probe scans the provided data for an ADTS syncword.
func Probe(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	for i := 0; i+1 < len(data); i++ {
		word := uint16(data[i])<<8 | uint16(data[i+1])
		if (word & 0xfff6) == 0xfff0 {
			return true
		}
	}
	return false
}

// ReadHeader reads an ADTS header from the provided bit reader.
func ReadHeader(reader *BitReader) (Header, error) {
	if reader == nil {
		return Header{}, fmt.Errorf("adts: nil reader")
	}

	sync, err := reader.ReadBits(12)
	if err != nil {
		return Header{}, err
	}
	if sync != 0xfff {
		return Header{}, fmt.Errorf("adts: invalid syncword 0x%x", sync)
	}

	if _, err := reader.ReadBits(3); err != nil { // mpeg version and layer
		return Header{}, err
	}

	protectionAbsent, err := reader.ReadBits(1)
	if err != nil {
		return Header{}, err
	}

	profile, err := reader.ReadBits(2)
	if err != nil {
		return Header{}, err
	}
	samplingIndex, err := reader.ReadBits(4)
	if err != nil {
		return Header{}, err
	}
	if _, err := reader.ReadBits(1); err != nil { // private bit
		return Header{}, err
	}
	chanConfig, err := reader.ReadBits(3)
	if err != nil {
		return Header{}, err
	}
	if _, err := reader.ReadBits(4); err != nil { // original/copy + home + copyright
		return Header{}, err
	}
	frameLength, err := reader.ReadBits(13)
	if err != nil {
		return Header{}, err
	}
	if _, err := reader.ReadBits(11); err != nil { // buffer fullness
		return Header{}, err
	}
	numFrames, err := reader.ReadBits(2)
	if err != nil {
		return Header{}, err
	}

	if protectionAbsent == 0 {
		if _, err := reader.ReadBits(16); err != nil { // CRC
			return Header{}, err
		}
	}

	return Header{
		Profile:          int(profile) + 1,
		SamplingIndex:    int(samplingIndex),
		ChannelConfig:    int(chanConfig),
		FrameLength:      int(frameLength),
		NumFrames:        int(numFrames) + 1,
		ProtectionAbsent: protectionAbsent != 0,
	}, nil
}

// ReadHeaderFromBytes parses an ADTS header from a byte slice.
func ReadHeaderFromBytes(data []byte) (Header, error) {
	reader := NewBitReader(data)
	return ReadHeader(reader)
}

// AudioSpecificConfig returns a 2-byte MPEG-4 AudioSpecificConfig for the header.
func AudioSpecificConfig(header Header) ([2]byte, error) {
	if header.SamplingIndex < 0 || header.SamplingIndex >= len(tables.SampleRates) {
		return [2]byte{}, fmt.Errorf("adts: invalid sampling index %d", header.SamplingIndex)
	}
	if header.ChannelConfig < 0 || header.ChannelConfig > 7 {
		return [2]byte{}, fmt.Errorf("adts: invalid channel config %d", header.ChannelConfig)
	}
	var cookie [2]byte
	cookie[0] = byte(header.Profile<<3) | byte((header.SamplingIndex>>1)&7)
	cookie[1] = byte((header.SamplingIndex&1)<<7) | byte(header.ChannelConfig<<3)
	return cookie, nil
}

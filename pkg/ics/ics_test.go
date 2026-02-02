package ics

import "testing"

type bitReader struct {
	bits []uint8
	pos  int
}

func (b *bitReader) ReadBits(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v <<= 1
		if b.pos < len(b.bits) {
			v |= uint32(b.bits[b.pos])
		}
		b.pos++
	}
	return v
}

func (b *bitReader) Reset() {
	b.pos = 0
}

func bitsFromCode(bitLen int, code uint32) []uint8 {
	bits := make([]uint8, 0, bitLen)
	for i := bitLen - 1; i >= 0; i-- {
		bits = append(bits, uint8((code>>uint(i))&1))
	}
	return bits
}

func TestDecodeBandTypesSimple(t *testing.T) {
	config := Config{SampleIndex: 0, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ics.Info.WindowSequence = OnlyLongSequence
	ics.Info.GroupCount = 1
	ics.Info.MaxSFB = 4

	// bandType=1 (4 bits), incr=4 (5 bits)
	bits := append(bitsFromCode(4, 1), bitsFromCode(5, 4)...)
	br := &bitReader{bits: bits}

	if err := ics.decodeBandTypes(br); err != nil {
		t.Fatalf("decodeBandTypes failed: %v", err)
	}

	for i := 0; i < 4; i++ {
		if ics.BandTypes[i] != 1 {
			t.Fatalf("bandTypes[%d]=%d want 1", i, ics.BandTypes[i])
		}
		if ics.SectEnd[i] != 4 {
			t.Fatalf("sectEnd[%d]=%d want 4", i, ics.SectEnd[i])
		}
	}
}

func TestDecodeScaleFactorsZero(t *testing.T) {
	config := Config{SampleIndex: 0, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ics.Info.GroupCount = 1
	ics.Info.MaxSFB = 2
	ics.BandTypes[0] = ZeroBT
	ics.BandTypes[1] = ZeroBT
	ics.SectEnd[0] = 2
	ics.SectEnd[1] = 2
	ics.GlobalGain = 100

	br := &bitReader{}
	if err := ics.decodeScaleFactors(br); err != nil {
		t.Fatalf("decodeScaleFactors failed: %v", err)
	}

	if ics.ScaleFactors[0] != 0 || ics.ScaleFactors[1] != 0 {
		t.Fatalf("unexpected scalefactors: %v", ics.ScaleFactors[:2])
	}
}

func TestDecodeSpectralDataZero(t *testing.T) {
	config := Config{SampleIndex: 0, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ics.Info.GroupCount = 1
	ics.Info.GroupLength[0] = 1
	ics.Info.MaxSFB = 1
	ics.Info.SwbOffsets = []int{0, 4}
	ics.Info.SwbCount = 1

	ics.BandTypes[0] = ZeroBT
	ics.SectEnd[0] = 1

	for i := 0; i < 8; i++ {
		ics.Data[i] = 1
	}

	br := &bitReader{}
	if err := ics.decodeSpectralData(br); err != nil {
		t.Fatalf("decodeSpectralData failed: %v", err)
	}

	for i := 0; i < 4; i++ {
		if ics.Data[i] != 0 {
			t.Fatalf("data[%d]=%f want 0", i, ics.Data[i])
		}
	}
}

func BenchmarkDecodeBandTypes(b *testing.B) {
	config := Config{SampleIndex: 0, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		b.Fatalf("New failed: %v", err)
	}
	ics.Info.WindowSequence = OnlyLongSequence
	ics.Info.GroupCount = 1
	ics.Info.MaxSFB = 4

	bits := append(bitsFromCode(4, 1), bitsFromCode(5, 4)...)
	br := &bitReader{bits: bits}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Reset()
		_ = ics.decodeBandTypes(br)
	}
}

func BenchmarkDecodeSpectralDataZero(b *testing.B) {
	config := Config{SampleIndex: 0, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		b.Fatalf("New failed: %v", err)
	}

	ics.Info.GroupCount = 1
	ics.Info.GroupLength[0] = 1
	ics.Info.MaxSFB = 1
	ics.Info.SwbOffsets = []int{0, 4}
	ics.Info.SwbCount = 1
	ics.BandTypes[0] = ZeroBT
	ics.SectEnd[0] = 1

	br := &bitReader{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ics.decodeSpectralData(br)
	}
}

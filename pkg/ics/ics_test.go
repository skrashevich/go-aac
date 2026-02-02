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

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{"valid config", Config{SampleIndex: 4, FrameLength: 1024}, false},
		{"invalid frame length", Config{SampleIndex: 4, FrameLength: 0}, true},
		{"invalid frame length negative", Config{SampleIndex: 4, FrameLength: -1}, true},
		{"invalid sample index negative", Config{SampleIndex: -1, FrameLength: 1024}, true},
		{"invalid sample index too large", Config{SampleIndex: 100, FrameLength: 1024}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestICSInfoDecode(t *testing.T) {
	config := Config{SampleIndex: 4, FrameLength: 1024}

	t.Run("long window", func(t *testing.T) {
		info := NewInfo()
		bits := []uint8{}
		bits = append(bits, 0)                         // ics_reserved_bit
		bits = append(bits, 0, 0)                      // window_sequence = 0
		bits = append(bits, 1)                         // window_shape
		bits = append(bits, 0, 0, 1, 0, 1, 0)          // max_sfb = 10
		bits = append(bits, 0)                         // predictor_present = 0

		br := &bitReader{bits: bits}
		err := info.Decode(br, config, false)
		if err != nil {
			t.Fatalf("Decode() failed: %v", err)
		}

		if info.WindowSequence != OnlyLongSequence {
			t.Errorf("WindowSequence = %d, want %d", info.WindowSequence, OnlyLongSequence)
		}
		if info.WindowShape[1] != 1 {
			t.Errorf("WindowShape[1] = %d, want 1", info.WindowShape[1])
		}
		if info.MaxSFB != 10 {
			t.Errorf("MaxSFB = %d, want 10", info.MaxSFB)
		}
		if info.WindowCount != 1 {
			t.Errorf("WindowCount = %d, want 1", info.WindowCount)
		}
	})

	t.Run("eight short sequence", func(t *testing.T) {
		info := NewInfo()
		bits := []uint8{}
		bits = append(bits, 0)                         // ics_reserved_bit
		bits = append(bits, 1, 0)                      // window_sequence = 2
		bits = append(bits, 0)                         // window_shape
		bits = append(bits, 0, 1, 0, 1)                // max_sfb = 5
		// window_grouping: 7 bits
		bits = append(bits, 1, 0, 1, 0, 0, 1, 0)

		br := &bitReader{bits: bits}
		err := info.Decode(br, config, false)
		if err != nil {
			t.Fatalf("Decode() failed: %v", err)
		}

		if info.WindowSequence != EightShortSequence {
			t.Errorf("WindowSequence = %d, want %d", info.WindowSequence, EightShortSequence)
		}
		if info.WindowCount != 8 {
			t.Errorf("WindowCount = %d, want 8", info.WindowCount)
		}
		if info.MaxSFB != 5 {
			t.Errorf("MaxSFB = %d, want 5", info.MaxSFB)
		}
	})
}

func TestDecodeBandTypesInvalid(t *testing.T) {
	config := Config{SampleIndex: 4, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ics.Info.WindowSequence = OnlyLongSequence
	ics.Info.GroupCount = 1
	ics.Info.MaxSFB = 4

	t.Run("invalid band type 12", func(t *testing.T) {
		bits := append(bitsFromCode(4, 12), bitsFromCode(5, 4)...)
		br := &bitReader{bits: bits}
		err := ics.decodeBandTypes(br)
		if err == nil {
			t.Error("expected error for band type 12")
		}
	})

	t.Run("too many bands", func(t *testing.T) {
		bits := append(bitsFromCode(4, 1), bitsFromCode(5, 10)...)
		br := &bitReader{bits: bits}
		err := ics.decodeBandTypes(br)
		if err == nil {
			t.Error("expected error for too many bands")
		}
	})
}

func TestDecodeScaleFactorsIntensity(t *testing.T) {
	config := Config{SampleIndex: 4, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ics.Info.GroupCount = 1
	ics.Info.MaxSFB = 2
	ics.BandTypes[0] = IntensityBT
	ics.BandTypes[1] = IntensityBT
	ics.SectEnd[0] = 1
	ics.SectEnd[1] = 2
	ics.GlobalGain = 100

	// Create bits for huffman scale factor (60)
	bits := []uint8{}
	for i := 0; i < 20; i++ {
		bits = append(bits, 1, 1, 1, 1, 1, 1, 1, 0)
	}

	br := &bitReader{bits: bits}
	err = ics.decodeScaleFactors(br)
	if err != nil {
		t.Fatalf("decodeScaleFactors failed: %v", err)
	}
}

func TestDecodeScaleFactorsNoise(t *testing.T) {
	config := Config{SampleIndex: 4, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ics.Info.GroupCount = 1
	ics.Info.MaxSFB = 2
	ics.BandTypes[0] = NoiseBT
	ics.BandTypes[1] = NoiseBT
	ics.SectEnd[0] = 1
	ics.SectEnd[1] = 2
	ics.GlobalGain = 100

	bits := []uint8{}
	// First noise: 9 bits = 100
	bits = append(bits, bitsFromCode(9, 100)...)
	// Second noise: huffman scale factor
	for i := 0; i < 10; i++ {
		bits = append(bits, 1, 1, 1, 1, 1, 1, 1, 0)
	}

	br := &bitReader{bits: bits}
	err = ics.decodeScaleFactors(br)
	if err != nil {
		t.Fatalf("decodeScaleFactors failed: %v", err)
	}
}

func TestDecodePulseData(t *testing.T) {
	config := Config{SampleIndex: 4, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ics.Info.SwbCount = 50
	ics.Info.SwbOffsets = make([]int, 51)
	for i := range ics.Info.SwbOffsets {
		ics.Info.SwbOffsets[i] = i * 20
	}

	t.Run("valid pulse data", func(t *testing.T) {
		bits := []uint8{}
		bits = append(bits, bitsFromCode(2, 1)...)     // pulse_count = 2
		bits = append(bits, bitsFromCode(6, 10)...)    // pulse_start_sfb = 10
		bits = append(bits, bitsFromCode(5, 5)...)     // pulse_offset[0] = 5
		bits = append(bits, bitsFromCode(4, 3)...)     // pulse_amp[0] = 3
		bits = append(bits, bitsFromCode(5, 7)...)     // pulse_offset[1] = 7
		bits = append(bits, bitsFromCode(4, 5)...)     // pulse_amp[1] = 5

		br := &bitReader{bits: bits}
		err := ics.decodePulseData(br)
		if err != nil {
			t.Fatalf("decodePulseData failed: %v", err)
		}

		if len(ics.pulseOffset) != 2 {
			t.Errorf("len(pulseOffset) = %d, want 2", len(ics.pulseOffset))
		}
		if ics.pulseAmp[0] != 3 {
			t.Errorf("pulseAmp[0] = %d, want 3", ics.pulseAmp[0])
		}
	})

	t.Run("pulse SWB out of range", func(t *testing.T) {
		bits := []uint8{}
		bits = append(bits, bitsFromCode(2, 0)...)     // pulse_count = 1
		bits = append(bits, bitsFromCode(6, 60)...)    // pulse_start_sfb = 60 (too large)

		br := &bitReader{bits: bits}
		err := ics.decodePulseData(br)
		if err == nil {
			t.Error("expected error for pulse SWB out of range")
		}
	})

	t.Run("pulse offset out of range", func(t *testing.T) {
		bits := []uint8{}
		bits = append(bits, bitsFromCode(2, 0)...)     // pulse_count = 1
		bits = append(bits, bitsFromCode(6, 0)...)     // pulse_start_sfb = 0
		bits = append(bits, bitsFromCode(5, 31)...)    // pulse_offset = 31 (will be > 1023)
		bits = append(bits, bitsFromCode(4, 3)...)     // pulse_amp = 3

		ics.Info.SwbOffsets[0] = 1000
		br := &bitReader{bits: bits}
		err := ics.decodePulseData(br)
		if err == nil {
			t.Error("expected error for pulse offset out of range")
		}
	})
}

func TestDecodeSpectralDataNoise(t *testing.T) {
	config := Config{SampleIndex: 4, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ics.Info.GroupCount = 1
	ics.Info.GroupLength[0] = 1
	ics.Info.MaxSFB = 1
	ics.Info.SwbOffsets = []int{0, 8}
	ics.Info.SwbCount = 1

	ics.BandTypes[0] = NoiseBT
	ics.SectEnd[0] = 1
	ics.ScaleFactors[0] = 1.0

	br := &bitReader{}
	err = ics.decodeSpectralData(br)
	if err != nil {
		t.Fatalf("decodeSpectralData failed: %v", err)
	}

	// Check that data is not all zeros
	allZero := true
	for i := 0; i < 8; i++ {
		if ics.Data[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("expected non-zero noise data")
	}
}

func TestDecodeSpectralDataIntensity(t *testing.T) {
	config := Config{SampleIndex: 4, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ics.Info.GroupCount = 1
	ics.Info.GroupLength[0] = 1
	ics.Info.MaxSFB = 1
	ics.Info.SwbOffsets = []int{0, 4}
	ics.Info.SwbCount = 1

	ics.BandTypes[0] = IntensityBT2
	ics.SectEnd[0] = 1

	for i := 0; i < 8; i++ {
		ics.Data[i] = 1
	}

	br := &bitReader{}
	err = ics.decodeSpectralData(br)
	if err != nil {
		t.Fatalf("decodeSpectralData failed: %v", err)
	}

	for i := 0; i < 4; i++ {
		if ics.Data[i] != 0 {
			t.Errorf("data[%d] = %f, want 0", i, ics.Data[i])
		}
	}
}

func TestApplyTNS(t *testing.T) {
	config := Config{SampleIndex: 4, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	data := make([]float32, 1024)
	for i := range data {
		data[i] = 1.0
	}

	t.Run("TNS not present", func(t *testing.T) {
		ics.TnsPresent = false
		ics.ApplyTNS(data, true)
		// Should do nothing
		if data[0] != 1.0 {
			t.Error("data should not be modified when TNS not present")
		}
	})

	t.Run("TNS present", func(t *testing.T) {
		ics.TnsPresent = true
		ics.Info.WindowCount = 1
		ics.Info.WindowSequence = 0
		ics.Info.SwbCount = 10
		ics.Info.SwbOffsets = make([]int, 11)
		for i := range ics.Info.SwbOffsets {
			ics.Info.SwbOffsets[i] = i * 10
		}
		ics.Info.MaxSFB = 10

		ics.ApplyTNS(data, true)
		// Just ensure it doesn't panic
	})
}

func TestClampInt(t *testing.T) {
	tests := []struct {
		v, min, max, want int
	}{
		{5, 0, 10, 5},
		{-5, 0, 10, 0},
		{15, 0, 10, 10},
		{5, 5, 5, 5},
	}

	for _, tt := range tests {
		got := clampInt(tt.v, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("clampInt(%d, %d, %d) = %d, want %d", tt.v, tt.min, tt.max, got, tt.want)
		}
	}
}

func TestToIntSlice(t *testing.T) {
	input := []uint16{1, 2, 3, 4, 5}
	expected := []int{1, 2, 3, 4, 5}
	result := toIntSlice(input)

	if len(result) != len(expected) {
		t.Errorf("len(result) = %d, want %d", len(result), len(expected))
	}

	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("result[%d] = %d, want %d", i, result[i], expected[i])
		}
	}
}

func BenchmarkDecodeScaleFactors(b *testing.B) {
	config := Config{SampleIndex: 4, FrameLength: 1024}
	ics, err := New(config)
	if err != nil {
		b.Fatalf("New failed: %v", err)
	}

	ics.Info.GroupCount = 1
	ics.Info.MaxSFB = 2
	ics.BandTypes[0] = ZeroBT
	ics.BandTypes[1] = ZeroBT
	ics.SectEnd[0] = 2
	ics.SectEnd[1] = 2
	ics.GlobalGain = 100

	br := &bitReader{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ics.decodeScaleFactors(br)
	}
}

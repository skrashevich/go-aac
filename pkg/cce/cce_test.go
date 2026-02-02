package cce

import (
	"testing"

	"github.com/skrashevich/go-aac/pkg/ics"
)

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

func TestDecodeMinimal(t *testing.T) {
	config := testConfig()
	e, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// All-zero bitstream should decode without error.
	br := &bitReader{}
	if err := e.Decode(br, config); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if e.CoupledCount != 0 {
		t.Fatalf("CoupledCount=%d want 0", e.CoupledCount)
	}
	if e.CouplingPoint != 0 {
		t.Fatalf("CouplingPoint=%d want 0", e.CouplingPoint)
	}
}

func TestApplyIndependentCoupling(t *testing.T) {
	config := testConfig()
	e, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	data := make([]float32, 4)
	e.ICS.Data = []float32{1, 2, 3, 4}
	e.Gain = [][]float32{make([]float32, 1)}
	e.Gain[0][0] = 2

	if err := e.ApplyIndependentCoupling(0, data); err != nil {
		t.Fatalf("ApplyIndependentCoupling failed: %v", err)
	}
	if data[0] != 2 || data[1] != 4 || data[2] != 6 || data[3] != 8 {
		t.Fatalf("unexpected data: %v", data)
	}
}

func TestApplyDependentCoupling(t *testing.T) {
	config := testConfig()
	e, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	info := e.ICS.Info
	info.GroupCount = 1
	info.GroupLength[0] = 1
	info.MaxSFB = 1
	info.SwbOffsets = []int{0, 2}
	info.SwbCount = 1

	e.ICS.BandTypes[0] = 1
	e.ICS.Data = []float32{1, 2}
	data := []float32{10, 20}

	e.Gain = [][]float32{make([]float32, maxGainBands)}
	e.Gain[0][0] = 0.5

	if err := e.ApplyDependentCoupling(0, data); err != nil {
		t.Fatalf("ApplyDependentCoupling failed: %v", err)
	}
	if data[0] != 10.5 || data[1] != 21 {
		t.Fatalf("unexpected data: %v", data)
	}
}

func BenchmarkDecodeMinimal(b *testing.B) {
	config := testConfig()
	e, err := New(config)
	if err != nil {
		b.Fatalf("New failed: %v", err)
	}
	br := &bitReader{}
	for i := 0; i < b.N; i++ {
		br.Reset()
		_ = e.Decode(br, config)
	}
}

func testConfig() ics.Config {
	return ics.Config{SampleIndex: 0, FrameLength: 1024, Profile: 0}
}

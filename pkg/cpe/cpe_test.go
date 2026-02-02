package cpe

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

func TestDecodeNoCommonWindow(t *testing.T) {
	config := testConfig()
	e, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	br := &bitReader{}
	if err := e.Decode(br, config); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if e.CommonWindow {
		t.Fatal("expected CommonWindow=false")
	}
	for i := 0; i < len(e.MSUsed); i++ {
		if e.MSUsed[i] {
			t.Fatalf("msUsed[%d] = true, want false", i)
		}
	}
}

func TestDecodeCommonWindowAllOnes(t *testing.T) {
	config := testConfig()
	e, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	// commonWindow=1, reserved=0, windowSequence=0, windowShape=0,
	// maxSFB=0, predictor=0, mask=2 (all ones)
	bits := []uint8{
		1,
		0, 0, 0, 0,
		0, 0, 0, 0, 0, 0,
		0,
		1, 0,
	}
	br := &bitReader{bits: bits}
	if err := e.Decode(br, config); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if !e.CommonWindow {
		t.Fatal("expected CommonWindow=true")
	}
	if !e.MaskPresent {
		t.Fatal("expected MaskPresent=true")
	}
	for i := 0; i < len(e.MSUsed); i++ {
		if !e.MSUsed[i] {
			t.Fatalf("msUsed[%d] = false, want true", i)
		}
	}
}

func TestDecodeReservedMask(t *testing.T) {
	config := testConfig()
	e, err := New(config)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	// commonWindow=1, reserved=0, windowSequence=0, windowShape=0,
	// maxSFB=0, predictor=0, mask=3 (reserved)
	bits := []uint8{
		1,
		0, 0, 0, 0,
		0, 0, 0, 0, 0, 0,
		0,
		1, 1,
	}
	br := &bitReader{bits: bits}
	if err := e.Decode(br, config); err == nil {
		t.Fatal("expected error for reserved mask type")
	}
}

func BenchmarkDecodeNoCommonWindow(b *testing.B) {
	config := testConfig()
	e, err := New(config)
	if err != nil {
		b.Fatalf("New failed: %v", err)
	}
	br := &bitReader{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Reset()
		_ = e.Decode(br, config)
	}
}

func BenchmarkDecodeCommonWindow(b *testing.B) {
	config := testConfig()
	e, err := New(config)
	if err != nil {
		b.Fatalf("New failed: %v", err)
	}
	bits := []uint8{
		1,
		0, 0, 0, 0,
		0, 0, 0, 0, 0, 0,
		0,
		1, 0,
	}
	br := &bitReader{bits: bits}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		br.Reset()
		_ = e.Decode(br, config)
	}
}

func testConfig() ics.Config {
	return ics.Config{SampleIndex: 0, FrameLength: 1024, Profile: 0}
}

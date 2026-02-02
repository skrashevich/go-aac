package huffman

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

func readerFromCode(bitLen int, code uint32, extra []uint8) *bitReader {
	bits := make([]uint8, 0, bitLen+len(extra))
	for i := bitLen - 1; i >= 0; i-- {
		bits = append(bits, uint8((code>>uint(i))&1))
	}
	bits = append(bits, extra...)
	return &bitReader{bits: bits}
}

func TestDecodeScaleFactor(t *testing.T) {
	entries := []sfEntry{hcbSF[0], hcbSF[5], hcbSF[10], hcbSF[len(hcbSF)-1]}
	for _, entry := range entries {
		br := readerFromCode(int(entry.bits), entry.code, nil)
		got := DecodeScaleFactor(br)
		if got != int(entry.value) {
			t.Fatalf("DecodeScaleFactor bits=%d code=%d: got %d want %d", entry.bits, entry.code, got, entry.value)
		}
	}
}

func TestDecodeSpectralData_Quad(t *testing.T) {
	br := readerFromCode(1, 0, nil)
	data := make([]int, 4)
	if err := DecodeSpectralData(br, 1, data, 0); err != nil {
		t.Fatalf("DecodeSpectralData: %v", err)
	}
	if data[0] != 0 || data[1] != 0 || data[2] != 0 || data[3] != 0 {
		t.Fatalf("unexpected quad data: %+v", data)
	}
}

func TestDecodeSpectralData_UnsignedSign(t *testing.T) {
	// HCB3 entry [4, 8, 1, 0, 0, 0] with a sign bit set to make it negative.
	br := readerFromCode(4, 8, []uint8{1})
	data := make([]int, 4)
	if err := DecodeSpectralData(br, 3, data, 0); err != nil {
		t.Fatalf("DecodeSpectralData: %v", err)
	}
	if data[0] != -1 {
		t.Fatalf("expected signed value -1, got %d", data[0])
	}
}

func TestDecodeSpectralData_Escape(t *testing.T) {
	// HCB11 entry [5, 4, 16, 16] with positive sign bits and minimal escape.
	// Escape sequence: stop bit 0 then 4 zero bits (j=16).
	extra := []uint8{
		0, 0, // sign bits for two values
		0, 0, 0, 0, 0, // escape for first value
		0, 0, 0, 0, 0, // escape for second value
	}
	br := readerFromCode(5, 4, extra)
	data := make([]int, 2)
	if err := DecodeSpectralData(br, 11, data, 0); err != nil {
		t.Fatalf("DecodeSpectralData: %v", err)
	}
	if data[0] != 16 || data[1] != 16 {
		t.Fatalf("unexpected escape values: %+v", data)
	}
}

func TestDecodeSpectralData_InvalidCodebook(t *testing.T) {
	br := readerFromCode(1, 0, nil)
	data := make([]int, 2)
	if err := DecodeSpectralData(br, 12, data, 0); err == nil {
		t.Fatal("expected error for invalid codebook")
	}
}

func BenchmarkDecodeScaleFactor(b *testing.B) {
	entry := hcbSF[10]
	br := readerFromCode(int(entry.bits), entry.code, nil)
	for i := 0; i < b.N; i++ {
		br.Reset()
		_ = DecodeScaleFactor(br)
	}
}

func BenchmarkDecodeSpectralDataQuad(b *testing.B) {
	br := readerFromCode(1, 0, nil)
	data := make([]int, 4)
	for i := 0; i < b.N; i++ {
		br.Reset()
		_ = DecodeSpectralData(br, 1, data, 0)
	}
}

func BenchmarkDecodeSpectralDataEscape(b *testing.B) {
	extra := []uint8{
		0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	}
	br := readerFromCode(5, 4, extra)
	data := make([]int, 2)
	for i := 0; i < b.N; i++ {
		br.Reset()
		_ = DecodeSpectralData(br, 11, data, 0)
	}
}

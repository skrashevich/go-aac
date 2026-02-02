package adts

import "testing"

func TestProbe(t *testing.T) {
	data := []byte{0x00, 0x11, 0x22, 0xff, 0xf1, 0x33}
	if !Probe(data) {
		t.Fatal("expected Probe to find syncword")
	}
	if Probe([]byte{0x00, 0x01, 0x02}) {
		t.Fatal("expected Probe to return false")
	}
}

func TestReadHeaderFromBytes(t *testing.T) {
	bits := newBitWriter()
	bits.write(12, 0xfff)
	bits.write(1, 0) // mpeg id
	bits.write(2, 0) // layer
	bits.write(1, 1) // protection absent
	bits.write(2, 1) // profile (AAC-LC => 2)
	bits.write(4, 4) // sampling index
	bits.write(1, 0) // private
	bits.write(3, 2) // channel config
	bits.write(4, 0) // original/copy + home + copyright
	bits.write(13, 100)
	bits.write(11, 0)
	bits.write(2, 0) // numFrames-1

	data := bits.bytes()
	header, err := ReadHeaderFromBytes(data)
	if err != nil {
		t.Fatalf("ReadHeaderFromBytes failed: %v", err)
	}
	if header.Profile != 2 {
		t.Fatalf("Profile=%d want 2", header.Profile)
	}
	if header.SamplingIndex != 4 {
		t.Fatalf("SamplingIndex=%d want 4", header.SamplingIndex)
	}
	if header.ChannelConfig != 2 {
		t.Fatalf("ChannelConfig=%d want 2", header.ChannelConfig)
	}
	if header.FrameLength != 100 {
		t.Fatalf("FrameLength=%d want 100", header.FrameLength)
	}
	if header.NumFrames != 1 {
		t.Fatalf("NumFrames=%d want 1", header.NumFrames)
	}
	if !header.ProtectionAbsent {
		t.Fatalf("ProtectionAbsent=false want true")
	}
}

func TestReadHeaderInvalidSync(t *testing.T) {
	reader := NewBitReader([]byte{0x00, 0x00})
	if _, err := ReadHeader(reader); err == nil {
		t.Fatal("expected error for invalid syncword")
	}
}

func TestAudioSpecificConfig(t *testing.T) {
	header := Header{Profile: 2, SamplingIndex: 4, ChannelConfig: 2}
	asc, err := AudioSpecificConfig(header)
	if err != nil {
		t.Fatalf("AudioSpecificConfig failed: %v", err)
	}
	if asc[0] != 0x12 || asc[1] != 0x10 {
		t.Fatalf("asc=%#02x %#02x want 0x12 0x10", asc[0], asc[1])
	}
}

func BenchmarkReadHeader(b *testing.B) {
	bits := newBitWriter()
	bits.write(12, 0xfff)
	bits.write(1, 0)
	bits.write(2, 0)
	bits.write(1, 1)
	bits.write(2, 1)
	bits.write(4, 4)
	bits.write(1, 0)
	bits.write(3, 2)
	bits.write(4, 0)
	bits.write(13, 100)
	bits.write(11, 0)
	bits.write(2, 0)
	data := bits.bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := NewBitReader(data)
		_, _ = ReadHeader(reader)
	}
}

func BenchmarkProbe(b *testing.B) {
	data := []byte{0x00, 0x11, 0x22, 0xff, 0xf1, 0x33, 0x44, 0x55}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Probe(data)
	}
}

func BenchmarkReadHeaderFromBytes(b *testing.B) {
	bits := newBitWriter()
	bits.write(12, 0xfff)
	bits.write(1, 0)
	bits.write(2, 0)
	bits.write(1, 1)
	bits.write(2, 1)
	bits.write(4, 4)
	bits.write(1, 0)
	bits.write(3, 2)
	bits.write(4, 0)
	bits.write(13, 100)
	bits.write(11, 0)
	bits.write(2, 0)
	data := bits.bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ReadHeaderFromBytes(data)
	}
}

func BenchmarkAudioSpecificConfig(b *testing.B) {
	header := Header{Profile: 2, SamplingIndex: 4, ChannelConfig: 2}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = AudioSpecificConfig(header)
	}
}

type bitWriter struct {
	bits []uint8
}

func newBitWriter() *bitWriter {
	return &bitWriter{bits: make([]uint8, 0, 64)}
}

func (w *bitWriter) write(n int, value uint32) {
	for i := n - 1; i >= 0; i-- {
		bit := (value >> uint(i)) & 1
		w.bits = append(w.bits, uint8(bit))
	}
}

func (w *bitWriter) bytes() []byte {
	length := (len(w.bits) + 7) / 8
	out := make([]byte, length)
	for i, bit := range w.bits {
		if bit != 0 {
			byteIndex := i / 8
			bitIndex := 7 - (i % 8)
			out[byteIndex] |= 1 << uint(bitIndex)
		}
	}
	return out
}

package decoder

import "fmt"

// Bitstream provides bit-level access over a byte slice.
type Bitstream struct {
	data   []byte
	bitPos int
	err    error
}

// NewBitstream creates a new bitstream over data.
func NewBitstream(data []byte) *Bitstream {
	return &Bitstream{data: data}
}

// ReadBits reads n bits in MSB-first order.
func (b *Bitstream) ReadBits(n int) uint32 {
	if b.err != nil {
		return 0
	}
	if n < 0 || n > 32 {
		b.err = fmt.Errorf("bitstream: invalid bit count %d", n)
		return 0
	}
	if b.bitPos+n > len(b.data)*8 {
		b.err = fmt.Errorf("bitstream: insufficient data")
		return 0
	}

	var v uint32
	for i := 0; i < n; i++ {
		byteIndex := b.bitPos / 8
		bitIndex := 7 - (b.bitPos % 8)
		bit := (b.data[byteIndex] >> uint(bitIndex)) & 1
		v = (v << 1) | uint32(bit)
		b.bitPos++
	}
	return v
}

// PeekBits reads n bits without advancing.
func (b *Bitstream) PeekBits(n int) uint32 {
	if b.err != nil {
		return 0
	}
	if n < 0 || n > 32 {
		b.err = fmt.Errorf("bitstream: invalid bit count %d", n)
		return 0
	}
	if b.bitPos+n > len(b.data)*8 {
		return 0
	}
	pos := b.bitPos
	v := b.ReadBits(n)
	b.bitPos = pos
	b.err = nil
	return v
}

// Advance moves the bit position forward.
func (b *Bitstream) Advance(bits int) {
	if b.err != nil {
		return
	}
	if bits < 0 || b.bitPos+bits > len(b.data)*8 {
		b.err = fmt.Errorf("bitstream: advance out of range")
		return
	}
	b.bitPos += bits
}

// Align advances to the next byte boundary.
func (b *Bitstream) Align() {
	if b.err != nil {
		return
	}
	mod := b.bitPos % 8
	if mod != 0 {
		b.bitPos += 8 - mod
	}
}

// Error returns any read error encountered.
func (b *Bitstream) Error() error {
	return b.err
}

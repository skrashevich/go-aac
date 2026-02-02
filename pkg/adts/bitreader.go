package adts

import "fmt"

// BitReader provides bit-level reads over a byte slice.
type BitReader struct {
	data   []byte
	bitPos int
}

// NewBitReader creates a new bit reader over data.
func NewBitReader(data []byte) *BitReader {
	return &BitReader{data: data}
}

// ReadBits reads n bits in MSB-first order.
func (r *BitReader) ReadBits(n int) (uint32, error) {
	if n < 0 || n > 32 {
		return 0, fmt.Errorf("adts: invalid bit count %d", n)
	}
	if r.bitPos+n > len(r.data)*8 {
		return 0, fmt.Errorf("adts: insufficient data")
	}

	var v uint32
	for i := 0; i < n; i++ {
		byteIndex := r.bitPos / 8
		bitIndex := 7 - (r.bitPos % 8)
		bit := (r.data[byteIndex] >> uint(bitIndex)) & 1
		v = (v << 1) | uint32(bit)
		r.bitPos++
	}
	return v, nil
}

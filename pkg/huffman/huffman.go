// Package huffman implements AAC Huffman decoding for spectral data and
// scalefactors.
//
// This is a direct port of the Huffman module from AAC.js by Devon Govett
// (LGPL v3).
package huffman

import "fmt"

// BitReader provides bit-level access to the AAC bitstream.
//
// ReadBits should return the next n bits in MSB-first order.
type BitReader interface {
	ReadBits(n int) uint32
}

type hcbEntry struct {
	bits   uint8
	code   uint32
	values [4]int16
}

type sfEntry struct {
	bits  uint8
	code  uint32
	value int16
}

const (
	quadLen = 4
	pairLen = 2
)

var unsigned = []bool{false, false, true, true, false, false, true, true, true, true, true}

var codebooks = [][]hcbEntry{hcb1, hcb2, hcb3, hcb4, hcb5, hcb6, hcb7, hcb8, hcb9, hcb10, hcb11}

// DecodeScaleFactor reads a Huffman coded scalefactor value from the bitstream.
func DecodeScaleFactor(stream BitReader) int {
	offset := findOffsetSF(stream, hcbSF)
	return int(hcbSF[offset].value)
}

// DecodeSpectralData decodes Huffman coded spectral coefficients for the
// specified codebook and stores the result into data at the given offset.
//
// For codebooks 1-4, four values are decoded (quad). For codebooks 5-11, two
// values are decoded (pair). For unsigned codebooks, additional sign bits are
// read from the stream. Codebook 11 supports escape sequences for values of
// magnitude 16.
func DecodeSpectralData(stream BitReader, cb int, data []int, off int) error {
	if cb < 1 || cb > len(codebooks) {
		return fmt.Errorf("huffman: unknown spectral codebook: %d", cb)
	}

	need := pairLen
	if cb < 5 {
		need = quadLen
	}
	if off < 0 || off+need > len(data) {
		return fmt.Errorf("huffman: data slice too small for codebook %d at offset %d", cb, off)
	}

	HCB := codebooks[cb-1]
	offset := findOffset(stream, HCB)

	data[off] = int(HCB[offset].values[0])
	data[off+1] = int(HCB[offset].values[1])
	if cb < 5 {
		data[off+2] = int(HCB[offset].values[2])
		data[off+3] = int(HCB[offset].values[3])
	}

	// sign and escape handling
	if cb < 11 {
		if unsigned[cb-1] {
			signValues(stream, data, off, need)
		}
		return nil
	}

	// Codebook 11 (escape) uses sign bits and escape sequences.
	signValues(stream, data, off, need)

	if absInt(data[off]) == 16 {
		data[off] = getEscape(stream, data[off])
	}
	if absInt(data[off+1]) == 16 {
		data[off+1] = getEscape(stream, data[off+1])
	}

	return nil
}

func findOffset(stream BitReader, table []hcbEntry) int {
	off := 0
	length := int(table[off].bits)
	cw := stream.ReadBits(length)

	for cw != table[off].code {
		off++
		nextLen := int(table[off].bits)
		shift := nextLen - length
		if shift > 0 {
			cw = (cw << uint(shift)) | stream.ReadBits(shift)
		}
		length = nextLen
	}

	return off
}

func findOffsetSF(stream BitReader, table []sfEntry) int {
	off := 0
	length := int(table[off].bits)
	cw := stream.ReadBits(length)

	for cw != table[off].code {
		off++
		nextLen := int(table[off].bits)
		shift := nextLen - length
		if shift > 0 {
			cw = (cw << uint(shift)) | stream.ReadBits(shift)
		}
		length = nextLen
	}

	return off
}

func signValues(stream BitReader, data []int, off int, length int) {
	for i := off; i < off+length; i++ {
		if data[i] != 0 && stream.ReadBits(1) != 0 {
			data[i] = -data[i]
		}
	}
}

func getEscape(stream BitReader, s int) int {
	i := 4
	for stream.ReadBits(1) != 0 {
		i++
	}

	j := int(stream.ReadBits(i)) | (1 << i)
	if s < 0 {
		return -j
	}
	return j
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

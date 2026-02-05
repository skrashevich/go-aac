// Package tns implements Temporal Noise Shaping for AAC decoding.
//
// This is a direct port of the TNS module from AAC.js by Devon Govett
// (LGPL v3).
package tns

import "fmt"

// BitReader provides bit-level access to the AAC bitstream.
type BitReader interface {
	ReadBits(n int) uint32
}

// Info contains the windowing information required by TNS.
type Info struct {
	WindowCount    int
	WindowSequence int
	SwbCount       int
	SwbOffsets     []int
}

const (
	tnsMaxOrder = 20
)

var (
	shortBits = [3]int{1, 4, 3}
	longBits  = [3]int{2, 6, 5}
)

var (
	tnsCoef13 = []float32{0.00000000, -0.43388373, 0.64278758, 0.34202015}

	tnsCoef03 = []float32{
		0.00000000, -0.43388373, -0.78183150, -0.97492790,
		0.98480773, 0.86602539, 0.64278758, 0.34202015,
	}

	tnsCoef14 = []float32{
		0.00000000, -0.20791170, -0.40673664, -0.58778524,
		0.67369562, 0.52643216, 0.36124167, 0.18374951,
	}

	tnsCoef04 = []float32{
		0.00000000, -0.20791170, -0.40673664, -0.58778524,
		-0.74314481, -0.86602539, -0.95105654, -0.99452192,
		0.99573416, 0.96182561, 0.89516330, 0.79801720,
		0.67369562, 0.52643216, 0.36124167, 0.18374951,
	}
)

var tnsTables = [][]float32{tnsCoef03, tnsCoef04, tnsCoef13, tnsCoef14}

var (
	tnsMaxBands1024 = []int{31, 31, 34, 40, 42, 51, 46, 46, 42, 42, 42, 39, 39}
	tnsMaxBands128  = []int{9, 9, 10, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14}
)

// TNS stores decoded filter parameters and implements TNS filtering.
type TNS struct {
	maxBands  int
	nFilt     [8]int
	length    [8][4]int
	direction [8][4]bool
	order     [8][4]int
	coef      [8][4][tnsMaxOrder]float32

	lpc [tnsMaxOrder]float32
	tmp [tnsMaxOrder]float32
}

// New creates a TNS processor for the given sample rate index.
func New(sampleIndex int) (*TNS, error) {
	if sampleIndex < 0 || sampleIndex >= len(tnsMaxBands1024) {
		return nil, fmt.Errorf("tns: invalid sample index %d", sampleIndex)
	}
	return &TNS{maxBands: tnsMaxBands1024[sampleIndex]}, nil
}

// NewELD creates a TNS processor for AAC-ELD with the given sample rate index
// and frame length (512 or 480).
func NewELD(sampleIndex int, frameLength int) (*TNS, error) {
	var table []int
	switch frameLength {
	case 512:
		table = tnsMaxBands512
	case 480:
		table = tnsMaxBands480
	default:
		return nil, fmt.Errorf("tns: unsupported ELD frame length %d", frameLength)
	}
	if sampleIndex < 0 || sampleIndex >= len(table) {
		return nil, fmt.Errorf("tns: invalid sample index %d", sampleIndex)
	}
	return &TNS{maxBands: table[sampleIndex]}, nil
}

var (
	tnsMaxBands512 = []int{31, 31, 31, 31, 32, 37, 31, 31, 31, 31, 31, 31, 31}
	tnsMaxBands480 = []int{31, 31, 31, 31, 32, 37, 30, 30, 30, 30, 30, 30, 30}
)

// Decode reads TNS data from the bitstream for the given window info.
func (t *TNS) Decode(stream BitReader, info Info) error {
	bits := longBits
	if info.WindowSequence == 2 {
		bits = shortBits
	}

	for w := 0; w < info.WindowCount; w++ {
		nFilt := int(stream.ReadBits(bits[0]))
		t.nFilt[w] = nFilt
		if nFilt == 0 {
			continue
		}

		coefRes := int(stream.ReadBits(1))

		for filt := 0; filt < nFilt; filt++ {
			t.length[w][filt] = int(stream.ReadBits(bits[1]))
			order := int(stream.ReadBits(bits[2]))
			if order > tnsMaxOrder {
				return fmt.Errorf("tns: filter order out of range: %d", order)
			}
			t.order[w][filt] = order

			if order == 0 {
				continue
			}

			t.direction[w][filt] = stream.ReadBits(1) != 0
			coefCompress := int(stream.ReadBits(1))
			coefLen := coefRes + 3 - coefCompress
			tableIdx := 2*coefCompress + coefRes
			table := tnsTables[tableIdx]

			for i := 0; i < order; i++ {
				idx := int(stream.ReadBits(coefLen))
				if idx < 0 || idx >= len(table) {
					return fmt.Errorf("tns: coef index out of range: %d", idx)
				}
				t.coef[w][filt][i] = table[idx]
			}
		}
	}

	return nil
}

// Process applies TNS filtering to the spectral data.
func (t *TNS) Process(info Info, maxSFB int, data []float32, decode bool) {
	mmm := maxSFB
	if t.maxBands < mmm {
		mmm = t.maxBands
	}

	for w := 0; w < info.WindowCount; w++ {
		bottom := info.SwbCount
		nFilt := t.nFilt[w]

		for filt := 0; filt < nFilt; filt++ {
			top := bottom
			bottom = top - t.length[w][filt]
			if bottom < 0 {
				bottom = 0
			}

			order := t.order[w][filt]
			if order == 0 {
				continue
			}

			// calculate lpc coefficients
			autoc := t.coef[w][filt]
			for i := 0; i < order; i++ {
				r := -autoc[i]
				t.lpc[i] = r

				for j := 0; j < (i+1)>>1; j++ {
					f := t.lpc[j]
					b := t.lpc[i-1-j]
					t.lpc[j] = f + r*b
					t.lpc[i-1-j] = b + r*f
				}
			}

			startBand := bottom
			if startBand > mmm {
				startBand = mmm
			}
			endBand := top
			if endBand > mmm {
				endBand = mmm
			}

			start := info.SwbOffsets[startBand]
			end := info.SwbOffsets[endBand]
			size := end - start
			if size <= 0 {
				continue
			}

			inc := 1
			if t.direction[w][filt] {
				inc = -1
				start = end - 1
			}
			start += w * 128

			if decode {
				for m := 0; m < size; m++ {
					for i := 1; i <= minInt(m, order); i++ {
						data[start] -= data[start-i*inc] * t.lpc[i-1]
					}
					start += inc
				}
			} else {
				for i := 0; i < order+1 && i < len(t.tmp); i++ {
					t.tmp[i] = 0
				}
				for m := 0; m < size; m++ {
					t.tmp[0] = data[start]
					for i := 1; i <= minInt(m, order); i++ {
						data[start] += t.tmp[i] * t.lpc[i-1]
					}
					for i := order; i > 0; i-- {
						t.tmp[i] = t.tmp[i-1]
					}
					start += inc
				}
			}
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Package mdct implements the Modified Discrete Cosine Transform (MDCT)
// used in AAC audio decoding.
//
// The MDCT is a lapped transform based on the type-IV discrete cosine
// transform (DCT-IV), with the additional property that the transform
// is applied to overlapping blocks. This package implements the inverse
// MDCT (IMDCT) which converts frequency-domain spectral coefficients
// back to time-domain audio samples.
//
// Supported transform lengths are 2048, 256, 1920, 240, 1024, 128, 960, and 120.
// These correspond to:
//   - 2048/256: Long/short blocks in standard AAC-LC
//   - 1920/240: Long/short blocks in AAC-LD (Low Delay)
//   - 1024/128: Long/short blocks in AAC-ELD (512-sample frames)
//   - 960/120:  Long/short blocks in AAC-ELD (480-sample frames)
//
// This is a direct port of the MDCT module from AAC.js by Devon Govett
// (LGPL v3).
package mdct

import (
	"fmt"
	"math"

	"github.com/skrashevich/go-aac/pkg/fft"
	"github.com/skrashevich/go-aac/pkg/tables"
)

// MDCT performs the Inverse Modified Discrete Cosine Transform.
//
// The transform converts N/2 spectral coefficients to N time-domain samples
// using a pre-FFT complex multiplication, inverse FFT, post-FFT complex
// multiplication, and reordering stage.
type MDCT struct {
	// N is the full transform length (number of output samples).
	N int
	// N2 is N/2 (number of input spectral coefficients).
	N2 int
	// N4 is N/4 (FFT length).
	N4 int
	// N8 is N/8.
	N8 int

	// sincos contains precomputed twiddle factors for the MDCT.
	// Each entry is [cos, sin] for the pre/post-FFT multiplication.
	sincos [][2]float32

	// fft is the FFT processor for the N/4 point complex FFT.
	fft *fft.FFT

	// buf is a scratch buffer for intermediate complex values.
	// Size is N/4 complex numbers.
	buf [][2]float32

	// tmp is a temporary 2-element buffer for post-FFT multiplication.
	tmp [2]float32
}

// New creates a new MDCT processor for the given transform length.
//
// Supported lengths are 2048, 256, 1920, 240, 1024, 128, 960, and 120. These
// are the only sizes used by the AAC decoder:
//   - 2048/256 for standard AAC-LC
//   - 1920/240 for AAC-LD (Low Delay)
//   - 1024/128 and 960/120 for AAC-ELD
//
// Returns an error if an unsupported length is provided.
func New(length int) (*MDCT, error) {
	m := &MDCT{
		N:  length,
		N2: length >> 1,
		N4: length >> 2,
		N8: length >> 3,
	}

	// Select the appropriate precomputed twiddle factor table.
	var srcTable [][2]float64
	useGenerated := false
	switch length {
	case 2048:
		srcTable = tables.MDCTTable2048
	case 256:
		srcTable = tables.MDCTTable256
	case 1920:
		srcTable = tables.MDCTTable1920
	case 240:
		srcTable = tables.MDCTTable240
	case 1024, 128, 960, 120:
		useGenerated = true
	default:
		return nil, fmt.Errorf("mdct: unsupported length %d (supported: 2048, 256, 1920, 240, 1024, 128, 960, 120)", length)
	}

	// Convert float64 table to float32 for faster processing.
	if useGenerated {
		m.sincos = generateSineCosTable(length)
	} else {
		m.sincos = make([][2]float32, len(srcTable))
		for i, v := range srcTable {
			m.sincos[i][0] = float32(v[0])
			m.sincos[i][1] = float32(v[1])
		}
	}

	// Create the FFT processor for N/4 points.
	var err error
	m.fft, err = fft.New(m.N4)
	if err != nil {
		return nil, fmt.Errorf("mdct: failed to create FFT: %w", err)
	}

	// Pre-allocate the scratch buffer.
	m.buf = make([][2]float32, m.N4)

	return m, nil
}

func generateSineCosTable(length int) [][2]float32 {
	size := length >> 2
	table := make([][2]float32, size)
	scale := float32(math.Sqrt(2.0 / float64(length)))
	for k := 0; k < size; k++ {
		angle := 2.0 * math.Pi * (float64(k) + 0.125) / float64(length)
		table[k][0] = scale * float32(math.Cos(angle))
		table[k][1] = scale * float32(math.Sin(angle))
	}
	return table
}

// Length returns the transform length this MDCT was configured for.
func (m *MDCT) Length() int {
	return m.N
}

// Process performs the inverse MDCT on input spectral coefficients.
//
// The input slice should contain N/2 spectral coefficients starting at
// inOffset. The output slice will receive N time-domain samples starting
// at outOffset.
//
// The transform consists of:
//  1. Pre-IFFT complex multiplication with twiddle factors
//  2. Complex inverse FFT (N/4 point)
//  3. Post-IFFT complex multiplication with twiddle factors
//  4. Reordering to produce final time-domain output
//
// Note: The output needs to be combined with the previous frame using
// overlap-add to produce the final PCM samples. This is typically done
// by the filter bank module.
func (m *MDCT) Process(input []float32, inOffset int, output []float32, outOffset int) {
	N2 := m.N2
	N4 := m.N4
	N8 := m.N8
	buf := m.buf
	sincos := m.sincos

	// Pre-IFFT complex multiplication.
	// Combines spectral coefficients with twiddle factors to prepare
	// for the FFT stage.
	for k := 0; k < N4; k++ {
		// Real part: input[N2-1-2k]*cos - input[2k]*sin
		// Imag part: input[2k]*cos + input[N2-1-2k]*sin
		in0 := input[inOffset+2*k]
		in1 := input[inOffset+N2-1-2*k]
		cos := sincos[k][0]
		sin := sincos[k][1]

		buf[k][1] = in0*cos + in1*sin
		buf[k][0] = in1*cos - in0*sin
	}

	// Complex IFFT (non-scaling).
	// The FFT is performed in-place on the buf array.
	m.fft.Process(buf, false)

	// Post-IFFT complex multiplication.
	// Applies additional twiddle factor rotation.
	for k := 0; k < N4; k++ {
		tmp0 := buf[k][0]
		tmp1 := buf[k][1]
		cos := sincos[k][0]
		sin := sincos[k][1]

		buf[k][1] = tmp1*cos + tmp0*sin
		buf[k][0] = tmp0*cos - tmp1*sin
	}

	// Reordering stage.
	// Distributes the complex FFT output to the proper positions in
	// the time-domain output buffer.
	for k := 0; k < N8; k += 2 {
		// First quarter
		output[outOffset+2*k] = buf[N8+k][1]
		output[outOffset+2+2*k] = buf[N8+1+k][1]

		output[outOffset+1+2*k] = -buf[N8-1-k][0]
		output[outOffset+3+2*k] = -buf[N8-2-k][0]

		// Second quarter
		output[outOffset+N4+2*k] = buf[k][0]
		output[outOffset+N4+2+2*k] = buf[1+k][0]

		output[outOffset+N4+1+2*k] = -buf[N4-1-k][1]
		output[outOffset+N4+3+2*k] = -buf[N4-2-k][1]

		// Third quarter
		output[outOffset+N2+2*k] = buf[N8+k][0]
		output[outOffset+N2+2+2*k] = buf[N8+1+k][0]

		output[outOffset+N2+1+2*k] = -buf[N8-1-k][1]
		output[outOffset+N2+3+2*k] = -buf[N8-2-k][1]

		// Fourth quarter
		output[outOffset+N2+N4+2*k] = -buf[k][1]
		output[outOffset+N2+N4+2+2*k] = -buf[1+k][1]

		output[outOffset+N2+N4+1+2*k] = buf[N4-1-k][0]
		output[outOffset+N2+N4+3+2*k] = buf[N4-2-k][0]
	}
}

// ProcessSlice is a convenience method that processes the entire input slice.
//
// The input should have exactly N/2 elements. Returns a new slice of N
// time-domain samples.
func (m *MDCT) ProcessSlice(input []float32) []float32 {
	output := make([]float32, m.N)
	m.Process(input, 0, output, 0)
	return output
}

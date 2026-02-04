// Package fft implements the Fast Fourier Transform used internally by
// the AAC decoder's MDCT stage.
//
// Supported transform lengths are 64, 512, 256, 60, 480, and 240. These
// correspond to the MDCT block sizes used in AAC (256, 2048, 1024, 128, 240,
// 1920, 960, 120) divided by 4. The implementation uses a radix-4
// decimation-in-time algorithm with bit-reversal permutation.
//
// This is a direct port of the FFT module from AAC.js by Devon Govett
// (LGPL v3).
package fft

import (
	"fmt"
	"math"
)

// FFT performs forward or inverse FFT of a specific length.
//
// Supported lengths: 64, 512, 256, 60, 480, 240.
//
// Example usage:
//
//	f, err := New(512)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	data := make([][2]float32, 512)
//	// ... fill data with complex samples (data[i][0]=real, data[i][1]=imag) ...
//	f.Process(data, false) // inverse FFT
type FFT struct {
	length int

	// roots holds precomputed twiddle factors (roots of unity).
	// For short transforms (64, 60): each entry is [real, -imag, 0] (third element unused).
	// For long transforms (512, 480): each entry is [real, -imag, imag].
	roots [][3]float32

	// rev is a scratch buffer used for bit-reversal permutation.
	rev [][2]float32

	// Scratch variables for the radix-4 butterfly, allocated once to avoid
	// per-call heap allocations.
	a, b, c, d, e1, e2 [2]float32
}

// New creates a new FFT processor for the given transform length.
//
// Supported lengths are 64, 512, 256, 60, 480, and 240. These are the only
// sizes required by the AAC decoder's MDCT module (MDCT lengths 256, 2048,
// 240, 1920, 1024, 128, 960, 120 each use N/4 as the FFT length).
//
// Returns an error if an unsupported length is provided.
func New(length int) (*FFT, error) {
	f := &FFT{
		length: length,
	}

	switch length {
	case 64:
		f.roots = generateTableShort(64)
	case 512:
		f.roots = generateTableLong(512)
	case 256:
		f.roots = generateTableLong(256)
	case 60:
		f.roots = generateTableShort(60)
	case 480:
		f.roots = generateTableLong(480)
	case 240:
		f.roots = generateTableShort(240)
	default:
		return nil, fmt.Errorf("fft: unsupported length %d (supported: 64, 512, 256, 60, 480, 240)", length)
	}

	// Allocate the bit-reversal scratch buffer.
	f.rev = make([][2]float32, length)

	return f, nil
}

// Length returns the transform length this FFT was configured for.
func (f *FFT) Length() int {
	return f.length
}

// generateTableShort creates twiddle-factor tables for short FFT lengths
// (64, 60). The table stores complex roots of unity e^{-2*pi*i*k/N} as
// pairs [real, -imag]. To maintain a uniform [3]float32 layout with the
// long table, each entry is stored as [real, -imag, 0].
//
// The recurrence relation used is:
//
//	re[k] = re[k-1]*cos(t) + im[k-1]*sin(t)
//	im[k] = im[k-1]*cos(t) - re[k-1]*sin(t)
//
// where t = 2*pi/N and im is the running (positive) imaginary part.
// The stored value is -im (negated imaginary part).
func generateTableShort(length int) [][3]float32 {
	t := 2.0 * math.Pi / float64(length)
	cosT := math.Cos(t)
	sinT := math.Sin(t)

	table := make([][3]float32, length)

	table[0][0] = 1.0 // real
	table[0][1] = 0.0 // -imag
	table[0][2] = 0.0 // unused

	lastImag := 0.0 // running positive imaginary part

	for i := 1; i < length; i++ {
		re := float64(table[i-1][0])*cosT + lastImag*sinT
		lastImag = lastImag*cosT - float64(table[i-1][0])*sinT
		table[i][0] = float32(re)
		table[i][1] = float32(-lastImag)
		table[i][2] = 0.0 // unused for short tables
	}

	return table
}

// generateTableLong creates twiddle-factor tables for long FFT lengths
// (512, 480). The table stores complex roots of unity e^{-2*pi*i*k/N}
// as triples [real, -imag, imag].
//
// The extra third element (positive imaginary part) is used when
// performing a forward FFT (imOffset=2). For inverse FFT, the second
// element (negated imaginary part, imOffset=1) is used.
func generateTableLong(length int) [][3]float32 {
	t := 2.0 * math.Pi / float64(length)
	cosT := math.Cos(t)
	sinT := math.Sin(t)

	table := make([][3]float32, length)

	table[0][0] = 1.0 // real
	table[0][1] = 0.0 // -imag
	table[0][2] = 0.0 // imag

	for i := 1; i < length; i++ {
		prevRe := float64(table[i-1][0])
		prevIm := float64(table[i-1][2]) // positive imaginary part

		re := prevRe*cosT + prevIm*sinT
		im := prevIm*cosT - prevRe*sinT

		table[i][0] = float32(re)
		table[i][1] = float32(-im) // negated imaginary
		table[i][2] = float32(im)  // positive imaginary
	}

	return table
}

// Process performs an in-place FFT (forward=true) or IFFT (forward=false)
// on the given complex data.
//
// The input slice must have exactly f.Length() elements, where each element
// is [2]float32{real, imag}. The transform is performed in-place, so the
// input slice is modified directly.
//
// In the AAC decoder context, this is always called with forward=false
// (inverse FFT) from the MDCT module. The forward path is included for
// completeness and uses a different twiddle-factor column and scaling.
//
// Algorithm:
//  1. Bit-reversal permutation of the input
//  2. Bottom radix-4 butterfly pass (groups of 4)
//  3. Iterative radix-2 butterfly passes from bottom to top, using
//     precomputed twiddle factors
func (f *FFT) Process(input [][2]float32, forward bool) {
	length := f.length
	rev := f.rev
	roots := f.roots

	if !isPowerOfTwo(length) {
		f.processDFT(input, forward)
		return
	}

	// imOffset selects which imaginary component of the twiddle factor to use.
	// forward: index 2 (positive imag, only valid for long tables)
	// inverse: index 1 (negated imag)
	imOffset := 1
	if forward {
		imOffset = 2
	}

	// scale is applied at each butterfly in the iterative passes.
	// forward: multiply by length; inverse: multiply by 1 (no scaling).
	scale := float32(1)
	if forward {
		scale = float32(length)
	}

	// --- Bit-reversal permutation ---
	// Standard bit-reversal using the incrementing algorithm.
	ii := 0
	for i := 0; i < length; i++ {
		rev[i][0] = input[ii][0]
		rev[i][1] = input[ii][1]

		k := length >> 1
		for ii >= k && k > 0 {
			ii -= k
			k >>= 1
		}
		ii += k
	}

	// Copy bit-reversed data back into input.
	for i := 0; i < length; i++ {
		input[i][0] = rev[i][0]
		input[i][1] = rev[i][1]
	}

	// --- Bottom radix-4 butterfly ---
	// Process groups of 4 elements without twiddle factor multiplication.
	a := &f.a
	b := &f.b
	c := &f.c
	d := &f.d
	e1 := &f.e1
	e2 := &f.e2

	for i := 0; i < length; i += 4 {
		a[0] = input[i][0] + input[i+1][0]
		a[1] = input[i][1] + input[i+1][1]
		b[0] = input[i+2][0] + input[i+3][0]
		b[1] = input[i+2][1] + input[i+3][1]
		c[0] = input[i][0] - input[i+1][0]
		c[1] = input[i][1] - input[i+1][1]
		d[0] = input[i+2][0] - input[i+3][0]
		d[1] = input[i+2][1] - input[i+3][1]

		input[i][0] = a[0] + b[0]
		input[i][1] = a[1] + b[1]
		input[i+2][0] = a[0] - b[0]
		input[i+2][1] = a[1] - b[1]

		e1[0] = c[0] - d[1]
		e1[1] = c[1] + d[0]
		e2[0] = c[0] + d[1]
		e2[1] = c[1] - d[0]

		if forward {
			input[i+1][0] = e2[0]
			input[i+1][1] = e2[1]
			input[i+3][0] = e1[0]
			input[i+3][1] = e1[1]
		} else {
			input[i+1][0] = e1[0]
			input[i+1][1] = e1[1]
			input[i+3][0] = e2[0]
			input[i+3][1] = e2[1]
		}
	}

	// --- Iterative butterfly passes (radix-2) from bottom to top ---
	// Starting from groups of 4, double the group size each iteration.
	for i := 4; i < length; i <<= 1 {
		shift := i << 1
		m := length / shift

		for j := 0; j < length; j += shift {
			for k := 0; k < i; k++ {
				km := k * m
				rootRe := roots[km][0]
				rootIm := roots[km][imOffset]

				idx := i + j + k
				jk := j + k

				// Complex multiplication: z = input[idx] * root
				zRe := input[idx][0]*rootRe - input[idx][1]*rootIm
				zIm := input[idx][0]*rootIm + input[idx][1]*rootRe

				// Butterfly: combine input[jk] with z
				input[idx][0] = (input[jk][0] - zRe) * scale
				input[idx][1] = (input[jk][1] - zIm) * scale
				input[jk][0] = (input[jk][0] + zRe) * scale
				input[jk][1] = (input[jk][1] + zIm) * scale
			}
		}
	}
}

func (f *FFT) processDFT(input [][2]float32, forward bool) {
	n := f.length
	out := f.rev

	sign := -1.0
	if !forward {
		sign = 1.0
	}

	scale := float32(1)
	if forward {
		scale = float32(n)
	}

	for k := 0; k < n; k++ {
		var sumRe, sumIm float64
		for j := 0; j < n; j++ {
			angle := sign * 2.0 * math.Pi * float64(k*j) / float64(n)
			cos := math.Cos(angle)
			sin := math.Sin(angle)
			re := float64(input[j][0])
			im := float64(input[j][1])
			sumRe += re*cos - im*sin
			sumIm += re*sin + im*cos
		}
		out[k][0] = float32(sumRe) * scale
		out[k][1] = float32(sumIm) * scale
	}

	for i := 0; i < n; i++ {
		input[i][0] = out[i][0]
		input[i][1] = out[i][1]
	}
}

func isPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

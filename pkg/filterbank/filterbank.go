// Package filterbank implements the AAC synthesis filter bank.
//
// It applies IMDCT, windowing, and overlap-add to produce time-domain
// samples from spectral coefficients.
//
// This is a direct port of the FilterBank module from AAC.js by Devon Govett
// (LGPL v3). AAC-ELD low-delay filterbank support is based on the fdk-aac
// reference implementation.
package filterbank

import (
	"fmt"
	"math"

	"github.com/skrashevich/go-aac/pkg/mdct"
	"github.com/skrashevich/go-aac/pkg/tables"
)

const (
	OnlyLongSequence   = 0
	LongStartSequence  = 1
	EightShortSequence = 2
	LongStopSequence   = 3
)

// WindowInfo provides the window sequence and shapes needed by the filter bank.
// WindowShape[0] is the previous frame shape, WindowShape[1] is the current.
type WindowInfo struct {
	WindowSequence int
	WindowShape    [2]int
}

// FilterBank performs IMDCT and windowing for AAC decoding.
type FilterBank struct {
	length      int
	shortLength int
	mid         int
	trans       int

	isELD bool

	mdctShort *mdct.MDCT
	mdctLong  *mdct.MDCT

	overlaps [][]float32
	buf      []float32

	// ELD-specific: low-delay synthesis window and extended overlap buffer
	ldWin     []float32
	ldOverlap [][]float32
}

var (
	sine1024 = generateSineWindow(1024)
	sine128  = generateSineWindow(128)
	kbd1024  = generateKBDWindow(4, 1024)
	kbd128   = generateKBDWindow(6, 128)

	longWindows  = [][]float32{sine1024, kbd1024}
	shortWindows = [][]float32{sine128, kbd128}
)

// New creates a FilterBank for the given number of channels.
//
// AAC small frames are not supported, and will return an error if requested.
func New(smallFrames bool, channels int) (*FilterBank, error) {
	if smallFrames {
		return nil, fmt.Errorf("filterbank: small frames not supported")
	}
	if channels <= 0 {
		return nil, fmt.Errorf("filterbank: invalid channel count %d", channels)
	}

	f := &FilterBank{
		length:      1024,
		shortLength: 128,
	}

	f.mid = (f.length - f.shortLength) / 2
	f.trans = f.shortLength / 2

	var err error
	f.mdctShort, err = mdct.New(f.shortLength * 2)
	if err != nil {
		return nil, fmt.Errorf("filterbank: mdct short: %w", err)
	}
	f.mdctLong, err = mdct.New(f.length * 2)
	if err != nil {
		return nil, fmt.Errorf("filterbank: mdct long: %w", err)
	}

	f.overlaps = make([][]float32, channels)
	for i := 0; i < channels; i++ {
		f.overlaps[i] = make([]float32, f.length)
	}
	if f.length > 0 {
		f.buf = make([]float32, 2*f.length)
	}

	return f, nil
}

// NewELD creates a FilterBank for AAC-ELD with the given frame length and channels.
// frameLength should be 512 or 480.
func NewELD(frameLength int, channels int) (*FilterBank, error) {
	if channels <= 0 {
		return nil, fmt.Errorf("filterbank: invalid channel count %d", channels)
	}

	var ldWin []float32
	switch frameLength {
	case 512:
		ldWin = tables.LowDelaySynthesis512()
	case 480:
		ldWin = tables.LowDelaySynthesis480()
	default:
		return nil, fmt.Errorf("filterbank: unsupported ELD frame length %d (must be 512 or 480)", frameLength)
	}

	f := &FilterBank{
		length: frameLength,
		isELD:  true,
		ldWin:  ldWin,
	}

	var err error
	f.mdctLong, err = mdct.New(frameLength * 2)
	if err != nil {
		return nil, fmt.Errorf("filterbank: mdct eld: %w", err)
	}

	f.buf = make([]float32, 2*frameLength)

	// ELD uses an extended overlap buffer of 2*N samples
	f.ldOverlap = make([][]float32, channels)
	for i := 0; i < channels; i++ {
		f.ldOverlap[i] = make([]float32, 2*frameLength)
	}

	// Standard overlap buffer for compatibility
	f.overlaps = make([][]float32, channels)
	for i := 0; i < channels; i++ {
		f.overlaps[i] = make([]float32, frameLength)
	}

	return f, nil
}

// Process runs the filter bank for a single channel.
//
// input must contain at least 1024 spectral values, output must contain at
// least 1024 samples.
func (f *FilterBank) Process(info WindowInfo, input, output []float32, channel int) error {
	if channel < 0 || channel >= len(f.overlaps) {
		return fmt.Errorf("filterbank: invalid channel %d", channel)
	}
	if len(input) < f.length {
		return fmt.Errorf("filterbank: input length %d < %d", len(input), f.length)
	}
	if len(output) < f.length {
		return fmt.Errorf("filterbank: output length %d < %d", len(output), f.length)
	}

	windowShape := info.WindowShape[1]
	windowShapePrev := info.WindowShape[0]
	if windowShape < 0 || windowShape >= len(longWindows) {
		return fmt.Errorf("filterbank: invalid window shape %d", windowShape)
	}
	if windowShapePrev < 0 || windowShapePrev >= len(longWindows) {
		return fmt.Errorf("filterbank: invalid previous window shape %d", windowShapePrev)
	}

	longWin := longWindows[windowShape]
	shortWin := shortWindows[windowShape]
	longWinPrev := longWindows[windowShapePrev]
	shortWinPrev := shortWindows[windowShapePrev]

	length := f.length
	shortLen := f.shortLength
	mid := f.mid
	trans := f.trans
	buf := f.buf
	overlap := f.overlaps[channel]

	switch info.WindowSequence {
	case OnlyLongSequence:
		f.mdctLong.Process(input, 0, buf, 0)

		for i := 0; i < length; i++ {
			output[i] = overlap[i] + (buf[i] * longWinPrev[i])
		}

		for i := 0; i < length; i++ {
			overlap[i] = buf[length+i] * longWin[length-1-i]
		}

	case LongStartSequence:
		f.mdctLong.Process(input, 0, buf, 0)

		for i := 0; i < length; i++ {
			output[i] = overlap[i] + (buf[i] * longWinPrev[i])
		}

		for i := 0; i < mid; i++ {
			overlap[i] = buf[length+i]
		}

		for i := 0; i < shortLen; i++ {
			overlap[mid+i] = buf[length+mid+i] * shortWin[shortLen-1-i]
		}

		for i := 0; i < mid; i++ {
			overlap[mid+shortLen+i] = 0
		}

	case EightShortSequence:
		for i := 0; i < 8; i++ {
			f.mdctShort.Process(input, i*shortLen, buf, 2*i*shortLen)
		}

		for i := 0; i < mid; i++ {
			output[i] = overlap[i]
		}

		for i := 0; i < shortLen; i++ {
			output[mid+i] = overlap[mid+i] + buf[i]*shortWinPrev[i]
			output[mid+1*shortLen+i] = overlap[mid+shortLen*1+i] + (buf[shortLen*1+i] * shortWin[shortLen-1-i]) + (buf[shortLen*2+i] * shortWin[i])
			output[mid+2*shortLen+i] = overlap[mid+shortLen*2+i] + (buf[shortLen*3+i] * shortWin[shortLen-1-i]) + (buf[shortLen*4+i] * shortWin[i])
			output[mid+3*shortLen+i] = overlap[mid+shortLen*3+i] + (buf[shortLen*5+i] * shortWin[shortLen-1-i]) + (buf[shortLen*6+i] * shortWin[i])

			if i < trans {
				output[mid+4*shortLen+i] = overlap[mid+shortLen*4+i] + (buf[shortLen*7+i] * shortWin[shortLen-1-i]) + (buf[shortLen*8+i] * shortWin[i])
			}
		}

		for i := 0; i < shortLen; i++ {
			if i >= trans {
				overlap[mid+4*shortLen+i-length] = (buf[shortLen*7+i] * shortWin[shortLen-1-i]) + (buf[shortLen*8+i] * shortWin[i])
			}

			overlap[mid+5*shortLen+i-length] = (buf[shortLen*9+i] * shortWin[shortLen-1-i]) + (buf[shortLen*10+i] * shortWin[i])
			overlap[mid+6*shortLen+i-length] = (buf[shortLen*11+i] * shortWin[shortLen-1-i]) + (buf[shortLen*12+i] * shortWin[i])
			overlap[mid+7*shortLen+i-length] = (buf[shortLen*13+i] * shortWin[shortLen-1-i]) + (buf[shortLen*14+i] * shortWin[i])
			overlap[mid+8*shortLen+i-length] = (buf[shortLen*15+i] * shortWin[shortLen-1-i])
		}

		for i := 0; i < mid; i++ {
			overlap[mid+shortLen+i] = 0
		}

	case LongStopSequence:
		f.mdctLong.Process(input, 0, buf, 0)

		for i := 0; i < mid; i++ {
			output[i] = overlap[i]
		}

		for i := 0; i < shortLen; i++ {
			output[mid+i] = overlap[mid+i] + (buf[mid+i] * shortWinPrev[i])
		}

		for i := 0; i < mid; i++ {
			output[mid+shortLen+i] = overlap[mid+shortLen+i] + buf[mid+shortLen+i]
		}

		for i := 0; i < length; i++ {
			overlap[i] = buf[length+i] * longWin[length-1-i]
		}

	default:
		return fmt.Errorf("filterbank: unknown window sequence %d", info.WindowSequence)
	}

	return nil
}

// ProcessELD runs the low-delay filter bank for AAC-ELD.
//
// The ELD filterbank uses:
//  1. Standard IMDCT to produce 2N time-domain samples from N spectral coefficients
//  2. Low-delay synthesis window application (3N coefficients)
//  3. Overlap-add with 2N-sample overlap buffer
//
// This implements the InvMdctTransformLowDelay algorithm from the fdk-aac
// reference implementation.
func (f *FilterBank) ProcessELD(input, output []float32, channel int) error {
	if !f.isELD {
		return fmt.Errorf("filterbank: ProcessELD called on non-ELD filterbank")
	}
	if channel < 0 || channel >= len(f.ldOverlap) {
		return fmt.Errorf("filterbank: invalid channel %d", channel)
	}

	n := f.length // frame length (512 or 480)
	buf := f.buf
	ldWin := f.ldWin
	overlap := f.ldOverlap[channel]

	// Step 1: IMDCT - N spectral coefficients -> 2N time-domain samples
	f.mdctLong.Process(input, 0, buf, 0)

	// Step 2 & 3: Apply low-delay synthesis window and overlap-add.
	// The LD window has 3*N coefficients organized as follows:
	// - ldWin[0..N-1]: window for the first section
	// - ldWin[N..2N-1]: window for the middle section
	// - ldWin[2N..3N-1]: window for the last section
	//
	// The algorithm:
	// 1. Shift the overlap buffer: move overlap[0..N-1] -> overlap[N..2N-1]
	//    (conceptually, the old "current" becomes the "previous")
	// 2. Apply windowed IMDCT output to the overlap buffer
	// 3. Read output from the overlap buffer

	// Shift overlap buffer
	copy(overlap[n:2*n], overlap[0:n])

	// Apply IMDCT output with LD synthesis window to overlap buffer
	for i := 0; i < n; i++ {
		overlap[i] = 0
	}

	// Accumulate windowed IMDCT output into overlap
	// Section 1: buf[0..N-1] * ldWin[0..N-1] -> added to overlap[0..N-1]
	for i := 0; i < n; i++ {
		overlap[i] += buf[i] * ldWin[i]
	}

	// Section 2: buf[N..2N-1] * ldWin[N..2N-1] -> added to overlap[0..N-1]
	for i := 0; i < n; i++ {
		overlap[i] += buf[n+i] * ldWin[n+i]
	}

	// Section 3: previous overlap contribution with ldWin[2N..3N-1]
	for i := 0; i < n; i++ {
		overlap[i] += overlap[n+i] * ldWin[2*n+i]
	}

	// Output the current frame
	copy(output, overlap[0:n])

	return nil
}

func generateSineWindow(length int) []float32 {
	window := make([]float32, length)
	div := math.Pi / (2.0 * float64(length))
	for i := 0; i < length; i++ {
		window[i] = float32(math.Sin((float64(i) + 0.5) * div))
	}
	return window
}

func generateKBDWindow(alpha float64, length int) []float32 {
	pin := math.Pi / float64(length)
	out := make([]float32, length)
	f := make([]float64, length)
	alpha2 := (alpha * pin) * (alpha * pin)

	sum := 0.0
	for n := 0; n < length; n++ {
		tmp := float64(n) * float64(length-n) * alpha2
		bessel := 1.0
		for j := 50; j > 0; j-- {
			bessel = bessel*tmp/float64(j*j) + 1
		}
		sum += bessel
		f[n] = sum
	}

	sum++
	for n := 0; n < length; n++ {
		out[n] = float32(math.Sqrt(f[n] / sum))
	}

	return out
}

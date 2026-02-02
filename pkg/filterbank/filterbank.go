// Package filterbank implements the AAC synthesis filter bank.
//
// It applies IMDCT, windowing, and overlap-add to produce time-domain
// samples from spectral coefficients.
//
// This is a direct port of the FilterBank module from AAC.js by Devon Govett
// (LGPL v3).
package filterbank

import (
	"fmt"
	"math"

	"github.com/skrashevich/go-aac/pkg/mdct"
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

	mdctShort *mdct.MDCT
	mdctLong  *mdct.MDCT

	overlaps [][]float32
	buf      []float32
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

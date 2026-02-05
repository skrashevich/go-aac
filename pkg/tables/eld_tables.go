package tables

import "math"

// ELD scalefactor window band offset tables for 512-sample frames.
// Source: fdk-aac libAACdec/src/aac_rom.cpp

var swbOffset512_48 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48,
	52, 56, 60, 68, 76, 84, 92, 100, 112, 124, 136, 148, 164,
	184, 208, 236, 268, 300, 332, 364, 396, 428, 460, 512,
}

var swbOffset512_32 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36,
	40, 44, 48, 52, 56, 64, 72, 80, 88, 96,
	108, 120, 132, 144, 160, 176, 192, 212, 236, 260,
	288, 320, 352, 384, 416, 448, 480, 512,
}

var swbOffset512_24 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40,
	44, 52, 60, 68, 80, 92, 104, 120, 140, 164, 192,
	224, 256, 288, 320, 352, 384, 416, 448, 480, 512,
}

// SWBOffset512 contains scalefactor window band offsets for ELD 512-sample
// frames, indexed by sample rate index (0-12).
var SWBOffset512 = [][]uint16{
	swbOffset512_48, // 96000 Hz
	swbOffset512_48, // 88200 Hz
	swbOffset512_48, // 64000 Hz
	swbOffset512_48, // 48000 Hz
	swbOffset512_48, // 44100 Hz
	swbOffset512_32, // 32000 Hz
	swbOffset512_24, // 24000 Hz
	swbOffset512_24, // 22050 Hz
	swbOffset512_24, // 16000 Hz
	swbOffset512_24, // 12000 Hz
	swbOffset512_24, // 11025 Hz
	swbOffset512_24, // 8000 Hz
	swbOffset512_24, // 7350 Hz
}

// SWBWindowCount512 contains the number of scalefactor window bands for
// ELD 512-sample frames for each sample rate index (0-12).
var SWBWindowCount512 = []uint8{36, 36, 36, 36, 36, 37, 31, 31, 31, 31, 31, 31, 31}

// ELD scalefactor window band offset tables for 480-sample frames.

var swbOffset480_48 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48,
	52, 56, 64, 72, 80, 88, 96, 108, 120, 132, 144, 156, 172,
	188, 212, 240, 272, 304, 336, 368, 400, 432, 480,
}

var swbOffset480_32 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36,
	40, 44, 48, 52, 56, 60, 64, 72, 80, 88,
	96, 104, 112, 124, 136, 148, 164, 180, 200, 224,
	256, 288, 320, 352, 384, 416, 448, 480,
}

var swbOffset480_24 = []uint16{
	0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40,
	44, 52, 60, 68, 80, 92, 104, 120, 140, 164, 192,
	224, 256, 288, 320, 352, 384, 416, 448, 480,
}

// SWBOffset480 contains scalefactor window band offsets for ELD 480-sample
// frames, indexed by sample rate index (0-12).
var SWBOffset480 = [][]uint16{
	swbOffset480_48, // 96000 Hz
	swbOffset480_48, // 88200 Hz
	swbOffset480_48, // 64000 Hz
	swbOffset480_48, // 48000 Hz
	swbOffset480_48, // 44100 Hz
	swbOffset480_32, // 32000 Hz
	swbOffset480_24, // 24000 Hz
	swbOffset480_24, // 22050 Hz
	swbOffset480_24, // 16000 Hz
	swbOffset480_24, // 12000 Hz
	swbOffset480_24, // 11025 Hz
	swbOffset480_24, // 8000 Hz
	swbOffset480_24, // 7350 Hz
}

// SWBWindowCount480 contains the number of scalefactor window bands for
// ELD 480-sample frames for each sample rate index (0-12).
var SWBWindowCount480 = []uint8{35, 35, 35, 35, 35, 37, 30, 30, 30, 30, 30, 30, 30}

// TNSMaxBands512 contains the maximum number of TNS bands for ELD 512-sample
// frames, indexed by sample rate index (0-12).
var TNSMaxBands512 = []int{31, 31, 31, 31, 32, 37, 31, 31, 31, 31, 31, 31, 31}

// TNSMaxBands480 contains the maximum number of TNS bands for ELD 480-sample
// frames, indexed by sample rate index (0-12).
var TNSMaxBands480 = []int{31, 31, 31, 31, 32, 37, 30, 30, 30, 30, 30, 30, 30}

// GenerateMDCTTable computes MDCT twiddle factors for a given transform length N.
// Each entry is [cos, sin] where:
//
//	cos = sqrt(2/N) * cos(2*pi*(k+1/8)/N)
//	sin = sqrt(2/N) * sin(2*pi*(k+1/8)/N)
//
// for k = 0..N/4-1.
func GenerateMDCTTable(n int) [][2]float64 {
	n4 := n / 4
	scale := math.Sqrt(2.0 / float64(n))
	table := make([][2]float64, n4)
	for k := 0; k < n4; k++ {
		angle := 2.0 * math.Pi * (float64(k) + 0.125) / float64(n)
		table[k][0] = scale * math.Cos(angle)
		table[k][1] = scale * math.Sin(angle)
	}
	return table
}

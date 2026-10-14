package decoder

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
	"math"
	"os"
	"testing"
)

// The fixtures encode independent 1000/1500 Hz tones as AAC-ELD (no SBR).
// .eld stores ASC length (one byte), ASC, then big-endian uint16 AU lengths.
// References are native FFmpeg float PCM, before any playback resampling.
func TestELDMatchesFFmpeg(t *testing.T) {
	for _, tc := range []struct {
		name            string
		frame, channels int
	}{
		{"mono24-512", 512, 1}, {"stereo24-480", 480, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile("testdata/" + tc.name + ".eld")
			if err != nil {
				t.Fatal(err)
			}
			d := New()
			if err = d.SetASC(data[1 : 1+int(data[0])]); err != nil {
				t.Fatal(err)
			}
			if d.Config.FrameLength != tc.frame || d.Config.ChanConfig != tc.channels {
				t.Fatalf("configuration: %+v", d.Config)
			}
			compressed, err := os.ReadFile("testdata/" + tc.name + ".f32.gz")
			if err != nil {
				t.Fatal(err)
			}
			r, err := gzip.NewReader(bytes.NewReader(compressed))
			if err != nil {
				t.Fatal(err)
			}
			ref, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatal(err)
			}
			data = data[1+int(data[0]):]
			index := 0
			var energy, noise float64
			for len(data) > 0 {
				size := int(binary.BigEndian.Uint16(data))
				samples, err := d.DecodeFrame(data[2 : 2+size])
				if err != nil {
					t.Fatal(err)
				}
				if len(samples) != tc.frame*tc.channels {
					t.Fatalf("frame samples %d", len(samples))
				}
				for _, v := range samples {
					want := math.Float32frombits(binary.LittleEndian.Uint32(ref[index*4:]))
					index++
					if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
						t.Fatal("non-finite PCM")
					}
					energy += float64(want) * float64(want)
					diff := float64(v - want)
					noise += diff * diff
				}
				data = data[2+size:]
			}
			if index*4 != len(ref) {
				t.Fatalf("decoded %d samples, reference %d", index, len(ref)/4)
			}
			snr := 10 * math.Log10(energy/noise)
			t.Logf("SNR against FFmpeg: %.1f dB", snr)
			if energy == 0 || snr < 60 {
				t.Fatalf("ELD differs from FFmpeg: SNR %.1f dB", snr)
			}
		})
	}
}

func TestELDRejectsUnsupportedToolsAndTruncation(t *testing.T) {
	for _, flag := range []byte{8, 4, 2, 1} {
		if err := New().SetASC([]byte{0xf8, 0xec, 0x20 | flag, 0}); err == nil {
			t.Fatalf("accepted ELD resilience/SBR flag %x", flag)
		}
	}
	asc := []byte{0xf8, 0xec, 0x20, 0}
	for n := range len(asc) {
		if err := New().SetASC(asc[:n]); err == nil {
			t.Fatalf("accepted truncated ASC length %d", n)
		}
	}
	d := New()
	if err := d.SetASC(asc); err != nil {
		t.Fatal(err)
	}
	if _, err := d.DecodeFrame(nil); err == nil {
		t.Fatal("accepted empty ELD AU")
	}
}

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/skrashevich/go-aac/pkg/adts"
	"github.com/skrashevich/go-aac/pkg/decoder"
)

func main() {
	raw := flag.Bool("raw", false, "output raw PCM int16 LE to stdout instead of WAV")
	verbose := flag.Bool("v", false, "verbose: print sample rate, channels, frame count")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: aac2pcm [options] input.aac [output.wav]\n\n")
		fmt.Fprintf(os.Stderr, "Decode AAC (ADTS) file to PCM.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nIf --raw is set, PCM int16 LE is written to stdout and output file is not needed.\n")
	}
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputPath := args[0]
	var outputPath string
	if !*raw {
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Error: output file required (or use --raw for stdout)\n")
			os.Exit(1)
		}
		outputPath = args[1]
	}

	data, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	if !adts.Probe(data) {
		fmt.Fprintf(os.Stderr, "Error: input file does not contain ADTS data\n")
		os.Exit(1)
	}

	dec := decoder.New()
	var allSamples []float32
	offset := 0
	frameCount := 0

	for offset < len(data) {
		if len(data)-offset < 7 {
			break
		}

		hdr, err := adts.ReadHeaderFromBytes(data[offset:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading ADTS header at offset %d: %v\n", offset, err)
			break
		}

		if hdr.FrameLength < 7 || offset+hdr.FrameLength > len(data) {
			fmt.Fprintf(os.Stderr, "Error: invalid frame length %d at offset %d\n", hdr.FrameLength, offset)
			break
		}

		frameData := data[offset : offset+hdr.FrameLength]

		samples, err := dec.DecodeFrame(frameData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding frame %d at offset %d: %v\n", frameCount, offset, err)
			offset += hdr.FrameLength
			frameCount++
			continue
		}

		allSamples = append(allSamples, samples...)
		offset += hdr.FrameLength
		frameCount++
	}

	if len(allSamples) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no samples decoded\n")
		os.Exit(1)
	}

	cfg := dec.Config
	numChannels := chanConfigToCount(cfg.ChanConfig)
	sampleRate := cfg.SampleRate

	if *verbose {
		fmt.Fprintf(os.Stderr, "Sample rate:  %d Hz\n", sampleRate)
		fmt.Fprintf(os.Stderr, "Channels:     %d (config %d)\n", numChannels, cfg.ChanConfig)
		fmt.Fprintf(os.Stderr, "Frames:       %d\n", frameCount)
		fmt.Fprintf(os.Stderr, "Total samples: %d (per channel: %d)\n", len(allSamples), len(allSamples)/numChannels)
	}

	int16Samples := floatToInt16(allSamples)

	if *raw {
		err = binary.Write(os.Stdout, binary.LittleEndian, int16Samples)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing raw PCM: %v\n", err)
			os.Exit(1)
		}
		return
	}

	f, err := os.Create(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	err = writeWAV(f, int16Samples, sampleRate, numChannels)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing WAV: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Written: %s\n", outputPath)
	}
}

func chanConfigToCount(chanConfig int) int {
	switch chanConfig {
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 3
	case 4:
		return 4
	case 5:
		return 5
	case 6:
		return 6
	case 7:
		return 8
	default:
		return 2
	}
}

func clamp(v float64, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func floatToInt16(samples []float32) []int16 {
	out := make([]int16, len(samples))
	for i, v := range samples {
		out[i] = int16(clamp(float64(v)*32767.0, -32768, 32767))
	}
	return out
}

func writeWAV(w io.Writer, samples []int16, sampleRate, numChannels int) error {
	dataSize := uint32(len(samples) * 2)
	fileSize := 36 + dataSize

	bitsPerSample := uint16(16)
	blockAlign := uint16(numChannels) * bitsPerSample / 8
	byteRate := uint32(sampleRate) * uint32(blockAlign)

	// RIFF header
	if _, err := w.Write([]byte("RIFF")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, fileSize); err != nil {
		return err
	}
	if _, err := w.Write([]byte("WAVE")); err != nil {
		return err
	}

	// fmt sub-chunk
	if _, err := w.Write([]byte("fmt ")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(16)); err != nil { // sub-chunk size
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(1)); err != nil { // PCM format
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(numChannels)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(sampleRate)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, bitsPerSample); err != nil {
		return err
	}

	// data sub-chunk
	if _, err := w.Write([]byte("data")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, dataSize); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, samples); err != nil {
		return err
	}

	return nil
}

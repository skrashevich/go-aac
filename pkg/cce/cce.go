// Package cce implements the AAC Channel Coupling Element.
//
// This is a direct port of the CCE module from AAC.js by Devon Govett
// (LGPL v3).
package cce

import (
	"fmt"
	"math"

	"github.com/skrashevich/go-aac/pkg/huffman"
	"github.com/skrashevich/go-aac/pkg/ics"
)

const (
	BeforeTNS  = 0
	AfterTNS   = 1
	AfterIMDCT = 2
)

const maxGainBands = 120

var cceScale = [4]float32{
	1.09050773266525765921,
	1.18920711500272106672,
	1.4142135623730950488,
	2.0,
}

// Element represents a Channel Coupling Element (CCE).
type Element struct {
	ICS         *ics.ICStream
	ChannelPair [8]bool
	IDSelect    [8]int
	ChSelect    [8]int
	Gain        [][]float32

	CouplingPoint int
	CoupledCount  int
}

// New creates a new CCE element for the given AAC config.
func New(config ics.Config) (*Element, error) {
	icsStream, err := ics.New(config)
	if err != nil {
		return nil, err
	}

	return &Element{ICS: icsStream}, nil
}

// Decode reads the CCE data from the bitstream.
func (e *Element) Decode(stream ics.BitReader, config ics.Config) error {
	e.CouplingPoint = int(2 * stream.ReadBits(1))
	e.CoupledCount = int(stream.ReadBits(3))

	gainCount := 0
	for i := 0; i <= e.CoupledCount; i++ {
		gainCount++
		e.ChannelPair[i] = stream.ReadBits(1) != 0
		e.IDSelect[i] = int(stream.ReadBits(4))
		if e.ChannelPair[i] {
			e.ChSelect[i] = int(stream.ReadBits(2))
			if e.ChSelect[i] == 3 {
				gainCount++
			}
		} else {
			e.ChSelect[i] = 2
		}
	}

	e.CouplingPoint += int(stream.ReadBits(1))
	e.CouplingPoint |= e.CouplingPoint >> 1

	sign := stream.ReadBits(1) != 0
	scale := cceScale[stream.ReadBits(2)]

	if err := e.ICS.Decode(stream, config, false); err != nil {
		return err
	}

	groupCount := e.ICS.Info.GroupCount
	maxSFB := e.ICS.Info.MaxSFB
	bandTypes := e.ICS.BandTypes

	e.Gain = make([][]float32, gainCount)
	for i := 0; i < gainCount; i++ {
		idx := 0
		cge := 1
		gain := 0
		gainCache := float32(1)

		if i > 0 {
			if e.CouplingPoint != AfterIMDCT {
				cge = int(stream.ReadBits(1))
			}
			if cge != 0 {
				gain = huffman.DecodeScaleFactor(stream) - 60
			}
			gainCache = float32(math.Pow(float64(scale), float64(-gain)))
		}

		gainSlice := make([]float32, maxGainBands)
		e.Gain[i] = gainSlice

		if e.CouplingPoint == AfterIMDCT {
			gainSlice[0] = gainCache
			continue
		}

		for g := 0; g < groupCount; g++ {
			for sfb := 0; sfb < maxSFB; sfb++ {
				if bandTypes[idx] != ics.ZeroBT {
					if cge == 0 {
						t := huffman.DecodeScaleFactor(stream) - 60
						if t != 0 {
							s := float32(1)
							gain += t
							gt := gain
							if !sign {
								if gt&1 != 0 {
									s = -1
								}
								gt = int(uint(gt) >> 1)
							}
							gainCache = float32(math.Pow(float64(scale), float64(-gt))) * s
						}
					}
					gainSlice[idx] = gainCache
				}
				idx++
			}
		}
	}

	return nil
}

// ApplyIndependentCoupling applies coupling after IMDCT.
func (e *Element) ApplyIndependentCoupling(index int, data []float32) error {
	if index < 0 || index >= len(e.Gain) {
		return fmt.Errorf("cce: gain index out of range: %d", index)
	}
	if len(e.ICS.Data) == 0 {
		return nil
	}

	gain := e.Gain[index][0]
	limit := len(data)
	if len(e.ICS.Data) < limit {
		limit = len(e.ICS.Data)
	}

	for i := 0; i < limit; i++ {
		data[i] += gain * e.ICS.Data[i]
	}

	return nil
}

// ApplyDependentCoupling applies coupling before/after TNS.
func (e *Element) ApplyDependentCoupling(index int, data []float32) error {
	if index < 0 || index >= len(e.Gain) {
		return fmt.Errorf("cce: gain index out of range: %d", index)
	}
	info := e.ICS.Info
	swbOffsets := info.SwbOffsets
	groupCount := info.GroupCount
	maxSFB := info.MaxSFB
	bandTypes := e.ICS.BandTypes
	iqData := e.ICS.Data
	gains := e.Gain[index]

	idx := 0
	offset := 0
	for g := 0; g < groupCount; g++ {
		length := info.GroupLength[g]
		for sfb := 0; sfb < maxSFB; sfb++ {
			if bandTypes[idx] != ics.ZeroBT {
				gain := gains[idx]
				start := swbOffsets[sfb]
				end := swbOffsets[sfb+1]
				for group := 0; group < length; group++ {
					base := offset + group*128
					for k := start; k < end; k++ {
						pos := base + k
						if pos < len(data) && pos < len(iqData) {
							data[pos] += gain * iqData[pos]
						}
					}
				}
			}
			idx++
		}

		offset += length * 128
	}

	return nil
}

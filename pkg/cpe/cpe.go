// Package cpe implements the AAC Channel Pair Element.
//
// This is a direct port of the CPE module from AAC.js by Devon Govett
// (LGPL v3).
package cpe

import (
	"fmt"

	"github.com/skrashevich/go-aac/pkg/ics"
)

const (
	maxMSMask = 128

	maskTypeAll0     = 0
	maskTypeUsed     = 1
	maskTypeAll1     = 2
	maskTypeReserved = 3
)

// Element represents a Channel Pair Element (CPE).
type Element struct {
	MSUsed []bool
	Left   *ics.ICStream
	Right  *ics.ICStream

	CommonWindow bool
	MaskPresent  bool
}

// New creates a new CPE element for the given AAC config.
func New(config ics.Config) (*Element, error) {
	left, err := ics.New(config)
	if err != nil {
		return nil, err
	}
	right, err := ics.New(config)
	if err != nil {
		return nil, err
	}

	return &Element{
		MSUsed: make([]bool, maxMSMask),
		Left:   left,
		Right:  right,
	}, nil
}

// Decode reads the CPE from the bitstream.
func (e *Element) Decode(stream ics.BitReader, config ics.Config) error {
	left := e.Left
	right := e.Right
	msUsed := e.MSUsed

	e.CommonWindow = stream.ReadBits(1) != 0
	if e.CommonWindow {
		if err := left.Info.Decode(stream, config, true); err != nil {
			return err
		}
		right.Info = left.Info

		mask := int(stream.ReadBits(2))
		e.MaskPresent = mask != 0

		switch mask {
		case maskTypeUsed:
			length := left.Info.GroupCount * left.Info.MaxSFB
			if length > len(msUsed) {
				length = len(msUsed)
			}
			for i := 0; i < length; i++ {
				msUsed[i] = stream.ReadBits(1) != 0
			}
		case maskTypeAll0, maskTypeAll1:
			val := mask != 0
			for i := 0; i < maxMSMask; i++ {
				msUsed[i] = val
			}
		case maskTypeReserved:
			fallthrough
		default:
			return fmt.Errorf("cpe: reserved ms mask type: %d", mask)
		}
	} else {
		e.MaskPresent = false
		for i := 0; i < maxMSMask; i++ {
			msUsed[i] = false
		}
	}

	if err := left.Decode(stream, config, e.CommonWindow); err != nil {
		return err
	}
	if err := right.Decode(stream, config, e.CommonWindow); err != nil {
		return err
	}

	return nil
}

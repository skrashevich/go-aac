// AAC-ELD filterbank adapted from FFmpeg's imdct_and_windowing_eld.
// Copyright (c) 2005-2006 Oded Shimon
// Copyright (c) 2006-2007 Maxim Gavrilov
// Copyright (c) 2008-2013 Alex Converse
// Copyright (c) 2008-2010 Paul Kendall
// Copyright (c) 2010 Janne Grunau
// Copyright (c) 2013 MIPS Technologies, Inc., California
// Licensed under LGPL-2.1-or-later; distributed here under LGPL-3.0.
// Source: FFmpeg ec33c154fa3c5d6989b95a053c46e4c10d8aa73d,
// libavcodec/aac/aacdec_dsp_template.c.
package filterbank

import (
	"fmt"

	"github.com/skrashevich/go-aac/pkg/mdct"
	"github.com/skrashevich/go-aac/pkg/tables"
)

// NewELD creates the low-delay filterbank for 512/480 sample AAC-ELD.
func NewELD(frameLength, channels int) (*FilterBank, error) {
	if channels < 1 || channels > 2 {
		return nil, fmt.Errorf("filterbank: ELD requires mono or stereo")
	}
	f := &FilterBank{length: frameLength}
	switch frameLength {
	case 512:
		f.eldWindow = tables.ELDWindow512[:]
	case 480:
		f.eldWindow = tables.ELDWindow480[:]
	default:
		return nil, fmt.Errorf("filterbank: unsupported ELD frame length %d", frameLength)
	}
	var err error
	f.mdctLong, err = mdct.New(frameLength * 2)
	if err != nil {
		return nil, err
	}
	f.buf = make([]float32, frameLength*2)
	f.eldInput = make([]float32, frameLength)
	f.overlaps = make([][]float32, channels)
	for i := range f.overlaps {
		f.overlaps[i] = make([]float32, frameLength*3)
	}
	return f, nil
}

// ProcessELD applies the ELD transform, asymmetric window and three-frame history.
func (f *FilterBank) ProcessELD(input, output []float32, channel int) error {
	n := f.length
	if f.eldWindow == nil || channel < 0 || channel >= len(f.overlaps) || len(input) < n || len(output) < n {
		return fmt.Errorf("filterbank: invalid ELD buffers or channel")
	}
	in := f.eldInput
	copy(in, input[:n])
	for i := 0; i < n/2; i += 2 {
		in[i], in[n-1-i] = -in[n-1-i], in[i]
		in[i+1], in[n-2-i] = in[n-2-i], -in[i+1]
	}
	f.mdctLong.Process(in, 0, f.buf, 0)
	buf := f.buf[n/2 : n/2+n]
	for i := 0; i < n; i += 2 {
		buf[i] = -buf[i]
	}
	saved, window := f.overlaps[channel], f.eldWindow
	n2, n4 := n/2, n/4
	for i := n4; i < n2; i++ {
		output[i-n4] = buf[n2-1-i]*window[i-n4] + saved[i+n2]*window[i+n-n4] - saved[n+n2-1-i]*window[i+2*n-n4] - saved[2*n+n2+i]*window[i+3*n-n4]
	}
	for i := 0; i < n2; i++ {
		output[n4+i] = buf[i]*window[i+n2-n4] - saved[n-1-i]*window[i+n2+n-n4] - saved[n+i]*window[i+n2+2*n-n4] + saved[3*n-1-i]*window[i+n2+3*n-n4]
	}
	for i := 0; i < n4; i++ {
		output[n2+n4+i] = buf[i+n2]*window[i+n-n4] - saved[n2-1-i]*window[i+2*n-n4] - saved[n+n2+i]*window[i+3*n-n4]
	}
	copy(saved[n:], saved[:2*n])
	copy(saved, buf)
	return nil
}

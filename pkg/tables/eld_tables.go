// AAC-ELD tables adapted from FFmpeg libavcodec/aactab.c.
// Copyright (c) 2005-2006 Oded Shimon
// Copyright (c) 2006-2007 Maxim Gavrilov
// Licensed under LGPL-2.1-or-later; distributed here under LGPL-3.0.
// Source revision: ec33c154fa3c5d6989b95a053c46e4c10d8aa73d
package tables

var swbOffset512_48 = []uint16{0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56, 60, 68, 76, 84, 92, 100, 112, 124, 136, 148, 164, 184, 208, 236, 268, 300, 332, 364, 396, 428, 460, 512}
var swbOffset512_32 = []uint16{0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56, 64, 72, 80, 88, 96, 108, 120, 132, 144, 160, 176, 192, 212, 236, 260, 288, 320, 352, 384, 416, 448, 480, 512}
var swbOffset512_24 = []uint16{0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 52, 60, 68, 80, 92, 104, 120, 140, 164, 192, 224, 256, 288, 320, 352, 384, 416, 448, 480, 512}

var SWBOffset512 = [][]uint16{swbOffset512_48, swbOffset512_48, swbOffset512_48, swbOffset512_48, swbOffset512_48, swbOffset512_32, swbOffset512_24, swbOffset512_24, swbOffset512_24, swbOffset512_24, swbOffset512_24, swbOffset512_24, swbOffset512_24}
var SWBWindowCount512 = []uint8{36, 36, 36, 36, 36, 37, 31, 31, 31, 31, 31, 31, 31}

var swbOffset480_48 = []uint16{0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56, 64, 72, 80, 88, 96, 108, 120, 132, 144, 156, 172, 188, 212, 240, 272, 304, 336, 368, 400, 432, 480}
var swbOffset480_32 = []uint16{0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56, 60, 64, 72, 80, 88, 96, 104, 112, 124, 136, 148, 164, 180, 200, 224, 256, 288, 320, 352, 384, 416, 448, 480}
var swbOffset480_24 = []uint16{0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 52, 60, 68, 80, 92, 104, 120, 140, 164, 192, 224, 256, 288, 320, 352, 384, 416, 448, 480}

var SWBOffset480 = [][]uint16{swbOffset480_48, swbOffset480_48, swbOffset480_48, swbOffset480_48, swbOffset480_48, swbOffset480_32, swbOffset480_24, swbOffset480_24, swbOffset480_24, swbOffset480_24, swbOffset480_24, swbOffset480_24, swbOffset480_24}
var SWBWindowCount480 = []uint8{35, 35, 35, 35, 35, 37, 30, 30, 30, 30, 30, 30, 30}

package tables

import "testing"

func TestSWBOffset512Lengths(t *testing.T) {
	if len(SWBOffset512) != 13 {
		t.Fatalf("SWBOffset512 length = %d, want 13", len(SWBOffset512))
	}
	for i, tbl := range SWBOffset512 {
		count := int(SWBWindowCount512[i])
		if len(tbl) != count+1 {
			t.Errorf("SWBOffset512[%d] length = %d, want %d", i, len(tbl), count+1)
		}
		// Last entry must equal 512
		if tbl[len(tbl)-1] != 512 {
			t.Errorf("SWBOffset512[%d] last = %d, want 512", i, tbl[len(tbl)-1])
		}
		// Offsets must be monotonically increasing
		for j := 1; j < len(tbl); j++ {
			if tbl[j] <= tbl[j-1] {
				t.Errorf("SWBOffset512[%d][%d] = %d not > %d", i, j, tbl[j], tbl[j-1])
			}
		}
	}
}

func TestSWBOffset480Lengths(t *testing.T) {
	if len(SWBOffset480) != 13 {
		t.Fatalf("SWBOffset480 length = %d, want 13", len(SWBOffset480))
	}
	for i, tbl := range SWBOffset480 {
		count := int(SWBWindowCount480[i])
		if len(tbl) != count+1 {
			t.Errorf("SWBOffset480[%d] length = %d, want %d", i, len(tbl), count+1)
		}
		// Last entry must equal 480
		if tbl[len(tbl)-1] != 480 {
			t.Errorf("SWBOffset480[%d] last = %d, want 480", i, tbl[len(tbl)-1])
		}
		// Offsets must be monotonically increasing
		for j := 1; j < len(tbl); j++ {
			if tbl[j] <= tbl[j-1] {
				t.Errorf("SWBOffset480[%d][%d] = %d not > %d", i, j, tbl[j], tbl[j-1])
			}
		}
	}
}

func TestTNSMaxBandsELD(t *testing.T) {
	if len(TNSMaxBands512) != 13 {
		t.Fatalf("TNSMaxBands512 length = %d, want 13", len(TNSMaxBands512))
	}
	if len(TNSMaxBands480) != 13 {
		t.Fatalf("TNSMaxBands480 length = %d, want 13", len(TNSMaxBands480))
	}
	for i := 0; i < 13; i++ {
		if TNSMaxBands512[i] <= 0 {
			t.Errorf("TNSMaxBands512[%d] = %d, want > 0", i, TNSMaxBands512[i])
		}
		if TNSMaxBands480[i] <= 0 {
			t.Errorf("TNSMaxBands480[%d] = %d, want > 0", i, TNSMaxBands480[i])
		}
	}
}

func TestGenerateMDCTTable(t *testing.T) {
	for _, n := range []int{1024, 960} {
		tbl := GenerateMDCTTable(n)
		n4 := n / 4
		if len(tbl) != n4 {
			t.Errorf("GenerateMDCTTable(%d) length = %d, want %d", n, len(tbl), n4)
		}
		// First entry: cos and sin should be close to sqrt(2/N) and ~0
		if tbl[0][0] <= 0 {
			t.Errorf("GenerateMDCTTable(%d)[0][0] = %f, want > 0", n, tbl[0][0])
		}
		if tbl[0][1] <= 0 {
			t.Errorf("GenerateMDCTTable(%d)[0][1] = %f, want > 0", n, tbl[0][1])
		}
	}
}

func TestLowDelaySynthesis512(t *testing.T) {
	win := LowDelaySynthesis512()
	if len(win) != 1536 {
		t.Fatalf("LowDelaySynthesis512 length = %d, want 1536", len(win))
	}
	// All values should be in [-1, 1]
	for i, v := range win {
		if v < -1.0 || v > 1.0 {
			t.Errorf("LowDelaySynthesis512[%d] = %f, out of range [-1, 1]", i, v)
			break
		}
	}
}

func TestLowDelaySynthesis480(t *testing.T) {
	win := LowDelaySynthesis480()
	if len(win) != 1440 {
		t.Fatalf("LowDelaySynthesis480 length = %d, want 1440", len(win))
	}
	// All values should be in [-1, 1]
	for i, v := range win {
		if v < -1.0 || v > 1.0 {
			t.Errorf("LowDelaySynthesis480[%d] = %f, out of range [-1, 1]", i, v)
			break
		}
	}
}

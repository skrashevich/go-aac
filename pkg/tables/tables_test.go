package tables

import (
	"math"
	"testing"
)

// Test constants
const (
	float64Epsilon = 1e-9
	float32Epsilon = 1e-6
)

// floatEquals compares two float64 values with epsilon tolerance
func floatEquals(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

// float32Equals compares two float32 values with epsilon tolerance
func float32Equals(a, b, epsilon float32) bool {
	return float32(math.Abs(float64(a-b))) < epsilon
}

// TestMDCTTable2048Size verifies that MDCTTable2048 contains exactly 512 pairs
func TestMDCTTable2048Size(t *testing.T) {
	expected := 512
	actual := len(MDCTTable2048)
	if actual != expected {
		t.Errorf("MDCTTable2048 size mismatch: expected %d, got %d", expected, actual)
	}
}

// TestMDCTTable2048Values verifies that MDCTTable2048 values match JavaScript version
func TestMDCTTable2048Values(t *testing.T) {
	// Test first 10 elements
	first10 := [][2]float64{
		{0.031249997702054, 0.000011984224612},
		{0.031249813866531, 0.000107857810004},
		{0.031249335895858, 0.000203730380198},
		{0.031248563794535, 0.000299601032804},
		{0.031247497569829, 0.000395468865451},
		{0.031246137231775, 0.000491332975794},
		{0.031244482793177, 0.000587192461525},
		{0.031242534269608, 0.000683046420376},
		{0.031240291679407, 0.000778893950134},
		{0.031237755043684, 0.000874734148645},
	}

	for i, expected := range first10 {
		if !floatEquals(MDCTTable2048[i][0], expected[0], float64Epsilon) {
			t.Errorf("MDCTTable2048[%d][0] mismatch: expected %.15f, got %.15f", i, expected[0], MDCTTable2048[i][0])
		}
		if !floatEquals(MDCTTable2048[i][1], expected[1], float64Epsilon) {
			t.Errorf("MDCTTable2048[%d][1] mismatch: expected %.15f, got %.15f", i, expected[1], MDCTTable2048[i][1])
		}
	}

	// Test last 5 elements (indices 507-511)
	last5 := [][2]float64{
		{0.000467367346520, 0.031246504888762},
		{0.000371502221008, 0.031247791699571},
		{0.000275633598775, 0.031248784394264},
		{0.000179762382174, 0.031249482963498},
		{0.000083889473581, 0.031249887400697},
	}

	for i, expected := range last5 {
		idx := 507 + i
		if !floatEquals(MDCTTable2048[idx][0], expected[0], float64Epsilon) {
			t.Errorf("MDCTTable2048[%d][0] mismatch: expected %.15f, got %.15f", idx, expected[0], MDCTTable2048[idx][0])
		}
		if !floatEquals(MDCTTable2048[idx][1], expected[1], float64Epsilon) {
			t.Errorf("MDCTTable2048[%d][1] mismatch: expected %.15f, got %.15f", idx, expected[1], MDCTTable2048[idx][1])
		}
	}

	// Test middle elements (indices 255, 256, 257)
	middle := map[int][2]float64{
		255: {0.022156326107988, 0.022037688476709},
		256: {0.022088611160696, 0.022105559413676},
		257: {0.022020688306983, 0.022173222284699},
	}

	for idx, expected := range middle {
		if !floatEquals(MDCTTable2048[idx][0], expected[0], float64Epsilon) {
			t.Errorf("MDCTTable2048[%d][0] mismatch: expected %.15f, got %.15f", idx, expected[0], MDCTTable2048[idx][0])
		}
		if !floatEquals(MDCTTable2048[idx][1], expected[1], float64Epsilon) {
			t.Errorf("MDCTTable2048[%d][1] mismatch: expected %.15f, got %.15f", idx, expected[1], MDCTTable2048[idx][1])
		}
	}
}

// TestMDCTTable256Size verifies that MDCTTable256 contains exactly 64 pairs
func TestMDCTTable256Size(t *testing.T) {
	expected := 64
	actual := len(MDCTTable256)
	if actual != expected {
		t.Errorf("MDCTTable256 size mismatch: expected %d, got %d", expected, actual)
	}
}

// TestMDCTTable256Values verifies that MDCTTable256 values match JavaScript version
func TestMDCTTable256Values(t *testing.T) {
	// Test first 10 elements
	first10 := [][2]float64{
		{0.088387931675923, 0.000271171628935},
		{0.088354655998507, 0.002440238387037},
		{0.088268158780110, 0.004607835236780},
		{0.088128492123423, 0.006772656498875},
		{0.087935740158418, 0.008933398165942},
		{0.087690018991670, 0.011088758687994},
		{0.087391476636423, 0.013237439756448},
		{0.087040292923427, 0.015378147086172},
		{0.086636679392621, 0.017509591195118},
		{0.086180879165703, 0.019630488181053},
	}

	for i, expected := range first10 {
		if !floatEquals(MDCTTable256[i][0], expected[0], float64Epsilon) {
			t.Errorf("MDCTTable256[%d][0] mismatch: expected %.15f, got %.15f", i, expected[0], MDCTTable256[i][0])
		}
		if !floatEquals(MDCTTable256[i][1], expected[1], float64Epsilon) {
			t.Errorf("MDCTTable256[%d][1] mismatch: expected %.15f, got %.15f", i, expected[1], MDCTTable256[i][1])
		}
	}

	// Test last 5 elements (indices 59-63)
	last5 := [][2]float64{
		{0.010550494103830, 0.087756407596056},
		{0.008393666439096, 0.087988899093631},
		{0.006231782743558, 0.088168389368510},
		{0.004066145255116, 0.088294770302461},
		{0.001898058472816, 0.088367965768336},
	}

	for i, expected := range last5 {
		idx := 59 + i
		if !floatEquals(MDCTTable256[idx][0], expected[0], float64Epsilon) {
			t.Errorf("MDCTTable256[%d][0] mismatch: expected %.15f, got %.15f", idx, expected[0], MDCTTable256[idx][0])
		}
		if !floatEquals(MDCTTable256[idx][1], expected[1], float64Epsilon) {
			t.Errorf("MDCTTable256[%d][1] mismatch: expected %.15f, got %.15f", idx, expected[1], MDCTTable256[idx][1])
		}
	}
}

// TestMDCTTable1920Size verifies that MDCTTable1920 contains exactly 480 pairs
func TestMDCTTable1920Size(t *testing.T) {
	expected := 480
	actual := len(MDCTTable1920)
	if actual != expected {
		t.Errorf("MDCTTable1920 size mismatch: expected %d, got %d", expected, actual)
	}
}

// TestMDCTTable1920Values verifies that MDCTTable1920 values match JavaScript version
func TestMDCTTable1920Values(t *testing.T) {
	// Test first 10 elements
	first10 := [][2]float64{
		{0.032274858518097, 0.000013202404176},
		{0.032274642494505, 0.000118821372483},
		{0.032274080835421, 0.000224439068308},
		{0.032273173546860, 0.000330054360572},
		{0.032271920638538, 0.000435666118218},
		{0.032270322123873, 0.000541273210231},
		{0.032268378019984, 0.000646874505642},
		{0.032266088347691, 0.000752468873546},
		{0.032263453131514, 0.000858055183114},
		{0.032260472389941, 0.000963632303600},
	}

	for i, expected := range first10 {
		if !floatEquals(MDCTTable1920[i][0], expected[0], float64Epsilon) {
			t.Errorf("MDCTTable1920[%d][0] mismatch: expected %.15f, got %.15f", i, expected[0], MDCTTable1920[i][0])
		}
		if !floatEquals(MDCTTable1920[i][1], expected[1], float64Epsilon) {
			t.Errorf("MDCTTable1920[%d][1] mismatch: expected %.15f, got %.15f", i, expected[1], MDCTTable1920[i][1])
		}
	}

	// Test last 5 elements (indices 475-479)
	last5 := [][2]float64{
		{0.000514871936481, 0.032270754152261},
		{0.000409263572030, 0.032272266266801},
		{0.000303650824695, 0.032273432771295},
		{0.000198034825504, 0.032274253653254},
		{0.000092416705518, 0.032274728903884},
	}

	for i, expected := range last5 {
		idx := 475 + i
		if !floatEquals(MDCTTable1920[idx][0], expected[0], float64Epsilon) {
			t.Errorf("MDCTTable1920[%d][0] mismatch: expected %.15f, got %.15f", idx, expected[0], MDCTTable1920[idx][0])
		}
		if !floatEquals(MDCTTable1920[idx][1], expected[1], float64Epsilon) {
			t.Errorf("MDCTTable1920[%d][1] mismatch: expected %.15f, got %.15f", idx, expected[1], MDCTTable1920[idx][1])
		}
	}
}

// TestMDCTTable240Size verifies that MDCTTable240 contains exactly 60 pairs
func TestMDCTTable240Size(t *testing.T) {
	expected := 60
	actual := len(MDCTTable240)
	if actual != expected {
		t.Errorf("MDCTTable240 size mismatch: expected %d, got %d", expected, actual)
	}
}

// TestMDCTTable240Values verifies that MDCTTable240 values match JavaScript version
func TestMDCTTable240Values(t *testing.T) {
	// Test first 10 elements
	first10 := [][2]float64{
		{0.091286604111815, 0.000298735779793},
		{0.091247502481454, 0.002688238127538},
		{0.091145864370807, 0.005075898091152},
		{0.090981759437558, 0.007460079287760},
		{0.090755300151030, 0.009839147718664},
		{0.090466641715108, 0.012211472889198},
		{0.090115981961863, 0.014575428926191},
		{0.089703561215976, 0.016929395692256},
		{0.089229662130024, 0.019271759896156},
		{0.088694609490769, 0.021600916198470},
	}

	for i, expected := range first10 {
		if !floatEquals(MDCTTable240[i][0], expected[0], float64Epsilon) {
			t.Errorf("MDCTTable240[%d][0] mismatch: expected %.15f, got %.15f", i, expected[0], MDCTTable240[i][0])
		}
		if !floatEquals(MDCTTable240[i][1], expected[1], float64Epsilon) {
			t.Errorf("MDCTTable240[%d][1] mismatch: expected %.15f, got %.15f", i, expected[1], MDCTTable240[i][1])
		}
	}

	// Test last 5 elements (indices 55-59)
	last5 := [][2]float64{
		{0.011619112781631, 0.090544627402740},
		{0.009244949170797, 0.090817752935000},
		{0.006864449533597, 0.091028636515846},
		{0.004479245345574, 0.091177133616206},
		{0.002090971306534, 0.091263142463585},
	}

	for i, expected := range last5 {
		idx := 55 + i
		if !floatEquals(MDCTTable240[idx][0], expected[0], float64Epsilon) {
			t.Errorf("MDCTTable240[%d][0] mismatch: expected %.15f, got %.15f", idx, expected[0], MDCTTable240[idx][0])
		}
		if !floatEquals(MDCTTable240[idx][1], expected[1], float64Epsilon) {
			t.Errorf("MDCTTable240[%d][1] mismatch: expected %.15f, got %.15f", idx, expected[1], MDCTTable240[idx][1])
		}
	}
}

// TestSWBOffset1024Size verifies that SWBOffset1024 contains 12 elements
func TestSWBOffset1024Size(t *testing.T) {
	expected := 12
	actual := len(SWBOffset1024)
	if actual != expected {
		t.Errorf("SWBOffset1024 size mismatch: expected %d, got %d", expected, actual)
	}
}

// TestSWBOffset128Size verifies that SWBOffset128 contains 12 elements
func TestSWBOffset128Size(t *testing.T) {
	expected := 12
	actual := len(SWBOffset128)
	if actual != expected {
		t.Errorf("SWBOffset128 size mismatch: expected %d, got %d", expected, actual)
	}
}

// TestSWBOffset1024SubarraySizes verifies that each subarray has the expected number of elements
func TestSWBOffset1024SubarraySizes(t *testing.T) {
	// Expected sizes based on the Go implementation
	expectedSizes := []int{42, 42, 48, 50, 50, 52, 48, 48, 44, 44, 44, 41}

	for i, expectedSize := range expectedSizes {
		actualSize := len(SWBOffset1024[i])
		if actualSize != expectedSize {
			t.Errorf("SWBOffset1024[%d] size mismatch: expected %d, got %d", i, expectedSize, actualSize)
		}
	}
}

// TestSWBOffset128SubarraySizes verifies that each subarray has the expected number of elements
func TestSWBOffset128SubarraySizes(t *testing.T) {
	// Expected sizes based on the Go implementation
	expectedSizes := []int{13, 13, 13, 15, 15, 15, 16, 16, 16, 16, 16, 16}

	for i, expectedSize := range expectedSizes {
		actualSize := len(SWBOffset128[i])
		if actualSize != expectedSize {
			t.Errorf("SWBOffset128[%d] size mismatch: expected %d, got %d", i, expectedSize, actualSize)
		}
	}
}

// TestSWBOffset1024Values verifies key values in the SWB offset tables
func TestSWBOffset1024Values(t *testing.T) {
	tests := []struct {
		index    int
		position int
		expected uint16
	}{
		{0, 0, 0},      // 96000 Hz - first element
		{0, 41, 1024},  // 96000 Hz - last element
		{3, 0, 0},      // 48000 Hz - first element
		{3, 49, 1024},  // 48000 Hz - last element
		{11, 0, 0},     // 8000 Hz - first element
		{11, 40, 1024}, // 8000 Hz - last element
	}

	for _, tt := range tests {
		actual := SWBOffset1024[tt.index][tt.position]
		if actual != tt.expected {
			t.Errorf("SWBOffset1024[%d][%d] mismatch: expected %d, got %d",
				tt.index, tt.position, tt.expected, actual)
		}
	}
}

// TestSWBOffset128Values verifies key values in the short SWB offset tables
func TestSWBOffset128Values(t *testing.T) {
	tests := []struct {
		index    int
		position int
		expected uint16
	}{
		{0, 0, 0},    // 96000 Hz - first element
		{0, 12, 128}, // 96000 Hz - last element
		{3, 0, 0},    // 48000 Hz - first element
		{3, 14, 128}, // 48000 Hz - last element
		{11, 0, 0},   // 8000 Hz - first element
		{11, 15, 128}, // 8000 Hz - last element
	}

	for _, tt := range tests {
		actual := SWBOffset128[tt.index][tt.position]
		if actual != tt.expected {
			t.Errorf("SWBOffset128[%d][%d] mismatch: expected %d, got %d",
				tt.index, tt.position, tt.expected, actual)
		}
	}
}

// TestSWBWindowCountsSizes verifies the sizes of window count arrays
func TestSWBWindowCountsSizes(t *testing.T) {
	if len(SWBShortWindowCount) != 12 {
		t.Errorf("SWBShortWindowCount size mismatch: expected 12, got %d", len(SWBShortWindowCount))
	}
	if len(SWBLongWindowCount) != 12 {
		t.Errorf("SWBLongWindowCount size mismatch: expected 12, got %d", len(SWBLongWindowCount))
	}
}

// TestSWBWindowCountsValues verifies the values in window count arrays
func TestSWBWindowCountsValues(t *testing.T) {
	expectedShort := []uint8{12, 12, 12, 14, 14, 14, 15, 15, 15, 15, 15, 15}
	expectedLong := []uint8{41, 41, 47, 49, 49, 51, 47, 47, 43, 43, 43, 40}

	for i := 0; i < 12; i++ {
		if SWBShortWindowCount[i] != expectedShort[i] {
			t.Errorf("SWBShortWindowCount[%d] mismatch: expected %d, got %d",
				i, expectedShort[i], SWBShortWindowCount[i])
		}
		if SWBLongWindowCount[i] != expectedLong[i] {
			t.Errorf("SWBLongWindowCount[%d] mismatch: expected %d, got %d",
				i, expectedLong[i], SWBLongWindowCount[i])
		}
	}
}

// TestScalefactorTableSize verifies that ScalefactorTable contains 428 elements
func TestScalefactorTableSize(t *testing.T) {
	expected := 428
	actual := len(ScalefactorTable)
	if actual != expected {
		t.Errorf("ScalefactorTable size mismatch: expected %d, got %d", expected, actual)
	}
}

// TestScalefactorTableFormula verifies that ScalefactorTable[i] == 2^((i-200)/4)
func TestScalefactorTableFormula(t *testing.T) {
	testIndices := []int{0, 50, 100, 150, 200, 250, 300, 350, 400, 427}

	for _, i := range testIndices {
		expected := float32(math.Pow(2, float64(i-200)/4))
		actual := ScalefactorTable[i]
		if !float32Equals(actual, expected, float32Epsilon) {
			t.Errorf("ScalefactorTable[%d] formula mismatch: expected %.9f, got %.9f",
				i, expected, actual)
		}
	}
}

// TestScalefactorTableValues verifies specific values from the computed table
func TestScalefactorTableValues(t *testing.T) {
	tests := []struct {
		index    int
		expected float32
	}{
		{0, float32(math.Pow(2, -50))},    // 2^(-200/4)
		{200, 1.0},                         // 2^0 = 1
		{204, float32(math.Pow(2, 1))},    // 2^(4/4) = 2
		{400, float32(math.Pow(2, 50))},   // 2^(200/4)
	}

	for _, tt := range tests {
		actual := ScalefactorTable[tt.index]
		if !float32Equals(actual, tt.expected, float32Epsilon) {
			t.Errorf("ScalefactorTable[%d] mismatch: expected %.9f, got %.9f",
				tt.index, tt.expected, actual)
		}
	}
}

// TestIQTableSize verifies that IQTable contains 8191 elements
func TestIQTableSize(t *testing.T) {
	expected := 8191
	actual := len(IQTable)
	if actual != expected {
		t.Errorf("IQTable size mismatch: expected %d, got %d", expected, actual)
	}
}

// TestIQTableFormula verifies that IQTable[i] == i^(4/3)
func TestIQTableFormula(t *testing.T) {
	testIndices := []int{0, 1, 10, 100, 500, 1000, 2000, 4000, 8000, 8190}

	for _, i := range testIndices {
		expected := float32(math.Pow(float64(i), 4.0/3.0))
		actual := IQTable[i]
		if !float32Equals(actual, expected, float32Epsilon) {
			t.Errorf("IQTable[%d] formula mismatch: expected %.9f, got %.9f",
				i, expected, actual)
		}
	}
}

// TestIQTableValues verifies specific values from the computed table
func TestIQTableValues(t *testing.T) {
	tests := []struct {
		index    int
		expected float32
	}{
		{0, 0},                                        // 0^(4/3) = 0
		{1, 1},                                        // 1^(4/3) = 1
		{8, float32(math.Pow(8, 4.0/3.0))},           // 8^(4/3) = 16
		{27, float32(math.Pow(27, 4.0/3.0))},         // 27^(4/3) = 81
		{64, float32(math.Pow(64, 4.0/3.0))},         // 64^(4/3) = 256
	}

	for _, tt := range tests {
		actual := IQTable[tt.index]
		if !float32Equals(actual, tt.expected, float32Epsilon) {
			t.Errorf("IQTable[%d] mismatch: expected %.9f, got %.9f",
				tt.index, tt.expected, actual)
		}
	}
}

// TestSampleRatesSize verifies that SampleRates contains 13 elements
func TestSampleRatesSize(t *testing.T) {
	expected := 13
	actual := len(SampleRates)
	if actual != expected {
		t.Errorf("SampleRates size mismatch: expected %d, got %d", expected, actual)
	}
}

// TestSampleRatesValues verifies all sample rate values match JavaScript version
func TestSampleRatesValues(t *testing.T) {
	expected := []int32{
		96000, 88200, 64000, 48000, 44100, 32000,
		24000, 22050, 16000, 12000, 11025, 8000, 7350,
	}

	for i, exp := range expected {
		actual := SampleRates[i]
		if actual != exp {
			t.Errorf("SampleRates[%d] mismatch: expected %d, got %d", i, exp, actual)
		}
	}
}

// TestSampleRatesIndexMapping verifies the mapping between index and sample rate
func TestSampleRatesIndexMapping(t *testing.T) {
	tests := []struct {
		index      int
		sampleRate int32
		desc       string
	}{
		{0, 96000, "96 kHz"},
		{1, 88200, "88.2 kHz"},
		{2, 64000, "64 kHz"},
		{3, 48000, "48 kHz"},
		{4, 44100, "44.1 kHz"},
		{5, 32000, "32 kHz"},
		{6, 24000, "24 kHz"},
		{7, 22050, "22.05 kHz"},
		{8, 16000, "16 kHz"},
		{9, 12000, "12 kHz"},
		{10, 11025, "11.025 kHz"},
		{11, 8000, "8 kHz"},
		{12, 7350, "7.35 kHz"},
	}

	for _, tt := range tests {
		actual := SampleRates[tt.index]
		if actual != tt.sampleRate {
			t.Errorf("SampleRates[%d] (%s) mismatch: expected %d, got %d",
				tt.index, tt.desc, tt.sampleRate, actual)
		}
	}
}

// BenchmarkScalefactorTableComputation benchmarks the scalefactor table generation
func BenchmarkScalefactorTableComputation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		table := make([]float32, 428)
		for j := 0; j < 428; j++ {
			table[j] = float32(math.Pow(2, float64(j-200)/4))
		}
	}
}

// BenchmarkIQTableComputation benchmarks the IQ table generation
func BenchmarkIQTableComputation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		table := make([]float32, 8191)
		for j := 0; j < 8191; j++ {
			table[j] = float32(math.Pow(float64(j), 4.0/3.0))
		}
	}
}

// BenchmarkScalefactorTableLookup benchmarks scalefactor table access
func BenchmarkScalefactorTableLookup(b *testing.B) {
	var sum float32
	for i := 0; i < b.N; i++ {
		sum += ScalefactorTable[i%428]
	}
	// Prevent compiler optimization
	_ = sum
}

// BenchmarkIQTableLookup benchmarks IQ table access
func BenchmarkIQTableLookup(b *testing.B) {
	var sum float32
	for i := 0; i < b.N; i++ {
		sum += IQTable[i%8191]
	}
	// Prevent compiler optimization
	_ = sum
}

package fft

import (
	"math"
	"math/cmplx"
	"testing"
)

// ---------- Constructor / New ----------

func TestNew_SupportedLengths(t *testing.T) {
	for _, n := range []int{64, 512, 256, 60, 480, 240} {
		f, err := New(n)
		if err != nil {
			t.Fatalf("New(%d) returned unexpected error: %v", n, err)
		}
		if f.Length() != n {
			t.Fatalf("New(%d).Length() = %d, want %d", n, f.Length(), n)
		}
	}
}

func TestNew_UnsupportedLength(t *testing.T) {
	unsupported := []int{0, 1, 2, 32, 128, 1024, -1, 100}
	for _, n := range unsupported {
		f, err := New(n)
		if err == nil {
			t.Errorf("New(%d) expected error, got nil (FFT=%+v)", n, f)
		}
		if f != nil {
			t.Errorf("New(%d) expected nil FFT on error, got non-nil", n)
		}
	}
}

// ---------- Table Generation ----------

func TestGenerateTableShort_Length(t *testing.T) {
	for _, n := range []int{60, 64, 240} {
		table := generateTableShort(n)
		if len(table) != n {
			t.Errorf("generateTableShort(%d): got length %d, want %d", n, len(table), n)
		}
	}
}

func TestGenerateTableShort_FirstEntry(t *testing.T) {
	for _, n := range []int{60, 64, 240} {
		table := generateTableShort(n)
		if table[0][0] != 1.0 || table[0][1] != 0.0 {
			t.Errorf("generateTableShort(%d)[0] = [%f, %f], want [1, 0]", n, table[0][0], table[0][1])
		}
	}
}

func TestGenerateTableShort_UnitCircle(t *testing.T) {
	// Each twiddle factor should lie on the unit circle: |re|^2 + |im|^2 ~ 1
	for _, n := range []int{60, 64, 240} {
		table := generateTableShort(n)
		for i := 0; i < n; i++ {
			re := float64(table[i][0])
			// table[i][1] is -imag, so imag = -table[i][1]
			im := -float64(table[i][1])
			mag := re*re + im*im
			if math.Abs(mag-1.0) > 1e-4 {
				t.Errorf("generateTableShort(%d)[%d]: magnitude = %f, want ~1.0", n, i, mag)
			}
		}
	}
}

func TestGenerateTableShort_MatchesDirect(t *testing.T) {
	// Verify against directly computed twiddle factors: e^{-2*pi*i*k/N}
	for _, n := range []int{60, 64, 240} {
		table := generateTableShort(n)
		for k := 0; k < n; k++ {
			angle := -2.0 * math.Pi * float64(k) / float64(n)
			wantRe := float32(math.Cos(angle))
			wantNegIm := float32(-math.Sin(angle)) // store -imag

			if !approxEq32(table[k][0], wantRe, 1e-4) {
				t.Errorf("generateTableShort(%d)[%d][0] (re) = %f, want %f", n, k, table[k][0], wantRe)
			}
			// The JS code stores -lastImag as f[i][1], where lastImag tracks
			// the positive imaginary part. So table[k][1] = -imag = sin(angle)
			// since angle is negative: sin(-2*pi*k/N)
			if !approxEq32(table[k][1], wantNegIm, 1e-4) {
				t.Errorf("generateTableShort(%d)[%d][1] (-im) = %f, want %f", n, k, table[k][1], wantNegIm)
			}
		}
	}
}

func TestGenerateTableLong_Length(t *testing.T) {
	for _, n := range []int{480, 512, 256} {
		table := generateTableLong(n)
		if len(table) != n {
			t.Errorf("generateTableLong(%d): got length %d, want %d", n, len(table), n)
		}
	}
}

func TestGenerateTableLong_FirstEntry(t *testing.T) {
	for _, n := range []int{480, 512, 256} {
		table := generateTableLong(n)
		if table[0][0] != 1.0 || table[0][1] != 0.0 || table[0][2] != 0.0 {
			t.Errorf("generateTableLong(%d)[0] = [%f, %f, %f], want [1, 0, 0]",
				n, table[0][0], table[0][1], table[0][2])
		}
	}
}

func TestGenerateTableLong_NegImRelation(t *testing.T) {
	// For long tables, entry[1] = -entry[2] (negated imaginary).
	for _, n := range []int{480, 512, 256} {
		table := generateTableLong(n)
		for i := 0; i < n; i++ {
			if !approxEq32(table[i][1], -table[i][2], 1e-7) {
				t.Errorf("generateTableLong(%d)[%d]: [1]=%f should be -[2]=%f",
					n, i, table[i][1], table[i][2])
			}
		}
	}
}

func TestGenerateTableLong_UnitCircle(t *testing.T) {
	for _, n := range []int{480, 512} {
		table := generateTableLong(n)
		for i := 0; i < n; i++ {
			re := float64(table[i][0])
			im := float64(table[i][2])
			mag := re*re + im*im
			if math.Abs(mag-1.0) > 1e-4 {
				t.Errorf("generateTableLong(%d)[%d]: magnitude = %f, want ~1.0", n, i, mag)
			}
		}
	}
}

// ---------- Process: basic smoke tests ----------

func TestProcess_AllZeros(t *testing.T) {
	// FFT of all zeros should remain all zeros.
	for _, n := range []int{64, 512} {
		f, err := New(n)
		if err != nil {
			t.Fatal(err)
		}
		data := make([][2]float32, n)
		f.Process(data, false)
		for i := range data {
			if data[i][0] != 0 || data[i][1] != 0 {
				t.Errorf("IFFT of zeros (n=%d): data[%d] = [%f, %f], want [0, 0]",
					n, i, data[i][0], data[i][1])
				break
			}
		}
	}
}

func TestProcess_SingleDC(t *testing.T) {
	// A constant signal in the frequency domain (all bins = [1,0]) should
	// give a delta function after IFFT: output[0] = [N, 0] and rest ~0.
	// (non-scaling IFFT as used in the JS code)
	for _, n := range []int{64, 512} {
		f, err := New(n)
		if err != nil {
			t.Fatal(err)
		}
		data := make([][2]float32, n)
		for i := range data {
			data[i][0] = 1.0
			data[i][1] = 0.0
		}
		f.Process(data, false)

		// After non-scaling IFFT, output[0] should be N.
		nf := float32(n)
		if !approxEq32(data[0][0], nf, 0.5) {
			t.Errorf("IFFT constant (n=%d): data[0][0] = %f, want ~%f", n, data[0][0], nf)
		}
		// All other bins should be near zero.
		for i := 1; i < n; i++ {
			if math.Abs(float64(data[i][0])) > 0.5 || math.Abs(float64(data[i][1])) > 0.5 {
				t.Errorf("IFFT constant (n=%d): data[%d] = [%f, %f], want ~[0, 0]",
					n, i, data[i][0], data[i][1])
				break
			}
		}
	}
}

// ---------- Process: comparison with naive DFT ----------

// naiveDFT computes the DFT or IDFT using the textbook O(N^2) formula.
// If inverse is true, computes the inverse (non-normalized) DFT.
func naiveDFT(input []complex128, inverse bool) []complex128 {
	n := len(input)
	output := make([]complex128, n)
	sign := -1.0
	if inverse {
		sign = 1.0
	}
	for k := 0; k < n; k++ {
		var sum complex128
		for j := 0; j < n; j++ {
			angle := sign * 2.0 * math.Pi * float64(k) * float64(j) / float64(n)
			w := cmplx.Rect(1.0, angle)
			sum += input[j] * w
		}
		output[k] = sum
	}
	return output
}

func TestProcess_IFFT_MatchesNaive_64(t *testing.T) {
	testIFFTMatchesNaive(t, 64)
}

func TestProcess_IFFT_MatchesNaive_60(t *testing.T) {
	testIFFTMatchesNaive(t, 60)
}

func TestProcess_IFFT_MatchesNaive_512(t *testing.T) {
	testIFFTMatchesNaive(t, 512)
}

func TestProcess_IFFT_MatchesNaive_480(t *testing.T) {
	testIFFTMatchesNaive(t, 480)
}

func testIFFTMatchesNaive(t *testing.T, n int) {
	t.Helper()

	f, err := New(n)
	if err != nil {
		t.Fatal(err)
	}

	// Build test input: some deterministic non-trivial data.
	input := make([][2]float32, n)
	inputC := make([]complex128, n)
	for i := 0; i < n; i++ {
		re := float32(math.Sin(float64(i)*0.1)) * 10
		im := float32(math.Cos(float64(i)*0.3)) * 5
		input[i][0] = re
		input[i][1] = im
		inputC[i] = complex(float64(re), float64(im))
	}

	// Compute via our FFT (IFFT mode).
	f.Process(input, false)

	// Compute via naive IDFT.
	expected := naiveDFT(inputC, true)

	// Compare. float32 precision is about 7 decimal digits, so we use a
	// relative/absolute tolerance appropriate for accumulated error in
	// lengths up to 512.
	tol := float64(0.05) // generous tolerance for float32 accumulated error
	if n > 128 {
		tol = 0.5 // longer transforms accumulate more rounding error
	}

	for i := 0; i < n; i++ {
		gotRe := float64(input[i][0])
		gotIm := float64(input[i][1])
		wantRe := real(expected[i])
		wantIm := imag(expected[i])

		if math.Abs(gotRe-wantRe) > tol || math.Abs(gotIm-wantIm) > tol {
			t.Errorf("IFFT (n=%d)[%d]: got [%f, %f], want [%f, %f] (tol=%f)",
				n, i, gotRe, gotIm, wantRe, wantIm, tol)
			if i > 5 {
				t.Logf("... (stopping after 5 mismatches)")
				break
			}
		}
	}
}

// ---------- Process: roundtrip forward+inverse (for long lengths only) ----------

func TestProcess_Roundtrip_512(t *testing.T) {
	testRoundtrip(t, 512)
}

func TestProcess_Roundtrip_480(t *testing.T) {
	testRoundtrip(t, 480)
}

func testRoundtrip(t *testing.T, n int) {
	t.Helper()

	f, err := New(n)
	if err != nil {
		t.Fatal(err)
	}

	// Build test input.
	original := make([][2]float32, n)
	data := make([][2]float32, n)
	for i := 0; i < n; i++ {
		re := float32(math.Sin(float64(i)*0.7)) * 3
		im := float32(math.Cos(float64(i)*0.2)) * 7
		original[i][0] = re
		original[i][1] = im
		data[i][0] = re
		data[i][1] = im
	}

	// Forward FFT, then inverse FFT.
	f.Process(data, true)
	f.Process(data, false)

	// After forward + inverse with the scaling used in this implementation,
	// we need to understand the combined scaling. The forward pass scales
	// each butterfly level by N, and the inverse pass does not scale (scale=1).
	// The number of radix-2 levels after the initial radix-4 pass is
	// log2(N)-2 for power-of-two, or similar for non-power-of-two.
	//
	// Since the scaling is non-standard (specific to AAC decoder usage),
	// we verify the roundtrip qualitatively: the shape should be preserved
	// even if there is an overall scale factor.
	//
	// Find the scale factor from the first non-zero element.
	var scaleFactor float64
	for i := 0; i < n; i++ {
		if math.Abs(float64(original[i][0])) > 0.01 {
			scaleFactor = float64(data[i][0]) / float64(original[i][0])
			break
		}
	}

	if scaleFactor == 0 {
		t.Fatal("could not determine scale factor")
	}

	tol := math.Abs(scaleFactor) * 0.01 // 1% relative tolerance
	for i := 0; i < n; i++ {
		gotRe := float64(data[i][0]) / scaleFactor
		gotIm := float64(data[i][1]) / scaleFactor
		wantRe := float64(original[i][0])
		wantIm := float64(original[i][1])

		if math.Abs(gotRe-wantRe) > tol || math.Abs(gotIm-wantIm) > tol {
			t.Errorf("Roundtrip (n=%d)[%d]: got [%f, %f], want [%f, %f] (scale=%e, tol=%f)",
				n, i, gotRe, gotIm, wantRe, wantIm, scaleFactor, tol)
			if i > 5 {
				break
			}
		}
	}
}

// ---------- Bit-reversal correctness ----------

func TestBitReversal_PowerOfTwo(t *testing.T) {
	// For N=8 (not a supported FFT length, but we can test the bit-reversal
	// logic by using N=64 and checking specific indices).
	n := 64
	f, err := New(n)
	if err != nil {
		t.Fatal(err)
	}

	// Create input where each element is uniquely identifiable.
	input := make([][2]float32, n)
	for i := 0; i < n; i++ {
		input[i][0] = float32(i)
		input[i][1] = 0
	}

	// Capture the bit-reversed order by running only the bit-reversal part.
	// We can do this indirectly: create a copy and process, then check the
	// permutation is consistent.
	// Instead, let us verify known bit-reversal pairs for N=64.
	// bit_reverse(1, 6bits) = 32, bit_reverse(2, 6bits) = 16, etc.
	//
	// For the full FFT, we instead just verify the overall result is correct
	// by comparing with the naive DFT (already tested above). This test
	// ensures that the Process function does not crash and produces valid output.
	f.Process(input, false)

	// Just verify no NaN or Inf values.
	for i := 0; i < n; i++ {
		if math.IsNaN(float64(input[i][0])) || math.IsInf(float64(input[i][0]), 0) {
			t.Errorf("n=%d: data[%d][0] is NaN/Inf", n, i)
		}
		if math.IsNaN(float64(input[i][1])) || math.IsInf(float64(input[i][1]), 0) {
			t.Errorf("n=%d: data[%d][1] is NaN/Inf", n, i)
		}
	}
}

// ---------- Edge cases ----------

func TestProcess_SingleFrequency(t *testing.T) {
	// Place a single tone at bin 1 and verify the IFFT produces a sinusoid.
	n := 64
	f, err := New(n)
	if err != nil {
		t.Fatal(err)
	}

	data := make([][2]float32, n)
	data[1][0] = float32(n) // amplitude at bin 1

	f.Process(data, false)

	// After non-scaling IFFT of a single tone at bin 1, we expect:
	// x[k] = N * e^{2*pi*i*k*1/N} = N * (cos(2*pi*k/N) + i*sin(2*pi*k/N))
	// Since we put N in bin 1, output should be cos/sin with amplitude N.
	// But the FFT convention here may differ, so we just check the shape:
	// the real and imaginary parts should oscillate sinusoidally.
	nf := float64(n)
	tol := 0.5
	for k := 0; k < n; k++ {
		angle := 2.0 * math.Pi * float64(k) / nf
		wantRe := nf * math.Cos(angle)
		wantIm := nf * math.Sin(angle)
		gotRe := float64(data[k][0])
		gotIm := float64(data[k][1])

		if math.Abs(gotRe-wantRe) > tol || math.Abs(gotIm-wantIm) > tol {
			t.Errorf("SingleFreq (n=%d)[%d]: got [%f, %f], want [%f, %f]",
				n, k, gotRe, gotIm, wantRe, wantIm)
			if k > 5 {
				break
			}
		}
	}
}

func TestProcess_Parseval(t *testing.T) {
	// Parseval's theorem: sum |x[n]|^2 = (1/N) * sum |X[k]|^2
	// For the non-scaling IFFT used here: sum |x[n]|^2 = (1/N^2) * N * sum |X[k]|^2
	// Actually, let us just verify energy is preserved up to the known factor.
	n := 64
	f, err := New(n)
	if err != nil {
		t.Fatal(err)
	}

	data := make([][2]float32, n)
	var energyBefore float64
	for i := 0; i < n; i++ {
		re := float32(math.Sin(float64(i) * 0.5))
		im := float32(math.Cos(float64(i) * 0.3))
		data[i][0] = re
		data[i][1] = im
		energyBefore += float64(re)*float64(re) + float64(im)*float64(im)
	}

	f.Process(data, false)

	var energyAfter float64
	for i := 0; i < n; i++ {
		re := float64(data[i][0])
		im := float64(data[i][1])
		energyAfter += re*re + im*im
	}

	// For non-scaling IFFT: energyAfter should equal N * energyBefore.
	expectedRatio := float64(n)
	actualRatio := energyAfter / energyBefore

	if math.Abs(actualRatio-expectedRatio)/expectedRatio > 0.01 {
		t.Errorf("Parseval (n=%d): energy ratio = %f, want ~%f", n, actualRatio, expectedRatio)
	}
}

// ---------- Benchmarks ----------

func BenchmarkProcess_IFFT_64(b *testing.B) {
	benchmarkProcess(b, 64, false)
}

func BenchmarkProcess_IFFT_512(b *testing.B) {
	benchmarkProcess(b, 512, false)
}

func BenchmarkProcess_IFFT_60(b *testing.B) {
	benchmarkProcess(b, 60, false)
}

func BenchmarkProcess_IFFT_480(b *testing.B) {
	benchmarkProcess(b, 480, false)
}

func benchmarkProcess(b *testing.B, n int, forward bool) {
	b.Helper()
	f, err := New(n)
	if err != nil {
		b.Fatal(err)
	}
	data := make([][2]float32, n)
	for i := range data {
		data[i][0] = float32(i)
		data[i][1] = float32(-i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Process(data, forward)
	}
}

// ---------- Helpers ----------

func approxEq32(a, b float32, tol float64) bool {
	return math.Abs(float64(a)-float64(b)) <= tol
}

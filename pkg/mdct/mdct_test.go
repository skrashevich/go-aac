package mdct

import (
	"math"
	"math/rand"
	"testing"

	"github.com/skrashevich/go-aac/pkg/tables"
)

func referenceIMDCT(input []float32, n int) []float32 {
	var sincos [][2]float64
	switch n {
	case 2048:
		sincos = tables.MDCTTable2048
	case 256:
		sincos = tables.MDCTTable256
	case 1920:
		sincos = tables.MDCTTable1920
	case 240:
		sincos = tables.MDCTTable240
	default:
		return nil
	}

	N2 := n / 2
	N4 := n / 4
	N8 := n / 8

	pre := make([]complex128, N4)
	for k := 0; k < N4; k++ {
		in0 := float64(input[2*k])
		in1 := float64(input[N2-1-2*k])
		cos := sincos[k][0]
		sin := sincos[k][1]

		re := in1*cos - in0*sin
		im := in0*cos + in1*sin
		pre[k] = complex(re, im)
	}

	// Inverse DFT (unscaled), matching fft.Process with forward=false.
	fft := make([]complex128, N4)
	for k := 0; k < N4; k++ {
		sum := complex(0, 0)
		for n := 0; n < N4; n++ {
			angle := 2.0 * math.Pi * float64(n*k) / float64(N4)
			sum += pre[n] * complex(math.Cos(angle), math.Sin(angle))
		}
		fft[k] = sum
	}

	bufRe := make([]float64, N4)
	bufIm := make([]float64, N4)
	for k := 0; k < N4; k++ {
		tmp0 := real(fft[k])
		tmp1 := imag(fft[k])
		cos := sincos[k][0]
		sin := sincos[k][1]

		bufIm[k] = tmp1*cos + tmp0*sin
		bufRe[k] = tmp0*cos - tmp1*sin
	}

	output := make([]float32, n)
	for k := 0; k < N8; k += 2 {
		output[2*k] = float32(bufIm[N8+k])
		output[2+2*k] = float32(bufIm[N8+1+k])

		output[1+2*k] = float32(-bufRe[N8-1-k])
		output[3+2*k] = float32(-bufRe[N8-2-k])

		output[N4+2*k] = float32(bufRe[k])
		output[N4+2+2*k] = float32(bufRe[1+k])

		output[N4+1+2*k] = float32(-bufIm[N4-1-k])
		output[N4+3+2*k] = float32(-bufIm[N4-2-k])

		output[N2+2*k] = float32(bufRe[N8+k])
		output[N2+2+2*k] = float32(bufRe[N8+1+k])

		output[N2+1+2*k] = float32(-bufIm[N8-1-k])
		output[N2+3+2*k] = float32(-bufIm[N8-2-k])

		output[N2+N4+2*k] = float32(-bufIm[k])
		output[N2+N4+2+2*k] = float32(-bufIm[1+k])

		output[N2+N4+1+2*k] = float32(bufRe[N4-1-k])
		output[N2+N4+3+2*k] = float32(bufRe[N4-2-k])
	}

	return output
}

func maxAbsDiff(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.Inf(1))
	}
	max := float32(0)
	for i := range a {
		d := a[i] - b[i]
		if d < 0 {
			d = -d
		}
		if d > max {
			max = d
		}
	}
	return max
}

func maxAbs(a []float32) float32 {
	max := float32(0)
	for _, v := range a {
		if v < 0 {
			v = -v
		}
		if v > max {
			max = v
		}
	}
	return max
}

func TestNewUnsupportedLength(t *testing.T) {
	if _, err := New(512); err == nil {
		t.Fatal("expected error for unsupported length")
	}
}

func TestNewELDLengths(t *testing.T) {
	for _, n := range []int{1024, 960} {
		m, err := New(n)
		if err != nil {
			t.Fatalf("New(%d) failed: %v", n, err)
		}
		if m.Length() != n {
			t.Errorf("New(%d).Length() = %d, want %d", n, m.Length(), n)
		}
	}
}

func TestProcessMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	sizes := []int{2048, 256, 1920, 240}
	for _, n := range sizes {
		m, err := New(n)
		if err != nil {
			t.Fatalf("New(%d) failed: %v", n, err)
		}

		input := make([]float32, n/2)
		for i := range input {
			input[i] = float32(rng.Float64()*2 - 1)
		}

		output := make([]float32, n)
		m.Process(input, 0, output, 0)

		ref := referenceIMDCT(input, n)
		maxDiff := maxAbsDiff(output, ref)
		maxRef := maxAbs(ref)
		tol := maxRef*1e-3 + 1e-3
		if maxDiff > tol {
			t.Fatalf("N=%d max diff %g exceeds tolerance %g", n, maxDiff, tol)
		}
	}
}

func TestProcessSliceMatchesProcess(t *testing.T) {
	m, err := New(256)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	input := make([]float32, 128)
	for i := range input {
		input[i] = float32(i) / 128.0
	}

	out1 := make([]float32, 256)
	m.Process(input, 0, out1, 0)
	out2 := m.ProcessSlice(input)

	if diff := maxAbsDiff(out1, out2); diff != 0 {
		t.Fatalf("ProcessSlice mismatch: max diff %g", diff)
	}
}

func benchmarkProcess(b *testing.B, n int) {
	m, err := New(n)
	if err != nil {
		b.Fatalf("New(%d) failed: %v", n, err)
	}
	input := make([]float32, n/2)
	output := make([]float32, n)
	for i := range input {
		input[i] = float32(i%31) / 31.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Process(input, 0, output, 0)
	}
}

func BenchmarkProcess2048(b *testing.B) { benchmarkProcess(b, 2048) }
func BenchmarkProcess256(b *testing.B)  { benchmarkProcess(b, 256) }
func BenchmarkProcess1920(b *testing.B) { benchmarkProcess(b, 1920) }
func BenchmarkProcess240(b *testing.B)  { benchmarkProcess(b, 240) }

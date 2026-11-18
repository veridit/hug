package embed

import (
	"math"
	"math/rand"
	"sort"
	"testing"
)

func randUnit(r *rand.Rand, dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = float32(r.NormFloat64())
	}
	return Normalize(v)
}

func TestQuantizedCosineIsClose(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	maxErr := 0.0
	for i := 0; i < 200; i++ {
		a, b := randUnit(r, 768), randUnit(r, 768)
		exact := float64(Dot(a, b))
		q := float64(DotQ(Quantize(a), Quantize(b)))
		maxErr = math.Max(maxErr, math.Abs(exact-q))
	}
	if maxErr > 0.01 {
		t.Fatalf("int8 cosine max error %.4f > 0.01", maxErr)
	}
	t.Logf("int8 cosine max abs error over 200 pairs: %.5f", maxErr)
}

func TestRoundTripF32(t *testing.T) {
	v := []float32{0.1, -0.2, 0.3}
	got := DecodeF32(EncodeF32(v))
	for i := range v {
		if got[i] != v[i] {
			t.Fatalf("%v != %v", got, v)
		}
	}
}

// The sketch prefilter must keep the true top-5 neighbours inside its
// shortlist (recall@5 within the 1024-candidate window used by search).
func TestSketchPrefilterRecall(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	const n, dim, keep = 5000, 768, 1024
	corpus := make([][]float32, n)
	for i := range corpus {
		corpus[i] = randUnit(r, dim)
	}
	hits, total := 0, 0
	for trial := 0; trial < 20; trial++ {
		// A query near a random corpus point, so true neighbours are meaningful.
		base := corpus[r.Intn(n)]
		q := make([]float32, dim)
		for i := range q {
			q[i] = base[i] + 0.05*float32(r.NormFloat64())
		}
		Normalize(q)
		type s struct {
			i   int
			sim float32
			ham int
		}
		all := make([]s, n)
		qs := Sketch(q)
		for i, v := range corpus {
			all[i] = s{i, Dot(q, v), Hamming(qs, Sketch(v))}
		}
		bySim := append([]s(nil), all...)
		sort.Slice(bySim, func(a, b int) bool { return bySim[a].sim > bySim[b].sim })
		sort.Slice(all, func(a, b int) bool { return all[a].ham < all[b].ham })
		short := map[int]bool{}
		for _, x := range all[:keep] {
			short[x.i] = true
		}
		for _, x := range bySim[:5] {
			total++
			if short[x.i] {
				hits++
			}
		}
	}
	recall := float64(hits) / float64(total)
	t.Logf("sketch prefilter recall@5 (keep %d of %d): %.3f", keep, n, recall)
	if recall < 0.95 {
		t.Fatalf("recall@5 %.3f < 0.95", recall)
	}
}

// 4vd1mz: the unrolled DotQ equals the plain loop bit for bit, for every
// length (tails of 0-7 bytes) and for extreme values (-127/127).
func TestDotQMatchesPlainLoop(t *testing.T) {
	plain := func(a, b []byte) float32 {
		if len(a) < 4 || len(b) < 4 {
			return 0
		}
		sa := math.Float32frombits(uint32(a[0]) | uint32(a[1])<<8 | uint32(a[2])<<16 | uint32(a[3])<<24)
		sb := math.Float32frombits(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
		var s int32
		for i := 4; i < min(len(a), len(b)); i++ {
			s += int32(int8(a[i])) * int32(int8(b[i]))
		}
		return float32(s) * sa * sb
	}
	r := rand.New(rand.NewSource(1))
	for dim := 0; dim <= 1030; dim += 1 + dim/16 {
		for k := 0; k < 20; k++ {
			a, b := Quantize(randUnit(r, max(dim, 1))[:dim]), Quantize(randUnit(r, max(dim+k%3, 1))[:dim+k%3])
			if k == 0 {
				for i := 4; i < len(a); i++ {
					a[i] = 0x81 // -127
				}
				for i := 4; i < len(b); i++ {
					b[i] = 0x81
				}
			}
			if got, want := DotQ(a, b), plain(a, b); got != want {
				t.Fatalf("dim %d: DotQ %v, plain %v", dim, got, want)
			}
		}
	}
}

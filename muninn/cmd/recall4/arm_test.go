package main

import (
	"math"
	"testing"

	"github.com/veridit/hug/muninn/internal/embed"
)

// The calibration coordinate maps the arm's cosine c to a*c+b exactly, for
// b of either sign, and truncation renormalises before it.
func TestArmShapeAffine(t *testing.T) {
	q := embed.Normalize([]float32{0.3, -0.2, 0.9, 0.1})
	d := embed.Normalize([]float32{0.25, 0.1, 0.8, -0.4})
	for _, ab := range [][2]float64{{1.3, -0.2}, {0.8, 0.15}} {
		a := arm{Dim: 3, A: ab[0], B: ab[1]}
		qt := embed.Normalize(append([]float32(nil), q[:3]...))
		dt := embed.Normalize(append([]float32(nil), d[:3]...))
		want := ab[0]*float64(embed.Dot(qt, dt)) + ab[1]
		got := float64(embed.Dot(a.shape(q, true), a.shape(d, false)))
		if math.Abs(got-want) > 1e-6 {
			t.Fatalf("a=%v b=%v: cosine %v, want %v", ab[0], ab[1], got, want)
		}
	}
	if v := (arm{Dim: 2}).shape(q, true); len(v) != 2 || math.Abs(float64(embed.Dot(v, v))-1) > 1e-6 {
		t.Fatalf("truncate: %v", v)
	}
}

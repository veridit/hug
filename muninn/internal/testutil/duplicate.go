package testutil

import (
	"context"
	"math"
	"strings"
)

// DuplicateEmbedder gives all fixture documents one direction and candidates
// containing synthetic-near a controlled cosine. Never a real-model calibration.
type DuplicateEmbedder struct{ Cosine float64 }

func (DuplicateEmbedder) Model() string { return "synthetic-duplicate-api" }
func (e DuplicateEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for _, s := range texts {
		if strings.Contains(s, "synthetic-near") {
			out = append(out, []float32{float32(e.Cosine), float32(math.Sqrt(1 - e.Cosine*e.Cosine))})
		} else {
			out = append(out, []float32{1, 0})
		}
	}
	return out, nil
}

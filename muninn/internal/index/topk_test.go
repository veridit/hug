package index

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/veridit/hug/muninn/internal/embed"
)

// 4vd1mz: topK keeps the best NeighborK as it goes; it must return exactly
// what sorting every vector returned, ties by id included. Vectors are
// drawn from 4 values per dimension so equal similarities are common.
func TestTopKMatchesFullSort(t *testing.T) {
	sortAll := func(q []byte, id int64, vecs []vecRow) []nbr {
		best := make([]nbr, 0, len(vecs))
		for _, v := range vecs {
			if v.id != id {
				best = append(best, nbr{v.id, embed.DotQ(q, v.q)})
			}
		}
		sort.Slice(best, func(i, j int) bool {
			if best[i].sim == best[j].sim {
				return best[i].id < best[j].id
			}
			return best[i].sim > best[j].sim
		})
		return best[:min(len(best), NeighborK)]
	}
	r := rand.New(rand.NewSource(3))
	for _, n := range []int{0, 1, 5, NeighborK, NeighborK + 1, 40, 500} {
		var vecs []vecRow
		for i := 0; i < n; i++ {
			v := make([]float32, 8)
			for j := range v {
				v[j] = float32(r.Intn(4) - 1)
			}
			vecs = append(vecs, vecRow{int64(r.Intn(3 * (n + 1))), embed.Quantize(v)})
		}
		for k := 0; k < min(n, 20); k++ {
			q, id := vecs[k].q, vecs[k].id
			if got, want := fmt.Sprint(topK(q, id, vecs)), fmt.Sprint(sortAll(q, id, vecs)); got != want {
				t.Fatalf("n=%d query %d:\n got %s\nwant %s", n, k, got, want)
			}
		}
	}
}

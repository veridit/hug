// Package embed turns text into vectors and stores them compactly.
package embed

import (
	"encoding/binary"
	"math"
	"math/bits"
)

// Normalize scales v to unit length in place and returns it.
func Normalize(v []float32) []float32 {
	var s float64
	for _, x := range v {
		s += float64(x) * float64(x)
	}
	n := math.Sqrt(s)
	if n < 1e-12 {
		return v
	}
	for i := range v {
		v[i] = float32(float64(v[i]) / n)
	}
	return v
}

// Dot is the inner product; for normalised vectors it is the cosine.
func Dot(a, b []float32) float32 {
	n := min(len(a), len(b))
	var s float32
	for i := 0; i < n; i++ {
		s += a[i] * b[i]
	}
	return s
}

// EncodeF32 packs a vector as little-endian float32.
func EncodeF32(v []float32) []byte {
	b := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(x))
	}
	return b
}

// DecodeF32 unpacks EncodeF32.
func DecodeF32(b []byte) []float32 {
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
	}
	return v
}

// Quantize maps a unit vector to int8 with a per-vector scale stored in the
// first 4 bytes (4x smaller than float32, cosine error well under 1%).
func Quantize(v []float32) []byte {
	var maxAbs float32
	for _, x := range v {
		if a := float32(math.Abs(float64(x))); a > maxAbs {
			maxAbs = a
		}
	}
	scale := maxAbs / 127
	if scale == 0 {
		scale = 1
	}
	out := make([]byte, 4+len(v))
	binary.LittleEndian.PutUint32(out, math.Float32bits(scale))
	for i, x := range v {
		q := math.Round(float64(x / scale))
		out[4+i] = byte(int8(max(-127, min(127, q))))
	}
	return out
}

// DotQ is the dot product of two Quantize outputs. Four independent
// accumulators let the CPU overlap the multiplies (4vd1mz: 441 -> 256 ns
// for a 768-dim pair); integer sums are exact, so the result is the same
// as a plain loop's (TestDotQMatchesPlainLoop).
func DotQ(a, b []byte) float32 {
	if len(a) < 4 || len(b) < 4 {
		return 0
	}
	sa := math.Float32frombits(binary.LittleEndian.Uint32(a))
	sb := math.Float32frombits(binary.LittleEndian.Uint32(b))
	n := min(len(a), len(b))
	a, b = a[4:n], b[4:n]
	var s0, s1, s2, s3 int32
	i := 0
	for ; i+8 <= len(a); i += 8 {
		x, y := a[i:i+8:i+8], b[i:i+8:i+8]
		s0 += int32(int8(x[0]))*int32(int8(y[0])) + int32(int8(x[4]))*int32(int8(y[4]))
		s1 += int32(int8(x[1]))*int32(int8(y[1])) + int32(int8(x[5]))*int32(int8(y[5]))
		s2 += int32(int8(x[2]))*int32(int8(y[2])) + int32(int8(x[6]))*int32(int8(y[6]))
		s3 += int32(int8(x[3]))*int32(int8(y[3])) + int32(int8(x[7]))*int32(int8(y[7]))
	}
	for ; i < len(a); i++ {
		s0 += int32(int8(a[i])) * int32(int8(b[i]))
	}
	return float32(s0+s1+s2+s3) * sa * sb
}

// Sketch is a 1-bit sign signature (32x smaller than float32). Hamming
// distance between sketches tracks angle, so it is a cheap prefilter.
func Sketch(v []float32) []byte {
	out := make([]byte, (len(v)+7)/8)
	for i, x := range v {
		if x > 0 {
			out[i/8] |= 1 << (i % 8)
		}
	}
	return out
}

// Hamming counts differing bits between two sketches.
func Hamming(a, b []byte) int {
	n := min(len(a), len(b))
	d := 0
	i := 0
	for ; i+8 <= n; i += 8 {
		d += bits.OnesCount64(binary.LittleEndian.Uint64(a[i:]) ^ binary.LittleEndian.Uint64(b[i:]))
	}
	for ; i < n; i++ {
		d += bits.OnesCount8(a[i] ^ b[i])
	}
	return d
}

package main

import (
	"math"
)

// ---- Tunables ---------------------------------------------------------------

// Default headroom applied *inside* the resampler.
// -3 dB ≈ 0.7071 -> 23170 in Q15. Bump to -6 dB (16384) if you still hit the rails.
const headroomQ15 = 23170 // Q15 scale factor

// Use Mitchell–Netravali (B=C=1/3). Swap to Catmull–Rom if you want more bite.
var mnW [256][4]int16 // Q15 phase table (256 phases, 4 taps)

func init() { initMitchellNetravali() }

// Precompute MN weights (Q15), normalized so DC gain is exactly 1.0.
func initMitchellNetravali() {
	const B, C = 1.0 / 3.0, 1.0 / 3.0
	for i := 0; i < 256; i++ {
		t := float64(i) / 256.0
		offs := [4]float64{-1 + t, t, 1 - t, 2 - t}

		sum := 0.0
		wf := [4]float64{}
		for k := 0; k < 4; k++ {
			x := math.Abs(offs[k])
			var h float64
			if x < 1 {
				h = ((12-9*B-6*C)*x*x*x + (-18+12*B+6*C)*x*x + (6 - 2*B)) / 6
			} else if x < 2 {
				h = ((-B-6*C)*x*x*x + (6*B+30*C)*x*x + (-12*B-48*C)*x + (8*B + 24*C)) / 6
			} else {
				h = 0
			}
			wf[k] = h
			sum += h
		}
		// Normalize to exactly 1.0 to keep DC stable.
		inv := 1.0 / sum
		si := 0
		for k := 0; k < 4; k++ {
			mnW[i][k] = int16(math.Round(wf[k] * inv * 32768.0))
			si += int(mnW[i][k])
		}
		// Nudge main tap to hit exactly 32768 in Q15.
		if si != 32768 {
			mnW[i][1] += int16(32768 - si)
		}
	}
}

// ---- Resampler --------------------------------------------------------------

// ResampleCubicInt16 resamples mono int16 from srcRate -> dstRate using
// Mitchell–Netravali cubic with built-in headroom (Q15).
// It clamps edges (hold) to avoid boundary clicks.
func ResampleCubicInt16(src []int16, srcRate, dstRate int) []int16 {
	if len(src) == 0 || srcRate == dstRate {
		out := make([]int16, len(src))
		copy(out, src)
		return out
	}

	// Output length so last valid center contributes. Conservative and safe.
	n := int((int64(len(src)-1)*int64(dstRate))/int64(srcRate)) + 1
	if n <= 0 {
		return nil
	}
	dst := make([]int16, n)

	// Q32.32 phase accumulator
	step := (uint64(srcRate) << 32) / uint64(dstRate)
	var phase uint64

	// Clamp-index accessor (hold at ends) to avoid edge discontinuities.
	get := func(i int) int16 {
		if i < 0 {
			i = 0
		} else if i >= len(src) {
			i = len(src) - 1
		}
		return src[i]
	}

	for i := 0; i < n; i++ {
		base := int(phase >> 32)
		fracIdx := byte(phase >> 24) // top 8 frac bits -> 256-phase table
		w := mnW[fracIdx]

		p0 := int32(get(base - 1))
		p1 := int32(get(base + 0))
		p2 := int32(get(base + 1))
		p3 := int32(get(base + 2))

		// Accumulate in Q15.
		acc := int64(w[0])*int64(p0) +
			int64(w[1])*int64(p1) +
			int64(w[2])*int64(p2) +
			int64(w[3])*int64(p3)

		// Apply headroom in Q15 *before* quantizing to int16.
		// acc: Q15 * int16 units. Multiply by headroomQ15 (Q15) -> Q30.
		// Convert Q30 -> int16 with rounding.
		y := int32((acc*int64(headroomQ15) + (1 << 29)) >> 30)

		// Saturate to int16 for safety (should rarely trigger with pad).
		if y > 32767 {
			y = 32767
		} else if y < -32768 {
			y = -32768
		}
		dst[i] = int16(y)

		phase += step
	}

	return dst
}

// Optional variant: choose headroom in dB at call site.
func ResampleCubicInt16PadDB(src []int16, srcRate, dstRate int, headroomDB float64) []int16 {
	h := math.Pow(10, headroomDB/20) // e.g., -6 => 0.5012
	hQ15 := int64(math.Round(h * 32768))
	if hQ15 <= 0 {
		hQ15 = 1
	}
	if len(src) == 0 || srcRate == dstRate {
		out := make([]int16, len(src))
		copy(out, src)
		return out
	}
	n := int((int64(len(src)-1)*int64(dstRate))/int64(srcRate)) + 1
	dst := make([]int16, n)
	step := (uint64(srcRate) << 32) / uint64(dstRate)
	var phase uint64
	get := func(i int) int16 {
		if i < 0 {
			return src[0]
		}
		if i >= len(src) {
			return src[len(src)-1]
		}
		return src[i]
	}
	for i := 0; i < n; i++ {
		base := int(phase >> 32)
		w := mnW[byte(phase>>24)]
		acc := int64(w[0])*int64(get(base-1)) +
			int64(w[1])*int64(get(base+0)) +
			int64(w[2])*int64(get(base+1)) +
			int64(w[3])*int64(get(base+2))
		y := int32((acc*hQ15 + (1 << 29)) >> 30)
		if y > 32767 {
			y = 32767
		} else if y < -32768 {
			y = -32768
		}
		dst[i] = int16(y)
		phase += step
	}
	return dst
}

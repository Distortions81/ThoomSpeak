package main

import (
	"math"
)

// ---------------- Tunables ----------------

// Polyphase resolution: 1024 fractional steps
const phases = 1024

// Lanczos "a" parameter: 3 => 6 taps total (k=-2..+3)
const a = 3
const taps = 2 * a // 6

// Low-pass cutoff as a fraction of source Nyquist
// <1.0 trims near-Nyquist to reduce ringing on crusty 8 kHz assets
const cutoff = 0.95

// Precomputed table: [phase][tap] in Q15
var lzW [phases][taps]int16

func init() { initLanczosTable() }

// sincπ(x) = sin(πx)/(πx), with sincπ(0)=1
func sincpi(x float64) float64 {
	if x == 0 {
		return 1.0
	}
	px := math.Pi * x
	return math.Sin(px) / px
}

// Lanczos kernel with bandwidth control.
// L(x) = sinc(c*x) * sinc((c/a)*x), |x|<a; else 0
func lanczos(x float64) float64 {
	ax := math.Abs(x)
	if ax >= a {
		return 0
	}
	return sincpi(cutoff*x) * sincpi((cutoff/float64(a))*x)
}

// Precompute weights in Q15; normalize each phase to exact DC=1.0.
func initLanczosTable() {
	for p := 0; p < phases; p++ {
		// Fractional offset in [0,1)
		f := float64(p) / phases
		// Taps around base sample: k = -a+1 .. +a
		sum := 0.0
		wf := [taps]float64{}
		idx := 0
		for k := -a + 1; k <= a; k++ {
			x := float64(k) - f // relative position
			w := lanczos(x)
			wf[idx] = w
			sum += w
			idx++
		}
		// Normalize and quantize to Q15
		inv := 1.0 / sum
		total := 0
		for i := 0; i < taps; i++ {
			lzW[p][i] = int16(math.Round(wf[i] * inv * 32768.0))
			total += int(lzW[p][i])
		}
		// Nudge the central-ish tap (k=0 -> index a-1) to force exact sum=32768
		diff := 32768 - total
		lzW[p][a-1] += int16(diff)
	}
}

// ResampleLanczosInt16PadDB resamples mono int16 from srcRate→dstRate using
// Lanczos-3 (band-limited) with a *built-in dB pad* applied BEFORE int16 quantization.
// padDB: negative for attenuation (e.g. -3, -6). Non-negative is clamped to 0 dB.
func ResampleLanczosInt16PadDB(src []int16, srcRate, dstRate int, padDB float64) []int16 {
	if len(src) == 0 || srcRate == dstRate {
		out := make([]int16, len(src))
		copy(out, src)
		return out
	}

	// Output length (safe; includes last contributing center sample).
	n := int((int64(len(src)-1)*int64(dstRate))/int64(srcRate)) + 1
	if n <= 0 {
		return nil
	}
	dst := make([]int16, n)

	// Q32.32 phase step
	step := (uint64(srcRate) << 32) / uint64(dstRate)
	var phase uint64

	// Compute Q15 scale from dB once (padDB <= 0 expected).
	scale := math.Pow(10, padDB/20) // e.g., -3 dB ≈ 0.7071
	if scale > 1 {
		scale = 1
	}
	if scale <= 0 {
		scale = 1.0 / (1 << 30) // tiny, avoid zero
	}
	scaleQ15 := int64(math.Round(scale * 32768.0))

	// Clamp accessor (hold endpoints) to avoid boundary clicks.
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
		fracIdx := int((phase >> (32 - 10)) & (phases - 1)) // use top 10 frac bits
		w := lzW[fracIdx]

		// Accumulate 6 taps in Q15
		acc := int64(0)
		j := 0
		for k := -a + 1; k <= a; k++ { // -2..+3
			s := int32(get(base + k))
			acc += int64(w[j]) * int64(s)
			j++
		}

		// Apply pad in Q15 BEFORE int16 quantization: Q15*Q15 = Q30.
		y := int32((acc*scaleQ15 + (1 << 29)) >> 30)

		// Saturate to int16 (should be rare with pad)
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

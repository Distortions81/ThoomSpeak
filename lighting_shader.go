package main

import (
	_ "embed"
	"gothoom/climg"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

const maxLights = 128

//go:embed data/shaders/light.kage
var lightShaderSrc []byte

var (
	lightingShader *ebiten.Shader
	lightingTmp    *ebiten.Image
	frameLights    []lightSource
	frameDarks     []darkSource
	prevLights     []lightSource
	prevDarks      []darkSource
)

// Global multipliers to make lights/darks reach farther on screen.
const (
	lightRadiusScale = 3
	darkRadiusScale  = 3
	// Stronger scaling for shader-based night attenuation. At 100% night,
	// total effective darkening approaches this factor depending on layout.
	// Increased baseline shader night strength to produce a very dark
	// overall scene at 100% night.
	shaderNightStrength = 10
)

func init() {
	var err error
	lightingShader, err = ebiten.NewShader(lightShaderSrc)
	if err != nil {
		panic(err)
	}
}

// ReloadLightingShader recompiles the lighting shader from disk and swaps it in.
// Falls back to the embedded shader source if reading from disk fails.
func ReloadLightingShader() error {
	// Try to reload from the source file for live iteration
	if b, err := os.ReadFile("data/shaders/light.kage"); err == nil {
		if sh, err2 := ebiten.NewShader(b); err2 == nil {
			lightingShader = sh
			return nil
		} else {
			return err2
		}
	}
	// Fallback: use embedded shader source
	sh, err := ebiten.NewShader(lightShaderSrc)
	if err != nil {
		return err
	}
	lightingShader = sh
	return nil
}

type lightSource struct {
	X, Y    float32
	Radius  float32
	R, G, B float32
}

type darkSource struct {
	X, Y   float32
	Radius float32
	Alpha  float32
}

func ensureLightingTmp(w, h int) {
	if lightingTmp == nil || lightingTmp.Bounds().Dx() != w || lightingTmp.Bounds().Dy() != h {
		lightingTmp = ebiten.NewImage(w, h)
	}
}

func applyLightingShader(dst *ebiten.Image, lights []lightSource, darks []darkSource, t float32) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	ensureLightingTmp(w, h)
	lightingTmp.DrawImage(dst, nil)

	// Interpolate between previous and current lighting with one-frame
	// persistence for disappearing sources and fade-in for new sources.
    il := make([]lightSource, 0, maxLights)
    id := make([]darkSource, 0, maxLights)

	// Lights: match by nearest neighbor to preserve identity across frames.
	usedPrev := make([]bool, len(prevLights))
	for i := range lights {
		// find best previous match within a threshold
		best := -1
		bestD := float32(1e30)
		thr := lights[i].Radius * 0.5
		thr2 := thr * thr
		for j := range prevLights {
			if usedPrev[j] {
				continue
			}
			dx := prevLights[j].X - lights[i].X
			dy := prevLights[j].Y - lights[i].Y
			d2 := dx*dx + dy*dy
			if d2 < bestD {
				bestD = d2
				best = j
			}
		}
        if best >= 0 && bestD <= thr2 {
            // Matched: interpolate all fields with the standard smoothing factor t
            pl := prevLights[best]
            usedPrev[best] = true
            il = append(il, lightSource{
                X:      lerp(pl.X, lights[i].X, t),
                Y:      lerp(pl.Y, lights[i].Y, t),
                Radius: lerp(pl.Radius, lights[i].Radius, t),
                R:      lerp(pl.R, lights[i].R, t),
                G:      lerp(pl.G, lights[i].G, t),
                B:      lerp(pl.B, lights[i].B, t),
            })
        } else {
            // New: take current values (no special fade)
            il = append(il, lights[i])
        }
        if len(il) >= maxLights {
            break
        }
    }

    // Darks: same matching and fade behavior using alpha
    usedPrevD := make([]bool, len(prevDarks))
    for i := range darks {
		best := -1
		bestD := float32(1e30)
		thr := darks[i].Radius * 0.5
		thr2 := thr * thr
		for j := range prevDarks {
			if usedPrevD[j] {
				continue
			}
			dx := prevDarks[j].X - darks[i].X
			dy := prevDarks[j].Y - darks[i].Y
			d2 := dx*dx + dy*dy
			if d2 < bestD {
				bestD = d2
				best = j
			}
		}
        if best >= 0 && bestD <= thr2 {
            // Matched dark: interpolate all fields with t
            pd := prevDarks[best]
            usedPrevD[best] = true
            id = append(id, darkSource{
                X:      lerp(pd.X, darks[i].X, t),
                Y:      lerp(pd.Y, darks[i].Y, t),
                Radius: lerp(pd.Radius, darks[i].Radius, t),
                Alpha:  lerp(pd.Alpha, darks[i].Alpha, t),
            })
        } else {
            // New dark: take current values
            id = append(id, darks[i])
        }
        if len(id) >= maxLights {
            break
        }
    }

	uniforms := map[string]any{
		"LightCount": len(il),
		"DarkCount":  len(id),
	}
	var lposX, lposY, lradius, lr, lg, lb [maxLights]float32
	for i := 0; i < len(il) && i < maxLights; i++ {
		ls := il[i]
		lposX[i] = ls.X
		lposY[i] = ls.Y
		lradius[i] = ls.Radius * float32(lightRadiusScale)
		lr[i] = ls.R
		lg[i] = ls.G
		lb[i] = ls.B
	}
	var dposX, dposY, dradius, da [maxLights]float32
	for i := 0; i < len(id) && i < maxLights; i++ {
		ds := id[i]
		dposX[i] = ds.X
		dposY[i] = ds.Y
		dradius[i] = ds.Radius * float32(darkRadiusScale)
		da[i] = ds.Alpha
	}
	uniforms["LightPosX"] = lposX
	uniforms["LightPosY"] = lposY
	uniforms["LightRadius"] = lradius
	uniforms["LightR"] = lr
	uniforms["LightG"] = lg
	uniforms["LightB"] = lb
	uniforms["DarkPosX"] = dposX
	uniforms["DarkPosY"] = dposY
	uniforms["DarkRadius"] = dradius
	uniforms["DarkAlpha"] = da

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = lightingTmp
	op.Uniforms = uniforms
	dst.DrawRectShader(w, h, lightingShader, op)

    // Save current as previous for next interpolation step
    prevLights = cloneLights(lights)
    prevDarks = cloneDarks(darks)
}

func lerp(a, b, t float32) float32 { return a + (b-a)*t }

func cloneLights(in []lightSource) []lightSource {
	if len(in) == 0 {
		return nil
	}
	out := make([]lightSource, len(in))
	copy(out, in)
	return out
}

func cloneDarks(in []darkSource) []darkSource {
	if len(in) == 0 {
		return nil
	}
	out := make([]darkSource, len(in))
	copy(out, in)
	return out
}

// addNightDarkSources appends dark sources to produce a smooth inverse-square
// vignette-like darkening using the shader path. The overall strength scales
// with the current/effective night level and ambientNightStrength.
func addNightDarkSources(w, h int) {
	lvl := currentNightLevel()
	if lvl <= 0 {
		return
	}
	// Convert to [0..1] strength; reuse ambientNightStrength as baseline.
	// Use a higher strength specifically for shader night so 100% looks dark.
	alpha := float32(float64(lvl) / 100.0 * float64(shaderNightStrength))
	if alpha <= 0 {
		return
	}
	// Use four corner dark sources with shared alpha to bias edges darker.
	// Radius based on screen diagonal yields gentle center falloff.
	diag := float32(math.Hypot(float64(w), float64(h)))
	// Center dark: provide near-total ambient darkening across the scene.
	centerRadius := diag * 1.5
	centerAlpha := alpha * 1.0
	frameDarks = append(frameDarks, darkSource{X: float32(w) / 2, Y: float32(h) / 2, Radius: centerRadius, Alpha: centerAlpha})

	// Corner vignettes: minimal edge emphasis
	cornerRadius := diag * 1.1
	cornerAlpha := alpha * 0.02 / 4
	corners := [][2]float32{{0, 0}, {float32(w), 0}, {0, float32(h)}, {float32(w), float32(h)}}
	for _, c := range corners {
		frameDarks = append(frameDarks, darkSource{X: c[0], Y: c[1], Radius: cornerRadius, Alpha: cornerAlpha})
	}
}

func addLightSource(pictID uint32, x, y float64, size int) {
	if !gs.shaderLighting || clImages == nil {
		return
	}
	flags := clImages.Flags(pictID)
	if flags&climg.PictDefFlagEmitsLight == 0 {
		return
	}
	li, ok := clImages.Lighting(pictID)
	if !ok {
		return
	}
	radius := float32(li.Radius)
	if radius == 0 {
		radius = float32(size)
	}
	radius *= float32(gs.GameScale)
	cx := float32(x)
	cy := float32(y)
	if flags&climg.PictDefFlagLightDarkcaster != 0 {
		if len(frameDarks) < maxLights {
			alpha := float32(li.Color[3]) / 255
			frameDarks = append(frameDarks, darkSource{X: cx, Y: cy, Radius: radius, Alpha: alpha})
		}
	} else {
		if len(frameLights) < maxLights {
			r := float32(li.Color[0]) / 255
			g := float32(li.Color[1]) / 255
			b := float32(li.Color[2]) / 255
			frameLights = append(frameLights, lightSource{X: cx, Y: cy, Radius: radius, R: r, G: g, B: b})
		}
	}
}

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

	// Use the already-interpolated sprite/mobile positions directly.
	// Interpolation for motion has been applied when enqueuing lights,
	// so avoid re-interpolating here to keep shader lights aligned
	// exactly with rendered objects.
	il := make([]lightSource, 0, min(maxLights, len(lights)))
	for i := 0; i < len(lights) && i < maxLights; i++ {
		il = append(il, lights[i])
	}
	id := make([]darkSource, 0, min(maxLights, len(darks)))
	for i := 0; i < len(darks) && i < maxLights; i++ {
		id = append(id, darks[i])
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

}

// min helper to avoid importing math just for ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

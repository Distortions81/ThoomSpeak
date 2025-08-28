package main

import (
	_ "embed"
	"github.com/hajimehoshi/ebiten/v2"
	"gothoom/climg"
)

const maxLights = 32

//go:embed data/shaders/light.kage
var lightShaderSrc []byte

var (
	lightingShader *ebiten.Shader
	lightingTmp    *ebiten.Image
	frameLights    []lightSource
	frameDarks     []darkSource
)

func init() {
	var err error
	lightingShader, err = ebiten.NewShader(lightShaderSrc)
	if err != nil {
		panic(err)
	}
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

func applyLightingShader(dst *ebiten.Image, lights []lightSource, darks []darkSource) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	ensureLightingTmp(w, h)
	lightingTmp.DrawImage(dst, nil)

	uniforms := map[string]any{
		"LightCount": len(lights),
		"DarkCount":  len(darks),
	}
	var lposX, lposY, lradius, lr, lg, lb [maxLights]float32
	for i := 0; i < len(lights) && i < maxLights; i++ {
		ls := lights[i]
		lposX[i] = ls.X
		lposY[i] = ls.Y
		lradius[i] = ls.Radius
		lr[i] = ls.R
		lg[i] = ls.G
		lb[i] = ls.B
	}
	var dposX, dposY, dradius, da [maxLights]float32
	for i := 0; i < len(darks) && i < maxLights; i++ {
		ds := darks[i]
		dposX[i] = ds.X
		dposY[i] = ds.Y
		dradius[i] = ds.Radius
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

package main

import (
	"bytes"
	"embed"
	"image"
	"image/draw"
	"image/png"
)

//go:embed icon.png term.png
var iconFS embed.FS

var (
	iconPix  []byte
	iconW    int
	iconH    int
	termIcon *image.RGBA
)

func loadIcons() {
	if iconPix != nil {
		return
	}
	iconPix, iconW, iconH = pngRGBA("icon.png")
	termIcon = decodePNG("term.png")
}

func pngRGBA(name string) ([]byte, int, int) {
	img := decodePNG(name)
	if img == nil {
		return nil, 0, 0
	}
	return append([]byte(nil), img.Pix...), img.Bounds().Dx(), img.Bounds().Dy()
}

func decodePNG(name string) *image.RGBA {
	raw, err := iconFS.ReadFile(name)
	if err != nil {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	b := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	return rgba
}

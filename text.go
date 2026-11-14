package main

import (
	"image"
	"image/color"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func drawString(dst *image.RGBA, s string, cx, cy, scale int, col color.Color, center bool) {
	if scale < 1 {
		scale = 1
	}
	face := basicfont.Face7x13
	advance := font.MeasureString(face, s).Ceil() * scale
	height := face.Height * scale
	x := cx
	y := cy
	if center {
		x = cx - advance/2
		y = cy - height/2
	}
	tmp := image.NewRGBA(image.Rect(0, 0, advance/scale+2, face.Height+2))
	d := &font.Drawer{
		Dst:  tmp,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(1, face.Ascent),
	}
	d.DrawString(s)
	for py := 0; py < tmp.Bounds().Dy(); py++ {
		for px := 0; px < tmp.Bounds().Dx(); px++ {
			_, _, _, a := tmp.At(px, py).RGBA()
			if a == 0 {
				continue
			}
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					dst.Set(x+px*scale+sx, y+py*scale+sy, col)
				}
			}
		}
	}
}

func wrapLines(s string, width int) []string {
	if width < 1 {
		width = 1
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\t", "    ")
	var out []string
	for _, raw := range strings.Split(s, "\n") {
		if raw == "" {
			out = append(out, "")
			continue
		}
		rs := []rune(raw)
		for len(rs) > width {
			out = append(out, string(rs[:width]))
			rs = rs[width:]
		}
		out = append(out, string(rs))
	}
	return out
}

package main

import (
	"image"
	"image/color"
	"image/draw"

	xdraw "golang.org/x/image/draw"
)

type gridCell struct {
	Icon image.Rectangle
	Hit  image.Rectangle
}

func pagerReserve(h int) int {
	r := h / 9
	if r < 52 {
		r = 52
	}
	return r
}

func iconArea(area image.Rectangle) image.Rectangle {
	return image.Rect(area.Min.X, area.Min.Y, area.Max.X, area.Max.Y-pagerReserve(area.Dy()))
}

func gridFit(area image.Rectangle) (icon, gap, labelH, cols, rows, perPage int) {
	ia := iconArea(area)
	short := ia.Dx()
	if ia.Dy() < short {
		short = ia.Dy()
	}
	icon = short / 5
	if icon < 56 {
		icon = 56
	}
	if icon > 168 {
		icon = 168
	}
	gap = icon / 4
	if gap < 16 {
		gap = 16
	}
	labelH = 22
	if icon >= 96 {
		labelH = 28
	}
	padX := ia.Dx() / 16
	if padX < 16 {
		padX = 16
	}
	padY := ia.Dy() / 12
	if padY < 16 {
		padY = 16
	}
	innerW := ia.Dx() - 2*padX
	innerH := ia.Dy() - 2*padY
	cellW := icon + gap
	cellH := icon + labelH + gap
	cols = innerW / cellW
	if cols < 1 {
		cols = 1
	}
	rows = innerH / cellH
	if rows < 1 {
		rows = 1
	}
	perPage = cols * rows
	return icon, gap, labelH, cols, rows, perPage
}

func gridPages(nItems, perPage int) int {
	if perPage < 1 {
		perPage = 1
	}
	if nItems <= 0 {
		return 1
	}
	return (nItems+perPage-1)/perPage + 1
}

func pageIsHelp(page, nItems, perPage int) bool {
	return page >= gridPages(nItems, perPage)-1
}

func pageItems[T any](all []T, page, perPage int) []T {
	if perPage < 1 || pageIsHelp(page, len(all), perPage) {
		return nil
	}
	start := page * perPage
	if start >= len(all) {
		return nil
	}
	end := start + perPage
	if end > len(all) {
		end = len(all)
	}
	return all[start:end]
}

func gridCells(area image.Rectangle, n int) []gridCell {
	if n <= 0 {
		return nil
	}
	ia := iconArea(area)
	icon, gap, labelH, cols, _, perPage := gridFit(area)
	if n > perPage {
		n = perPage
	}
	if cols > n {
		cols = n
	}
	padX := ia.Dx() / 16
	if padX < 16 {
		padX = 16
	}
	padY := ia.Dy() / 12
	if padY < 16 {
		padY = 16
	}
	cellW := icon + gap
	out := make([]gridCell, n)
	for i := 0; i < n; i++ {
		c, r := i%cols, i/cols
		x0 := ia.Min.X + padX + c*cellW
		y0 := ia.Min.Y + padY + r*(icon+labelH+gap)
		iconR := image.Rect(x0, y0, x0+icon, y0+icon)
		hit := image.Rect(x0-gap/4, y0-gap/4, x0+icon+gap/4, y0+icon+labelH+gap/4)
		out[i] = gridCell{Icon: iconR, Hit: hit}
	}
	return out
}

func pagerButtons(area image.Rectangle) (prev, next image.Rectangle) {
	pr := pagerReserve(area.Dy())
	sz := pr * 2 / 3
	if sz > 48 {
		sz = 48
	}
	if sz < 32 {
		sz = 32
	}
	y0 := area.Max.Y - pr + (pr-sz)/2
	inset := area.Dx() / 10
	prev = image.Rect(area.Min.X+inset, y0, area.Min.X+inset+sz, y0+sz)
	next = image.Rect(area.Max.X-inset-sz, y0, area.Max.X-inset, y0+sz)
	return prev, next
}

func drawPager(dst *image.RGBA, area image.Rectangle, page, pages int) {
	if pages < 1 {
		pages = 1
	}
	prev, next := pagerButtons(area)
	arrow := color.RGBA{R: 0xe8, G: 0xee, B: 0xf4, A: 0xff}
	dim := color.RGBA{R: 0x5a, G: 0x62, B: 0x70, A: 0xff}
	pc, nc := arrow, arrow
	if page <= 0 {
		pc = dim
	}
	if page >= pages-1 {
		nc = dim
	}
	drawChevron(dst, prev, true, pc)
	drawChevron(dst, next, false, nc)
	drawString(dst, itoa(page+1)+"/"+itoa(pages), (area.Min.X+area.Max.X)/2, (prev.Min.Y+prev.Max.Y)/2, 1, arrow, true)
}

func itoa(n int) string {
	if n < 0 {
		n = 0
	}
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func drawChevron(dst *image.RGBA, r image.Rectangle, left bool, col color.Color) {
	cx := (r.Min.X + r.Max.X) / 2
	cy := (r.Min.Y + r.Max.Y) / 2
	w := r.Dx() / 4
	h := r.Dy() / 3
	for y := -h; y <= h; y++ {
		t := h - abs(y)
		if t < 1 {
			t = 1
		}
		if left {
			draw.Draw(dst, image.Rect(cx+w-t, cy+y, cx+w, cy+y+1), &image.Uniform{C: col}, image.Point{}, draw.Src)
		} else {
			draw.Draw(dst, image.Rect(cx-w, cy+y, cx-w+t, cy+y+1), &image.Uniform{C: col}, image.Point{}, draw.Src)
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func drawHelp(dst *image.RGBA, area image.Rectangle, title string, lines []string) {
	ia := iconArea(area)
	cx := (ia.Min.X + ia.Max.X) / 2
	y := ia.Min.Y + ia.Dy()/6
	drawString(dst, title, cx, y, 2, color.RGBA{R: 0xe8, G: 0xee, B: 0xf4, A: 0xff}, true)
	y += 40
	ink := color.RGBA{R: 0xa0, G: 0xb0, B: 0xc0, A: 0xff}
	for _, ln := range lines {
		drawString(dst, ln, cx, y, 1, ink, true)
		y += 22
	}
}

func (it *scriptItem) sizedIcon(n int) *image.RGBA {
	if it == nil || it.icon == nil || n < 1 {
		return nil
	}
	if it.iconN == n && it.iconScaled != nil {
		return it.iconScaled
	}
	dst := image.NewRGBA(image.Rect(0, 0, n, n))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), it.icon, it.icon.Bounds(), xdraw.Src, nil)
	it.iconScaled = dst
	it.iconN = n
	return dst
}

func drawScriptIcon(dst *image.RGBA, it *scriptItem, r image.Rectangle) {
	ic := it.sizedIcon(r.Dx())
	if ic != nil {
		draw.Draw(dst, r, ic, ic.Bounds().Min, draw.Over)
	}
}

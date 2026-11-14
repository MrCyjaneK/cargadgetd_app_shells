package main

import (
	"image"
	"image/color"
	"image/draw"
	"sync"
)

type scriptItem struct {
	path       string
	name       string
	icon       *image.RGBA
	iconScaled *image.RGBA
	iconN      int
}

type shells struct {
	mu          sync.Mutex
	w, ht       int
	items       []scriptItem
	page        int
	tapX        int
	swiping     bool
	view        string // "grid" or "out"
	title       string
	body        string
	running     bool
	loaded      bool
	scanID      int
	runID       int
	placeholder *image.RGBA
}

func newShells() *shells {
	loadIcons()
	return &shells{view: "grid", placeholder: termIcon}
}

func (s *shells) setSize(w, ht int) {
	s.mu.Lock()
	s.w, s.ht = w, ht
	s.mu.Unlock()
}

func (s *shells) resume() {
	s.mu.Lock()
	s.scanID++
	id := s.scanID
	s.runID++
	s.view = "grid"
	s.running = false
	s.page = 0
	s.mu.Unlock()
	go s.scan(id)
}

func (s *shells) pause() {}

func (s *shells) scan(id int) {
	paths := ListScripts(ShellsDir())
	ph := s.placeholder
	items := make([]scriptItem, 0, len(paths))
	for _, p := range paths {
		items = append(items, scriptItem{
			path: p,
			name: ProbeName(p),
			icon: ProbeIcon(p, ph),
		})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if id != s.scanID {
		return
	}
	s.items = items
	s.loaded = true
}

func (s *shells) draw(dst *image.RGBA) {
	b := dst.Bounds()
	w, ht := b.Dx(), b.Dy()
	s.setSize(w, ht)
	draw.Draw(dst, b, &image.Uniform{C: color.RGBA{R: 0x18, G: 0x1c, B: 0x24, A: 0xff}}, image.Point{}, draw.Src)

	s.mu.Lock()
	view := s.view
	items := append([]scriptItem(nil), s.items...)
	page := s.page
	title := s.title
	body := s.body
	loaded := s.loaded
	s.mu.Unlock()

	area := image.Rect(0, 0, w, ht)
	if view == "out" {
		s.drawOut(dst, area, title, body)
		return
	}
	if !loaded {
		drawString(dst, "Loading…", w/2, ht/2, 2, color.RGBA{R: 0xa0, G: 0xb0, B: 0xc0, A: 0xff}, true)
		return
	}
	s.drawGrid(dst, area, items, page)
}

func (s *shells) drawGrid(dst *image.RGBA, area image.Rectangle, items []scriptItem, page int) {
	_, _, _, _, _, per := gridFit(area)
	n := len(items)
	pages := gridPages(n, per)
	if page < 0 {
		page = 0
	}
	if pages < 1 {
		pages = 1
	}
	if page >= pages {
		page = pages - 1
	}
	if pageIsHelp(page, n, per) {
		if n == 0 {
			drawHelp(dst, area, "No scripts", []string{
				"Put executables in",
				"~/.local/cargadgetd/apps/shells",
				"Each must handle:",
				"$0 v0 name",
				"$0 v0 icon   (PNG, or exit 1)",
				"$0 v0 run    (text on stdout)",
			})
		} else {
			drawHelp(dst, area, "Shells", []string{
				"Scripts live in",
				"~/.local/cargadgetd/apps/shells",
				"$0 v0 name / icon / run",
				"Swipe or tap arrows to page.",
			})
		}
	} else {
		slice := pageItems(items, page, per)
		cells := gridCells(area, len(slice))
		nameCol := color.RGBA{R: 0xe8, G: 0xee, B: 0xf4, A: 0xff}
		for i := range slice {
			c := cells[i]
			drawScriptIcon(dst, &slice[i], c.Icon)
			scale := 1
			if c.Icon.Dx() >= 96 {
				scale = 2
			}
			drawString(dst, slice[i].name, (c.Icon.Min.X+c.Icon.Max.X)/2, c.Icon.Max.Y+12, scale, nameCol, true)
		}
	}
	drawPager(dst, area, page, pages)
}

func backRect(area image.Rectangle) image.Rectangle {
	h := 48
	if area.Dy() < 200 {
		h = 36
	}
	return image.Rect(area.Min.X+12, area.Min.Y+8, area.Min.X+12+110, area.Min.Y+8+h)
}

func (s *shells) drawOut(dst *image.RGBA, area image.Rectangle, title, body string) {
	back := backRect(area)
	btn := color.RGBA{R: 0x2a, G: 0x33, B: 0x40, A: 0xff}
	draw.Draw(dst, back, &image.Uniform{C: btn}, image.Point{}, draw.Src)
	drawString(dst, "< Back", (back.Min.X+back.Max.X)/2, (back.Min.Y+back.Max.Y)/2, 1, color.RGBA{R: 0xe8, G: 0xee, B: 0xf4, A: 0xff}, true)
	if title == "" {
		title = "Output"
	}
	drawString(dst, title, (area.Min.X+area.Max.X)/2, (back.Min.Y+back.Max.Y)/2, 2, color.RGBA{R: 0xe8, G: 0xee, B: 0xf4, A: 0xff}, true)

	if body == "" {
		body = "(no output)"
	}
	pad := 16
	top := back.Max.Y + 16
	cols := (area.Dx() - 2*pad) / 7
	lines := wrapLines(body, cols)
	ink := color.RGBA{R: 0xc8, G: 0xd0, B: 0xd8, A: 0xff}
	y := top
	maxY := area.Max.Y - 16
	for i, ln := range lines {
		if y+14 > maxY {
			drawString(dst, "…", pad, y, 1, ink, false)
			_ = i
			break
		}
		drawString(dst, ln, pad, y, 1, ink, false)
		y += 16
	}
}

func (s *shells) touch(x, y, action int) {
	s.mu.Lock()
	w, ht := s.w, s.ht
	view := s.view
	s.mu.Unlock()
	if w < 1 {
		w, ht = 800, 480
	}
	area := image.Rect(0, 0, w, ht)
	if view == "out" {
		if action == 0 && image.Pt(x, y).In(backRect(area)) {
			s.mu.Lock()
			s.runID++
			s.running = false
			s.view = "grid"
			s.mu.Unlock()
		}
		return
	}

	s.mu.Lock()
	items := append([]scriptItem(nil), s.items...)
	page := s.page
	loaded := s.loaded
	s.mu.Unlock()
	if !loaded {
		return
	}

	_, _, _, _, _, per := gridFit(area)
	pages := gridPages(len(items), per)

	if action == 1 {
		s.mu.Lock()
		s.tapX = x
		s.swiping = false
		s.mu.Unlock()
		return
	}
	if action == 2 {
		s.mu.Lock()
		if abs(x-s.tapX) > 24 {
			s.swiping = true
		}
		s.mu.Unlock()
		return
	}
	if action != 0 {
		return
	}

	s.mu.Lock()
	sw := s.swiping
	dx := x - s.tapX
	s.swiping = false
	s.mu.Unlock()
	if sw && abs(dx) > area.Dx()/8 {
		if dx < 0 {
			page++
		} else {
			page--
		}
		s.setPage(page, pages)
		return
	}

	prev, next := pagerButtons(area)
	pt := image.Pt(x, y)
	if pt.In(prev) {
		s.setPage(page-1, pages)
		return
	}
	if pt.In(next) {
		s.setPage(page+1, pages)
		return
	}
	if pageIsHelp(page, len(items), per) {
		return
	}
	slice := pageItems(items, page, per)
	cells := gridCells(area, len(slice))
	for i := range slice {
		if pt.In(cells[i].Hit) {
			s.startRun(slice[i])
			return
		}
	}
}

func (s *shells) setPage(page, pages int) {
	if pages < 1 {
		pages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	s.mu.Lock()
	s.page = page
	s.mu.Unlock()
}

func (s *shells) startRun(it scriptItem) {
	s.mu.Lock()
	s.runID++
	rid := s.runID
	s.running = true
	s.view = "out"
	s.title = it.name
	s.body = "Running…"
	path := it.path
	s.mu.Unlock()
	go func() {
		text, err := Run(path)
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.runID != rid {
			return
		}
		s.running = false
		if text == "" && err == nil {
			s.body = "(no output)"
			return
		}
		if err != nil {
			if text != "" {
				s.body = text + "\n(" + err.Error() + ")"
			} else {
				s.body = err.Error()
			}
			return
		}
		s.body = text
	}()
}

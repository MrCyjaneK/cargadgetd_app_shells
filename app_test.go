package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/mrcyjanek/cargadgetd_sdk/cfg"
)

func useShellsDir(t *testing.T, dir string) {
	t.Helper()
	cfg.SetOwnRootForTest(t, t.TempDir())
	if err := cfg.OwnSet("shells", "dir", dir); err != nil {
		t.Fatal(err)
	}
}

func writeExec(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	if got := expandHome("~/.local/cargadgetd/apps/shells"); got != filepath.Join(home, ".local/cargadgetd/apps/shells") {
		t.Fatal(got)
	}
	if got := expandHome("/usr/lib"); got != "/usr/lib" {
		t.Fatal(got)
	}
}

func TestListSkipsNonExecAndHidden(t *testing.T) {
	dir := t.TempDir()
	writeExec(t, dir, "keep", "#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(filepath.Join(dir, "nope"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hidden"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := ListScripts(dir)
	if len(got) != 1 || filepath.Base(got[0]) != "keep" {
		t.Fatalf("got %v", got)
	}
}

func TestProbeFallbacks(t *testing.T) {
	dir := t.TempDir()
	p := writeExec(t, dir, "foo", "#!/bin/sh\nexit 1\n")
	if ProbeName(p) != "foo" {
		t.Fatalf("name=%q", ProbeName(p))
	}
	ph := image.NewRGBA(image.Rect(0, 0, 4, 4))
	ic := ProbeIcon(p, ph)
	if ic != ph {
		t.Fatal("icon should be placeholder")
	}
}

func TestProbeNameIconRun(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "icon.png")
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	p := writeExec(t, dir, "hasicon", `#!/bin/sh
case "$2" in
  name) echo "Custom Name"; exit 0 ;;
  icon) cat "$(dirname "$0")/icon.png"; exit 0 ;;
  run) echo ran-ok; exit 0 ;;
  *) exit 2 ;;
esac
`)
	if ProbeName(p) != "Custom Name" {
		t.Fatalf("name=%q", ProbeName(p))
	}
	ph := image.NewRGBA(image.Rect(0, 0, 2, 2))
	ic := ProbeIcon(p, ph)
	if ic == ph || ic.Bounds().Dx() != 8 {
		t.Fatalf("icon fallback or size %v", ic.Bounds())
	}
	out, err := Run(p)
	if err != nil || strings.TrimSpace(out) != "ran-ok" {
		t.Fatalf("run %q err=%v", out, err)
	}
}

func waitLoaded(t *testing.T, a *shells) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		ok := a.loaded
		a.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatal("scan timeout")
}

func waitBody(t *testing.T, a *shells, sub string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		body := a.body
		running := a.running
		a.mu.Unlock()
		if !running && strings.Contains(body, sub) {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	t.Fatalf("body=%q running=%v want %q", a.body, a.running, sub)
}

func TestShellsEmptyHelp(t *testing.T) {
	dir := t.TempDir()
	useShellsDir(t, dir)
	a := newShells()
	a.w, a.ht = 800, 480
	a.resume()
	waitLoaded(t, a)
	dst := image.NewRGBA(image.Rect(0, 0, 800, 480))
	a.draw(dst)
	a.mu.Lock()
	n := len(a.items)
	a.mu.Unlock()
	if n != 0 {
		t.Fatalf("items=%d", n)
	}
}

func TestShellsRunAndBack(t *testing.T) {
	dir := t.TempDir()
	writeExec(t, dir, "os-release", `#!/bin/sh
case "$2" in
  name) echo "OS release"; exit 0 ;;
  icon) exit 1 ;;
  run) echo NAME=testos; exit 0 ;;
esac
`)
	useShellsDir(t, dir)
	a := newShells()
	a.w, a.ht = 1184, 720
	a.resume()
	waitLoaded(t, a)

	dst := image.NewRGBA(image.Rect(0, 0, 1184, 720))
	a.draw(dst)
	area := image.Rect(0, 0, 1184, 720)
	cells := gridCells(area, 1)
	if len(cells) != 1 {
		t.Fatalf("cells=%d", len(cells))
	}
	hit := cells[0].Hit
	cx, cy := hit.Min.X+hit.Dx()/2, hit.Min.Y+hit.Dy()/2
	a.touch(cx, cy, 1)
	a.mu.Lock()
	view := a.view
	a.mu.Unlock()
	if view != "grid" {
		t.Fatalf("opened on down: view=%q", view)
	}
	a.touch(cx, cy, 0)
	waitBody(t, a, "NAME=testos")

	back := backRect(area)
	a.touch((back.Min.X+back.Max.X)/2, (back.Min.Y+back.Max.Y)/2, 0)
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.view != "grid" {
		t.Fatalf("view=%q", a.view)
	}
}

func TestShellsPager(t *testing.T) {
	a := newShells()
	a.w, a.ht = 1184, 720
	a.loaded = true
	area := image.Rect(0, 0, a.w, a.ht)
	_, _, _, _, _, per := gridFit(area)
	for i := 0; i < per+1; i++ {
		a.items = append(a.items, scriptItem{path: "/x", name: "s", icon: a.placeholder})
	}
	if pages := gridPages(len(a.items), per); pages != 3 {
		t.Fatalf("pages=%d per=%d", pages, per)
	}
	x0 := a.w / 2
	a.touch(x0, 200, 1)
	a.touch(x0-a.w/4, 200, 2)
	a.touch(x0-a.w/4, 200, 0)
	if a.page != 1 {
		t.Fatalf("page=%d", a.page)
	}
	dst := image.NewRGBA(area)
	a.draw(dst)
}

func TestEmbeddedIcons(t *testing.T) {
	loadIcons()
	if len(iconPix) < 32*32*4 {
		t.Fatalf("app icon pix %d", len(iconPix))
	}
	if termIcon == nil || termIcon.Bounds().Dx() < 32 {
		t.Fatal("missing term.png")
	}
	p, w, h := iconABI()
	if p == nil || w != iconW || h != iconH {
		t.Fatalf("icon ABI %v %dx%d want %dx%d", p, w, h, iconW, iconH)
	}
	if uintptr(p) == uintptr(unsafe.Pointer(&iconPix[0])) {
		t.Fatal("icon must be C.malloc, not Go heap")
	}
	n := w * h * 4
	if !bytes.Equal(unsafe.Slice((*byte)(p), n), iconPix) {
		t.Fatal("icon pixels mismatch")
	}
}

func TestWrapLines(t *testing.T) {
	got := wrapLines("aa\n\nbbbbbb", 4)
	if len(got) != 4 || got[0] != "aa" || got[1] != "" || got[2] != "bbbb" || got[3] != "bb" {
		t.Fatalf("%q", got)
	}
}

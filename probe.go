package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mrcyjanek/cargadgetd_sdk/cfg"
)

const (
	nameTimeout = 2 * time.Second
	iconTimeout = 2 * time.Second
	runTimeout  = 8 * time.Second
	maxName     = 4 << 10
	maxIcon     = 1 << 20
	maxRun      = 64 << 10
)

const defaultShellsDir = "~/.local/cargadgetd/apps/shells"

// ShellsDir is plugins/shells.yaml key dir, or the default, with ~ expanded.
func ShellsDir() string {
	dir := defaultShellsDir
	if v, ok := cfg.OwnGet("shells", "dir"); ok && strings.TrimSpace(v) != "" {
		dir = strings.TrimSpace(v)
	}
	return expandHome(dir)
}

func expandHome(p string) string {
	p = strings.TrimSpace(p)
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == "~" {
		return home
	}
	return filepath.Join(home, p[2:])
}

// ListScripts returns executable regular files in dir, sorted by name.
func ListScripts(dir string) []string {
	if dir == "" {
		return nil
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range ents {
		name := e.Name()
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if info.Mode()&0o111 == 0 {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = filepath.Join(dir, n)
	}
	return out
}

func ProbeName(path string) string {
	base := filepath.Base(path)
	out, err := runVerb(path, "name", nameTimeout, maxName)
	if err != nil {
		return base
	}
	line := strings.TrimSpace(out)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	if line == "" {
		return base
	}
	return line
}

func ProbeIcon(path string, placeholder *image.RGBA) *image.RGBA {
	out, err := runVerbBytes(path, "icon", iconTimeout, maxIcon)
	if err != nil || len(out) == 0 {
		return placeholder
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		return placeholder
	}
	b := img.Bounds()
	if b.Dx() < 1 || b.Dy() < 1 {
		return placeholder
	}
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	return rgba
}

func Run(path string) (string, error) {
	out, err := runVerbBytes(path, "run", runTimeout, maxRun)
	text := string(out)
	if err != nil {
		if text == "" {
			return "", err
		}
		return text, err
	}
	return text, nil
}

func runVerb(path, verb string, timeout time.Duration, max int) (string, error) {
	b, err := runVerbBytes(path, verb, timeout, max)
	return string(b), err
}

func runVerbBytes(path, verb string, timeout time.Duration, max int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "v0", verb)
	cmd.Dir = filepath.Dir(path)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	b := buf.Bytes()
	if len(b) > max {
		b = b[:max]
	}
	if ctx.Err() == context.DeadlineExceeded {
		return b, fmt.Errorf("timed out")
	}
	if err != nil {
		return b, err
	}
	return b, nil
}

package main

/*
#cgo CFLAGS: -I${SRCDIR}/include
#define CARGADGETD_APP_IMPLEMENT_PLAY
#include "cargadgetd/cg_app.h"
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
*/
import "C"

import (
	"image"
	"unsafe"
)

var (
	gHost      *C.struct_cargadgetd_host
	app        = newShells()
	shellsName = C.CString("Shells")
	iconC      *C.uint8_t
)

func playFromHost(pcm []int16, nframes, ch, rate int) {
	if gHost == nil || nframes <= 0 || len(pcm) == 0 {
		return
	}
	C.cargadgetd_host_play(gHost, (*C.int16_t)(unsafe.Pointer(&pcm[0])), C.int(nframes), C.int(ch), C.int(rate))
}

func ensureIconC() {
	if iconC != nil {
		return
	}
	loadIcons()
	if len(iconPix) == 0 {
		return
	}
	n := C.size_t(len(iconPix))
	iconC = (*C.uint8_t)(C.malloc(n))
	if iconC == nil {
		return
	}
	C.memcpy(unsafe.Pointer(iconC), unsafe.Pointer(&iconPix[0]), n)
}

//export cargadgetd_app_shells_icon
func cargadgetd_app_shells_icon(w, h *C.int) *C.uint8_t {
	ensureIconC()
	if w != nil {
		*w = C.int(iconW)
	}
	if h != nil {
		*h = C.int(iconH)
	}
	return iconC
}

func iconABI() (unsafe.Pointer, int, int) {
	var w, h C.int
	return unsafe.Pointer(cargadgetd_app_shells_icon(&w, &h)), int(w), int(h)
}

//export cargadgetd_app_shells_init
func cargadgetd_app_shells_init(host *C.struct_cargadgetd_host) {
	gHost = host
	_ = playFromHost
}

//export cargadgetd_app_shells_draw
func cargadgetd_app_shells_draw(rgba *C.uint32_t, w, h, stride C.int) {
	if rgba == nil || w < 1 || h < 1 || stride < w*4 {
		return
	}
	pix := unsafe.Slice((*uint8)(unsafe.Pointer(rgba)), int(h)*int(stride))
	dst := &image.RGBA{
		Pix:    pix,
		Stride: int(stride),
		Rect:   image.Rect(0, 0, int(w), int(h)),
	}
	app.draw(dst)
}

//export cargadgetd_app_shells_touch
func cargadgetd_app_shells_touch(x, y, action C.int) {
	app.touch(int(x), int(y), int(action))
}

//export cargadgetd_app_shells_pause
func cargadgetd_app_shells_pause() { app.pause() }

//export cargadgetd_app_shells_resume
func cargadgetd_app_shells_resume() { app.resume() }

//export cargadgetd_app_shells_name
func cargadgetd_app_shells_name() *C.char { return shellsName }

func main() {}

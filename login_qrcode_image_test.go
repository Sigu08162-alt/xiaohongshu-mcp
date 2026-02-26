package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestEnsurePNGQuietZone_AddsPaddingWhenDarkTouchesEdges(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 20, 20))
	// White background.
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			src.Set(x, y, color.White)
		}
	}
	// Black area touching all edges (simulates clipped QR).
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			if x < 6 || x > 13 || y < 6 || y > 13 {
				src.Set(x, y, color.Black)
			}
		}
	}

	var in bytes.Buffer
	if err := png.Encode(&in, src); err != nil {
		t.Fatalf("encode source png: %v", err)
	}

	out, err := ensurePNGQuietZone(in.Bytes(), 12)
	if err != nil {
		t.Fatalf("ensure quiet zone: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode output png: %v", err)
	}
	b := img.Bounds()
	if b.Dx() <= 20 || b.Dy() <= 20 {
		t.Fatalf("expected padded image bigger than source, got %dx%d", b.Dx(), b.Dy())
	}

	// Edge must be white after padding.
	if edgeHasDarkPixel(img) {
		t.Fatal("expected no dark pixels on outer edges after padding")
	}
}

func TestEnsurePNGQuietZone_LeavesImageWhenAlreadyHasQuietZone(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 40, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			src.Set(x, y, color.White)
		}
	}
	// Black square in center with existing quiet zone.
	for y := 10; y < 30; y++ {
		for x := 10; x < 30; x++ {
			src.Set(x, y, color.Black)
		}
	}
	var in bytes.Buffer
	if err := png.Encode(&in, src); err != nil {
		t.Fatalf("encode source png: %v", err)
	}

	out, err := ensurePNGQuietZone(in.Bytes(), 12)
	if err != nil {
		t.Fatalf("ensure quiet zone: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode output png: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 40 || b.Dy() != 40 {
		t.Fatalf("expected original size unchanged, got %dx%d", b.Dx(), b.Dy())
	}
}

func edgeHasDarkPixel(img image.Image) bool {
	b := img.Bounds()
	isDark := func(x, y int) bool {
		r, g, bb, _ := img.At(x, y).RGBA()
		rr := uint8(r >> 8)
		gg := uint8(g >> 8)
		bb8 := uint8(bb >> 8)
		return rr < 200 || gg < 200 || bb8 < 200
	}

	for x := b.Min.X; x < b.Max.X; x++ {
		if isDark(x, b.Min.Y) || isDark(x, b.Max.Y-1) {
			return true
		}
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		if isDark(b.Min.X, y) || isDark(b.Max.X-1, y) {
			return true
		}
	}
	return false
}

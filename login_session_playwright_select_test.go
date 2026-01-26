package main

import (
	"testing"

	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
)

type fakeQRElement struct {
	box *browser.BoundingBox
}

func (f fakeQRElement) Screenshot() ([]byte, error) {
	return []byte("img"), nil
}

func (f fakeQRElement) BoundingBox() (*browser.BoundingBox, error) {
	return f.box, nil
}

func TestPickLargestElement(t *testing.T) {
	small := fakeQRElement{box: &browser.BoundingBox{Width: 80, Height: 80}}
	large := fakeQRElement{box: &browser.BoundingBox{Width: 160, Height: 160}}

	got := pickLargestElement([]qrElement{small, large})
	if got == nil {
		t.Fatalf("expected element")
	}
	if got != large {
		t.Fatalf("expected largest element")
	}
}

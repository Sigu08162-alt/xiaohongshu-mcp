package main

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"testing"
)

type fakeQRPage struct {
	fullData     []byte
	elementCalls int
}

func (f *fakeQRPage) Navigate(ctx context.Context, url string) error { return nil }
func (f *fakeQRPage) WaitLoad(ctx context.Context) error             { return nil }
func (f *fakeQRPage) Has(ctx context.Context, selector string) (bool, error) {
	return false, nil
}
func (f *fakeQRPage) Close() error { return nil }

func (f *fakeQRPage) HasRegex(ctx context.Context, selector, jsRegex string) (bool, error) {
	return false, nil
}
func (f *fakeQRPage) Element(ctx context.Context, selector string) (qrElement, error) {
	f.elementCalls++
	return nil, errors.New("no element")
}
func (f *fakeQRPage) Elements(ctx context.Context, selector string) ([]qrElement, error) {
	return nil, nil
}
func (f *fakeQRPage) ElementByRegex(ctx context.Context, selector, jsRegex string) (qrElement, error) {
	f.elementCalls++
	return nil, errors.New("no element")
}
func (f *fakeQRPage) Frames(ctx context.Context) ([]qrFrame, error) { return nil, nil }

func (f *fakeQRPage) ScreenshotFullPage(path string) error {
	return os.WriteFile(path, f.fullData, 0644)
}

func TestQRCode_UsesForcedFullPageScreenshot(t *testing.T) {
	orig := forceFullPageQRCode
	forceFullPageQRCode = true
	t.Cleanup(func() { forceFullPageQRCode = orig })

	page := &fakeQRPage{fullData: []byte("fullpage")}
	s := &playwrightLoginSession{page: page}

	got, err := s.QRCode(context.Background())
	if err != nil {
		t.Fatalf("QRCode err: %v", err)
	}
	want := base64.StdEncoding.EncodeToString(page.fullData)
	if got.Image != want {
		t.Fatalf("unexpected image")
	}
	if page.elementCalls != 0 {
		t.Fatalf("expected no element lookup")
	}
}

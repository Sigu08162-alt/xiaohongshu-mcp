package main

import (
	"encoding/base64"
	"os"
	"testing"
)

type fakeScreenshotter struct {
	data []byte
}

func (f fakeScreenshotter) ScreenshotFullPage(path string) error {
	return os.WriteFile(path, f.data, 0644)
}

func TestFullPageScreenshotBase64(t *testing.T) {
	data := []byte("pngdata")
	got, err := fullPageScreenshotBase64(fakeScreenshotter{data: data})
	if err != nil {
		t.Fatalf("fullPageScreenshotBase64 err: %v", err)
	}
	want := base64.StdEncoding.EncodeToString(data)
	if got != want {
		t.Fatalf("unexpected base64: %s", got)
	}
}

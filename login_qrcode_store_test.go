package main

import (
	"os"
	"testing"
	"time"
)

func TestLoginQRCodeStorePutAndGetBySession(t *testing.T) {
	store := newLoginQRCodeStore()
	store.Put("sess-1", []byte{1, 2, 3}, time.Minute)

	got, ok := store.Get("sess-1")
	if !ok {
		t.Fatal("expected cached qrcode")
	}
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("unexpected bytes: %v", got)
	}

	// Returned slice should be a copy.
	got[0] = 9
	got2, ok := store.Get("sess-1")
	if !ok {
		t.Fatal("expected cached qrcode on second read")
	}
	if got2[0] != 1 {
		t.Fatalf("store returned mutable backing array, got=%v", got2)
	}
}

func TestLoginQRCodeStoreGetLatest(t *testing.T) {
	store := newLoginQRCodeStore()
	store.Put("sess-1", []byte{1}, time.Minute)
	store.Put("sess-2", []byte{2}, time.Minute)

	sessionID, data, ok := store.GetLatest()
	if !ok {
		t.Fatal("expected latest qrcode")
	}
	if sessionID != "sess-2" {
		t.Fatalf("expected latest session sess-2, got %q", sessionID)
	}
	if len(data) != 1 || data[0] != 2 {
		t.Fatalf("unexpected latest data: %v", data)
	}
}

func TestLoginQRCodeStoreExpiresEntries(t *testing.T) {
	store := newLoginQRCodeStore()
	base := time.Date(2026, 2, 26, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return base }
	store.Put("sess-1", []byte{1}, time.Second)

	store.now = func() time.Time { return base.Add(2 * time.Second) }
	if _, ok := store.Get("sess-1"); ok {
		t.Fatal("expected expired qrcode to be removed")
	}
	if _, _, ok := store.GetLatest(); ok {
		t.Fatal("expected no latest qrcode after expiry")
	}
}

func TestResolvePublicBaseURLFromListenAddr(t *testing.T) {
	_ = os.Unsetenv("XHS_PUBLIC_BASE_URL")

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: "http://localhost:18060"},
		{name: "port only", in: ":19000", want: "http://localhost:19000"},
		{name: "host port", in: "127.0.0.1:18060", want: "http://127.0.0.1:18060"},
		{name: "all interfaces", in: "0.0.0.0:18060", want: "http://localhost:18060"},
		{name: "full url", in: "https://example.com/base", want: "https://example.com/base"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := resolvePublicBaseURLFromListenAddr(tt.in)
			if got != tt.want {
				t.Fatalf("resolvePublicBaseURLFromListenAddr(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

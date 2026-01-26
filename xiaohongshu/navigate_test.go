package xiaohongshu

import "testing"

func TestResolveProfileUserID_UsesRequested(t *testing.T) {
	userID := resolveProfileUserID("current", "requested")
	if userID != "requested" {
		t.Fatalf("unexpected user id: %q", userID)
	}
}

func TestResolveProfileUserID_FallsBackToCurrent(t *testing.T) {
	userID := resolveProfileUserID("current", "")
	if userID != "current" {
		t.Fatalf("unexpected user id: %q", userID)
	}
}

func TestBuildProfileURL(t *testing.T) {
	url := buildProfileURL("user123")
	expected := "https://www.xiaohongshu.com/user/profile/user123"
	if url != expected {
		t.Fatalf("unexpected url: %q", url)
	}
}

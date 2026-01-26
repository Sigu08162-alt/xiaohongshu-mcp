package main

import (
	"context"
	"encoding/base64"
	"reflect"
	"strings"
	"testing"
)

type fakeLoginProvider struct {
	result loginQRResult
	err    error
}

func (f fakeLoginProvider) GetQRCode(ctx context.Context) (loginQRResult, error) {
	return f.result, f.err
}

func strPtr(s string) *string {
	return &s
}

func TestLoginQrcodeHandler_TextForSecurityStage(t *testing.T) {
	service := &XiaohongshuService{
		loginManager: fakeLoginProvider{
			result: loginQRResult{
				LoginQrcodeResponse: LoginQrcodeResponse{
					Timeout:    "4m0s",
					IsLoggedIn: false,
					Img:        "img",
					Stage:      "security",
				},
			},
		},
	}
	app := &AppServer{xiaohongshuService: service}

	result := app.handleGetLoginQrcode(context.Background())
	if result == nil || len(result.Content) == 0 {
		t.Fatalf("expected content")
	}
	if !strings.Contains(result.Content[0].Text, "安全认证") {
		t.Fatalf("expected security text")
	}
}

func TestLoginQrcodeHandler_TextIncludesStatusAndSession(t *testing.T) {
	service := &XiaohongshuService{
		loginManager: fakeLoginProvider{
			result: loginQRResult{
				LoginQrcodeResponse: LoginQrcodeResponse{
					Timeout:    "4m0s",
					IsLoggedIn: false,
					Img:        "img",
					Stage:      "security",
					Status:     loginStatusSecurityNeeded,
					SessionID:  "sess-1",
				},
			},
		},
	}
	app := &AppServer{xiaohongshuService: service}

	result := app.handleGetLoginQrcode(context.Background())
	if result == nil || len(result.Content) == 0 {
		t.Fatalf("expected content")
	}
	if !strings.Contains(result.Content[0].Text, "状态:") {
		t.Fatalf("expected status text")
	}
	if !strings.Contains(result.Content[0].Text, "sess-1") {
		t.Fatalf("expected session id")
	}
}

func TestParseSyncCookiesPayload_Base64(t *testing.T) {
	data := []byte(`[{"name":"a"}]`)
	args := SyncCookiesArgs{CookiesBase64: base64.StdEncoding.EncodeToString(data)}

	got, err := parseSyncCookiesPayload(args)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("unexpected payload")
	}
}

func TestParseSyncCookiesPayload_JSON(t *testing.T) {
	data := []byte(`[{"name":"a"}]`)
	args := SyncCookiesArgs{CookiesJSON: string(data)}

	got, err := parseSyncCookiesPayload(args)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("unexpected payload")
	}
}

func TestParseSyncCookiesPayload_Missing(t *testing.T) {
	_, err := parseSyncCookiesPayload(SyncCookiesArgs{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPublishContentArgs_HasLocationAndMarkerTagsFields(t *testing.T) {
	typ := reflect.TypeOf(PublishContentArgs{})
	if _, ok := typ.FieldByName("Location"); !ok {
		t.Fatalf("missing Location field")
	}
	if _, ok := typ.FieldByName("MarkerTags"); !ok {
		t.Fatalf("missing MarkerTags field")
	}
}

func TestBuildPublishContentArgsMap_IncludesLocationAndMarkerTags(t *testing.T) {
	args := PublishContentArgs{
		Title:      "t",
		Content:    "c",
		Images:     []string{"1.jpg"},
		Tags:       []string{"标签1"},
		Location:   strPtr("深圳湾公园"),
		MarkerTags: []string{"深圳湾公园", "张三"},
		ScheduleAt: strPtr("2026-01-01T00:00:00Z"),
	}

	got := buildPublishContentArgsMap(args)
	if got["location"] != "深圳湾公园" {
		t.Fatalf("unexpected location: %v", got["location"])
	}

	markerTags, ok := got["marker_tags"].([]interface{})
	if !ok {
		t.Fatalf("unexpected marker_tags type: %T", got["marker_tags"])
	}
	wantTags := []interface{}{"深圳湾公园", "张三"}
	if !reflect.DeepEqual(markerTags, wantTags) {
		t.Fatalf("unexpected marker_tags: %v", markerTags)
	}
}

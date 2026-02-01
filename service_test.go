package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	apppublish "github.com/xpzouying/xiaohongshu-mcp/internal/app/publish"
	"github.com/xpzouying/xiaohongshu-mcp/internal/app/testkit"
	domainpublish "github.com/xpzouying/xiaohongshu-mcp/internal/domain/publish"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/polling"
)

func testPollingModules() PollingModules {
	base := polling.Module{
		TimeoutMs:  1000,
		IntervalMs: 100,
		MaxRetries: 1,
		Delays:     map[string]int{"wait_1000ms": 1000},
	}
	return PollingModules{
		Publish:     base,
		Draft:       base,
		Video:       base,
		Interaction: base,
		Analytics:   base,
		Auth:        base,
	}
}

func TestPublishContent_UsesUsecase(t *testing.T) {
	gw := &testkit.FakePublishGateway{}
	uc := apppublish.Usecase{Gateway: gw, Limits: domainpublish.Limits{MaxTags: 10, MinImages: 1, MaxImages: 9}}
	service, err := NewXiaohongshuServiceWithModules(&uc, testPollingModules())
	if err != nil {
		t.Fatalf("service init err: %v", err)
	}
	req := &PublishRequest{
		Title:   "t",
		Content: "c",
		Images:  []string{"/tmp/placeholder.jpg"},
	}
	if _, err := service.PublishContent(context.Background(), req); err != nil {
		t.Fatalf("publish err: %v", err)
	}
	if gw.ImageCalls != 1 {
		t.Fatalf("expected gateway call")
	}
}

func TestSyncCookies_WritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")
	os.Setenv("COOKIES_PATH", path)
	t.Cleanup(func() { os.Unsetenv("COOKIES_PATH") })

	service, err := NewXiaohongshuServiceWithModules(nil, testPollingModules())
	if err != nil {
		t.Fatalf("service init err: %v", err)
	}
	data := []byte(`[{"name":"a"}]`)
	gotPath, gotSize, err := service.SyncCookies(context.Background(), data)
	if err != nil {
		t.Fatalf("sync err: %v", err)
	}
	if gotPath != path {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotSize != int64(len(data)) {
		t.Fatalf("unexpected size: %d", gotSize)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read err: %v", err)
	}
	if string(content) != string(data) {
		t.Fatalf("unexpected content")
	}
}

func TestPublishRequest_HasLocationAndMarkerTagsFields(t *testing.T) {
	typ := reflect.TypeOf(PublishRequest{})
	if _, ok := typ.FieldByName("Location"); !ok {
		t.Fatalf("missing Location field")
	}
	if _, ok := typ.FieldByName("MarkerTags"); !ok {
		t.Fatalf("missing MarkerTags field")
	}
}

func TestPublishContent_MapsLocationAndMarkerTags(t *testing.T) {
	gw := &testkit.FakePublishGateway{}
	uc := apppublish.Usecase{Gateway: gw, Limits: domainpublish.Limits{MaxTags: 10, MinImages: 1, MaxImages: 9}}
	service, err := NewXiaohongshuServiceWithModules(&uc, testPollingModules())
	if err != nil {
		t.Fatalf("service init err: %v", err)
	}
	req := &PublishRequest{
		Title:      "t",
		Content:    "c",
		Images:     []string{"/tmp/placeholder.jpg"},
		Location:   "深圳湾公园",
		MarkerTags: []string{"深圳湾公园", "张三"},
	}
	if _, err := service.PublishContent(context.Background(), req); err != nil {
		t.Fatalf("publish err: %v", err)
	}
	if gw.LastImage.Location != "深圳湾公园" {
		t.Fatalf("unexpected location: %s", gw.LastImage.Location)
	}
	if len(gw.LastImage.MarkerTags) != 2 {
		t.Fatalf("unexpected marker tags: %v", gw.LastImage.MarkerTags)
	}
}

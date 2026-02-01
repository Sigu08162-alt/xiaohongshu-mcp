package publish

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xpzouying/xiaohongshu-mcp/internal/domain/publish"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
)

func testPollingModule() PollingModule {
	return PollingModule{
		TimeoutMs:  1000,
		IntervalMs: 10,
		MaxRetries: 1,
		Delays: map[string]int{
			"page_stable_ms":          1,
			"pre_submit_render_ms":    1,
			"post_content_render_ms":  1,
			"scroll_into_view_wait_ms": 1,
			"click_retry_wait_ms":     1,
			"tag_editor_ready_ms":     1,
			"tag_arrow_step_ms":       1,
			"tag_after_enter_ms":      1,
			"tag_hash_delay_ms":       1,
			"tag_char_delay_ms":       1,
			"tag_after_text_ms":       1,
			"tag_suggestion_click_ms": 1,
			"tag_after_tag_ms":        1,
		},
	}
}

type fakePage struct {
	Calls                []string
	ElementCalls         []string
	ElementsCalls        []string
	WaitForFunctionCalls int
	EvalCalls            []string
	URLValue             string
	ElementResults       map[string]*fakeElement
	ElementsResults      map[string][]*fakeElement
	HasResults           map[string]bool
	TextResults          map[string]string
	EvalResult           interface{}
	IsVisibleResults     map[string]bool
}

func (p *fakePage) Goto(url string) error {
	p.Calls = append(p.Calls, "goto")
	return nil
}

func (p *fakePage) Reload() error {
	p.Calls = append(p.Calls, "reload")
	return nil
}

func (p *fakePage) WaitLoad() error {
	p.Calls = append(p.Calls, "waitload")
	return nil
}

func (p *fakePage) WaitIdle() error {
	p.Calls = append(p.Calls, "waitidle")
	return nil
}

func (p *fakePage) WaitDOMStable(maxWait time.Duration, stabilityThreshold float64) error {
	p.Calls = append(p.Calls, "waitdomstable")
	return nil
}

func (p *fakePage) URL() string {
	if p.URLValue != "" {
		return p.URLValue
	}
	return "https://example.com?target=image&published=true"
}

func (p *fakePage) Click(selector string) error {
	p.Calls = append(p.Calls, "click:"+selector)
	return nil
}

func (p *fakePage) ClickForce(selector string) error {
	p.Calls = append(p.Calls, "clickforce:"+selector)
	return nil
}

func (p *fakePage) DoubleClick(selector string) error {
	p.Calls = append(p.Calls, "doubleclick:"+selector)
	return nil
}

func (p *fakePage) Fill(selector, value string) error {
	p.Calls = append(p.Calls, "fill:"+selector)
	return nil
}

func (p *fakePage) Type(selector, value string) error {
	p.Calls = append(p.Calls, "type:"+selector)
	return nil
}

func (p *fakePage) SetFiles(selector string, files []string) error {
	p.Calls = append(p.Calls, "files:"+selector)
	return nil
}

func (p *fakePage) Hover(selector string) error {
	p.Calls = append(p.Calls, "hover:"+selector)
	return nil
}

func (p *fakePage) Focus(selector string) error {
	p.Calls = append(p.Calls, "focus:"+selector)
	return nil
}

func (p *fakePage) Press(key string) error {
	p.Calls = append(p.Calls, "press:"+key)
	return nil
}

func (p *fakePage) ScrollIntoView(selector string) error {
	p.Calls = append(p.Calls, "scrollintoview:"+selector)
	return nil
}

func (p *fakePage) ScrollBy(deltaX, deltaY float64) error {
	p.Calls = append(p.Calls, "scrollby")
	return nil
}

func (p *fakePage) IsVisible(selector string) (bool, error) {
	if p.IsVisibleResults != nil {
		if value, ok := p.IsVisibleResults[selector]; ok {
			return value, nil
		}
	}
	return false, nil
}

func (p *fakePage) Text(selector string) (string, error) {
	if p.TextResults != nil {
		if value, ok := p.TextResults[selector]; ok {
			return value, nil
		}
	}
	return "", nil
}

func (p *fakePage) HTML(selector string) (string, error) {
	return "", nil
}

func (p *fakePage) Attribute(selector, name string) (string, error) {
	return "", nil
}

func (p *fakePage) WaitVisible(selector string) error {
	return nil
}

func (p *fakePage) WaitHidden(selector string) error {
	return nil
}

func (p *fakePage) WaitForSelector(selector string, timeout time.Duration) error {
	return nil
}

func (p *fakePage) WaitForFunction(expression string, timeout time.Duration) error {
	p.WaitForFunctionCalls++
	return nil
}

func (p *fakePage) Eval(expression string, args ...interface{}) (interface{}, error) {
	p.EvalCalls = append(p.EvalCalls, expression)
	return p.EvalResult, nil
}

func (p *fakePage) EvalOnSelector(selector, expression string, args ...interface{}) (interface{}, error) {
	p.EvalCalls = append(p.EvalCalls, expression)
	return p.EvalResult, nil
}

func (p *fakePage) Screenshot(path string) error {
	return nil
}

func (p *fakePage) ScreenshotFullPage(path string) error {
	return nil
}

func (p *fakePage) Element(selector string) (browser.Element, error) {
	p.ElementCalls = append(p.ElementCalls, selector)
	if p.ElementResults != nil {
		if elem, ok := p.ElementResults[selector]; ok {
			return elem, nil
		}
	}
	return &fakeElement{Selector: selector}, nil
}

func (p *fakePage) ElementByRegex(selector, jsRegex string) (browser.Element, error) {
	p.ElementCalls = append(p.ElementCalls, selector)
	return &fakeElement{Selector: selector}, nil
}

func (p *fakePage) Elements(selector string) ([]browser.Element, error) {
	p.ElementsCalls = append(p.ElementsCalls, selector)
	if p.ElementsResults != nil {
		if elems, ok := p.ElementsResults[selector]; ok {
			result := make([]browser.Element, 0, len(elems))
			for _, elem := range elems {
				result = append(result, elem)
			}
			return result, nil
		}
	}
	return []browser.Element{}, nil
}

func (p *fakePage) Has(selector string) (bool, error) {
	if p.HasResults != nil {
		if value, ok := p.HasResults[selector]; ok {
			return value, nil
		}
	}
	return false, nil
}

func (p *fakePage) HasRegex(selector, jsRegex string) (bool, error) {
	return false, nil
}

func (p *fakePage) Mouse() browser.Mouse {
	return &fakeMouse{}
}

func (p *fakePage) Keyboard() browser.Keyboard {
	return &fakeKeyboard{}
}

func (p *fakePage) Route(urlPattern string, handler browser.RouteHandler) error {
	return nil
}

func (p *fakePage) UnrouteAll() error {
	return nil
}

func (p *fakePage) WithContext(ctx context.Context) browser.Page {
	return p
}

func (p *fakePage) WithTimeout(timeout time.Duration) browser.Page {
	return p
}

func (p *fakePage) Close() error {
	return nil
}

func (p *fakePage) HasElementCall(selector string) bool {
	for _, call := range p.ElementCalls {
		if call == selector {
			return true
		}
	}
	return false
}

func (p *fakePage) HasWaitForFunctionCall() bool {
	return p.WaitForFunctionCalls > 0
}

type fakeEngine struct{ page *fakePage }

func (e *fakeEngine) Start() error {
	return nil
}

func (e *fakeEngine) NewPage() (browser.Page, error) {
	return e.page, nil
}

func (e *fakeEngine) Close() error {
	return nil
}

type fakeElement struct {
	Selector string
	TextVal  string
	Visible  bool
}

func (e *fakeElement) Click() error                            { return nil }
func (e *fakeElement) ClickForce() error                       { return nil }
func (e *fakeElement) DoubleClick() error                      { return nil }
func (e *fakeElement) Hover() error                            { return nil }
func (e *fakeElement) Focus() error                            { return nil }
func (e *fakeElement) Fill(value string) error                 { return nil }
func (e *fakeElement) Type(value string) error                 { return nil }
func (e *fakeElement) Press(key string) error                  { return nil }
func (e *fakeElement) Input(value string) error                { return nil }
func (e *fakeElement) SetFiles(files []string) error           { return nil }
func (e *fakeElement) ScrollIntoView() error                   { return nil }
func (e *fakeElement) WaitVisible() error                      { return nil }
func (e *fakeElement) WaitHidden() error                       { return nil }
func (e *fakeElement) WaitStable(duration time.Duration) error { return nil }
func (e *fakeElement) IsVisible() (bool, error)                { return e.Visible, nil }
func (e *fakeElement) Text() (string, error)                   { return e.TextVal, nil }
func (e *fakeElement) HTML() (string, error)                   { return "", nil }
func (e *fakeElement) Attribute(name string) (string, error)   { return "", nil }
func (e *fakeElement) Value() (string, error)                  { return "", nil }
func (e *fakeElement) Eval(expression string, args ...interface{}) (interface{}, error) {
	return nil, nil
}
func (e *fakeElement) Element(selector string) (browser.Element, error) {
	return &fakeElement{Selector: selector}, nil
}
func (e *fakeElement) Elements(selector string) ([]browser.Element, error) {
	return []browser.Element{}, nil
}
func (e *fakeElement) Remove() error { return nil }
func (e *fakeElement) BoundingBox() (*browser.BoundingBox, error) {
	return &browser.BoundingBox{}, nil
}
func (e *fakeElement) Frame() (browser.Page, error) { return nil, nil }

type fakeMouse struct{}

func (m *fakeMouse) MoveTo(x, y float64) error { return nil }
func (m *fakeMouse) Click(button browser.MouseButton, opts ...browser.MouseClickOption) error {
	return nil
}
func (m *fakeMouse) Down(button browser.MouseButton) error { return nil }
func (m *fakeMouse) Up(button browser.MouseButton) error   { return nil }
func (m *fakeMouse) Wheel(deltaX, deltaY float64) error    { return nil }

type fakeKeyboard struct{}

func (k *fakeKeyboard) Type(text string) error       { return nil }
func (k *fakeKeyboard) InsertText(text string) error { return nil }
func (k *fakeKeyboard) Press(key string) error       { return nil }
func (k *fakeKeyboard) Down(key string) error        { return nil }
func (k *fakeKeyboard) Up(key string) error          { return nil }

func TestGateway_PublishImage_UsesSelectors(t *testing.T) {
	engine := &fakeEngine{page: &fakePage{}}
	cfg := Config{
		PublishImageURL: "https://example.com",
		PublishVideoURL: "https://example.com",
		Selectors: map[string]string{
			"upload_input": "input[type=file]",
			"title_input":  "input[name=title]",
			"content":      "textarea[name=content]",
			"submit":       "button[type=submit]",
		},
		PublishPolling: testPollingModule(),
		DraftPolling:   testPollingModule(),
		VideoPolling:   testPollingModule(),
	}
	gw, err := NewGateway(cfg, engine)
	if err != nil {
		t.Fatalf("new gateway err: %v", err)
	}
	err = gw.PublishImage(context.Background(), publish.ImageContent{
		Title:      "t",
		Content:    "c",
		ImagePaths: []string{"1.jpg"},
	})
	if err != nil {
		t.Fatalf("publish err: %v", err)
	}
	if len(engine.page.Calls) == 0 {
		t.Fatalf("expected page calls")
	}
}

func TestGateway_PublishVideo_UsesSelectors(t *testing.T) {
	engine := &fakeEngine{page: &fakePage{}}
	cfg := Config{
		PublishImageURL: "https://example.com",
		PublishVideoURL: "https://example.com",
		Selectors: map[string]string{
			"upload_input": "input[type=file]",
			"title_input":  "input[name=title]",
			"content":      "textarea[name=content]",
			"submit":       "button[type=submit]",
		},
		PublishPolling: testPollingModule(),
		DraftPolling:   testPollingModule(),
		VideoPolling:   testPollingModule(),
	}
	gw, err := NewGateway(cfg, engine)
	if err != nil {
		t.Fatalf("new gateway err: %v", err)
	}
	err = gw.PublishVideo(context.Background(), publish.VideoContent{
		Title:     "t",
		Content:   "c",
		VideoPath: "1.mp4",
	})
	if err != nil {
		t.Fatalf("publish err: %v", err)
	}
	if len(engine.page.Calls) == 0 {
		t.Fatalf("expected page calls")
	}
}

func TestGateway_PublishImage_SetsLocation(t *testing.T) {
	dropdown := &fakeElement{TextVal: "深圳湾公园", Visible: true}
	engine := &fakeEngine{page: &fakePage{
		ElementsResults: map[string][]*fakeElement{
			".d-dropdown-wrapper": {dropdown},
		},
	}}
	cfg := Config{
		PublishImageURL: "https://example.com",
		PublishVideoURL: "https://example.com",
		Selectors: map[string]string{
			"upload_input": "input[type=file]",
			"title_input":  "input[name=title]",
			"content":      "textarea[name=content]",
			"submit":       "button[type=submit]",
		},
		PublishPolling: testPollingModule(),
		DraftPolling:   testPollingModule(),
		VideoPolling:   testPollingModule(),
	}
	gw, err := NewGateway(cfg, engine)
	if err != nil {
		t.Fatalf("new gateway err: %v", err)
	}
	err = gw.PublishImage(context.Background(), publish.ImageContent{
		Title:      "t",
		Content:    "c",
		ImagePaths: []string{"1.jpg"},
		Location:   "深圳湾公园",
	})
	if err != nil {
		t.Fatalf("publish err: %v", err)
	}
	if !engine.page.HasElementCall(".address-box input.d-text") {
		t.Fatalf("expected location input call")
	}
}

func TestGateway_PublishImage_SetsMarkerTags(t *testing.T) {
	engine := &fakeEngine{page: &fakePage{
		ElementsResults: map[string][]*fakeElement{
			".d-new-form-item": {
				{TextVal: "标记地点或标记朋友"},
			},
			"div[role=\"dialog\"] div[role=\"banner\"] ~ *": {
				{TextVal: "地点"},
				{TextVal: "用户"},
			},
		},
		EvalResult: "selected",
	}}
	cfg := Config{
		PublishImageURL: "https://example.com",
		PublishVideoURL: "https://example.com",
		Selectors: map[string]string{
			"upload_input": "input[type=file]",
			"title_input":  "input[name=title]",
			"content":      "textarea[name=content]",
			"submit":       "button[type=submit]",
		},
		PublishPolling: testPollingModule(),
		DraftPolling:   testPollingModule(),
		VideoPolling:   testPollingModule(),
	}
	gw, err := NewGateway(cfg, engine)
	if err != nil {
		t.Fatalf("new gateway err: %v", err)
	}
	err = gw.PublishImage(context.Background(), publish.ImageContent{
		Title:      "t",
		Content:    "c",
		ImagePaths: []string{"1.jpg"},
		MarkerTags: []string{"深圳湾公园"},
	})
	if err != nil {
		t.Fatalf("publish err: %v", err)
	}
	if !engine.page.HasWaitForFunctionCall() {
		t.Fatalf("expected marker dialog wait")
	}
}

func TestWaitForUploadComplete_NoUploading(t *testing.T) {
	page := &fakePage{
		IsVisibleResults: map[string]bool{
			".mask.uploading": false,
			"[class*='uploading']": false,
		},
		EvalResult: 1,
	}

	selectors := resolveUploadSelectors(map[string]string{})
	if err := waitForUploadComplete(page, selectors, 1, 5*time.Millisecond, time.Millisecond); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestWaitForUploadComplete_TimesOut(t *testing.T) {
	page := &fakePage{
		IsVisibleResults: map[string]bool{
			".mask.uploading": true,
		},
		EvalResult: 0,
	}

	selectors := resolveUploadSelectors(map[string]string{})
	err := waitForUploadComplete(page, selectors, 1, 3*time.Millisecond, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "图片上传中") {
		t.Fatalf("expected upload timeout error, got %v", err)
	}
}

func TestResolveUploadSelectors_Defaults(t *testing.T) {
	selectors := resolveUploadSelectors(map[string]string{})
	if selectors.UploadingMask != ".mask.uploading" {
		t.Fatalf("unexpected uploading mask: %s", selectors.UploadingMask)
	}
	if selectors.UploadingClass != "[class*='uploading']" {
		t.Fatalf("unexpected uploading class: %s", selectors.UploadingClass)
	}
	if selectors.UploadPreview != "img.preview" {
		t.Fatalf("unexpected upload preview: %s", selectors.UploadPreview)
	}
	if selectors.UploadingToast != ".creator-publish-toast" {
		t.Fatalf("unexpected uploading toast: %s", selectors.UploadingToast)
	}
}

func TestResolveUploadSelectors_Overrides(t *testing.T) {
	selectors := resolveUploadSelectors(map[string]string{
		"uploading_mask":      ".u-mask",
		"uploading_class":     ".u-uploading",
		"upload_preview":      ".u-preview",
		"uploading_toast":     ".u-toast",
	})
	if selectors.UploadingMask != ".u-mask" {
		t.Fatalf("unexpected uploading mask: %s", selectors.UploadingMask)
	}
	if selectors.UploadingClass != ".u-uploading" {
		t.Fatalf("unexpected uploading class: %s", selectors.UploadingClass)
	}
	if selectors.UploadPreview != ".u-preview" {
		t.Fatalf("unexpected upload preview: %s", selectors.UploadPreview)
	}
	if selectors.UploadingToast != ".u-toast" {
		t.Fatalf("unexpected uploading toast: %s", selectors.UploadingToast)
	}
}

func TestGetDelay_Missing(t *testing.T) {
	_, err := getDelay(PollingModule{}, "missing_key")
	if err == nil {
		t.Fatalf("expected error for missing delay key")
	}
}

func TestGetDelay_Value(t *testing.T) {
	d, err := getDelay(PollingModule{
		Delays: map[string]int{
			"ready_ms": 250,
		},
	}, "ready_ms")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 250*time.Millisecond {
		t.Fatalf("unexpected duration: %v", d)
	}
}

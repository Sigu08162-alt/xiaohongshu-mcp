package publish

import (
	"context"
	"testing"
	"time"

	"github.com/xpzouying/xiaohongshu-mcp/internal/domain/publish"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
)

type fakePage struct {
	Calls                 []string
	ElementCalls          []string
	ElementsCalls         []string
	WaitForFunctionCalls  int
	EvalCalls             []string
	URLValue              string
	ElementResults        map[string]*fakeElement
	ElementsResults       map[string][]*fakeElement
	HasResults            map[string]bool
	TextResults           map[string]string
	EvalResult            interface{}
	IsVisibleResults      map[string]bool
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

func (e *fakeElement) Click() error                          { return nil }
func (e *fakeElement) ClickForce() error                     { return nil }
func (e *fakeElement) DoubleClick() error                    { return nil }
func (e *fakeElement) Hover() error                          { return nil }
func (e *fakeElement) Focus() error                          { return nil }
func (e *fakeElement) Fill(value string) error               { return nil }
func (e *fakeElement) Type(value string) error               { return nil }
func (e *fakeElement) Press(key string) error                { return nil }
func (e *fakeElement) Input(value string) error              { return nil }
func (e *fakeElement) SetFiles(files []string) error         { return nil }
func (e *fakeElement) ScrollIntoView() error                 { return nil }
func (e *fakeElement) WaitVisible() error                    { return nil }
func (e *fakeElement) WaitHidden() error                     { return nil }
func (e *fakeElement) WaitStable(duration time.Duration) error { return nil }
func (e *fakeElement) IsVisible() (bool, error)              { return e.Visible, nil }
func (e *fakeElement) Text() (string, error)                 { return e.TextVal, nil }
func (e *fakeElement) HTML() (string, error)                 { return "", nil }
func (e *fakeElement) Attribute(name string) (string, error) { return "", nil }
func (e *fakeElement) Value() (string, error)                { return "", nil }
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

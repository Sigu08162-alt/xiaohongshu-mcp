package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
)

const (
	defaultMaxSize    = 2
	defaultIdleExpiry = 30 * time.Minute
	healthCheckURL    = "about:blank"
)

// entry holds a pooled page with metadata.
type entry struct {
	page     browser.Page
	lastUsed time.Time
}

// BrowserPool manages a pool of reusable browser pages backed by a single Engine.
// It is safe for concurrent use.
type BrowserPool struct {
	mu      sync.Mutex
	engine  browser.Engine
	pages   []*entry
	maxSize int
	expiry  time.Duration
	started bool
}

// New creates a BrowserPool with the given engine.
func New(engine browser.Engine, maxSize int) *BrowserPool {
	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}
	return &BrowserPool{
		engine:  engine,
		maxSize: maxSize,
		expiry:  defaultIdleExpiry,
	}
}

// Start initialises the underlying browser engine.
func (p *BrowserPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return nil
	}
	if err := p.engine.Start(); err != nil {
		return fmt.Errorf("browser pool: start engine: %w", err)
	}
	p.started = true
	return nil
}

// Acquire returns a healthy page from the pool, creating one if needed.
// The caller must call Release when done.
func (p *BrowserPool) Acquire(ctx context.Context) (browser.Page, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		if err := p.engine.Start(); err != nil {
			return nil, fmt.Errorf("browser pool: start engine: %w", err)
		}
		p.started = true
	}

	// Try to reuse an idle page.
	for len(p.pages) > 0 {
		e := p.pages[len(p.pages)-1]
		p.pages = p.pages[:len(p.pages)-1]

		if time.Since(e.lastUsed) > p.expiry {
			_ = e.page.Close()
			continue
		}
		if p.isHealthy(e.page) {
			return e.page, nil
		}
		_ = e.page.Close()
	}

	// No reusable page — create a new one.
	page, err := p.engine.NewPage()
	if err != nil {
		return nil, fmt.Errorf("browser pool: new page: %w", err)
	}
	return page, nil
}

// Release returns a page to the pool. If the pool is full the page is closed.
func (p *BrowserPool) Release(page browser.Page) {
	if page == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.pages) >= p.maxSize {
		_ = page.Close()
		return
	}
	p.pages = append(p.pages, &entry{page: page, lastUsed: time.Now()})
}

// Discard closes a page without returning it to the pool (use after errors).
func (p *BrowserPool) Discard(page browser.Page) {
	if page != nil {
		_ = page.Close()
	}
}

// Close shuts down all pooled pages and the engine.
func (p *BrowserPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, e := range p.pages {
		_ = e.page.Close()
	}
	p.pages = nil
	if p.started {
		_ = p.engine.Close()
		p.started = false
	}
}

// isHealthy does a lightweight check — navigate to about:blank succeeds.
func (p *BrowserPool) isHealthy(page browser.Page) bool {
	err := page.Goto(healthCheckURL)
	return err == nil
}

// WithPage is a convenience wrapper: acquires a page, runs fn, releases or
// discards depending on whether fn returned an error.
func (p *BrowserPool) WithPage(ctx context.Context, fn func(browser.Page) error) error {
	page, err := p.Acquire(ctx)
	if err != nil {
		return err
	}
	if err := fn(page); err != nil {
		p.Discard(page)
		return err
	}
	p.Release(page)
	return nil
}

package selector

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// HotLoader watches a YAML config file and reloads it when modified.
type HotLoader struct {
	mu       sync.RWMutex
	path     string
	modTime  time.Time
	config   *SelectorConfig
	interval time.Duration
	stop     chan struct{}
}

// NewHotLoader creates a loader that checks for file changes every interval.
// It performs an initial load and starts a background watcher goroutine.
func NewHotLoader(path string, interval time.Duration) (*HotLoader, error) {
	l := &HotLoader{
		path:     path,
		interval: interval,
		stop:     make(chan struct{}),
	}
	if err := l.reload(); err != nil {
		return nil, err
	}
	go l.watch()
	return l, nil
}

// Get returns the current config (thread-safe).
func (l *HotLoader) Get() *SelectorConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config
}

// Stop stops the background watcher goroutine.
func (l *HotLoader) Stop() {
	close(l.stop)
}

func (l *HotLoader) watch() {
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-ticker.C:
			info, err := os.Stat(l.path)
			if err != nil {
				slog.Warn("selector: stat failed", "path", l.path, "err", err)
				continue
			}
			if info.ModTime().After(l.modTime) {
				if err := l.reload(); err != nil {
					slog.Error("selector: reload failed", "err", err)
				} else {
					l.mu.RLock()
					version := l.config.Version
					l.mu.RUnlock()
					slog.Info("selector: config reloaded", "path", l.path, "version", version)
				}
			}
		}
	}
}

func (l *HotLoader) reload() error {
	info, err := os.Stat(l.path)
	if err != nil {
		return fmt.Errorf("selector: stat %s: %w", l.path, err)
	}

	data, err := os.ReadFile(l.path)
	if err != nil {
		return fmt.Errorf("selector: read %s: %w", l.path, err)
	}

	var cfg SelectorConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("selector: parse %s: %w", l.path, err)
	}

	l.mu.Lock()
	l.config = &cfg
	l.modTime = info.ModTime()
	l.mu.Unlock()
	return nil
}

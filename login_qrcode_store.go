package main

import (
	"encoding/base64"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultLoginQRCacheTTL = 4 * time.Minute
	defaultPublicBaseURL   = "http://localhost:18060"
)

type cachedLoginQRCode struct {
	png      []byte
	expireAt time.Time
}

type loginQRCodeStore struct {
	mu     sync.Mutex
	items  map[string]cachedLoginQRCode
	latest string
	now    func() time.Time
}

func newLoginQRCodeStore() *loginQRCodeStore {
	return &loginQRCodeStore{
		items: make(map[string]cachedLoginQRCode),
	}
}

func (s *loginQRCodeStore) currentTime() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *loginQRCodeStore) Put(sessionID string, png []byte, ttl time.Duration) {
	if strings.TrimSpace(sessionID) == "" || len(png) == 0 {
		return
	}
	if ttl <= 0 {
		ttl = defaultLoginQRCacheTTL
	}
	now := s.currentTime()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked(now)
	s.items[sessionID] = cachedLoginQRCode{
		png:      append([]byte(nil), png...),
		expireAt: now.Add(ttl),
	}
	s.latest = sessionID
}

func (s *loginQRCodeStore) Get(sessionID string) ([]byte, bool) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, false
	}
	now := s.currentTime()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked(now)
	item, ok := s.items[sessionID]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), item.png...), true
}

func (s *loginQRCodeStore) GetLatest() (string, []byte, bool) {
	now := s.currentTime()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked(now)
	if s.latest == "" {
		return "", nil, false
	}
	item, ok := s.items[s.latest]
	if !ok {
		return "", nil, false
	}
	return s.latest, append([]byte(nil), item.png...), true
}

func (s *loginQRCodeStore) cleanupLocked(now time.Time) {
	for id, item := range s.items {
		if !item.expireAt.After(now) {
			delete(s.items, id)
			if s.latest == id {
				s.latest = ""
			}
		}
	}
	if s.latest != "" {
		return
	}
	for id := range s.items {
		s.latest = id
		return
	}
}

func resolvePublicBaseURL() string {
	base := strings.TrimSpace(os.Getenv("XHS_PUBLIC_BASE_URL"))
	if base == "" {
		return defaultPublicBaseURL
	}
	return strings.TrimRight(base, "/")
}

func resolvePublicBaseURLFromListenAddr(listenAddr string) string {
	base := strings.TrimSpace(os.Getenv("XHS_PUBLIC_BASE_URL"))
	if base != "" {
		return strings.TrimRight(base, "/")
	}

	addr := strings.TrimSpace(listenAddr)
	if addr == "" {
		return defaultPublicBaseURL
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + strings.TrimRight(addr, "/")
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + host + ":" + port
}

func parseLoginQRTimeout(timeout string) time.Duration {
	d, err := time.ParseDuration(timeout)
	if err != nil || d <= 0 {
		return defaultLoginQRCacheTTL
	}
	return d
}

func decodeLoginQRCodeImage(img string) ([]byte, bool) {
	raw := strings.TrimSpace(img)
	raw = strings.TrimPrefix(raw, "data:image/png;base64,")
	if raw == "" {
		return nil, false
	}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, false
	}
	return data, true
}

func (s *AppServer) loginQRCodeImageURL(sessionID string) string {
	if strings.TrimSpace(sessionID) == "" {
		return ""
	}
	base := strings.TrimRight(s.publicBaseURL, "/")
	return base + "/api/v1/login/qrcode/image?session_id=" + url.QueryEscape(sessionID)
}

func (s *AppServer) cacheLoginQRCode(result *LoginQrcodeResponse) (string, bool) {
	if s == nil || s.loginQRStore == nil || result == nil || result.IsLoggedIn {
		return "", false
	}
	if strings.TrimSpace(result.SessionID) == "" {
		return "", false
	}

	data, ok := decodeLoginQRCodeImage(result.Img)
	if !ok {
		return "", false
	}

	s.loginQRStore.Put(result.SessionID, data, parseLoginQRTimeout(result.Timeout))
	return s.loginQRCodeImageURL(result.SessionID), true
}

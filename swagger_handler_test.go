package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSwaggerRedirect(t *testing.T) {
	router := setupRoutes(&AppServer{})

	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("expected status %d, got %d", http.StatusMovedPermanently, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/swagger/index.html" {
		t.Fatalf("expected redirect to /swagger/index.html, got %q", got)
	}
}

func TestSwaggerIndexServed(t *testing.T) {
	router := setupRoutes(&AppServer{})

	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html content type, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "SwaggerUIBundle") {
		t.Fatalf("expected swagger UI bootstrap content")
	}
}

func TestSwaggerDocServed(t *testing.T) {
	router := setupRoutes(&AppServer{})

	req := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected application/json content type, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "\"swagger\"") {
		t.Fatalf("expected swagger document payload")
	}
}

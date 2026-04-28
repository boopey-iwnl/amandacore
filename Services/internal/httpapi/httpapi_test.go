package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDecodeJSONRejectsWrongContentType(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"ok":true}`))
	request.Header.Set("Content-Type", "text/plain")

	var payload map[string]bool
	if err := DecodeJSON(request, &payload); err == nil {
		t.Fatal("expected content-type validation error")
	}
}

func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(strings.Repeat("x", int(DefaultMaxJSONBodyBytes)+1)))
	request.Header.Set("Content-Type", "application/json")

	var payload map[string]bool
	if err := DecodeJSON(request, &payload); err == nil {
		t.Fatal("expected oversized body error")
	}
}

func TestDecodeJSONRejectsMultipleValues(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"ok":true} {"again":true}`))
	request.Header.Set("Content-Type", "application/json")

	var payload map[string]bool
	if err := DecodeJSON(request, &payload); err == nil {
		t.Fatal("expected multiple JSON value error")
	}
}

func TestRateLimiterRejectsAfterWindowLimit(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	now := time.Unix(100, 0)

	if !limiter.Allow("login:test", now) || !limiter.Allow("login:test", now.Add(time.Second)) {
		t.Fatal("expected first two attempts to be accepted")
	}
	if limiter.Allow("login:test", now.Add(2*time.Second)) {
		t.Fatal("expected third attempt in window to be rejected")
	}
	if !limiter.Allow("login:test", now.Add(time.Minute)) {
		t.Fatal("expected next window to be accepted")
	}
}

func FuzzDecodeJSONMalformedInput(f *testing.F) {
	f.Add([]byte(`{"username":"amanda","password":"local"}`))
	f.Add([]byte(`{`))
	f.Add([]byte(`[]`))

	f.Fuzz(func(t *testing.T, body []byte) {
		request := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(string(body)))
		request.Header.Set("Content-Type", "application/json")

		var payload map[string]any
		_ = DecodeJSON(request, &payload)
	})
}

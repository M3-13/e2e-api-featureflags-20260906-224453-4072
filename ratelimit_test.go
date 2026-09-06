package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitWriteRequests(t *testing.T) {
	handler := RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/flags", nil)
	for i := 0; i < rateLimitBurst; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rr.Code)
		}
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on burst+1, got %d", rr.Code)
	}
	if rr.Body.String() != `{"error":"rate limit exceeded"}` {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

func TestRateLimitGetUnlimited(t *testing.T) {
	handler := RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	for i := 0; i < rateLimitBurst+20; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200 for GET, got %d", i+1, rr.Code)
		}
	}
}

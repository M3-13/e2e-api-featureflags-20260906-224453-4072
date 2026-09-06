package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func captureLog(t *testing.T, f func()) string {
	t.Helper()
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)
	f()
	return buf.String()
}

func TestLoggingCapturesMethodPathStatusAndDuration(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	rr := httptest.NewRecorder()
	logOutput := captureLog(t, func() {
		handler := Logging(next)
		req := httptest.NewRequest(http.MethodPost, "/flags", nil)
		handler.ServeHTTP(rr, req)
	})

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}

	if !strings.Contains(logOutput, "POST") {
		t.Fatalf("expected log to contain method POST, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "/flags") {
		t.Fatalf("expected log to contain path /flags, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "201") {
		t.Fatalf("expected log to contain status 201, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "ns") && !strings.Contains(logOutput, "µs") &&
		!strings.Contains(logOutput, "ms") && !strings.Contains(logOutput, "s") {
		t.Fatalf("expected log to contain a duration, got %q", logOutput)
	}
}

func TestLoggingDefaultsStatusTo200(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// no explicit WriteHeader; the wrapper must default to 200
	})

	rr := httptest.NewRecorder()
	logOutput := captureLog(t, func() {
		handler := Logging(next)
		req := httptest.NewRequest(http.MethodGet, "/flags", nil)
		handler.ServeHTTP(rr, req)
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if !strings.Contains(logOutput, "200") {
		t.Fatalf("expected log to contain status 200, got %q", logOutput)
	}
}

func TestLoggingDoesNotLogQueryParameters(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	logOutput := captureLog(t, func() {
		handler := Logging(next)
		req := httptest.NewRequest(http.MethodGet, "/flags/evaluate?user=geheim", nil)
		handler.ServeHTTP(rr, req)
	})

	if strings.Contains(logOutput, "geheim") {
		t.Fatalf("expected query value 'geheim' to be absent from log, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "/flags/evaluate") {
		t.Fatalf("expected log to contain path /flags/evaluate, got %q", logOutput)
	}
	if strings.Contains(logOutput, "?") {
		t.Fatalf("expected no query string in log, got %q", logOutput)
	}
}

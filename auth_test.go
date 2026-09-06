package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const testAPIKey = "test-api-key"

func TestMain(m *testing.M) {
	os.Setenv("FLAG_API_KEY", testAPIKey)
	os.Exit(m.Run())
}

func TestRequireAuthMissingKey(t *testing.T) {
	t.Setenv("FLAG_API_KEY", "secret")
	handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without key, got %d", rr.Code)
	}
	if rr.Body.String() != `{"error":"unauthorized"}` {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

func TestRequireAuthEmptyKey(t *testing.T) {
	t.Setenv("FLAG_API_KEY", "secret")
	handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	req.Header.Set("X-API-Key", "")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with empty key, got %d", rr.Code)
	}
}

func TestRequireAuthWrongKey(t *testing.T) {
	t.Setenv("FLAG_API_KEY", "secret")
	handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	req.Header.Set("X-API-Key", "wrong")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong key, got %d", rr.Code)
	}
}

func TestRequireAuthValidKey(t *testing.T) {
	t.Setenv("FLAG_API_KEY", "secret")
	handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	req.Header.Set("X-API-Key", "secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid key, got %d", rr.Code)
	}
}

func TestHealthzIsPublic(t *testing.T) {
	t.Setenv("FLAG_API_KEY", "secret")
	handler := newHandler()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for public /healthz, got %d", rr.Code)
	}
}

func TestProtectedRouteRequiresAuth(t *testing.T) {
	t.Setenv("FLAG_API_KEY", "secret")
	handler := newHandler()

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for protected route without key, got %d", rr.Code)
	}
}

func TestProtectedRouteWithKey(t *testing.T) {
	t.Setenv("FLAG_API_KEY", "secret")
	handler := newHandler()

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	req.Header.Set("X-API-Key", "secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for protected route with key, got %d", rr.Code)
	}
}

func TestProtectedRouteWithoutEnvKeyAnswers401(t *testing.T) {
	t.Setenv("FLAG_API_KEY", "")
	handler := newHandler()

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	req.Header.Set("X-API-Key", "anything")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when FLAG_API_KEY is unset, got %d", rr.Code)
	}
}

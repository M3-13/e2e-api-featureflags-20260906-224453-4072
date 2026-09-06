package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func resetStore() {
	flagStore = NewStore()
}

func doRequest(t *testing.T, method, path, body string, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rr := httptest.NewRecorder()
	newHandler().ServeHTTP(rr, req)
	return rr
}

func postFlag(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doRequest(t, http.MethodPost, "/flags", body, "application/json")
}

func decodeFlag(t *testing.T, body string) Flag {
	t.Helper()
	var f Flag
	if err := json.Unmarshal([]byte(body), &f); err != nil {
		t.Fatalf("failed to decode flag JSON %q: %v", body, err)
	}
	return f
}

func decodeError(t *testing.T, body string) string {
	t.Helper()
	var e map[string]string
	if err := json.Unmarshal([]byte(body), &e); err != nil {
		t.Fatalf("failed to decode error JSON %q: %v", body, err)
	}
	return e["error"]
}

func TestCreateFlagSuccess(t *testing.T) {
	resetStore()
	rr := postFlag(t, `{"key":"feature-x","enabled":true,"rollout_percent":50}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body %q)", rr.Code, rr.Body.String())
	}
	f := decodeFlag(t, rr.Body.String())
	if f.Key != "feature-x" || !f.Enabled || f.RolloutPercent != 50 {
		t.Fatalf("unexpected flag: %+v", f)
	}
}

func TestCreateFlagDefaultRollout(t *testing.T) {
	resetStore()
	rr := postFlag(t, `{"key":"feature-y","enabled":true}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body %q)", rr.Code, rr.Body.String())
	}
	f := decodeFlag(t, rr.Body.String())
	if f.RolloutPercent != 100 {
		t.Fatalf("expected default rollout 100, got %d", f.RolloutPercent)
	}
}

func TestCreateFlagTrimsKey(t *testing.T) {
	resetStore()
	rr := postFlag(t, `{"key":"  feature-z  ","enabled":true}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body %q)", rr.Code, rr.Body.String())
	}
	f := decodeFlag(t, rr.Body.String())
	if f.Key != "feature-z" {
		t.Fatalf("expected trimmed key, got %q", f.Key)
	}
}

func TestCreateFlagEmptyKey(t *testing.T) {
	resetStore()
	rr := postFlag(t, `{"key":"   ","enabled":true}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if decodeError(t, rr.Body.String()) == "" {
		t.Fatal("expected error message")
	}
}

func TestCreateFlagMissingEnabled(t *testing.T) {
	resetStore()
	rr := postFlag(t, `{"key":"no-enabled"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateFlagRolloutOutOfRange(t *testing.T) {
	resetStore()
	for _, v := range []string{"-1", "101"} {
		rr := postFlag(t, `{"key":"range","enabled":true,"rollout_percent":`+v+`}`)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for rollout_percent %s, got %d", v, rr.Code)
		}
	}
}

func TestCreateFlagDuplicate(t *testing.T) {
	resetStore()
	if rr := postFlag(t, `{"key":"dup","enabled":true}`); rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	rr := postFlag(t, `{"key":"dup","enabled":false}`)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

func TestCreateFlagInvalidJSON(t *testing.T) {
	resetStore()
	rr := postFlag(t, `{not json}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateFlagUnsupportedContentType(t *testing.T) {
	resetStore()
	rr := doRequest(t, http.MethodPost, "/flags", `{"key":"x","enabled":true}`, "text/plain")
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rr.Code)
	}
}

func TestCreateFlagJSONWithCharset(t *testing.T) {
	resetStore()
	rr := doRequest(t, http.MethodPost, "/flags", `{"key":"charset","enabled":true}`, "application/json; charset=utf-8")
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestCreateFlagBodyTooLarge(t *testing.T) {
	resetStore()
	body := `{"key":"big","enabled":true,"description":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	rr := postFlag(t, body)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
}

func TestCreateFlagTrailingDataOverLimit(t *testing.T) {
	resetStore()
	body := `{"key":"prefix","enabled":true}` + strings.Repeat(" ", maxBodyBytes)
	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(body))
	req.ContentLength = -1
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	newHandler().ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
}

func TestUpdateFlagTrailingDataOverLimit(t *testing.T) {
	resetStore()
	postFlag(t, `{"key":"upd","enabled":true}`)
	body := `{"enabled":false}` + strings.Repeat(" ", maxBodyBytes)
	req := httptest.NewRequest(http.MethodPut, "/flags/upd", strings.NewReader(body))
	req.ContentLength = -1
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	newHandler().ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
}

func TestListFlagsSorted(t *testing.T) {
	resetStore()
	for _, k := range []string{"charlie", "alpha", "bravo"} {
		rr := postFlag(t, `{"key":"`+k+`","enabled":true}`)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rr.Code)
		}
	}
	rr := doRequest(t, http.MethodGet, "/flags", "", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var flags []Flag
	if err := json.Unmarshal(rr.Body.Bytes(), &flags); err != nil {
		t.Fatalf("failed to decode list: %v", err)
	}
	want := []string{"alpha", "bravo", "charlie"}
	if len(flags) != len(want) {
		t.Fatalf("expected %d flags, got %d", len(want), len(flags))
	}
	for i, k := range want {
		if flags[i].Key != k {
			t.Fatalf("expected key %q at index %d, got %q", k, i, flags[i].Key)
		}
	}
}

func TestListFlagsEmpty(t *testing.T) {
	resetStore()
	rr := doRequest(t, http.MethodGet, "/flags", "", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if strings.TrimSpace(rr.Body.String()) != "[]" {
		t.Fatalf("expected empty JSON array, got %q", rr.Body.String())
	}
}

func TestGetFlag(t *testing.T) {
	resetStore()
	postFlag(t, `{"key":"known","enabled":true,"rollout_percent":30}`)

	rr := doRequest(t, http.MethodGet, "/flags/known", "", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if f := decodeFlag(t, rr.Body.String()); f.Key != "known" || f.RolloutPercent != 30 {
		t.Fatalf("unexpected flag: %+v", f)
	}
}

func TestGetFlagNotFound(t *testing.T) {
	resetStore()
	rr := doRequest(t, http.MethodGet, "/flags/missing", "", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if decodeError(t, rr.Body.String()) == "" {
		t.Fatal("expected error message")
	}
}

func TestUpdateFlagPartial(t *testing.T) {
	resetStore()
	postFlag(t, `{"key":"upd","enabled":true,"description":"old","rollout_percent":10}`)

	rr := doRequest(t, http.MethodPut, "/flags/upd", `{"enabled":false}`, "application/json")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body %q)", rr.Code, rr.Body.String())
	}
	f := decodeFlag(t, rr.Body.String())
	if f.Enabled {
		t.Fatal("expected enabled to be false")
	}
	if f.Description != "old" {
		t.Fatalf("expected description unchanged, got %q", f.Description)
	}
	if f.RolloutPercent != 10 {
		t.Fatalf("expected rollout_percent unchanged, got %d", f.RolloutPercent)
	}
}

func TestUpdateFlagNotFound(t *testing.T) {
	resetStore()
	rr := doRequest(t, http.MethodPut, "/flags/missing", `{"enabled":true}`, "application/json")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestUpdateFlagRolloutOutOfRange(t *testing.T) {
	resetStore()
	postFlag(t, `{"key":"upd","enabled":true}`)
	rr := doRequest(t, http.MethodPut, "/flags/upd", `{"rollout_percent":101}`, "application/json")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestUpdateFlagUnsupportedContentType(t *testing.T) {
	resetStore()
	postFlag(t, `{"key":"upd","enabled":true}`)
	rr := doRequest(t, http.MethodPut, "/flags/upd", `{"enabled":false}`, "text/plain")
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rr.Code)
	}
}

func TestUpdateFlagBodyTooLarge(t *testing.T) {
	resetStore()
	postFlag(t, `{"key":"upd","enabled":true}`)
	body := `{"description":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	rr := doRequest(t, http.MethodPut, "/flags/upd", body, "application/json")
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
}

func TestDeleteFlag(t *testing.T) {
	resetStore()
	postFlag(t, `{"key":"del","enabled":true}`)

	rr := doRequest(t, http.MethodDelete, "/flags/del", "", "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}

	rr = doRequest(t, http.MethodGet, "/flags/del", "", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rr.Code)
	}
}

func TestDeleteFlagNotFound(t *testing.T) {
	resetStore()
	rr := doRequest(t, http.MethodDelete, "/flags/missing", "", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	resetStore()
	rr := doRequest(t, http.MethodDelete, "/flags", "", "")
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

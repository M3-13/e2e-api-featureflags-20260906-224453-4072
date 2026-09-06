package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecideDisabledAlwaysFalse(t *testing.T) {
	if Decide("feature", "alice", false, 100) {
		t.Fatal("expected false when enabled is false")
	}
	if Decide("feature", "alice", false, 0) {
		t.Fatal("expected false when enabled is false")
	}
}

func TestDecideRolloutZeroNeverTrue(t *testing.T) {
	for _, user := range []string{"alice", "bob", "carol", "dave", "eve", "frank"} {
		if Decide("feature", user, true, 0) {
			t.Fatalf("expected false for rollout_percent 0, user %q", user)
		}
	}
}

func TestDecideRolloutHundredAlwaysTrue(t *testing.T) {
	for _, user := range []string{"alice", "bob", "carol", "dave", "eve", "frank"} {
		if !Decide("feature", user, true, 100) {
			t.Fatalf("expected true for rollout_percent 100, user %q", user)
		}
	}
}

func TestDecideDeterministic(t *testing.T) {
	users := []string{"alice", "bob", "carol", "dave", "eve", "frank", "grace", "heidi"}
	keys := []string{"feature-a", "feature-b", "feature-c"}

	for _, key := range keys {
		for _, user := range users {
			first := Decide(key, user, true, 50)
			for i := 0; i < 10; i++ {
				if got := Decide(key, user, true, 50); got != first {
					t.Fatalf("non-deterministic result for key %q user %q: %v then %v", key, user, first, got)
				}
			}
		}
	}
}

func seedFlag(t *testing.T, f Flag) {
	t.Helper()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.flags = make(map[string]Flag)
	store.flags[f.Key] = f
}

func performEvaluate(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	handler := newHandler()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestEvaluateMissingUserReturns400(t *testing.T) {
	rr := performEvaluate(t, "/flags/feature/evaluate")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestEvaluateEmptyUserReturns400(t *testing.T) {
	rr := performEvaluate(t, "/flags/feature/evaluate?user=")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestEvaluateUnknownKeyReturns404(t *testing.T) {
	seedFlag(t, Flag{Key: "known", Enabled: true, RolloutPercent: 100})
	rr := performEvaluate(t, "/flags/unknown/evaluate?user=alice")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestEvaluateDisabledFlagDecisionFalse(t *testing.T) {
	seedFlag(t, Flag{Key: "feature", Enabled: false, RolloutPercent: 100})
	rr := performEvaluate(t, "/flags/feature/evaluate?user=alice")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp evaluateResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Decision {
		t.Fatal("expected decision false for disabled flag")
	}
	if resp.Key != "feature" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestEvaluateRolloutZeroDecisionFalse(t *testing.T) {
	seedFlag(t, Flag{Key: "feature", Enabled: true, RolloutPercent: 0})
	rr := performEvaluate(t, "/flags/feature/evaluate?user=alice")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp evaluateResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Decision {
		t.Fatal("expected decision false for rollout_percent 0")
	}
}

func TestEvaluateRolloutHundredDecisionTrue(t *testing.T) {
	seedFlag(t, Flag{Key: "feature", Enabled: true, RolloutPercent: 100})
	rr := performEvaluate(t, "/flags/feature/evaluate?user=alice")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp evaluateResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Decision {
		t.Fatal("expected decision true for rollout_percent 100")
	}
	if resp.RolloutPercent != 100 {
		t.Fatalf("unexpected rollout_percent: %d", resp.RolloutPercent)
	}
}

func TestEvaluateDeterministicAcrossRequests(t *testing.T) {
	seedFlag(t, Flag{Key: "feature", Enabled: true, RolloutPercent: 50})

	var first *bool
	for i := 0; i < 5; i++ {
		rr := performEvaluate(t, "/flags/feature/evaluate?user=alice")
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var resp evaluateResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if first == nil {
			first = &resp.Decision
		} else if resp.Decision != *first {
			t.Fatal("decision changed across identical requests")
		}
	}
}

func TestEvaluateDoesNotPersistUser(t *testing.T) {
	seedFlag(t, Flag{Key: "feature", Enabled: true, RolloutPercent: 50})
	performEvaluate(t, "/flags/feature/evaluate?user=alice")

	store.mu.RLock()
	defer store.mu.RUnlock()
	for _, f := range store.flags {
		if f.Description == "alice" {
			t.Fatal("user value leaked into store")
		}
	}
}

func TestEvaluateDoesNotEchoUser(t *testing.T) {
	seedFlag(t, Flag{Key: "feature", Enabled: true, RolloutPercent: 100})
	rr := performEvaluate(t, "/flags/feature/evaluate?user=alice")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if strings.Contains(rr.Body.String(), "alice") {
		t.Fatalf("user value leaked into response body: %s", rr.Body.String())
	}
}

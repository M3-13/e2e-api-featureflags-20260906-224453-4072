package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestDecideDeterministic(t *testing.T) {
	a := Decide("feature", "alice", true, 50)
	b := Decide("feature", "alice", true, 50)
	if a != b {
		t.Fatalf("expected deterministic result, got %v then %v", a, b)
	}
}

func TestDecideEnabledFalse(t *testing.T) {
	for _, p := range []int{0, 50, 100} {
		if Decide("feature", "alice", false, p) {
			t.Fatalf("expected false when disabled, rollout %d", p)
		}
	}
}

func TestDecideRolloutZeroNeverTrue(t *testing.T) {
	for _, u := range []string{"alice", "bob", "carol", "dave", "erin"} {
		if Decide("feature", u, true, 0) {
			t.Fatalf("expected false for rollout 0, user %s", u)
		}
	}
}

func TestDecideRolloutHundredAlwaysTrue(t *testing.T) {
	for _, u := range []string{"alice", "bob", "carol", "dave", "erin"} {
		if !Decide("feature", u, true, 100) {
			t.Fatalf("expected true for rollout 100, user %s", u)
		}
	}
}

func TestDecideSameKeyDifferentUserMayDiffer(t *testing.T) {
	results := make(map[bool]bool)
	for _, u := range []string{"u1", "u2", "u3", "u4", "u5", "u6", "u7", "u8"} {
		results[Decide("feature", u, true, 50)] = true
	}
	if !results[true] || !results[false] {
		t.Fatalf("expected a mix of decisions across users, got %v", results)
	}
}

func TestEvaluateMissingUser(t *testing.T) {
	resetStore()
	postFlag(t, `{"key":"feature","enabled":true}`)
	rr := doRequest(t, http.MethodGet, "/flags/feature/evaluate", "", "")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if decodeError(t, rr.Body.String()) == "" {
		t.Fatal("expected error message")
	}
}

func TestEvaluateEmptyUser(t *testing.T) {
	resetStore()
	postFlag(t, `{"key":"feature","enabled":true}`)
	rr := doRequest(t, http.MethodGet, "/flags/feature/evaluate?user=", "", "")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestEvaluateUnknownKey(t *testing.T) {
	resetStore()
	rr := doRequest(t, http.MethodGet, "/flags/missing/evaluate?user=alice", "", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if decodeError(t, rr.Body.String()) == "" {
		t.Fatal("expected error message")
	}
}

func TestEvaluateSuccess(t *testing.T) {
	resetStore()
	postFlag(t, `{"key":"feature","enabled":true,"rollout_percent":100}`)
	rr := doRequest(t, http.MethodGet, "/flags/feature/evaluate?user=alice", "", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body %q)", rr.Code, rr.Body.String())
	}
	var resp evaluateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Key != "feature" {
		t.Fatalf("expected key feature, got %q", resp.Key)
	}
	if resp.User != "alice" {
		t.Fatalf("expected user alice, got %q", resp.User)
	}
	if !resp.Enabled {
		t.Fatal("expected enabled true")
	}
	if resp.RolloutPercent != 100 {
		t.Fatalf("expected rollout 100, got %d", resp.RolloutPercent)
	}
	if !resp.Decision {
		t.Fatal("expected decision true for rollout 100")
	}
}

func TestEvaluateDisabledDecisionFalse(t *testing.T) {
	resetStore()
	postFlag(t, `{"key":"feature","enabled":false,"rollout_percent":100}`)
	rr := doRequest(t, http.MethodGet, "/flags/feature/evaluate?user=alice", "", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp evaluateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Decision {
		t.Fatal("expected decision false when disabled")
	}
}

func TestEvaluateUserNotStored(t *testing.T) {
	resetStore()
	postFlag(t, `{"key":"feature","enabled":true,"rollout_percent":100}`)
	rr := doRequest(t, http.MethodGet, "/flags/feature/evaluate?user=alice", "", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	if _, ok := flagStore.Get("alice"); ok {
		t.Fatal("user value must not be stored in the store")
	}
	flags := flagStore.List()
	if len(flags) != 1 {
		t.Fatalf("expected exactly one flag in store, got %d", len(flags))
	}
	f := flags[0]
	if f.Key != "feature" || f.Description != "" {
		t.Fatalf("unexpected flag content: %+v", f)
	}
	if rr.Body.String() == "" {
		t.Fatal("expected non-empty response")
	}
}

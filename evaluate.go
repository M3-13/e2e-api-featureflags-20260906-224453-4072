package main

import (
	"crypto/sha256"
	"encoding/json"
	"net/http"
)

// store is the shared in-memory flag store used by the evaluate handler.
// It is declared here because this ticket must remain self-contained until
// the Flag-CRUD ticket merges its own store wiring; every handler shares this
// single instance.
var store = NewStore()

// Decide deterministically evaluates whether a flag is enabled for a user.
//
// If the flag itself is disabled, the decision is always false. Otherwise the
// decision is derived from SHA-256 over "key:user": the first 8 bytes are read
// as a big-endian uint64, taken modulo 100, and the flag is decided true when
// that value is strictly less than rolloutPercent. This makes the result
// deterministic for a given (key, user) pair and distributes users uniformly
// across the rollout percentage.
func Decide(key, user string, enabled bool, rolloutPercent int) bool {
	if !enabled {
		return false
	}

	sum := sha256.Sum256([]byte(key + ":" + user))

	var h uint64
	for _, b := range sum[:8] {
		h = h<<8 | uint64(b)
	}

	return h%100 < uint64(rolloutPercent)
}

// evaluateResponse is the JSON body returned by GET /flags/{key}/evaluate.
// The user identifier is intentionally not echoed back: it is used exclusively
// to compute the decision hash (see AC-16).
type evaluateResponse struct {
	Key            string `json:"key"`
	Enabled        bool   `json:"enabled"`
	RolloutPercent int    `json:"rollout_percent"`
	Decision       bool   `json:"decision"`
}

// Evaluate handles GET /flags/{key}/evaluate?user={id}.
//
// The user value is read from the query parameter and is used exclusively to
// compute the decision hash; it is never persisted or echoed back anywhere.
// A missing or empty user yields 400, an unknown key yields 404.
func Evaluate(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	if user == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"user query parameter is required"}`))
		return
	}

	key := r.PathValue("key")
	flag, ok := store.Get(key)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"flag not found"}`))
		return
	}

	decision := Decide(flag.Key, user, flag.Enabled, flag.RolloutPercent)

	resp := evaluateResponse{
		Key:            flag.Key,
		Enabled:        flag.Enabled,
		RolloutPercent: flag.RolloutPercent,
		Decision:       decision,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

package main

import (
	"crypto/sha256"
	"encoding/binary"
	"net/http"
)

// Decide computes a deterministic rollout decision for a given flag key and
// user id. When the flag is disabled the decision is always false. Otherwise
// the SHA-256 hash of "key:user" is taken, its first 8 bytes are read as a
// big-endian uint64, and the decision is true when that value modulo 100 is
// strictly less than rolloutPercent.
func Decide(key, user string, enabled bool, rolloutPercent int) bool {
	if !enabled {
		return false
	}
	sum := sha256.Sum256([]byte(key + ":" + user))
	n := binary.BigEndian.Uint64(sum[:8])
	return int(n%100) < rolloutPercent
}

type evaluateResponse struct {
	Key            string `json:"key"`
	User           string `json:"user"`
	Enabled        bool   `json:"enabled"`
	RolloutPercent int    `json:"rollout_percent"`
	Decision       bool   `json:"decision"`
}

// Evaluate handles GET /flags/{key}/evaluate. The user id is read from the
// "user" query parameter (missing or empty yields 400). The user value is used
// solely for the Decide computation and for the "user" field of the response;
// it is never stored in the in-memory store, never logged, and never kept
// beyond this request.
func Evaluate(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	if user == "" {
		writeError(w, http.StatusBadRequest, "user parameter is required")
		return
	}

	key := r.PathValue("key")
	f, ok := flagStore.Get(key)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}

	resp := evaluateResponse{
		Key:            f.Key,
		User:           user,
		Enabled:        f.Enabled,
		RolloutPercent: f.RolloutPercent,
		Decision:       Decide(key, user, f.Enabled, f.RolloutPercent),
	}
	writeJSON(w, http.StatusOK, resp)
}

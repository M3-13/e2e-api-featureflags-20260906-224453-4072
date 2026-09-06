package main

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"
)

const maxBodyBytes = 1 << 20

var flagStore = NewStore()

func writeNotImplemented(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error":"not implemented"}`))
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	b, _ := json.Marshal(map[string]string{"error": msg})
	w.Write(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func isJSONContentType(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return false
	}
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	return mt == "application/json"
}

func readJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if !isJSONContentType(r) {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported media type")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func CreateFlag(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Key            string `json:"key"`
		Enabled        *bool  `json:"enabled"`
		Description    string `json:"description"`
		RolloutPercent *int   `json:"rollout_percent"`
	}
	if !readJSONBody(w, r, &in) {
		return
	}

	key := strings.TrimSpace(in.Key)
	if key == "" {
		writeError(w, http.StatusBadRequest, "key must not be empty")
		return
	}
	if in.Enabled == nil {
		writeError(w, http.StatusBadRequest, "enabled is required")
		return
	}

	rollout := 100
	if in.RolloutPercent != nil {
		rollout = *in.RolloutPercent
	}
	if rollout < 0 || rollout > 100 {
		writeError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
		return
	}

	f := Flag{
		Key:            key,
		Enabled:        *in.Enabled,
		Description:    in.Description,
		RolloutPercent: rollout,
	}

	if err := flagStore.Create(f); err != nil {
		if errors.Is(err, ErrFlagExists) {
			writeError(w, http.StatusConflict, "flag already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, f)
}

func ListFlags(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, flagStore.List())
}

func GetFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	f, ok := flagStore.Get(key)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func UpdateFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	var p UpdatePatch
	if !readJSONBody(w, r, &p) {
		return
	}

	if p.RolloutPercent != nil && (*p.RolloutPercent < 0 || *p.RolloutPercent > 100) {
		writeError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
		return
	}

	f, ok := flagStore.Update(key, p)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}

	writeJSON(w, http.StatusOK, f)
}

func DeleteFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !flagStore.Delete(key) {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

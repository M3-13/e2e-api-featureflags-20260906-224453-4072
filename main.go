package main

import (
	"net/http"
)

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Write([]byte(`{"error":"method not allowed"}`))
}

func newHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)

	mux.HandleFunc("POST /flags", CreateFlag)
	mux.HandleFunc("GET /flags", ListFlags)
	mux.HandleFunc("GET /flags/{key}", GetFlag)
	mux.HandleFunc("PUT /flags/{key}", UpdateFlag)
	mux.HandleFunc("DELETE /flags/{key}", DeleteFlag)
	mux.HandleFunc("GET /flags/{key}/evaluate", Evaluate)

	mux.HandleFunc("/flags", methodNotAllowed)
	mux.HandleFunc("/flags/{key}", methodNotAllowed)
	mux.HandleFunc("/flags/{key}/evaluate", methodNotAllowed)

	return Logging(mux)
}

func main() {
	http.ListenAndServe(":8080", newHandler())
}

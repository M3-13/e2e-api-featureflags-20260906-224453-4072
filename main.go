package main

import (
	"log"
	"net/http"
	"os"
	"time"
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
	protected := http.NewServeMux()

	protected.HandleFunc("POST /flags", CreateFlag)
	protected.HandleFunc("GET /flags", ListFlags)
	protected.HandleFunc("GET /flags/{key}", GetFlag)
	protected.HandleFunc("PUT /flags/{key}", UpdateFlag)
	protected.HandleFunc("DELETE /flags/{key}", DeleteFlag)
	protected.HandleFunc("GET /flags/{key}/evaluate", Evaluate)

	protected.HandleFunc("/flags", methodNotAllowed)
	protected.HandleFunc("/flags/{key}", methodNotAllowed)
	protected.HandleFunc("/flags/{key}/evaluate", methodNotAllowed)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)

	mux.Handle("/", RequireAuth(RateLimit(protected)))

	return Logging(mux)
}

func main() {
	addr := os.Getenv("FLAG_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           newHandler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

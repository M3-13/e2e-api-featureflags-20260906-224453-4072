package main

import "net/http"

func Decide(key, user string, enabled bool, rolloutPercent int) bool {
	return false
}

func Evaluate(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

package main

import "net/http"

func writeNotImplemented(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`{"error":"not implemented"}`))
}

func CreateFlag(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

func ListFlags(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

func GetFlag(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

func UpdateFlag(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

func DeleteFlag(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

package api

import (
	"encoding/json"
	"net/http"
)

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	jsonData, _ := json.Marshal(data)
	w.Write([]byte(jsonData))
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	w.Write([]byte(message))
}

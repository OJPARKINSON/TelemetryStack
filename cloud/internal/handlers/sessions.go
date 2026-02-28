package handlers

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) GetSessions(w http.ResponseWriter, r *http.Request) {

}
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(sessionID)
}

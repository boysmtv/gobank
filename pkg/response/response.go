package response

import (
	"encoding/json"
	"net/http"
)

type Envelope map[string]any

func JSON(w http.ResponseWriter, status int, payload Envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, Envelope{
		"success": false,
		"message": message,
	})
}

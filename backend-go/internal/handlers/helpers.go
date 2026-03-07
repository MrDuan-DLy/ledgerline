package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/anthropics/accounting-tool/backend-go/internal/models"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

// WriteJSON is the exported version for use outside the handlers package.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	writeJSON(w, status, v)
}

func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, models.ErrorResponse{Detail: detail})
}

func writeFileBytes(path string, content []byte) error {
	return os.WriteFile(path, content, 0o600)
}

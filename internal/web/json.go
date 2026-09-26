package web

import (
	"encoding/json"
	"log"
	"net/http"
)

// writeJSON serializes v as an indented-free JSON response. It is used by the
// activity delivery endpoints (CTRL007) that serve machine-readable payloads
// rather than rendered HTML.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status/header are already committed, so the only sensible action
		// is to log; truncating the response does not change SOP state.
		log.Printf("[json] encode: %v", err)
	}
}

package httpapi

import (
	"encoding/json"
	"net/http"

	"musicgo/internal/domain"
)

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorBody struct {
	Error errorPayload        `json:"error"`
	Quota *domain.SearchQuota `json:"quota,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	body := errorBody{
		Error: errorPayload{
			Code:    code,
			Message: message,
		},
	}
	writeJSON(w, status, body)
}

func writeSearchError(w http.ResponseWriter, status int, searchErr *domain.SearchError, quota *domain.SearchQuota) {
	body := errorBody{
		Error: errorPayload{
			Code:    searchErr.PublicCode(),
			Message: searchErr.Message,
		},
		Quota: quota,
	}
	writeJSON(w, status, body)
}

package httpapi

import (
	"net/http"
)

type HealthHandler struct {
	serviceName string
	environment  string
}

func NewHealthHandler(serviceName, environment string) http.Handler {
	return &HealthHandler{serviceName: serviceName, environment: environment}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, healthResponse{
		Status:      "ok",
		Service:     h.serviceName,
		Environment: h.environment,
	})
}

type healthResponse struct {
	Status      string `json:"status"`
	Service     string `json:"service"`
	Environment string `json:"environment"`
}

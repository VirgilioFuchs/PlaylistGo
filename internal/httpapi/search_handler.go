package httpapi

import (
	"errors"
	"net/http"

	"musicgo/internal/domain"
)

type SearchHandler struct {
	service SearchService
}

func NewSearchHandler(service SearchService) http.Handler {
	return &SearchHandler{service: service}
}

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	response, err := h.service.Search(r.Context(), r.URL.Query().Get("q"))
	if err == nil {
		writeJSON(w, http.StatusOK, response)
		return
	}

	var searchErr *domain.SearchError
	if !errors.As(err, &searchErr) {
		writeError(w, http.StatusServiceUnavailable, "upstream_unavailable", "Servico indisponivel no momento.")
		return
	}

	var quota *domain.SearchQuota
	if response.Quota.DailyLimit > 0 {
		quota = &response.Quota
	}

	switch searchErr.Kind {
	case domain.SearchErrorValidation:
		writeSearchError(w, http.StatusBadRequest, searchErr, nil)
	case domain.SearchErrorQuotaLimited:
		writeSearchError(w, http.StatusTooManyRequests, searchErr, quota)
	case domain.SearchErrorUpstreamInvalid:
		writeSearchError(w, http.StatusBadGateway, searchErr, quota)
	default:
		writeSearchError(w, http.StatusServiceUnavailable, searchErr, quota)
	}
}

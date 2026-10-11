package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	SearchCacheTTL      = 72 * time.Hour
	SearchMaxResults    = 15
	QuotaLimitedMessage = "As buscas novas acabaram hoje, mas voce ainda pode usar resultados recentes e editar suas playlists."
)

type NormalizedQuery string

type SearchCacheEntry struct {
	Query     NormalizedQuery
	Results   []TrackSnapshot
	CachedAt  time.Time
	ExpiresAt time.Time
}

type SearchQuota struct {
	DailyLimit  int    `json:"dailyLimit"`
	Remaining   int    `json:"remaining"`
	UsedPercent int    `json:"usedPercent"`
	AlertLevel  string `json:"alertLevel"`
}

type SearchCacheStatus struct {
	Hit bool `json:"hit"`
}

type SearchResponse struct {
	Data  []TrackSnapshot   `json:"data"`
	Quota SearchQuota       `json:"quota"`
	Cache SearchCacheStatus `json:"cache"`
}

type SearchBudgetWindow struct {
	WindowKey  string
	DailyLimit int
	Used       int
	ResetAt    time.Time
}

func (w SearchBudgetWindow) Remaining() int {
	remaining := w.DailyLimit - w.Used
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (w SearchBudgetWindow) PublicState() SearchQuota {
	usedPercent := 0
	if w.DailyLimit > 0 {
		usedPercent = (w.Used * 100) / w.DailyLimit
		if usedPercent > 100 {
			usedPercent = 100
		}
	}

	return SearchQuota{
		DailyLimit:  w.DailyLimit,
		Remaining:   w.Remaining(),
		UsedPercent: usedPercent,
		AlertLevel:  quotaAlertLevel(w.Remaining(), w.DailyLimit),
	}
}

type SearchErrorKind string

const (
	SearchErrorValidation          SearchErrorKind = "validation"
	SearchErrorQuotaLimited        SearchErrorKind = "quota_limited"
	SearchErrorUpstreamUnavailable SearchErrorKind = "upstream_unavailable"
	SearchErrorUpstreamInvalid     SearchErrorKind = "upstream_invalid"
)

type SearchError struct {
	Kind    SearchErrorKind
	Message string
	Err     error
}

func (e *SearchError) Error() string {
	return e.Message
}

func (e *SearchError) Unwrap() error {
	return e.Err
}

func (e *SearchError) PublicCode() string {
	switch e.Kind {
	case SearchErrorValidation:
		return "validation_error"
	case SearchErrorQuotaLimited:
		return "quota_limited"
	default:
		return "upstream_unavailable"
	}
}

func NewValidationError(message string) *SearchError {
	return &SearchError{Kind: SearchErrorValidation, Message: message}
}

func NewQuotaLimitedError(message string, err error) *SearchError {
	return &SearchError{Kind: SearchErrorQuotaLimited, Message: message, Err: err}
}

func NewUpstreamUnavailableError(message string, err error) *SearchError {
	return &SearchError{Kind: SearchErrorUpstreamUnavailable, Message: message, Err: err}
}

func NewUpstreamInvalidError(message string, err error) *SearchError {
	return &SearchError{Kind: SearchErrorUpstreamInvalid, Message: message, Err: err}
}

func NormalizeSearchQuery(raw string) (NormalizedQuery, error) {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(raw)), " "))
	if normalized == "" {
		return "", NewValidationError("Informe um termo de busca.")
	}
	return NormalizedQuery(normalized), nil
}

func quotaAlertLevel(remaining, limit int) string {
	if limit <= 0 || remaining <= 0 {
		return "0"
	}

	percentRemaining := remaining * 100

	switch {
	case percentRemaining <= limit*5:
		return "5"
	case percentRemaining <= limit*10:
		return "10"
	case percentRemaining <= limit*15:
		return "15"
	case percentRemaining <= limit*25:
		return "25"
	default:
		return ""
	}
}

func IsQuotaLimited(err error) bool {
	var searchErr *SearchError
	return errors.As(err, &searchErr) && searchErr.Kind == SearchErrorQuotaLimited
}

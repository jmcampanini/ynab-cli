package ynab

import (
	"fmt"
	"net/http"
	"strings"
)

// Error is an error response from the YNAB API. Status is the HTTP status,
// ID is the API's dotted error code such as "404.2", and Name and Detail are
// the API's own description.
type Error struct {
	Detail string
	ID     string
	Name   string
	Status int
}

// Error returns an actionable message for the error class, falling back to
// the API's own name and detail.
func (e *Error) Error() string {
	detail := e.Detail
	if detail == "" {
		detail = e.Name
	}
	switch e.Status {
	case http.StatusUnauthorized:
		return "the API rejected the token; check token in the config file or YNAB_TOKEN"
	case http.StatusForbidden:
		return "the API refused the request: " + detail + "; check the YNAB subscription and the token's access"
	case http.StatusNotFound:
		return "not found: " + detail
	case http.StatusConflict:
		return "conflict: " + detail + "; the resource changed since it was read"
	case http.StatusTooManyRequests:
		return "rate limited: the API allows 200 requests per hour per token; wait before retrying"
	case http.StatusServiceUnavailable:
		return "the YNAB API is unavailable: " + detail + "; retry later"
	}
	return fmt.Sprintf("API error %s %s: %s", e.ID, e.Name, strings.TrimSpace(detail))
}

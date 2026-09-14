package ynab

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(t *testing.T, status int, body string, seen *http.Request) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			*seen = *r
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return &Client{BaseURL: server.URL, Token: "secret", UserAgent: "ynab-cli/test"}
}

func TestPlansDecodesEnvelopeAndSendsHeaders(t *testing.T) {
	var seen http.Request
	client := serve(t, http.StatusOK, `{"data":{"plans":[{"id":"p1","name":"Household","last_modified_on":"2026-09-01T12:00:00Z","first_month":"2024-01-01","last_month":"2026-10-01","currency_format":{"iso_code":"USD","decimal_digits":2},"date_format":null}]}}`, &seen)

	plans, err := client.Plans(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(plans) != 1 || plans[0].Name != "Household" || plans[0].CurrencyFormat.ISOCode != "USD" || plans[0].DateFormat != nil {
		t.Errorf("Plans() = %+v, want one decoded plan", plans)
	}
	if seen.URL.Path != "/plans" || seen.Header.Get("Authorization") != "Bearer secret" || seen.Header.Get("User-Agent") != "ynab-cli/test" {
		t.Errorf("request = %s %s %v, want bearer token and user agent", seen.Method, seen.URL.Path, seen.Header)
	}
}

func TestAccountsDecodesMilliunits(t *testing.T) {
	client := serve(t, http.StatusOK, `{"data":{"accounts":[{"id":"a1","name":"Checking","type":"checking","on_budget":true,"closed":false,"note":null,"balance":-97810,"cleared_balance":100000,"uncleared_balance":-197810,"transfer_payee_id":"tp1","direct_import_linked":true,"direct_import_in_error":false,"last_reconciled_at":null,"deleted":false}]}}`, nil)

	accounts, err := client.Accounts(t.Context(), "p1")
	if err != nil {
		t.Fatal(err)
	}

	if len(accounts) != 1 || accounts[0].Balance != -97810 || !accounts[0].OnPlan || accounts[0].Note != nil || *accounts[0].TransferPayeeID != "tp1" {
		t.Errorf("Accounts() = %+v", accounts)
	}
}

func TestErrorClasses(t *testing.T) {
	cases := []struct {
		body   string
		status int
		want   string
	}{
		{`{"error":{"id":"401","name":"unauthorized","detail":"Unauthorized"}}`, 401, "YNAB_TOKEN"},
		{`{"error":{"id":"403.1","name":"subscription_lapsed","detail":"Subscription lapsed"}}`, 403, "Subscription lapsed"},
		{`{"error":{"id":"404.2","name":"resource_not_found","detail":"Resource not found"}}`, 404, "not found: Resource not found"},
		{`{"error":{"id":"409","name":"conflict","detail":"Conflict"}}`, 409, "conflict"},
		{`{"error":{"id":"429","name":"too_many_requests","detail":"Too many requests"}}`, 429, "200 requests per hour"},
		{`{"error":{"id":"503","name":"service_unavailable","detail":"Down"}}`, 503, "unavailable: Down"},
		{`<html>gateway</html>`, 502, "API error 502 Bad Gateway"},
	}
	for _, tc := range cases {
		client := serve(t, tc.status, tc.body, nil)

		_, err := client.Plans(t.Context())

		var apiErr *Error
		if !errors.As(err, &apiErr) || apiErr.Status != tc.status {
			t.Errorf("status %d: err = %v, want *Error with that status", tc.status, err)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("status %d: message %q lacks %q", tc.status, err.Error(), tc.want)
		}
	}
}

func TestMalformedSuccessBodyFails(t *testing.T) {
	client := serve(t, http.StatusOK, `{"data":`, nil)

	_, err := client.Plans(t.Context())

	var apiErr *Error
	if err == nil || errors.As(err, &apiErr) {
		t.Errorf("err = %v, want a decode error that is not an API error", err)
	}
}

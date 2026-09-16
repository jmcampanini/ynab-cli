package ynab

import (
	"errors"
	"io"
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

func TestWriteRequestsSendMethodBodyAndHeaders(t *testing.T) {
	var seen http.Request
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = *r
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"transaction_ids":["n1"],"transaction":{"id":"n1","date":"2026-09-15","amount":-1000,"cleared":"uncleared","approved":false,"account_id":"a1","account_name":"Checking","subtransactions":[]},"transactions":[{"id":"n1"}],"duplicate_import_ids":[]}}`))
	}))
	t.Cleanup(server.Close)
	client := &Client{BaseURL: server.URL, Token: "secret"}

	var save SaveTransaction
	save.SetAmount(-1000)
	save.AccountID, save.Date = "a1", "2026-09-15"
	save.FlagColor = NullValue[string]()
	created, err := client.CreateTransaction(t.Context(), "p1", save)
	if err != nil {
		t.Fatal(err)
	}

	if created.ID != "n1" || created.AccountName != "Checking" || created.Amount != -1000 {
		t.Errorf("CreateTransaction() = %+v, want the returned record", created)
	}
	if seen.Method != http.MethodPost || seen.URL.Path != "/plans/p1/transactions" || seen.Header.Get("Content-Type") != "application/json" {
		t.Errorf("request = %s %s %v, want a JSON POST", seen.Method, seen.URL.Path, seen.Header)
	}
	if want := `{"transaction":{"account_id":"a1","date":"2026-09-15","flag_color":null,"amount":-1000}}`; body != want {
		t.Errorf("body = %s, want %s", body, want)
	}

	changes := []SaveTransaction{{ID: "t1", Approved: boolPtr(true)}, {ID: "t2", FlagColor: SomeValue("red")}}
	if _, err := client.UpdateTransactions(t.Context(), "p1", changes); err != nil {
		t.Fatal(err)
	}
	if want := `{"transactions":[{"approved":true,"id":"t1"},{"flag_color":"red","id":"t2"}]}`; seen.Method != http.MethodPatch || body != want {
		t.Errorf("update = %s %s, want PATCH %s", seen.Method, body, want)
	}

	if _, err := client.DeleteTransaction(t.Context(), "p1", "t1"); err != nil {
		t.Fatal(err)
	}
	if seen.Method != http.MethodDelete || seen.URL.Path != "/plans/p1/transactions/t1" || body != "" {
		t.Errorf("delete = %s %s %q, want DELETE of the transaction with no body", seen.Method, seen.URL.Path, body)
	}

	ids, err := client.ImportTransactions(t.Context(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if seen.Method != http.MethodPost || seen.URL.Path != "/plans/p1/transactions/import" || len(ids) != 1 {
		t.Errorf("import = %s %s %v, want a POST returning one id", seen.Method, seen.URL.Path, ids)
	}
}

func boolPtr(value bool) *bool { return &value }

func TestWriteAnswersWithoutTheRecordFail(t *testing.T) {
	client := serve(t, http.StatusOK, `{"data":{}}`, nil)

	if _, err := client.DeleteTransaction(t.Context(), "p1", "t1"); err == nil || !strings.Contains(err.Error(), "returned no record") {
		t.Errorf("DeleteTransaction() without a record = %v", err)
	}
	if _, err := client.ImportTransactions(t.Context(), "p1"); err == nil || !strings.Contains(err.Error(), "transaction_ids") {
		t.Errorf("ImportTransactions() without ids = %v", err)
	}
	if _, err := client.CreateTransaction(t.Context(), "p1", SaveTransaction{}); err == nil || !strings.Contains(err.Error(), "returned no record") {
		t.Errorf("CreateTransaction() without a record = %v", err)
	}
}

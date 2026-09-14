package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// fixture is a fake YNAB API with one plan and three accounts. Plan "p2"
// exists so a name lookup can be exercised against several plans.
const (
	fixturePlans = `{"data":{"plans":[
{"id":"p1","name":"Household","last_modified_on":"2026-09-01T12:00:00Z","first_month":"2024-01-01","last_month":"2026-10-01",
 "date_format":{"format":"MM/DD/YYYY"},
 "currency_format":{"iso_code":"USD","example_format":"123,456.78","decimal_digits":2,"decimal_separator":".","symbol_first":true,"group_separator":",","currency_symbol":"$","display_symbol":true}},
{"id":"p2","name":"Side Business","last_modified_on":"2026-08-01T12:00:00Z","first_month":"2025-01-01","last_month":"2026-09-01","date_format":null,"currency_format":null}
]}}`
	fixtureAccounts = `{"data":{"accounts":[
{"id":"a1","name":"Chase Checking","type":"checking","on_budget":true,"closed":false,"note":"Main","balance":1234560,"cleared_balance":1000000,"uncleared_balance":234560,"transfer_payee_id":"tp1","direct_import_linked":true,"direct_import_in_error":false,"last_reconciled_at":"2026-08-31T00:00:00Z","deleted":false},
{"id":"a2","name":"Visa","type":"creditCard","on_budget":true,"closed":false,"note":null,"balance":-97810,"cleared_balance":-97810,"uncleared_balance":0,"transfer_payee_id":"tp2","direct_import_linked":true,"direct_import_in_error":true,"last_reconciled_at":null,"deleted":false},
{"id":"a3","name":"Old Savings","type":"savings","on_budget":false,"closed":true,"note":null,"balance":0,"cleared_balance":0,"uncleared_balance":0,"transfer_payee_id":null,"direct_import_linked":false,"direct_import_in_error":false,"last_reconciled_at":null,"deleted":false}
]}}`
)

// harness runs a fresh root against a fake API in an isolated environment
// and records the API paths each execution requested.
type harness struct {
	deps     dependencies
	requests []string
	terminal bool
	env      map[string]string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{env: map[string]string{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.requests = append(h.requests, r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer good-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":{"id":"401","name":"unauthorized","detail":"Unauthorized"}}`)
			return
		}
		switch r.URL.Path {
		case "/plans":
			_, _ = io.WriteString(w, fixturePlans)
		case "/plans/p1/accounts":
			_, _ = io.WriteString(w, fixtureAccounts)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":{"id":"404.2","name":"resource_not_found","detail":"Resource not found"}}`)
		}
	}))
	t.Cleanup(server.Close)
	h.deps = dependencies{
		baseURL:    server.URL,
		isTerminal: func(io.Writer) bool { return h.terminal },
		lookupEnv:  func(name string) string { return h.env[name] },
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "missing"))
	t.Setenv("YNAB_TOKEN", "good-token")
	t.Setenv("YNAB_PLAN", "household")
	return h
}

func (h *harness) execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRoot(h.deps)
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.ExecuteContext(t.Context())
	return stdout.String(), err
}

func TestEveryApplicationCommandDeclaresArgsAndGroupsAreRunnable(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(command *cobra.Command) {
		for _, child := range command.Commands() {
			if child.Name() == "help" || child.Name() == "completion" {
				continue
			}
			if child.Args == nil {
				t.Errorf("%s has no Args validator", child.CommandPath())
			}
			if child.HasSubCommands() && !child.Runnable() {
				t.Errorf("%s has subcommands but no runner", child.CommandPath())
			}
			walk(child)
		}
	}
	walk(NewRoot())
}

func TestExitCodesTopicPrintsSameHelpFromBothEntryPoints(t *testing.T) {
	h := newHarness(t)

	a, err := h.execute(t, "exit-codes")
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.execute(t, "help", "exit-codes")
	if err != nil {
		t.Fatal(err)
	}

	if a != b {
		t.Errorf("exit-codes and help exit-codes differ:\n%s\n%s", a, b)
	}
	for _, row := range []string{"0  Success", "1  Command failure", "2  Usage", "3  Writes disabled"} {
		if !strings.Contains(a, row) {
			t.Errorf("exit-codes topic lacks %q", row)
		}
	}
}

func TestEveryApplicationCommandHasWrappedLongHelp(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(command *cobra.Command) {
		if command.Name() == "help" || command.Name() == "completion" {
			return
		}
		if command.Long == "" {
			t.Errorf("%s lacks long help", command.CommandPath())
		}
		for _, text := range []string{command.Long, command.Example} {
			for _, line := range strings.Split(text, "\n") {
				if len(line) > 80 {
					t.Errorf("%s line exceeds 80 columns: %q", command.CommandPath(), line)
				}
			}
		}
		for _, child := range command.Commands() {
			walk(child)
		}
	}
	walk(NewRoot())
}

func TestUsageErrorsExitTwoBeforeAnyRequest(t *testing.T) {
	h := newHarness(t)
	cases := [][]string{
		{"unknown"}, {"plans", "extra"}, {"accounts", "list", "extra"}, {"accounts", "get"},
		{"accounts", "get", "a", "b"}, {"config", "extra"}, {"exit-codes", "extra"}, {"output-formats", "extra"},
		{"accounts", "list", "--jsonl", "--csv"}, {"--color", "bold", "plans", "list"}, {"plans", "list", "--nope"},
		{"config", "--config", ""},
	}

	for _, args := range cases {
		out, err := h.execute(t, args...)
		if err == nil || out != "" || ExitCode(err) != ExitUsage {
			t.Errorf("execute(%v) = %q, %v (exit %d), want a usage error", args, out, err, ExitCode(err))
		}
	}
	if len(h.requests) != 0 {
		t.Errorf("usage errors made requests: %v", h.requests)
	}
}

func TestMissingTokenAndPlanExitOneNamingTheSetting(t *testing.T) {
	h := newHarness(t)
	t.Setenv("YNAB_TOKEN", "")
	t.Setenv("YNAB_PLAN", "")

	_, err := h.execute(t, "plans", "list")
	if err == nil || ExitCode(err) != ExitFailure || !strings.Contains(err.Error(), "YNAB_TOKEN") || !strings.Contains(err.Error(), "ynab.toml") {
		t.Errorf("plans list without token = %v (exit %d)", err, ExitCode(err))
	}

	t.Setenv("YNAB_TOKEN", "good-token")
	_, err = h.execute(t, "accounts", "list")
	for _, want := range []string{"plan", "YNAB_PLAN", "--plan", "ynab plans list"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("accounts list without plan = %v, want mention of %q", err, want)
		}
	}
	if ExitCode(err) != ExitFailure || len(h.requests) != 0 {
		t.Errorf("exit %d, requests %v; want exit 1 and no request", ExitCode(err), h.requests)
	}
}

func TestAPIErrorsExitOne(t *testing.T) {
	h := newHarness(t)
	t.Setenv("YNAB_TOKEN", "bad-token")

	out, err := h.execute(t, "plans", "list")

	if err == nil || out != "" || ExitCode(err) != ExitFailure || !strings.Contains(err.Error(), "YNAB_TOKEN") {
		t.Errorf("plans list with bad token = %q, %v (exit %d)", out, err, ExitCode(err))
	}
}

func TestHelpAndVersionNeedNoTokenOrNetwork(t *testing.T) {
	h := newHarness(t)
	t.Setenv("YNAB_TOKEN", "")
	for _, args := range [][]string{{"--help"}, {"--version"}, {"plans"}, {"accounts"}, {"accounts", "get", "--help"}, {"output-formats"}, {"config", "--help"}} {
		out, err := h.execute(t, args...)
		if err != nil || out == "" {
			t.Errorf("execute(%v) = %q, %v", args, out, err)
		}
	}
	if len(h.requests) != 0 {
		t.Errorf("help made requests: %v", h.requests)
	}
}

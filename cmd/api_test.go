package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAPIGetSubstitutesThePlanAndPassesTheBodyThrough(t *testing.T) {
	h := newHarness(t)

	raw, err := h.execute(t, "api", "get", "plans/{plan}/accounts")
	if err != nil {
		t.Fatal(err)
	}
	pretty, err := h.execute(t, "api", "get", "/plans/{plan}/accounts", "--pretty")
	if err != nil {
		t.Fatal(err)
	}

	if raw != fixtureAccounts+"\n" {
		t.Errorf("api get =\n%s\nwant the fixture body as served plus a newline", raw)
	}
	if !strings.Contains(pretty, "\n        \"balance\": 1234560,\n") {
		t.Errorf("api get --pretty is not re-indented with raw milliunits:\n%s", pretty)
	}
	if want := []string{"/plans", "/plans/p1/accounts", "/plans", "/plans/p1/accounts"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v", h.requests, want)
	}
}

func TestAPINon2xxPrintsTheBodyAndExitsOne(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "api", "get", "/nope")

	if out != `{"error":{"id":"404.2","name":"resource_not_found","detail":"Resource not found"}}`+"\n" {
		t.Errorf("stdout = %q, want the error body as served", out)
	}
	if err == nil || ExitCode(err) != ExitFailure || err.Error() != "HTTP 404 Not Found" {
		t.Errorf("err = %v (exit %d), want HTTP 404 Not Found with exit 1", err, ExitCode(err))
	}
	if want := []string{"/nope"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v (no plan lookup without {plan})", h.requests, want)
	}
}

func TestAPIWritesGoThroughTheGate(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "api", "patch", "/plans/{plan}/x", "--body", `{"a":1}`)
	if err == nil || ExitCode(err) != ExitWritesDisabled || out != "" || len(h.requests) != 0 {
		t.Errorf("patch without allow_writes = %q, %v (exit %d), requests %v; want exit 3 and no request", out, err, ExitCode(err), h.requests)
	}

	h.requests = nil
	out, err = h.execute(t, "api", "patch", "/plans/{plan}/x", "--body", `{"a":1}`, "--dry-run", "--pretty")
	if err != nil {
		t.Fatal(err)
	}
	if want := "PATCH " + h.deps.baseURL + "/plans/p1/x\n{\n  \"a\": 1\n}\n"; out != want {
		t.Errorf("dry run =\n%s\nwant\n%s", out, want)
	}
	if want := []string{"/plans"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("dry run requests = %v, want only the plan lookup", h.requests)
	}

	h.requests, h.writes = nil, nil
	out, err = h.execute(t, "api", "post", "/plans/{plan}/x", "--body", `{"a":1}`, "--allow-writes")
	if err == nil || ExitCode(err) != ExitFailure || !strings.Contains(out, "resource_not_found") {
		t.Errorf("post to an unknown path = %q, %v (exit %d), want the 404 body and exit 1", out, err, ExitCode(err))
	}
	if want := []string{`POST /plans/p1/x {"a":1}`}; !reflect.DeepEqual(h.writes, want) {
		t.Errorf("writes = %v, want %v", h.writes, want)
	}
}

func TestAPIBodyComesFromAFileOrStdinAndMustBeJSON(t *testing.T) {
	h := newHarness(t)
	file := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(file, []byte(`{"from":"file"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	fromFile, err := h.execute(t, "api", "put", "/x", "--body-file", file, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	h.stdin = strings.NewReader(`{"from":"stdin"}`)
	fromStdin, err := h.execute(t, "api", "delete", "/x", "--body-file", "-", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	h.requests = nil
	out, err := h.execute(t, "api", "post", "/x", "--body", "not json", "--dry-run")

	if !strings.HasSuffix(fromFile, "\n{\"from\":\"file\"}\n") {
		t.Errorf("--body-file dry run =\n%s", fromFile)
	}
	if !strings.HasSuffix(fromStdin, "\n{\"from\":\"stdin\"}\n") {
		t.Errorf("--body-file - dry run =\n%s", fromStdin)
	}
	if err == nil || ExitCode(err) != ExitUsage || out != "" || len(h.requests) != 0 {
		t.Errorf("--body 'not json' = %q, %v (exit %d), requests %v; want a usage error before any request", out, err, ExitCode(err), h.requests)
	}
}

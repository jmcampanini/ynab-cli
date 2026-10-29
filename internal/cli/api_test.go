package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jmcampanini/ynab-cli/internal/status"
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

func TestAPINon2xxLeavesStdoutEmptyAndExitsOne(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "api", "get", "/nope")

	if out != "" {
		t.Errorf("stdout = %q, want no error payload", out)
	}
	if err == nil || status.Code(err) != status.ExitFailure || err.Error() != `HTTP 404 Not Found: "Resource not found"` {
		t.Errorf("err = %v (exit %d), want HTTP 404 Not Found with detail and exit 1", err, status.Code(err))
	}
	if want := []string{"/nope"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v (no plan lookup without {plan})", h.requests, want)
	}
}

func TestAPIWritesGoThroughTheGate(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "api", "patch", "/plans/{plan}/x", "--body", `{"a":1}`)
	if err == nil || status.Code(err) != status.ExitWritesDisabled || out != "" || len(h.requests) != 0 {
		t.Errorf("patch without allow_writes = %q, %v (exit %d), requests %v; want exit 3 and no request", out, err, status.Code(err), h.requests)
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
	if err == nil || status.Code(err) != status.ExitFailure || out != "" || !strings.Contains(err.Error(), "Resource not found") {
		t.Errorf("post to an unknown path = %q, %v (exit %d), want empty stdout, the 404 diagnostic, and exit 1", out, err, status.Code(err))
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
	if err == nil || status.Code(err) != status.ExitFailure || out != "" || len(h.requests) != 0 {
		t.Errorf("--body 'not json' = %q, %v (exit %d), requests %v; want exit 1 before any request", out, err, status.Code(err), h.requests)
	}
}

func TestAPIFailureDoesNotEmitResponseRecordsOrLeakToken(t *testing.T) {
	cases := []struct {
		name      string
		code      int
		body      string
		truncated bool
		want      string
	}{
		{"error detail", http.StatusBadRequest, `{"error":{"detail":"rejected good-token\ntry again"}}`, false, `HTTP 400 Bad Request: "rejected <redacted>\ntry again"`},
		{"error name", http.StatusBadRequest, `{"error":{"name":"invalid_request"}}`, false, `HTTP 400 Bad Request: "invalid_request"`},
		{"unexpected JSON", http.StatusBadRequest, `{"token":"good-token"}`, false, "HTTP 400 Bad Request"},
		{"malformed JSON", http.StatusBadRequest, `{"error":{"detail":"good-token`, false, "HTTP 400 Bad Request"},
		{"HTML error", http.StatusBadGateway, `<h1>good-token</h1>`, false, "HTTP 502 Bad Gateway"},
		{"partial success body", http.StatusOK, `{"data":{"id":"one"}}`, true, "unexpected EOF"},
		{"partial error body", http.StatusInternalServerError, `{"error":{"detail":"good-token"}}`, true, "unexpected EOF"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tc.truncated {
					w.Header().Set("Content-Length", "10000")
				}
				w.WriteHeader(tc.code)
				_, _ = io.WriteString(w, tc.body)
			}))
			t.Cleanup(server.Close)
			h.deps.baseURL = server.URL
			out, err := h.execute(t, "api", "get", "/response", "--pretty")
			if err == nil || status.Code(err) != status.ExitFailure || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v (exit %d); want exit 1 containing %q", err, status.Code(err), tc.want)
			}
			if out != "" || strings.Contains(err.Error(), "good-token") || strings.ContainsAny(err.Error(), "\r\n") {
				t.Errorf("stdout = %q, error = %q; want no payload, token, or diagnostic newline", out, err)
			}
		})
	}
}

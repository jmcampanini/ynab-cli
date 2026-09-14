package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigRedactsTokenInEveryOutputPath(t *testing.T) {
	h := newHarness(t)
	path := filepath.Join(t.TempDir(), "ynab.toml")
	if err := os.WriteFile(path, []byte("token = \"file-token\"\nplan = \"Household\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// The harness sets both variables; clear them so the file is the source.
	t.Setenv("YNAB_TOKEN", "")
	if err := os.Unsetenv("YNAB_PLAN"); err != nil {
		t.Fatal(err)
	}

	toml, err := h.execute(t, "config", "--config", path, "--provenance", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}
	jsonl, err := h.execute(t, "config", "--config", path, "--provenance", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	for name, out := range map[string]string{"toml": toml, "jsonl": jsonl} {
		if strings.Contains(out, "file-token") || !strings.Contains(out, "<redacted>") {
			t.Errorf("%s output leaks the token:\n%s", name, out)
		}
	}
	for _, want := range []string{`token = "<redacted>"`, `allow_writes = true`, "# allow_writes | true | <pflag>", "# token | \"<redacted>\" | " + path} {
		if !strings.Contains(toml, want) {
			t.Errorf("toml output lacks %q:\n%s", want, toml)
		}
	}
	var record configRecord
	if err := json.Unmarshal([]byte(jsonl), &record); err != nil || strings.Count(jsonl, "\n") != 1 {
		t.Fatalf("jsonl = %q, %v; want one object", jsonl, err)
	}
	if record.Plan != "Household" || record.Token != "<redacted>" || record.Sources["allow_writes"] != "<default>" || record.Sources["token"] != path {
		t.Errorf("jsonl record = %+v", record)
	}
	if len(h.requests) != 0 {
		t.Errorf("config made requests: %v", h.requests)
	}
}

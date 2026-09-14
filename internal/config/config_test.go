package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmcampanini/go-config-loader/configreporter"
	"github.com/jmcampanini/go-config-loader/pflagloader"
	"github.com/spf13/pflag"
)

// isolateEnv clears every YNAB_* variable the loader reads, so a
// developer's exported settings cannot leak into a test; each test then
// sets the variables it needs.
func isolateEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"YNAB_PLAN", "YNAB_TOKEN", "YNAB_ALLOW_WRITES"} {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
}

func flags(t *testing.T, args ...string) *pflag.FlagSet {
	t.Helper()
	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	if err := pflagloader.Register[Config](f); err != nil {
		t.Fatal(err)
	}
	if err := f.Parse(args); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestTokenHasNoFlag(t *testing.T) {
	f := flags(t)

	if f.Lookup("token") != nil {
		t.Error("--token is registered; the token must come only from TOML or YNAB_TOKEN")
	}
	if f.Lookup("plan") == nil || f.Lookup("allow-writes") == nil {
		t.Error("--plan and --allow-writes are not registered")
	}
}

func TestLayersProvenanceAndRoundTrip(t *testing.T) {
	isolateEnv(t)
	path := filepath.Join(t.TempDir(), "ynab.toml")
	if err := os.WriteFile(path, []byte("plan = \"From File\"\ntoken = \"file-token\"\nallow_writes = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("YNAB_PLAN", "From Env")
	t.Setenv("YNAB_TOKEN", "env-token")

	loaded, err := Load(path, flags(t, "--plan", "From Flag"))
	if err != nil {
		t.Fatal(err)
	}

	cfg := loaded.Config
	if cfg.Plan != "From Flag" || cfg.Token != "env-token" || !cfg.AllowWrites || loaded.Path != path {
		t.Fatalf("Load() = %+v, path %q, want flag over env over file", cfg, loaded.Path)
	}
	updates := loaded.Report.Updates
	if updates["plan"] != "<pflag>" || updates["token"] != "<env>" || updates["allowwrites"] != path {
		t.Errorf("provenance = %v", updates)
	}

	redacted := cfg.Redact()
	body, err := configreporter.New(redacted, loaded.Report).TOML()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "env-token") || !strings.Contains(string(body), `token = "<redacted>"`) {
		t.Errorf("report = %s, want the token redacted", body)
	}
	for _, row := range configreporter.New(redacted, loaded.Report).ProvenanceRows() {
		if strings.Contains(strings.Join(row, " "), "env-token") {
			t.Errorf("provenance row %v leaks the token", row)
		}
	}

	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	roundtrip, err := Load(path, flags(t))
	if err != nil || roundtrip.Config.Plan != "From Env" || roundtrip.Config.Token != "env-token" || !roundtrip.Config.AllowWrites {
		t.Fatalf("roundtrip = %+v, %v; the written file must parse", roundtrip.Config, err)
	}
}

func TestDiscoveryToleratesAbsentFileAndExplicitFileMustExist(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("YNAB_TOKEN", "env-token")
	discovered := filepath.Join(root, "ynab", "ynab.toml")

	loaded, err := Load("", flags(t))
	if err != nil || loaded.Path != discovered || loaded.Config.Token != "env-token" || loaded.Config.Plan != "" || loaded.Config.AllowWrites {
		t.Fatalf("Load(absent) = %+v, %v, want defaults plus env and path %q", loaded, err, discovered)
	}

	missing := filepath.Join(root, "missing.toml")
	if _, err := Load(missing, flags(t)); err == nil || !strings.Contains(err.Error(), missing) {
		t.Errorf("Load(missing explicit) = %v, want an error naming the file", err)
	}
}

func TestUnknownKeysAndBadValuesFail(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	path := filepath.Join(root, "ynab", "ynab.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}

	for _, body := range []string{"budget = \"x\"\n", "allow_writes = \"yes\"\n", "plan = [1]\n"} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load("", flags(t)); err == nil {
			t.Errorf("Load(%q) succeeded", body)
		}
	}
	t.Setenv("YNAB_ALLOW_WRITES", "maybe")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load("", flags(t)); err == nil {
		t.Error("Load with YNAB_ALLOW_WRITES=maybe succeeded")
	}
}

func TestUnquotedTokenParseErrorHidesTheValue(t *testing.T) {
	isolateEnv(t)
	path := filepath.Join(t.TempDir(), "ynab.toml")
	if err := os.WriteFile(path, []byte("token = deadbeefdeadbeef\nplan = 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path, flags(t))

	if err == nil || strings.Contains(err.Error(), "deadbeef") || !strings.Contains(err.Error(), "line 1") {
		t.Errorf("Load(unquoted token) = %v; want a parse error naming the line but not the value", err)
	}
	if err := os.WriteFile(path, []byte("plan = nope\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, flags(t)); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("Load(unquoted plan) = %v; want the parser's detail kept for other keys", err)
	}
}

func TestEmptyTokenEnvDoesNotOverrideFile(t *testing.T) {
	isolateEnv(t)
	path := filepath.Join(t.TempDir(), "ynab.toml")
	if err := os.WriteFile(path, []byte("token = \"file-token\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("YNAB_TOKEN", "")

	loaded, err := Load(path, flags(t))

	if err != nil || loaded.Config.Token != "file-token" || loaded.Report.Updates["token"] != path {
		t.Errorf("Load() = %+v, %v; want the file token to survive an empty YNAB_TOKEN", loaded, err)
	}
}

func TestRedactLeavesEmptyTokenVisible(t *testing.T) {
	if got := (Config{}).Redact(); got.Token != "" {
		t.Errorf("Redact(empty) = %q, want empty so a missing token is visible", got.Token)
	}
	if got := (Config{Token: "secret"}).Redact(); got.Token != Redacted {
		t.Errorf("Redact(set) = %q, want %q", got.Token, Redacted)
	}
}

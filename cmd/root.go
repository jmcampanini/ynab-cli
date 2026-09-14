// Package cmd connects the YNAB client and configuration to Cobra.
package cmd

import (
	"io"
	"os"
	"time"

	"github.com/jmcampanini/go-config-loader/pflagloader"
	"github.com/jmcampanini/ynab-cli/internal/config"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Version is injected by the versioned build target.
var Version = "dev"

// dependencies are the process-level facts commands consult, kept
// injectable so tests can point at a local server and a fake terminal.
type dependencies struct {
	// baseURL is the API root; empty means the production API.
	baseURL string
	// isTerminal reports whether a writer is an interactive terminal.
	isTerminal func(io.Writer) bool
	// lookupEnv reads one environment variable, empty when unset.
	lookupEnv func(string) string
	// now is the clock that resolves "current", "today", and "yesterday".
	now func() time.Time
}

func defaultDependencies() dependencies {
	return dependencies{
		baseURL: ynab.DefaultBaseURL,
		isTerminal: func(w io.Writer) bool {
			file, ok := w.(interface{ Fd() uintptr })
			return ok && term.IsTerminal(int(file.Fd()))
		},
		lookupEnv: os.Getenv,
		now:       time.Now,
	}
}

// app carries the dependencies and root-level flag values that leaf
// commands read at run time.
type app struct {
	color colorMode
	deps  dependencies
}

// NewRoot constructs a fresh command tree without performing command work.
func NewRoot() *cobra.Command {
	return newRoot(defaultDependencies())
}

func newRoot(deps dependencies) *cobra.Command {
	a := &app{color: colorAuto, deps: deps}
	root := &cobra.Command{
		Use: "ynab", Short: "Read and write a YNAB plan from the terminal",
		Long: `ynab reads and writes a YNAB plan from the terminal and emits JSON lines
and CSV for scripts and agents. Every command is 'ynab <noun> <verb>'; a
bare noun prints its help. This release reads plans, accounts, categories,
category groups, and months. Writes are not implemented yet; when they
arrive they stay disabled until allow_writes is set, and every write will
accept --dry-run.

ynab needs a personal access token from https://app.ynab.com/settings/developer
and network access to api.ynab.com. It runs no external programs, never
prompts, and keeps nothing on disk.

` + configHelp + "\n\n" + planHelp + "\n\n" + outputHelp,
		Example: `  ynab plans list
  ynab accounts list --plan Household
  ynab accounts list --jsonl | jq .balance
  ynab categories get "Bills: Internet"
  ynab months get
  ynab config --provenance`,
		SilenceErrors: true, SilenceUsage: true, Version: Version,
		DisableSuggestions: true,
		RunE:               func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	root.InitDefaultHelpFlag()
	root.InitDefaultVersionFlag()
	root.PersistentFlags().String("config", "", "Read this TOML file instead of the discovered one")
	root.PersistentFlags().Var(colorValue{mode: &a.color}, "color", "Color human output: auto, always, never")
	if err := pflagloader.Register[config.Config](root.PersistentFlags()); err != nil {
		// Registration depends only on this package's static configuration type.
		panic(err)
	}
	root.AddCommand(newConfig(), newPlans(a), newAccounts(a), newCategories(a), newCategoryGroups(a), newMonths(a), exitCodesTopic(), outputFormatsTopic())
	return root
}

// configPath returns the --config value, or empty to discover the file.
func configPath(cmd *cobra.Command) (string, error) {
	path, err := cmd.Flags().GetString("config")
	if err != nil {
		return "", err
	}
	if cmd.Flags().Changed("config") && path == "" {
		return "", usageError("--config requires a nonempty file path")
	}
	return path, nil
}

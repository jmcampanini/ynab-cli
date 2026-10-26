// Package cli connects the YNAB client and configuration to Cobra.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jmcampanini/go-config-loader/pflagloader"
	"github.com/jmcampanini/ynab-cli/internal/config"
	"github.com/jmcampanini/ynab-cli/internal/status"
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

// newRoot constructs a fresh command tree without performing command work.
func newRoot(deps dependencies) *cobra.Command {
	a := &app{color: colorAuto, deps: deps}
	root := &cobra.Command{
		Use: "ynab", Short: "Read and write a YNAB plan from the terminal",
		Long: `ynab reads and writes a YNAB plan from the terminal and emits JSON lines
and CSV for scripts and agents. Every command is 'ynab <noun> <verb>'; a
bare noun prints its help. The command groups below cover the plan's
data and the reports derived from it; anything the API offers beyond
them goes through 'ynab api', which sends a raw request and prints the
raw response. Writes stay disabled until allow_writes is set, and every
write accepts --dry-run; see 'ynab transactions --help'.

ynab needs a personal access token from https://app.ynab.com/settings/developer
and network access to api.ynab.com. It runs no external programs and keeps
nothing on disk. 'ynab completion bash|zsh|fish|powershell' prints a
completion script that also completes account, category, and payee names
from the plan.

` + configHelp + "\n\n" + planHelp + "\n\n" + outputHelp + `

Common tasks:
  List your plans:        ynab plans list
  Summarize the month:    ynab plans status
  List accounts:          ynab accounts list --plan Household
  Export the register:    ynab transactions list --since 2026-08-01 --csv
  Review the register:    ynab transactions review
  Preview an approval:    ynab transactions approve --all-unapproved --dry-run
  Preview a move:         ynab months move 50 --from Fun --to Bills --dry-run
  Fund every target:      ynab months fund --all-underfunded --allow-writes
  List what needs money:  ynab reports funding --underfunded
  Call the API directly:  ynab api get /user
  Check configuration:    ynab config --provenance

Command groups:
  plans            Plans the token can access:     ynab plans --help
  accounts         Accounts and their balances:    ynab accounts --help
  categories       Categories and their targets:   ynab categories --help
  category-groups  Groups of categories:           ynab category-groups --help
  months           Months and assigned money:      ynab months --help
  transactions     The register and its writes:    ynab transactions --help
  payees           Payees and their names:         ynab payees --help
  scheduled        Repeating transactions:         ynab scheduled --help
  money-movements  Recorded money movements:       ynab money-movements --help
  reports          Funding and spending reports:   ynab reports --help
  api              Raw requests to the YNAB API:   ynab api --help`,
		SilenceErrors: true, SilenceUsage: true, Version: Version,
		DisableSuggestions: true,
		RunE:               func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	root.InitDefaultHelpFlag()
	root.InitDefaultVersionFlag()
	// Cobra's closing usage line is shortened so the help of the
	// longest-named groups fits 80 columns.
	root.SetUsageTemplate(strings.Replace(root.UsageTemplate(),
		`Use "{{.CommandPath}} [command] --help" for more information about a command.`,
		`Use "{{.CommandPath}} [command] --help" for a command's help.`, 1))
	root.PersistentFlags().String("config", "", "Read this TOML file instead of the discovered one")
	root.PersistentFlags().Var(colorValue{mode: &a.color}, "color", "Color human output: auto, always, never")
	if err := pflagloader.Register[config.Config](root.PersistentFlags()); err != nil {
		// Registration depends only on this package's static configuration type.
		panic(err)
	}
	root.AddCommand(newConfig(), newPlans(a), newAccounts(a), newCategories(a), newCategoryGroups(a), newMonths(a), newTransactions(a), newPayees(a), newScheduled(a), newMoneyMovements(a), newReports(a), newAPI(a), exitCodesTopic(), outputFormatsTopic())
	return root
}

// Execute runs the command tree with process signals and reports failures.
func Execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := newRoot(defaultDependencies()).ExecuteContext(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ynab: %s\n", err)
	}
	return status.Code(err)
}

package cli

import (
	"github.com/jmcampanini/ynab-cli/internal/selection"
	"github.com/jmcampanini/ynab-cli/internal/status"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// writeSession is what every mutating command has in hand once the gate
// has passed: the client and the plan, plus the accounts, categories, and
// payees once a command has loaded them.
type writeSession struct {
	accounts   []ynab.Account
	app        *app
	categories []ynab.CategoryGroup
	client     *ynab.Client
	dryRun     bool
	payees     []ynab.Payee
	plan       ynab.Plan
}

// openWrite loads the configuration, applies the write gate, and reads
// the plan. Without --dry-run and without allow_writes it fails with
// exit status 3 before any request.
func (a *app) openWrite(cmd *cobra.Command, dryRun bool) (*writeSession, error) {
	loaded, client, err := a.connect(cmd)
	if err != nil {
		return nil, err
	}
	if !dryRun && !loaded.Config.AllowWrites {
		return nil, status.WritesDisabled(loaded.Path)
	}

	plan, err := selection.Plan(cmd.Context(), client, loaded.Config)
	if err != nil {
		return nil, err
	}
	return &writeSession{app: a, client: client, dryRun: dryRun, plan: plan}, nil
}

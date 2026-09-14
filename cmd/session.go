package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/config"
	"github.com/jmcampanini/ynab-cli/internal/names"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// connect loads the configuration and returns a client for its token.
func (a *app) connect(cmd *cobra.Command) (config.Loaded, *ynab.Client, error) {
	path, err := configPath(cmd)
	if err != nil {
		return config.Loaded{}, nil, err
	}
	loaded, err := config.Load(path, cmd.Root().PersistentFlags())
	if err != nil {
		return config.Loaded{}, nil, err
	}
	if loaded.Config.Token == "" {
		return config.Loaded{}, nil, fmt.Errorf("no token configured: set token in %s or %s", loaded.Path, config.TokenEnv)
	}

	client := &ynab.Client{BaseURL: a.deps.baseURL, Token: loaded.Config.Token, UserAgent: "ynab-cli/" + Version}
	return loaded, client, nil
}

// selectedPlan resolves the configured plan against the plans endpoint by
// ID or case-insensitive exact name.
func selectedPlan(ctx context.Context, client *ynab.Client, cfg config.Config) (ynab.Plan, error) {
	if cfg.Plan == "" {
		return ynab.Plan{}, fmt.Errorf("no plan configured: set plan in the config file, YNAB_PLAN, or --plan; 'ynab plans list' shows the plans")
	}
	plans, err := client.Plans(ctx)
	if err != nil {
		return ynab.Plan{}, err
	}

	matches := matchPlan(cfg.Plan, plans)
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return ynab.Plan{}, fmt.Errorf("plan %q not found; the token can read: %s", cfg.Plan, joinPlanNames(plans))
	default:
		return ynab.Plan{}, fmt.Errorf("plan %q matches %d plans; use an ID from 'ynab plans list': %s", cfg.Plan, len(matches), joinPlanNames(matches))
	}
}

// matchPlan returns the plans whose ID equals the query or whose name
// equals it ignoring case.
func matchPlan(query string, plans []ynab.Plan) []ynab.Plan {
	planNames := make([]string, len(plans))
	for i, plan := range plans {
		if plan.ID == query {
			return []ynab.Plan{plan}
		}
		planNames[i] = plan.Name
	}
	var matches []ynab.Plan
	for _, i := range names.Match(query, planNames) {
		matches = append(matches, plans[i])
	}
	return matches
}

func joinPlanNames(plans []ynab.Plan) string {
	quoted := make([]string, len(plans))
	for i, plan := range plans {
		quoted[i] = fmt.Sprintf("%q", plan.Name)
	}
	return strings.Join(quoted, ", ")
}

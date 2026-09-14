package cmd

import (
	"fmt"
	"strings"

	"github.com/jmcampanini/go-config-loader/configreporter"
	"github.com/jmcampanini/ynab-cli/internal/config"
	"github.com/spf13/cobra"
)

// configRecord is the --jsonl shape of the effective configuration.
type configRecord struct {
	AllowWrites bool              `json:"allow_writes"`
	Plan        string            `json:"plan"`
	Token       string            `json:"token"`
	Sources     map[string]string `json:"sources,omitempty"`
}

func newConfig() *cobra.Command {
	var output outputFlags
	var provenance bool
	command := &cobra.Command{
		Use: "config", Short: "Print the effective configuration",
		Long: `Print the effective configuration as redirectable TOML after applying
every layer. A configured token is always shown as "<redacted>", in TOML,
JSON, and provenance alike. --provenance adds each field's source as a
TOML comment, or a "sources" object with --jsonl. This command makes no
API request and needs no token or plan.

Example configuration:
  plan = "Household"
  token = "your-personal-access-token"
  allow_writes = false

` + configHelp + "\n\n" + outputHelp,
		Example: `  ynab config
  ynab config --provenance
  ynab config --config ./ynab.toml --plan Household --jsonl`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			path, err := configPath(cmd)
			if err != nil {
				return err
			}
			loaded, err := config.Load(path, cmd.Root().PersistentFlags())
			if err != nil {
				return err
			}

			redacted := loaded.Config.Redact()
			reporter := configreporter.New(redacted, loaded.Report)
			if output.jsonl {
				record := configRecord{AllowWrites: redacted.AllowWrites, Plan: redacted.Plan, Token: redacted.Token}
				if provenance {
					record.Sources = make(map[string]string, len(loaded.Report.Updates))
					for path, source := range loaded.Report.Updates {
						record.Sources[configKey(path)] = source
					}
				}
				return writeJSONL(cmd.OutOrStdout(), []configRecord{record})
			}
			if err := reporter.WriteTOML(cmd.OutOrStdout()); err != nil {
				return err
			}
			if provenance {
				for _, row := range reporter.ProvenanceRows() {
					row[0] = configKey(row[0])
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "# %s\n", strings.Join(row, " | ")); err != nil {
						return err
					}
				}
			}
			return nil
		}),
	}
	output.bind(command, false)
	command.Flags().BoolVar(&provenance, "provenance", false, "Include each field's source")
	return command
}

// configKey maps go-config-loader's provenance path, the lowercased Go
// field name, to the TOML key users write.
func configKey(path string) string {
	if path == "allowwrites" {
		return "allow_writes"
	}
	return path
}

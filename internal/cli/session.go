package cli

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/config"
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

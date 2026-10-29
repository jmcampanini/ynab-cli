package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// configPath returns the --config value, or empty to discover the file.
func configPath(cmd *cobra.Command) (string, error) {
	path, err := cmd.Flags().GetString("config")
	if err != nil {
		return "", err
	}
	if cmd.Flags().Changed("config") && path == "" {
		return "", errors.New("--config requires a nonempty file path")
	}
	return path, nil
}

// Package config loads the CLI's layered configuration: defaults, the TOML
// file, environment variables, then flags.
package config

import (
	"errors"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/jmcampanini/go-config-loader/configloader"
	"github.com/jmcampanini/go-config-loader/pflagloader"
	"github.com/spf13/pflag"
)

// TokenEnv is the only environment variable that sets Token. The token has
// no flag on purpose: a token on the command line lands in shell history.
const TokenEnv = "YNAB_TOKEN"

// Redacted replaces a configured token in every report.
const Redacted = "<redacted>"

// Config is the effective application configuration.
type Config struct {
	AllowWrites bool   `toml:"allow_writes" config:"allow-writes" help:"Enable mutating commands for this invocation"`
	Plan        string `toml:"plan" config:"plan" help:"Plan ID or exact plan name"`
	Token       string `toml:"token"`
}

// Redact returns a copy safe to print: a nonempty token becomes Redacted.
func (c Config) Redact() Config {
	if c.Token != "" {
		c.Token = Redacted
	}
	return c
}

// Loaded is the outcome of one configuration load.
type Loaded struct {
	Config Config
	// Path is the file that was read, or the discovered path when no
	// explicit file was given, whether or not it exists.
	Path   string
	Report configloader.LoadReport
}

// Load applies defaults, the TOML file, environment, then root flags. An
// empty path discovers $XDG_CONFIG_HOME/ynab/ynab.toml, which may be absent;
// an explicit path must exist.
func Load(path string, flags *pflag.FlagSet) (Loaded, error) {
	var fileLoader configloader.ConfigLoader[Config]
	var err error
	if path == "" {
		helper, helperErr := configloader.NewFileHelper("ynab", "ynab.toml")
		if helperErr != nil {
			return Loaded{}, helperErr
		}
		discovered := helper.XDGConfigFile()
		if len(discovered) > 0 {
			path = discovered[0]
		}
		fileLoader, err = configloader.NewMergeAllFilesLoader[Config](discovered)
	} else {
		fileLoader, err = configloader.NewRequiredFileLoader[Config](path)
	}
	if err != nil {
		return Loaded{}, fmt.Errorf("load config %q: %w", path, err)
	}

	env := configloader.OSEnv()
	envLoader, err := configloader.NewEnvironmentLoader[Config]("ynab", env)
	if err != nil {
		return Loaded{}, err
	}
	flagLoader, err := pflagloader.NewLoader[Config](flags)
	if err != nil {
		return Loaded{}, err
	}
	cfg, report, err := configloader.Load(Config{}, fileLoader, envLoader, flagLoader)
	if err != nil {
		return Loaded{}, fmt.Errorf("load config %q: %w", path, redactParseError(err))
	}

	// go-config-loader registers a flag for every env-backed field, so the
	// token's environment layer is applied here to keep --token from existing.
	// An empty variable counts as unset: no token is ever legitimately empty.
	if token := env[TokenEnv]; token != "" {
		cfg.Token = token
		report.Updates["token"] = configloader.SourceEnv
	}
	return Loaded{Config: cfg, Path: path, Report: report}, nil
}

// redactParseError hides the parser's message for a failure on the token
// key, because that message echoes the unparsed value, as in an unquoted
// token = abc123. Other keys keep the parser's detail.
func redactParseError(err error) error {
	var parseErr toml.ParseError
	if !errors.As(err, &parseErr) || parseErr.LastKey != "token" {
		return err
	}
	return fmt.Errorf("toml: line %d: the token value could not be parsed; write it as token = \"...\"", parseErr.Position.Line)
}

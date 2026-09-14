package cmd

// Shared fragments compose command long descriptions so repeated contract
// text cannot drift between commands.

const configHelp = `Configuration:
  Read $XDG_CONFIG_HOME/ynab/ynab.toml, or ~/.config/ynab/ynab.toml when
  XDG_CONFIG_HOME is empty; a missing file is fine. Relative XDG_CONFIG_HOME
  follows go-config-loader and resolves against the working directory.
  --config PATH replaces discovery and the file must exist. Precedence is
  defaults, TOML, environment, then flags. Keys: plan (ID or exact name),
  token, and allow_writes (default false). YNAB_PLAN, YNAB_TOKEN, and
  YNAB_ALLOW_WRITES override TOML; --plan and --allow-writes override the
  environment. There is no --token flag, so tokens stay out of shell
  history. 'ynab config' prints the effective values with the token
  redacted.`

const planHelp = `Plan selection:
  Commands that read a plan need plan set in the config file, YNAB_PLAN, or
  --plan. A plan name is matched case-insensitively against the exact names
  from 'ynab plans list'; an ID is used as is. Each invocation resolves the
  plan with one request to the plans endpoint. Nothing is remembered between
  invocations.`

const monthHelp = `Months:
  MONTH and --month accept current, YYYY-MM, or YYYY-MM-01, and default to
  current. current is the calendar month in UTC, which is how the API
  defines the current plan month. Any other form is a usage error.`

const outputHelp = `Output:
  Results go to stdout; diagnostics go to stderr. The default is a human
  table or field listing. --jsonl writes one JSON object per line and --csv
  writes a header row and one row per record; both use the same field names,
  never carry color, and are mutually exclusive. --color auto|always|never
  controls the human output; auto disables color when stdout is not a
  terminal, TERM is dumb, or NO_COLOR is set. Amounts are decimal currency
  units with outflows negative. See 'ynab help output-formats' for every
  record shape and 'ynab help exit-codes' for exit statuses.`

const dateHelp = `Dates:
  --since and --until accept YYYY-MM-DD, today, or yesterday. today and
  yesterday follow the local clock. Any other form is a usage error.`

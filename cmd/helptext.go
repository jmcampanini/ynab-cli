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
  MONTH and --month accept current, YYYY-MM, or YYYY-MM-01. current is the
  calendar month in UTC, which is how the API defines the current plan
  month. Any other form is a usage error.`

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

const dateFlagHelp = `Dates:
  --date accepts YYYY-MM-DD, today, or yesterday. today and yesterday
  follow the local clock. Any other form is a usage error.`

const writeHelp = `Writes:
  Mutating commands stay disabled until allow_writes is true in the config
  file or YNAB_ALLOW_WRITES, or --allow-writes is passed; otherwise they
  exit 3 after loading the configuration and before any request. --dry-run
  resolves names and validates input with real reads, builds the request,
  and prints the records the real run would print without sending it;
  --jsonl marks each such record dry_run: true and human output ends with
  "dry run, nothing changed". --dry-run wins over --allow-writes. Every
  mutating command prints the resulting records on stdout in the shapes
  of the read commands, so a caller can save them. There is no undo.`

const bulkHelp = `Bulk updates:
  IDs are sent in one bulk update per 100 transactions, a limit this CLI
  chooses; past one batch, progress goes to stderr. If a later batch
  fails, the records already applied are printed before the error with
  the IDs not changed, and the exit status is 1. A dry run, or a check
  that needs the current records, reads them with one plan listing
  request plus one request per ID older than the API's one-year default
  window; a dry run prints them with the change applied.`

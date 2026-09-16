# ynab-cli

ynab-cli is a command line for a YNAB plan. It reads accounts, categories,
months, and transactions, writes transactions and assigned amounts, and emits
JSON lines and CSV for scripts and agents. Writes are disabled until the
configuration enables them, and every write has a dry run. This release
reads plans, accounts, categories, months, transactions, payees, scheduled
transactions, and money movements, summarizes the month with
`ynab plans status`, and writes transactions: create, update, delete,
approve, categorize, clear, unclear, flag, and import; the
[milestones](plans/milestones.md) record the order the rest lands in, and
the [domain map](plans/domain-map.md) records the nouns, verbs, and
decisions.

Command help is the canonical reference: `ynab --help` and each command's
`--help` describe every user-facing contract, `ynab config --help` describes
the configuration file, discovery, and precedence, `ynab help exit-codes`
describes exit statuses, and `ynab help output-formats` describes every
`--jsonl` and `--csv` record.

## Install

ynab-cli distributes from HEAD only; there is no release channel or tagged
binary.

### Homebrew

```sh
brew tap jmcampanini/ynab-cli https://github.com/jmcampanini/ynab-cli
brew install --HEAD jmcampanini/ynab-cli/ynab
```

Upgrade to the latest commit:

```sh
brew upgrade --fetch-HEAD ynab
```

### From source

```sh
make build
# then copy ./build/ynab to a directory on your PATH
```

## Representative commands

| Command | Result |
|---|---|
| `ynab plans list` | List every plan the token can read, marking the configured one. |
| `ynab plans get` | Show the configured plan's months, currency, and date format. |
| `ynab accounts list` | List the plan's open accounts with balances as a table. |
| `ynab accounts list --closed` | Include closed accounts. |
| `ynab accounts list --jsonl \| jq .balance` | Emit one JSON object per account. |
| `ynab accounts list --csv > accounts.csv` | Write the same records as CSV. |
| `ynab accounts get "Chase Checking"` | Show every field of one account by exact name or ID. |
| `ynab categories list` | List this month's categories by group with assigned, activity, available, and targets. |
| `ynab categories list --month 2026-08 --jsonl \| jq 'select(.available < 0)'` | Find last month's overspent categories. |
| `ynab categories get "Bills: Internet"` | Show one category by ID, exact name, or `Group: Name`, with its target decoded. |
| `ynab category-groups list` | List the category groups with their category counts. |
| `ynab months list` | List every month's income, assigned, activity, ready to assign, and age of money. |
| `ynab months get` | Show this month's totals and its category rows. |
| `ynab transactions list --since 2026-08-01 --until 2026-08-31 --csv > august.csv` | Write a month of the register as CSV, one row per split line. |
| `ynab transactions list --account Visa --unapproved` | List the imports awaiting approval in one account. |
| `ynab transactions list --category Groceries --jsonl >> history.jsonl` | Append the transactions touching one category, splits included. |
| `ynab transactions get ID` | Show every field of one transaction, with its split lines. |
| `ynab transactions review` | List what needs approval or a category, oldest first, with counts. |
| `ynab transactions approve --all-unapproved --dry-run` | Show what approving every pending import would do, changing nothing. |
| `ynab transactions create --account Checking --date today --amount -12.50 --payee "Corner Store" --category Groceries --allow-writes` | Record a transaction; a new payee name creates the payee. |
| `ynab transactions categorize --category Groceries ID... --allow-writes` | Set a category on transactions by ID, 100 per request. |
| `ynab transactions delete ID --allow-writes --jsonl >> deleted.jsonl` | Delete a transaction and keep the record it printed. |
| `ynab payees list --unused` | List payees no transaction references. |
| `ynab scheduled list` | List the scheduled transactions with their next dates. |
| `ynab money-movements list --month current` | List this month's recorded moves between categories. |
| `ynab plans status` | Show ready to assign, overspent, underfunded, unapproved, uncategorized, and import errors on one screen. |
| `ynab config --provenance` | Print the effective configuration with each field's source. |

## Required external programs

None. ynab-cli needs network access to api.ynab.com and never prompts.

## Configuration

ynab-cli reads `$XDG_CONFIG_HOME/ynab/ynab.toml`, or `~/.config/ynab/ynab.toml`
when `XDG_CONFIG_HOME` is empty; a missing file is fine. `--config PATH`
replaces discovery, and the file must exist. `YNAB_PLAN`, `YNAB_TOKEN`, and
`YNAB_ALLOW_WRITES` override the file; `--plan` and `--allow-writes` override
the environment. There is no `--token` flag. A minimal file:

```toml
plan = "Household"
token = "your-personal-access-token"
```

Create a token at https://app.ynab.com/settings/developer. `ynab config --help`
documents the format and precedence, and `ynab config` prints the values in
effect with the token redacted.

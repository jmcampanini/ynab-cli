# ynab-cli

ynab reads and writes a YNAB plan from the terminal and emits JSON lines and CSV for scripts and agents. Writes are disabled until the configuration enables them, and every write has a dry run. ynab needs network access to api.ynab.com and never prompts.


## Install

ynab is configured for versioned releases with prebuilt binaries for Linux and macOS on amd64 and arm64. The first release has not been published; until then, use the HEAD formula or build from source. After the first stable release, install and upgrade with mise or the shared Homebrew tap:

```sh
mise use -g github:jmcampanini/ynab-cli
mise upgrade github:jmcampanini/ynab-cli
brew install --cask jmcampanini/tap/ynab
brew upgrade --cask jmcampanini/tap/ynab
```

Or download the archive for your platform from the [releases page](https://github.com/jmcampanini/ynab-cli/releases), verify it against the release's SHA-256 checksums file, and put `ynab` on your `PATH`. To upgrade, download the new release and replace the binary. Releases include GitHub build provenance attestations. If a browser download on macOS is quarantined, run `xattr -d com.apple.quarantine ynab` on the extracted binary before running it.


### Homebrew HEAD alternative

The existing formula remains available for building the latest commit, including its shell completions:

```sh
brew tap jmcampanini/ynab-cli https://github.com/jmcampanini/ynab-cli
brew install --HEAD jmcampanini/ynab-cli/ynab
```

Upgrade to the latest commit:

```sh
brew upgrade --fetch-HEAD jmcampanini/ynab-cli/ynab
```

To switch an existing HEAD installation to the released cask after the first stable release, uninstall the formula with `brew uninstall --formula jmcampanini/ynab-cli/ynab` before installing the cask. Both packages install the `ynab` executable.


### From source

```sh
brew install git mise
git clone https://github.com/jmcampanini/ynab-cli
cd ynab-cli
mise trust
mise install
mise run build
```

Copy `./build/ynab` to a directory on your `PATH`. To upgrade a source installation, pull the latest source, run `mise install` and `mise run build`, and replace the installed binary.


## Representative commands

`ynab plans status` names what needs attention, `ynab transactions review` lists the rows `ynab transactions approve` acts on, and `ynab reports funding --underfunded` lists what `ynab months fund --all-underfunded` would assign.

| Command | Result |
|---|---|
| `ynab plans list` | List every plan the token can read, marking the configured one. |
| `ynab plans status` | Show ready to assign, overspent, underfunded, unapproved, uncategorized, and import errors on one screen. |
| `ynab accounts list` | List the plan's open accounts with balances as a table. |
| `ynab categories list --month 2026-08 --jsonl \| jq 'select(.available < 0)'` | Find August 2026's overspent categories. |
| `ynab transactions list --since 2026-08-01 --until 2026-08-31 --csv > august.csv` | Write a month of the register as CSV, with split lines after their parent row. |
| `ynab transactions review` | List what needs approval or a category, oldest first, with counts. |
| `ynab transactions approve --all-unapproved --dry-run` | Show what approving every transaction awaiting approval would do, changing nothing. |
| `ynab months move 50 --from "Dining Out" --to Groceries --dry-run` | Show both rows as a move would leave them, changing nothing. |
| `ynab months cover "Dining Out" --from ready-to-assign --allow-writes` | Move exactly an overspent category's shortfall into it. |
| `ynab months fund --all-underfunded --allow-writes` | Assign what every visible underfunded target still needs, with the total. |
| `ynab transactions create --account Checking --date today --amount -12.50 --payee "Corner Store" --category Groceries --allow-writes` | Record a transaction; a new payee name creates the payee. |
| `ynab reports funding --underfunded` | List the targets still needing money this month with what they need. |
| `ynab reports spending --by payee --since 2026-01-01 --csv` | Total outflows and inflows per payee since a date, ready for a spreadsheet. |
| `ynab api get /user` | Print the raw API response, milliunits and all. |

The canonical reference is `ynab --help`, each command's `--help`, and the help topics `ynab help exit-codes` and `ynab help output-formats`.


## Configuration

ynab reads `$XDG_CONFIG_HOME/ynab/config.toml`, or `~/.config/ynab/config.toml` when `XDG_CONFIG_HOME` is empty; a missing file is fine. A minimal file:

```toml
plan = "Household"
token = "your-personal-access-token"
```

Create a token at https://app.ynab.com/settings/developer. `ynab config --help` documents the format and precedence.

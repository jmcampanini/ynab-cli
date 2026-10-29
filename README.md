# ynab-cli

ynab reads and writes a YNAB plan from the terminal and exports JSON, JSON lines, and CSV for scripts and agents. Writes are disabled until the configuration enables them, and every write has a dry run. ynab needs network access to api.ynab.com and never prompts.


## Install

ynab publishes versioned releases with prebuilt binaries for Linux and macOS on amd64 and arm64.

Install and upgrade with mise:

```sh
mise use -g github:jmcampanini/ynab-cli
mise upgrade github:jmcampanini/ynab-cli
```

If replacing the HEAD formula, first run `brew uninstall --formula jmcampanini/ynab-cli/ynab`; install and upgrade with Homebrew through the shared tap:

```sh
brew install --cask jmcampanini/tap/ynab
brew upgrade --cask jmcampanini/tap/ynab
```

Install or upgrade a [release archive](https://github.com/jmcampanini/ynab-cli/releases) with curl, shasum, and tar by selecting its version and platform below, verifying its checksum, and copying the binary into `~/.local/bin`, which must be on your `PATH`:

```sh
version=0.1.0
platform=darwin_arm64 # Or darwin_amd64, linux_amd64, linux_arm64.
archive="ynab_${version}_${platform}.tar.gz"
checksums="ynab_${version}_checksums.txt"
release="https://github.com/jmcampanini/ynab-cli/releases/download/v${version}"
curl -fLO "${release}/${archive}" &&
  curl -fLO "${release}/${checksums}" &&
  grep "  ${archive}$" "${checksums}" | shasum -a 256 -c - &&
  tar -xzf "${archive}" &&
  mkdir -p "$HOME/.local/bin" &&
  cp ynab "$HOME/.local/bin/ynab"
```

If replacing the cask, first run `brew uninstall --cask ynab`; install and upgrade the latest source commit with Homebrew's HEAD formula:

```sh
brew tap jmcampanini/ynab-cli https://github.com/jmcampanini/ynab-cli
brew install --HEAD jmcampanini/ynab-cli/ynab
brew upgrade --fetch-HEAD jmcampanini/ynab-cli/ynab
```

Build from source with Git and mise, then copy the binary into `~/.local/bin`, which must be on your `PATH`:

```sh
git clone https://github.com/jmcampanini/ynab-cli
cd ynab-cli
mise run build
mkdir -p "$HOME/.local/bin"
cp build/ynab "$HOME/.local/bin/ynab"
```


## Representative commands

`ynab plans status` names what needs attention, `ynab transactions review --all` lists the rows `ynab transactions approve` acts on, and `ynab reports funding --underfunded --all` lists what `ynab months fund --all-underfunded` would assign.

| Command | Result |
|---|---|
| `ynab plans list --all` | List every plan the token can read, marking the configured one. |
| `ynab plans status` | Show ready to assign, overspent, underfunded, unapproved, uncategorized, and import errors on one screen. |
| `ynab accounts list` | List the plan's open accounts with balances as a table. |
| `ynab categories list --month 2026-08 --all --jsonl \| jq 'select(.available < 0)'` | Find August 2026's overspent categories. |
| `ynab transactions list --since 2026-08-01 --until 2026-08-31 --all --csv --output-file august.csv` | Save a month of the register as CSV, with split lines in each transaction's `subtransactions` cell. |
| `ynab transactions review` | List what needs approval or a category, oldest first, with counts. |
| `ynab transactions approve --all-unapproved --dry-run` | Show what approving every transaction awaiting approval would do, changing nothing. |
| `ynab months move 50 --from "Dining Out" --to Groceries --dry-run` | Show both rows as a move would leave them, changing nothing. |
| `ynab months cover "Dining Out" --from ready-to-assign --allow-writes` | Move exactly an overspent category's shortfall into it. |
| `ynab months fund --all-underfunded --allow-writes` | Assign what every visible underfunded target still needs, with the total. |
| `ynab transactions create --account Checking --date today --amount -12.50 --payee "Corner Store" --category Groceries --allow-writes` | Record a transaction; a new payee name creates the payee. |
| `ynab reports funding --underfunded` | List the targets still needing money this month with what they need. |
| `ynab reports spending --by payee --since 2026-01-01 --all --csv` | Total outflows and inflows per payee since a date, ready for a spreadsheet. |
| `ynab api get /user` | Print the raw API response, milliunits and all. |

The canonical reference is `ynab --help`, each command's `--help`, and the help topics `ynab help exit-codes` and `ynab help output-formats`.


## Configuration

ynab reads `$XDG_CONFIG_HOME/ynab/config.toml`, or `~/.config/ynab/config.toml` when `XDG_CONFIG_HOME` is unset, empty, or relative; a missing discovered file is fine. A minimal file:

```toml
plan = "Household"
token = "your-personal-access-token"
```

Create a token at https://app.ynab.com/settings/developer. `ynab config --help` documents the format and precedence.

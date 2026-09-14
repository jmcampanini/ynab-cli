# Milestones

Each milestone is one slice of ynab-cli that a person can install and test by
hand as soon as it lands. Slices build on each other in order. The
[domain map](domain-map.md) owns the nouns, verbs, and decisions; this file
owns the order of delivery. Each milestone has a GitHub milestone of the same
name and one issue describing how an agent implements it.

Every slice follows the Gibson `go` and `cli` facets: Cobra, root `main.go`,
one command per file in `cmd/`, domain code in `internal/`, `make check` as
the verification contract, command help as the canonical documentation, and
the `exit-codes` and `output-formats` help topics kept current.


## 1. Read the plan ([#1](https://github.com/jmcampanini/ynab-cli/issues/1))

The repository scaffold plus enough reads to prove the token, the config, and
the output contract work.

Delivers: Go module and Makefile per Gibson, `check` workflow, configuration
(`plan`, `token`, `allow_writes`) with the `config` command, the YNAB HTTP
client with error mapping, `plans list`, `plans get`, `accounts list`,
`accounts get`, human tables, `--jsonl`, `--csv`, color handling, the
`exit-codes` and `output-formats` topics.

Test by hand: put a token in the config file, run `ynab plans list`, set
`plan`, run `ynab accounts list` and `ynab accounts list --jsonl | jq`.


## 2. Read the budget ([#2](https://github.com/jmcampanini/ynab-cli/issues/2))

Categories and months, which introduce name resolution, month arguments, and
currency formatting.

Delivers: `categories list`, `categories get`, `category-groups list`,
`months list`, `months get`, exact-name resolution with `Group: Name`,
month arguments (`current`, `YYYY-MM`), amounts formatted from plan settings,
target fields on category rows.

Test by hand: `ynab months get` shows this month's totals; `ynab categories
list --month 2026-08` shows last month's assigned, activity, and available;
`ynab categories get "Bills: Internet"` resolves by name.


## 3. Read transactions ([#3](https://github.com/jmcampanini/ynab-cli/issues/3))

The register, its filters, and the remaining read-only nouns.

Delivers: `transactions list` with every filter (account, category, payee,
since, until, unapproved, uncategorized, cleared, flag), `transactions get`,
`transactions review`, `payees list`, `payees get`, `scheduled list`,
`scheduled get`, `money-movements list`, `plans status`.

Test by hand: `ynab transactions list --since 2026-08-01 --csv >
august.csv`; `ynab transactions review` shows what needs attention;
`ynab plans status` gives the one-screen summary.


## 4. Write transactions ([#4](https://github.com/jmcampanini/ynab-cli/issues/4))

The write posture and the transaction verbs that carry the monthly work.

Delivers: `allow_writes` enforcement with exit code 3 and `--allow-writes`,
`--dry-run` on every mutating command, `transactions create` (including
splits and transfers), `update`, `delete`, `approve`, `categorize`, `clear`,
`unclear`, `flag`, `import`.

Test by hand: `ynab transactions create ... --dry-run` prints the record and
changes nothing; without `--allow-writes` it exits 3; with it, the
transaction appears in YNAB; `ynab transactions delete ID` prints it back.


## 5. Move money and maintain the plan ([#5](https://github.com/jmcampanini/ynab-cli/issues/5))

The month writes and the remaining create and update verbs.

Delivers: `months assign`, `months move`, `months cover`, `months fund`,
`categories create`, `categories update` (name, note, group, target),
`category-groups create`, `category-groups rename`, `payees create`,
`payees rename`, `accounts create`, `scheduled create`, `scheduled update`,
`scheduled delete`.

Test by hand: `ynab months move 50 --from "Dining Out" --to Groceries
--dry-run`, then for real, and the YNAB app shows both categories changed.


## 6. Reports and the escape hatch ([#6](https://github.com/jmcampanini/ynab-cli/issues/6))

Derived reads and the raw API passthrough.

Delivers: `reports funding`, `reports spending --by category|payee`,
`api get|post|patch|put|delete`, shell completion.

Test by hand: `ynab reports funding` lists underfunded targets;
`ynab reports spending --by payee --since 2026-01-01 --csv` opens in a
spreadsheet; `ynab api get /user` prints the raw response.

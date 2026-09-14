# ynab-cli

ynab-cli is a command line for a YNAB plan. It reads accounts, categories,
months, and transactions, writes transactions and assigned amounts, and emits
JSON lines and CSV for scripts and agents. Writes are disabled until the
configuration enables them, and every write has a dry run.

Nothing is built yet. The [domain map](plans/domain-map.md) records the
nouns, verbs, and decisions, and the [milestones](plans/milestones.md) record
the order of delivery. Once the first milestone lands, command help becomes
the canonical reference: `ynab --help`, `ynab config --help`,
`ynab help exit-codes`, and `ynab help output-formats`.

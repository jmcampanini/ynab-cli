# Domain map

The nouns, verbs, and concepts ynab-cli owns, how they map onto the YNAB API,
and the design decisions that shape every command. The [milestones](milestones.md)
slice this map into deliverable work.

Sources: the YNAB API v1 OpenAPI spec (v1.86.0) and docs at https://api.ynab.com,
a survey of nine YNAB CLIs and about twenty MCP servers, and the Gibson CLI wiki.
Decisions were settled on 2026-09-13.


## Scope

ynab-cli is a generic command line for a YNAB plan, usable by a person at a
terminal and by an agent through machine-readable output. It exposes what the
API exposes with consistent nouns and verbs. It does not carry opinions about
how to budget: no categorization rules, no anomaly detection, no household-
specific views. Those compose on top of it.


## Terminology

YNAB renamed budgets to plans in the API in March 2026 (`/plans/{plan_id}`,
`default_plan`). The CLI says `plan` everywhere and never `budget`.

Other terms, used consistently in help, output, and code:

| Term | Meaning |
| --- | --- |
| plan | The container for accounts, categories, months, and transactions. |
| assigned | Money given to a category in a month. The API calls it `budgeted`. |
| activity | Net transactions in a category for a month. |
| available | A category's running balance. The API calls it `balance`. |
| ready to assign | Plan money not yet assigned. The API calls it `to_be_budgeted`. |
| target | A category's funding rule. The API calls it a goal. |
| milliunits | The API's integer amount unit, 1000 per currency unit. Never shown to users. |


## Nouns

Ranked by how much they matter to the monthly workflow. R = read, W = write.

| Noun | What it is | API | Notes |
| --- | --- | --- | --- |
| plan | The container. | R only | `GET /plans/{id}` returns the whole plan in one call. Settings hold date and currency format. No create, rename, or delete. |
| account | Checking, savings, cash, credit cards, tracking accounts. | R, create | `on_budget`, `closed`, three balances, `last_reconciled_at`, direct import status. No rename, close, delete, or reconcile. |
| transaction | The unit of work. | Full CRUD, bulk create and update | Date, amount, payee, category, memo, cleared (`uncleared`, `cleared`, `reconciled`), approved, flag, import id, transfer link, subtransactions. Filters: `since_date`, `until_date`, `type=uncategorized|unapproved`, by account, category, payee, or month. Default window is one year back. |
| subtransaction | A line of a split transaction. | Create with the parent, or convert an unsplit transaction through `api put` | Cannot edit the lines of an existing split, or change the parent's date, amount, or category. |
| transfer | A transaction whose payee is another account. | Through transaction | Set `payee_id` to the target account's `transfer_payee_id`. The API creates the pair. |
| category | A plan line. | R, create, update (name, note, group, target) | Per month: assigned, activity, available. `hidden`, `internal`. No delete, hide, or reorder. |
| category group | A grouping of categories. | R, create, rename | Internal groups: the master group (Ready to Assign, Uncategorized) and Credit Card Payments. |
| target | A category's funding rule. | Update through category | Types `TB`, `TBD`, `MF`, `NEED`, `DEBT`. Reads include `goal_under_funded` and `goal_percentage_complete`. Writes cover target amount, date, and frequency only. |
| month | The plan month: income, assigned, activity, ready to assign, age of money, per-category rows. | R, plus one write | `PATCH months/{m}/categories/{c}` sets assigned. Month is `current` or `YYYY-MM-01`. |
| money movement | A recorded category-to-category move. | R only | Cannot be posted. A move is two assigned-amount writes. |
| payee | Who was paid. | R, create, rename | No delete or merge. Payee locations are mobile-only and out of scope. |
| scheduled transaction | A recurring or future transaction. | R only | The API offers create, update, and delete, without splits; the CLI does not maintain scheduled transactions (see Dismissed). |
| user | The token owner. | R (id only) | Useful for a connectivity check. |


## Verbs

Standard verbs, spelled the same on every noun that supports them:
`list`, `get`, `create`, `update`, `delete`.

YNAB-specific verbs on transactions. These are what the monthly work is
made of:

| Verb | Meaning | API |
| --- | --- | --- |
| approve | `approved: false` to `true` | bulk `PATCH transactions` |
| categorize | set `category_id` | bulk `PATCH transactions` |
| clear, unclear | toggle `cleared` | `PATCH` |
| flag | set `flag_color` | `PATCH` |
| import | trigger direct import on linked accounts | `POST transactions/import` |
| review | list unapproved and uncategorized together | two filtered `GET`s |

Money verbs on the month:

| Verb | Meaning | API |
| --- | --- | --- |
| assign | set a category's assigned amount, absolute or by delta | `PATCH` month category |
| move | shift an amount between two categories, or from ready to assign | two `PATCH`es, not atomic; report partial failure |
| cover | move into an overspent category up to its shortfall | derived from `available < 0`, then `PATCH`es |
| fund | assign a target's underfunded amount | derived from `goal_under_funded`, then `PATCH` |

Read-side derived verbs:

- `plans status`: ready to assign, overspent categories, underfunded targets,
  unapproved and uncategorized counts, accounts in import error.
- `reports funding`: per category for a month, assigned, activity, available,
  target, underfunded.
- `reports spending`: totals over a date range grouped by category or payee.


## Command surface

Every command is `ynab <noun> <verb>`. Nouns are plural groups. Bare groups
print help. This is uniform on purpose: help, completion, and documentation
follow one rule, and an agent needs one pattern.

```
ynab config                       effective TOML, --provenance
ynab plans list                   works without a configured plan
ynab plans get                    summary and settings
ynab plans status
ynab accounts list|get|create
ynab categories list|get|create|update
ynab category-groups list|create|rename
ynab months list|get
ynab months assign|move|cover|fund
ynab transactions list|get|create|update|delete
ynab transactions approve|categorize|clear|unclear|flag|import|review
ynab payees list|get|create|rename
ynab scheduled list|get
ynab money-movements list
ynab reports funding|spending
ynab api get|post|patch|put|delete PATH
ynab completion bash|zsh|fish|powershell
ynab help exit-codes
ynab help output-formats
```

`api` is the escape hatch. It passes a raw request through and prints the raw
response, milliunits included. Anything odd goes there instead of growing a
one-off command.

`completion` is Cobra's. Operands and flags that take an account, category,
category group, or payee name complete from the plan with one listing
request, and offer nothing when the request fails.


## Decisions

**Write posture.** Config `allow_writes` ships false. A mutating command run
without it fails with its own exit code and an error naming the setting.
`--allow-writes` overrides for one invocation. Every mutating command accepts
`--dry-run`, which prints the records the real run would print and makes no
request. There is no undo journal; deletes print the deleted record so a
caller can recreate it. Personal access tokens are full-scope across every
plan on the account and the API has no undo, so the safe state is the default.

**Amounts.** Input is dollars as typed: `450`, `-97.81`. Outflows are
negative, matching YNAB. Tables format with the plan's currency settings.
JSON carries one field, `amount`, as a two-decimal number written from the
exact milliunit value (`450.00`). Floats never appear in the code path.
Milliunits appear only in `api` output.

**Name resolution.** Every account, category, or payee operand accepts an ID
or the exact name, matched case-insensitively. No prefix or fuzzy matching:
on a write, a near miss must fail, not guess. Categories that share a name
across groups are written `Group: Name`. A failed lookup lists the closest
existing names in its error. Hidden, closed, and deleted entities are
excluded unless a flag asks for them.

**Configuration.** Per the Gibson configuration policy: defaults, then TOML
file, then environment, then flags, loaded with go-config-loader. Keys are
`plan` (ID or exact name), `token`, and `allow_writes`. Environment
`YNAB_PLAN` and `YNAB_TOKEN`. Flag `--plan`. There is no `--token` flag,
because tokens on the command line land in shell history. There is no
`last-used` default: `plan` is unset until configured, and every command that
needs one fails pointing at `plans list`. The `config` command redacts
`token`.

**Output.** Human tables on stdout by default. `--jsonl` on every command
writes one JSON object per line: one line per record for lists, one line for
`get`, `status`, and `config`. `--csv` on list commands. jq reads JSONL
natively and `>>` appends it, so there is no JSON array form. Diagnostics go
to stderr. Color follows the Gibson terminal output policy. The
`output-formats` help topic documents every record shape.

**Disk.** The CLI keeps no cache and reads no saved output back. A caller
who wants history saves `--jsonl` output and refreshes it with a date-filtered
re-fetch. The API's `server_knowledge` delta cursor is not exposed in this
pass; `api` returns it raw if needed.

**Rate limit.** 200 requests per hour per token. Most commands make one
request. Commands that need several resources, such as `plans status`, make
one typed request per resource and state the count in their help. The
one-call full plan export is preferred only when a command needs the
transaction history, which none does yet; it carries every transaction the
plan has ever had. Uncategorized listings exclude tracking-account entries
and transfers between plan accounts, matching the web app's category-needed
workflow. A 429 fails with an error naming the limit.

**Dates.** ISO `YYYY-MM-DD` for dates, `YYYY-MM` for months, plus `today`,
`yesterday`, and `current`.

**Exit codes.** 0 success, 1 command failure (API error, partial move),
2 usage, 3 writes disabled. Documented in the `exit-codes` topic.


## What the API cannot do

Do not promise these. State them in the owning command's help.

- Create, rename, or delete a plan.
- Rename, close, delete, or reconcile an account; set an account note; create
  loan or mortgage accounts; link or unlink direct import.
- Delete, hide, unhide, or reorder categories or groups.
- Record a money movement as its own entity; set month notes.
- Delete or merge payees; manage rename rules.
- Account-level reconcile. Only per-transaction `cleared: reconciled`.
- Edit the lines of an existing split.
- Create split scheduled transactions.
- Read pending bank transactions.
- Bulk approve in one request; it is a bulk `PATCH`, which is fine.


## Dismissed

Considered and left out, with the reason:

- Fuzzy name matching: silently wrong on writes.
- Milliunits in JSON: no consumer needs them; `api` has them.
- JSON array output: jq reads JSONL and `>>` appends it.
- `server_knowledge` cursor as a flag and field: date filters cover the disk
  workflow at household volume; add back as one flag and one field if needed.
- Local cache: a subsystem before the rate limit is a problem.
- Undo journal: a subsystem; deletes print the record instead.
- Keychain token storage and `auth login`: an interactive command and a
  platform dependency for a single-user tool.
- Payee-to-category rules, outlier reports, household-specific views: budget
  opinions that compose on top of the CLI.
- Scheduled transaction writes (`scheduled create|update|delete`), dropped on
  2026-09-15: the owner reads scheduled transactions and lets the
  transactions they enter be imported, and does not maintain them from the
  terminal. The API's update is a PUT that requires a date, so an update
  would have to choose one, a decision not worth making for an unused verb.

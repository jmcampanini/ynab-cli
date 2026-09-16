package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// maxImportIDLength is the API's limit on import_id.
const maxImportIDLength = 36

// splitLine is one parsed --split value before its category is resolved.
type splitLine struct {
	amount   string
	category string
	memo     string
}

// parseSplit parses CATEGORY=AMOUNT[:memo]. The first "=" ends the
// category, so "Group: Name" is fine, and the first ":" after it starts
// the memo, which an amount never contains.
func parseSplit(text string) (splitLine, error) {
	category, rest, found := strings.Cut(text, "=")
	if !found || category == "" || rest == "" {
		return splitLine{}, usageError(fmt.Sprintf("invalid --split %q: use CATEGORY=AMOUNT or CATEGORY=AMOUNT:memo", text))
	}
	amount, memo, _ := strings.Cut(rest, ":")
	return splitLine{amount: amount, category: category, memo: memo}, nil
}

func newTransactionsCreate(a *app) *cobra.Command {
	var output outputFlags
	var fields transactionFields
	var dryRun bool
	var importID string
	var splits []string
	command := &cobra.Command{
		Use: "create --account A --date D --amount N [flags]", Short: "Create a transaction, split, or transfer",
		Long: `Create one transaction in the configured plan and print it as stored,
in the shape of 'transactions get'. --account, --date, and --amount are
required; the amount is in currency units with outflows negative, at the
plan's decimal precision, and the API rejects a future date.

--payee takes a payee ID or name; a name that matches no payee creates
one. --transfer-to instead makes the transaction a transfer to that
account, and --category is then refused when both accounts are plan
accounts, since the API ignores it. --category takes an ID, an exact
name, or "Group: Name"; a Credit Card Payments category is refused, since
the API ignores it and a card payment is a transfer. --cleared defaults
to uncleared and --approved to false, as in the API. --flag sets a color.
--import-id is at most 36 characters; one already on a transaction is
refused by the API.

--split CATEGORY=AMOUNT[:memo] may repeat to create a split; the amounts
must sum to --amount and --category is then refused. The API cannot
change the lines of a split once it exists.

Requests: the plans endpoint, the accounts endpoint, the payees endpoint
with --payee, the categories endpoint with --category or --split, then
the create. A dry run makes every read and skips the create.

` + writeHelp + "\n\n" + planHelp + "\n\n" + dateFlagHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions create --account "Cash on Hand" --date today \
    --amount -1.00 --payee "CLI test" --category Groceries --dry-run
  ynab transactions create --account Checking --date 2026-09-14 --amount -100 \
    --payee Costco --split "Groceries=-60" --split "Household=-40:paper towels"
  ynab transactions create --account Checking --date today --amount -500 \
    --transfer-to Savings --allow-writes`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			if len(importID) > maxImportIDLength {
				return usageError(fmt.Sprintf("--import-id is %d characters; the API allows at most %d", len(importID), maxImportIDLength))
			}
			if len(splits) > 0 && cmd.Flags().Changed("category") {
				return usageError("--split and --category cannot be combined; a split's lines carry the categories")
			}
			lines := make([]splitLine, len(splits))
			for i, text := range splits {
				var err error
				if lines[i], err = parseSplit(text); err != nil {
					return err
				}
			}

			s, err := a.openTransactionWrite(cmd, dryRun)
			if err != nil {
				return err
			}
			change, transfer, err := fields.resolve(cmd, s)
			if err != nil {
				return err
			}
			if transfer != nil && change.CategoryID != nil {
				if err := transferCategoryConflict(s.accountByID(change.AccountID), *transfer); err != nil {
					return err
				}
			}
			change.ImportID = importID
			if change.Subtransactions, err = s.resolveSplits(cmd, lines, ynab.Amount(*change.Milliunits)); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if s.dryRun {
				summary := s.summary("created", "create", 1)
				if change.PayeeName != nil {
					summary += fmt.Sprintf(" and the payee %q", *change.PayeeName)
				}
				return s.printTransaction(out, output, s.preview(ynab.Transaction{}, change), summary)
			}
			stored, err := s.client.CreateTransaction(cmd.Context(), s.plan.ID, change)
			if err != nil {
				var apiErr *ynab.Error
				if errors.As(err, &apiErr) && apiErr.Status == http.StatusConflict && importID != "" {
					return fmt.Errorf("import id %q is already on a transaction: %w", importID, err)
				}
				return err
			}
			// The API may have matched the name to a payee through a rename
			// rule, so creation is claimed from the stored record.
			summary := s.summary("created", "create", 1)
			if stored.PayeeID != nil && !s.knownPayee(*stored.PayeeID) {
				summary += fmt.Sprintf(" and the payee %q", text(stored.PayeeName))
			}
			return s.printTransaction(out, output, stored, summary)
		}),
	}
	output.bind(command, false)
	fields.bind(command)
	command.Flags().StringVar(&importID, "import-id", "", "Import id, at most 36 characters")
	command.Flags().StringArrayVar(&splits, "split", nil, "Split line CATEGORY=AMOUNT[:memo]; repeatable")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be created without creating it")
	for _, name := range []string{"account", "date", "amount"} {
		_ = command.MarkFlagRequired(name)
	}
	completeTransactionFields(command, a)
	return command
}

// resolveSplits resolves each line's category and amount and checks that
// the lines sum to the transaction's amount.
func (s *writeSession) resolveSplits(cmd *cobra.Command, lines []splitLine, total ynab.Amount) ([]ynab.SaveSubtransaction, error) {
	if len(lines) == 0 {
		return nil, nil
	}

	subtransactions := make([]ynab.SaveSubtransaction, len(lines))
	var sum ynab.Amount
	for i, line := range lines {
		category, err := s.category(cmd.Context(), line.category)
		if err != nil {
			return nil, err
		}
		amount, err := s.amount(line.amount)
		if err != nil {
			return nil, err
		}
		sum += amount
		subtransactions[i] = ynab.SaveSubtransaction{CategoryID: &category.ID, Milliunits: int64(amount)}
		if line.memo != "" {
			memo := line.memo
			subtransactions[i].Memo = &memo
		}
	}
	if sum != total {
		return nil, usageError(fmt.Sprintf("--split lines total %s but --amount is %s", sum, total))
	}
	return subtransactions, nil
}

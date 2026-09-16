package cmd

import (
	"fmt"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// accountCreateTypes are the account types the API creates, a subset of
// the types it reports.
var accountCreateTypes = []string{"checking", "savings", "cash", "creditCard", "otherAsset", "otherLiability"}

// accountWriteRecord is an account record printed by a mutating command,
// marked when it is a dry-run preview.
type accountWriteRecord struct {
	accountRecord
	DryRun bool `json:"dry_run,omitempty"`
}

func newAccountsCreate(a *app) *cobra.Command {
	var output outputFlags
	var accountType, balanceText string
	var dryRun bool
	command := &cobra.Command{
		Use: "create NAME --type TYPE --balance AMOUNT", Short: "Create an account with an opening balance",
		Long: `Create an account and print it as stored, in the shape of 'accounts
get'. --type is one of the six the API creates: checking, savings, cash,
creditCard, otherAsset, or otherLiability; loan and mortgage accounts
cannot be created here. --balance is the opening balance in currency
units, negative for money owed. A name another account already has,
ignoring case, is refused, since the CLI could not then tell them apart.
Three requests: the plans endpoint, the accounts endpoint for the name
check, then the create.

This cannot be undone through the API: it cannot rename, close, or
delete an account, so a mistaken create must be fixed in the YNAB app.

` + writeHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab accounts create "Emergency Fund" --type savings --balance 0 --dry-run
  ynab accounts create "Travel Card" --type creditCard --balance -250.40 \
    --allow-writes`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return usageError("NAME must not be empty")
			}

			s, err := a.openWrite(cmd, dryRun)
			if err != nil {
				return err
			}
			if s.accounts, err = s.client.Accounts(cmd.Context(), s.plan.ID); err != nil {
				return err
			}
			for _, account := range s.accounts {
				if strings.EqualFold(account.Name, args[0]) {
					return fmt.Errorf("an account named %q already exists (%s)", account.Name, account.ID)
				}
			}
			balance, err := s.amount(balanceText)
			if err != nil {
				return err
			}

			rest := fmt.Sprintf(" %s account %q with %s", accountType, args[0], balance.Format(s.plan.CurrencyFormat))
			account := ynab.Account{Balance: balance, ClearedBalance: balance, Name: args[0], OnPlan: accountType != "otherAsset" && accountType != "otherLiability", Type: accountType}
			if !s.dryRun {
				if account, err = s.client.CreateAccount(cmd.Context(), s.plan.ID, ynab.SaveAccount{Milliunits: int64(balance), Name: args[0], Type: accountType}); err != nil {
					return err
				}
			}

			record := accountWriteRecord{accountRecord: newAccountRecord(account), DryRun: s.dryRun}
			out := cmd.OutOrStdout()
			if output.jsonl {
				return writeJSONL(out, []accountWriteRecord{record})
			}
			if err := writeFields(out, accountFields(record.accountRecord, s.plan.CurrencyFormat)); err != nil {
				return err
			}
			_, err = fmt.Fprintf(out, "\n%s\n", s.line("created", "create", rest))
			return err
		}),
	}
	output.bind(command, false)
	bindEnumFlag(command, &accountType, "type", "type", "Account type: "+strings.Join(accountCreateTypes, ", "), accountCreateTypes...)
	command.Flags().StringVar(&balanceText, "balance", "", "Opening balance in currency units")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be created without creating it")
	_ = command.MarkFlagRequired("type")
	_ = command.MarkFlagRequired("balance")
	return command
}

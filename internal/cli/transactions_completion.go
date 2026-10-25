package cli

import (
	"github.com/spf13/cobra"
)

// completeTransactionFields registers completers for the name-taking
// field flags of transactions create and update.
func completeTransactionFields(cmd *cobra.Command, a *app) {
	completeFlags(cmd, a.completeAccounts(), "account", "transfer-to")
	completeFlags(cmd, a.completeCategories(), "category")
	completeFlags(cmd, a.completePayees(), "payee")
}

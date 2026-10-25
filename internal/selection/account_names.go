package selection

import (
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// AccountNames maps account IDs to names so transfer targets, which the
// transaction endpoints identify only by ID, can be named.
type AccountNames map[string]string

// NewAccountNames indexes the names used to display transfer accounts.
func NewAccountNames(accounts []ynab.Account) AccountNames {
	byID := make(AccountNames, len(accounts))
	for _, account := range accounts {
		byID[account.ID] = account.Name
	}
	return byID
}

// Name returns the account's name, the ID itself when the account is
// unknown, and blank for nil.
func (n AccountNames) Name(id *string) string {
	if id == nil {
		return ""
	}
	if name, ok := n[*id]; ok {
		return name
	}
	return *id
}

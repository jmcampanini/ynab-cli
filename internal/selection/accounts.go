package selection

import (
	"github.com/jmcampanini/ynab-cli/internal/names"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// Account resolves query against account IDs, then names ignoring
// case, closed accounts included.
func Account(query string, accounts []ynab.Account) (ynab.Account, error) {
	candidates := make([]names.Candidate, len(accounts))
	for i, account := range accounts {
		candidates[i] = names.Candidate{ID: account.ID, Names: []string{account.Name}}
	}

	index, err := names.Resolve("account", query, candidates)
	if err != nil {
		return ynab.Account{}, err
	}
	return accounts[index], nil
}

package selection

import (
	"github.com/jmcampanini/ynab-cli/internal/names"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// Payee resolves query against payee IDs, then names ignoring case.
func Payee(query string, payees []ynab.Payee) (ynab.Payee, error) {
	candidates := make([]names.Candidate, len(payees))
	for i, payee := range payees {
		candidates[i] = names.Candidate{ID: payee.ID, Names: []string{payee.Name}}
	}

	index, err := names.Resolve("payee", query, candidates)
	if err != nil {
		return ynab.Payee{}, err
	}
	return payees[index], nil
}

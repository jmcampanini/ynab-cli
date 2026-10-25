package selection

import (
	"github.com/jmcampanini/ynab-cli/internal/names"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// CategoryGroup resolves query against group IDs, then names
// ignoring case, hidden groups included.
func CategoryGroup(query string, groups []ynab.CategoryGroup) (ynab.CategoryGroup, error) {
	candidates := make([]names.Candidate, len(groups))
	for i, group := range groups {
		candidates[i] = names.Candidate{ID: group.ID, Names: []string{group.Name}}
	}

	index, err := names.Resolve("category group", query, candidates)
	if err != nil {
		return ynab.CategoryGroup{}, err
	}
	return groups[index], nil
}

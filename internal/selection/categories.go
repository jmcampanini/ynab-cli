package selection

import (
	"github.com/jmcampanini/ynab-cli/internal/names"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// Category resolves query against every category, hidden ones
// included, by ID, exact name, or exact "Group: Name". A name shared
// across groups fails listing both qualified forms.
func Category(query string, groups []ynab.CategoryGroup) (ynab.Category, ynab.CategoryGroup, error) {
	type located struct {
		category ynab.Category
		group    ynab.CategoryGroup
	}
	var all []located
	var candidates []names.Candidate
	for _, group := range groups {
		for _, category := range group.Categories {
			all = append(all, located{category: category, group: group})
			candidates = append(candidates, names.Candidate{ID: category.ID, Names: []string{category.Name, group.Name + ": " + category.Name}})
		}
	}

	index, err := names.Resolve("category", query, candidates)
	if err != nil {
		return ynab.Category{}, ynab.CategoryGroup{}, err
	}
	return all[index].category, all[index].group, nil
}

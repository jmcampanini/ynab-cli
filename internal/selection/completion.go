package selection

import (
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// CategoryCompletions returns the name of each shown category, qualified by its
// group when any category, hidden ones included, shares the name.
func CategoryCompletions(groups []ynab.CategoryGroup) []string {
	shared := map[string]int{}
	for _, group := range groups {
		for _, category := range group.Categories {
			shared[strings.ToLower(category.Name)]++
		}
	}
	var names []string
	for _, group := range groups {
		for _, category := range group.Categories {
			switch {
			case category.Internal || category.Hidden || group.Hidden:
				continue
			case shared[strings.ToLower(category.Name)] > 1:
				names = append(names, group.Name+": "+category.Name)
			default:
				names = append(names, category.Name)
			}
		}
	}
	return names
}

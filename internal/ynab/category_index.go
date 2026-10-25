package ynab

// CategoriesByID indexes category amounts returned by a month endpoint.
func CategoriesByID(categories []Category) map[string]Category {
	byID := make(map[string]Category, len(categories))
	for _, category := range categories {
		byID[category.ID] = category
	}
	return byID
}

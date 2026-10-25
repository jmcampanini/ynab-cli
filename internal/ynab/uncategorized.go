package ynab

// TransactionsNeedingCategory removes accounts and transfers that do not
// use categories from the API's uncategorized listing. The plan side of a
// transfer to a tracking account still needs a category.
func TransactionsNeedingCategory(transactions []Transaction, accounts []Account) []Transaction {
	onPlan := make(map[string]bool, len(accounts))
	for _, account := range accounts {
		onPlan[account.ID] = account.OnPlan
	}

	var needed []Transaction
	for _, tx := range transactions {
		if included, known := onPlan[tx.AccountID]; known && !included {
			continue
		}
		if tx.TransferAccountID != nil && onPlan[tx.AccountID] && onPlan[*tx.TransferAccountID] {
			continue
		}
		needed = append(needed, tx)
	}
	return needed
}

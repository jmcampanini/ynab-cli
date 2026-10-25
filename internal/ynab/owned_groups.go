package ynab

// OwnedGroup reports whether the API owns the group: the master
// group, whose categories are internal, and Credit Card Payments. The
// API also marks every group it created from its plan template as
// internal, such as Bills, so the flag alone does not settle it.
func OwnedGroup(group CategoryGroup) bool {
	if group.Internal && group.Name == CreditCardPaymentsGroup {
		return true
	}
	for _, category := range group.Categories {
		if category.Internal {
			return true
		}
	}
	return false
}

// CreditCardPaymentsGroup is the API's internal group whose categories it
// ignores as a transaction's category.
const CreditCardPaymentsGroup = "Credit Card Payments"

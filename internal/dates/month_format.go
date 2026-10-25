package dates

// ShortMonth shortens the API's first-of-month date to YYYY-MM.
func ShortMonth(date string) string {
	if len(date) >= 7 {
		return date[:7]
	}
	return date
}

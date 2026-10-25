package names

import (
	"fmt"
	"unicode/utf8"

	"github.com/jmcampanini/ynab-cli/internal/status"
)

// CheckLength rejects an empty name or one past the API's limit,
// counted in characters, as a usage error before any request.
func CheckLength(name string, limit int) error {
	if name == "" {
		return status.Usage("NAME must not be empty")
	}
	if count := utf8.RuneCountInString(name); count > limit {
		return status.Usage(fmt.Sprintf("NAME is %d characters; the API allows at most %d", count, limit))
	}
	return nil
}

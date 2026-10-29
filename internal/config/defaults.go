package config

import "strconv"

// DefaultMonth selects the current month when a month operand is omitted.
const DefaultMonth = "current"

// DefaultRecordLimit caps a read when neither --limit nor --all is supplied.
const DefaultRecordLimit = 1000

// CommandDefaults declares operational defaults without making them TOML keys.
// Each caller gets its own map so command construction cannot change defaults
// used by a later invocation.
func CommandDefaults() map[string]string {
	return map[string]string{
		"by":    "category",
		"color": "auto",
		"limit": strconv.Itoa(DefaultRecordLimit),
		"month": DefaultMonth,
	}
}

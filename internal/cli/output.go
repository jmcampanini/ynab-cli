package cli

import (
	"github.com/spf13/cobra"
)

// outputFlags selects the record format. Both flags off means the human
// rendering.
type outputFlags struct {
	csv   bool
	jsonl bool
}

// bind adds --jsonl, and --csv when listing, as mutually exclusive flags.
func (f *outputFlags) bind(cmd *cobra.Command, listing bool) {
	cmd.Flags().BoolVar(&f.jsonl, "jsonl", false, "Write one JSON object per line")
	if listing {
		cmd.Flags().BoolVar(&f.csv, "csv", false, "Write CSV with a header row")
		cmd.MarkFlagsMutuallyExclusive("jsonl", "csv")
	}
}

package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/jmcampanini/ynab-cli/internal/outputfile"
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/spf13/cobra"
)

// outputFlags selects the record format and, for reads, collection scope
// and destination. No format flag means the human rendering.
type outputFlags struct {
	csv       bool
	jsonl     bool
	json      bool
	noHeader  bool
	path      string
	overwrite bool
	limit     int
	all       bool
	hasMore   bool
}

// bind adds --jsonl, and --csv when listing, as mutually exclusive flags.
func (f *outputFlags) bind(cmd *cobra.Command, listing bool) {
	cmd.Flags().BoolVar(&f.jsonl, "jsonl", false, "Write one JSON object per line")
	if listing {
		cmd.Flags().BoolVar(&f.csv, "csv", false, "Write CSV with a header row")
		cmd.MarkFlagsMutuallyExclusive("jsonl", "csv")
	}
}

// bindRead adds the record collection contract to a read command. Writes
// keep bind's existing formats and never cap the records they change.
func (f *outputFlags) bindRead(cmd *cobra.Command, listing bool) {
	f.bind(cmd, true)
	cmd.Flags().BoolVar(&f.json, "json", false, "Write one JSON collection")
	cmd.MarkFlagsMutuallyExclusive("json", "jsonl", "csv")
	cmd.Flags().BoolVar(&f.noHeader, "no-header", false, "Omit the CSV header")
	cmd.Flags().StringVar(&f.path, "output-file", "", "Write machine output to this file")
	cmd.Flags().BoolVar(&f.overwrite, "overwrite", false, "Replace an existing output file")
	cmd.Long += "\n\n" + readOutputHelp
	if listing {
		cmd.Flags().IntVar(&f.limit, "limit", 0, "Maximum records to return")
		useCommandDefault(cmd.Flags().Lookup("limit"))
		cmd.Flags().BoolVar(&f.all, "all", false, "Return every matching record")
		cmd.MarkFlagsMutuallyExclusive("limit", "all")
		cmd.Long += "\n\n" + collectionHelp
	}
	runner := cmd.RunE
	cmd.RunE = run(func(cmd *cobra.Command, args []string) error {
		if f.noHeader && !f.csv {
			return errors.New("--no-header requires --csv")
		}
		if f.overwrite && f.path == "" {
			return errors.New("--overwrite requires --output-file")
		}
		if cmd.Flags().Changed("output-file") && f.path == "" {
			return errors.New("--output-file must name a file")
		}
		if f.path != "" && !f.machine() {
			return errors.New("--output-file requires --json, --jsonl, or --csv")
		}
		if listing && f.limit < 0 {
			return errors.New("--limit must be zero or greater")
		}
		f.hasMore = false
		execute := func() error {
			if err := runner(cmd, args); err != nil {
				return err
			}
			if f.hasMore && !f.json {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "more records exist (hasMore=true, nextCursor=null); use --all to return every match")
			}
			return nil
		}
		if f.path == "" {
			return execute()
		}
		return outputfile.Write(f.path, f.overwrite, func(w io.Writer) error {
			out := cmd.OutOrStdout()
			cmd.SetOut(w)
			defer cmd.SetOut(out)
			return execute()
		})
	})
}

func (f outputFlags) machine() bool { return f.json || f.jsonl || f.csv }

// limitRecords caps presentation after every source read and filter has
// completed. It never caps a query used by a mutating command.
func limitRecords[T any](f *outputFlags, records []T) []T {
	if !f.all && len(records) > f.limit {
		f.hasMore = true
		return records[:f.limit]
	}
	return records
}

func writeRecords[T any](out io.Writer, f outputFlags, records []T) error {
	switch {
	case f.json:
		return render.JSON(out, records, f.hasMore)
	case f.csv:
		return render.CSV(out, records, f.noHeader)
	default:
		return render.JSONL(out, records)
	}
}

const readOutputHelp = `Record exports:
  --json writes one object with records, hasMore, nextCursor, and complete.
  --jsonl writes one object per line. --csv writes the same record fields
  with a header; --no-header omits it. Choose only one format. Nested
  objects and arrays are compact JSON in one CSV cell; nulls are empty.
  No matches produce records: [] in JSON, no JSONL bytes, or a CSV header.
  --output-file PATH requires one of these formats and leaves stdout empty.
  Existing files require --overwrite. Failed runs leave the file unchanged.
  A failed source read writes no records in any format.`

const collectionHelp = `Collection scope:
  Return at most 1000 matching records by default. --limit N sets another
  maximum, including zero; --all returns every record matching the command's
  date range and filters. These options never change a write's selection.
  The API returns its matching collection in one response, without a resume
  cursor. A limit is applied after reading and filtering, not at the API.
  Truncation sets hasMore: true and nextCursor: null in JSON, or reports
  those values on stderr for other formats, and still exits 0. Use --all
  to collect every match; --cursor is unavailable. Report summary totals
  describe the full matching collection even when its displayed rows stop
  at the limit.`

package categoryview

// WriteRecord is a category record printed by a mutating
// command, marked when it is a dry-run preview.
type WriteRecord struct {
	Record
	DryRun bool `json:"dry_run,omitempty"`
}

// WriteRecords marks category records as previews when dryRun is true.
func WriteRecords(records []Record, dryRun bool) []WriteRecord {
	marked := make([]WriteRecord, len(records))
	for i, record := range records {
		marked[i] = WriteRecord{Record: record, DryRun: dryRun}
	}
	return marked
}

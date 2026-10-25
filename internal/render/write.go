package render

// WriteSummary is the closing line of a write that is not counted in
// transactions: past and base are the verb's forms, such as "moved" and
// "move", and rest completes the sentence.
func WriteSummary(dryRun bool, past, base, rest string) string {
	if dryRun {
		return "dry run, nothing changed: would " + base + rest
	}
	return past + rest
}

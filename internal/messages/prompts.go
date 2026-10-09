package messages

// The prompt runner's own lines (cmd/prompts): a per-row ✓/✗ line with the flow,
// the summary, and the confirmation of a CSV/SQL write. The words live in the
// locale; the row layout stays in the command.

// PromptRunOk is one matching row: PromptRunOk("classification", "contact:add").
func PromptRunOk(flow, obtained string) string {
	return format("prompts.run.ok", flow, obtained)
}

// PromptRunMismatch is one failing row: expected vs obtained.
func PromptRunMismatch(flow, expected, obtained string) string {
	return format("prompts.run.mismatch", flow, expected, obtained)
}

// PromptRunSummary is the totals line: PromptRunSummary(45, 48, 3).
func PromptRunSummary(ok, total, failures int) string {
	return format("prompts.run.summary", ok, total, failures)
}

// PromptsFileEmpty explains that the -flow filter matched nothing.
func PromptsFileEmpty(flow, path string) string {
	return format("prompts.run.file_empty", flow, path)
}

// PromptCSVWritten confirms a -csv write.
func PromptCSVWritten(path string) string {
	return format("prompts.run.csv_written", path)
}

// PromptSQLWritten confirms a seed-SQL generation.
func PromptSQLWritten(count int, path string) string {
	return format("prompts.run.sql_written", count, path)
}
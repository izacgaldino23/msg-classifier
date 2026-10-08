package messages

// TransactionSaved is the summary once a transaction is persisted.
func TransactionSaved(id uint) string { return format("transaction.saved", id) }

// TransactionSavedNoID is the fallback for a malformed outcome carrying no transaction.
func TransactionSavedNoID() string { return T("transaction.saved_no_id") }

// TransactionFound is the summary listing how many transactions matched.
func TransactionFound(count int, term string) string { return format("transaction.found", count, term) }

// TransactionNotFound is the summary when nothing matched the composed filter.
func TransactionNotFound(term string) string { return format("transaction.not_found", term) }

// TransactionNoData names what the message was missing, so the user appends it and
// sends the same message again.
func TransactionNoData(missing string) string { return format("transaction.no_data", missing) }

// TransactionDuplicate is the summary when the transaction already exists (DC-008).
func TransactionDuplicate() string { return T("transaction.duplicate") }

// The Missing fragments below travel in outcome.Missing and are interpolated into
// TransactionNoData by each surface — they are user text, so they live in the locale,
// but the service decides which one applies.
func MissingMessage() string { return T("transaction.missing_message") }

// MissingAmount is the fragment for a transaction message with no monetary value.
func MissingAmount() string { return T("transaction.missing_amount") }

// MissingFilter is the fragment for a require message with nothing to filter by.
func MissingFilter() string { return T("transaction.missing_filter") }
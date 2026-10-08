package messages

// The REPL's own text: the banner, the confirmation prompt, the error line and the
// field labels it prints in a record block. The words live in the locale; the two-space
// indent and the ":" separator stay in the terminal renderer.

// Banner greets the session when the CLI starts.
func Banner() string { return T("cli.banner") }

// Cancelled confirms a dropped duplicate pending (DC-008).
func Cancelled() string { return T("cli.cancelled") }

// DupOptions are the three answers the confirmation loop accepts.
func DupOptions() string { return T("cli.dup_options") }

// SectionExtraction heads the per-segment trace.
func SectionExtraction() string { return T("cli.extraction") }

// ErrorLine is the one line the REPL prints when a message fails.
func ErrorLine(err error) string { return format("cli.error_line", err) }

// Classification is the status line above the outcome block. The choices stay raw on
// purpose: the terminal is a developer surface, where contact/add is more useful than
// Contato/Adicionar.
func Classification(category, kind string, categoryConfidence, kindConfidence float64) string {
	return format("cli.classification", category, kind, categoryConfidence*100, kindConfidence*100)
}

// Field is the label printed next to a value: Field("name"), Field("party")…
func Field(name string) string { return T("cli.field." + name) }
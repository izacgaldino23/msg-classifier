package messages

// Label returns the PT-BR label for a classification or sub-type choice, falling back
// to the input unchanged when it is not mapped. It takes any so a nil lookup (missing
// classification) renders "" instead of erroring.
func Label(choice any) string {
	s, ok := choice.(string)
	if !ok {
		return ""
	}
	// An unmapped choice renders itself, not its key — a badge showing "label.whatever"
	// is worse than one showing the raw choice.
	if label, ok := texts["label."+s]; ok {
		return label
	}
	return s
}
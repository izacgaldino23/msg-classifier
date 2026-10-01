package services

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"
)

// jevRequester is the minimal Jev seam for the extractor (dynamic questions).
type jevRequester interface {
	MakeJevRequest(request *jev.JevRequest) (*jev.JevResponse, error)
}

// Span locates an extracted value within the original message.
type Span struct {
	Start int
	End   int
}

// NameResult carries the extracted name and the per-segment trace for the UI.
type NameResult struct {
	Name     string
	Segments []models.SegmentScore
}

// ContactExtractor pulls phone/email via regex and the name via a Jev Noul fan-out.
type ContactExtractor struct {
	jev jevRequester
}

func NewContactExtractor(client jevRequester) *ContactExtractor {
	return &ContactExtractor{jev: client}
}

// phonePattern matches a BR phone: optional +55, optional trunk 0, DDD 11-99,
// mobile (9 + 8 digits) or landline (8 digits), tolerant of parens/spaces/dashes/dots.
var phonePattern = regexp.MustCompile(`(?:\+?55[\s.-]?)?0?[\s.-]?(?:\(?[1-9][0-9]\)?[\s.-]?)?(?:9[\s.-]?[0-9]{4}[\s.-]?[0-9]{4}|[1-9][0-9]{3}[\s.-]?[0-9]{4})`)

var emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

// trimPunctuation is stripped from the ends of included name segments; periods
// stay ("Jr." is a legitimate name ending).
const trimPunctuation = `"'),;:!?-—`

// ExtractPhone returns the first valid BR phone normalized to digits (10/11) and its span.
func (e *ContactExtractor) ExtractPhone(message string) (string, Span, bool) {
	for _, loc := range phonePattern.FindAllStringIndex(message, -1) {
		digits := normalizePhone(message[loc[0]:loc[1]])
		if digits == "" {
			continue
		}
		return digits, Span{Start: loc[0], End: loc[1]}, true
	}
	return "", Span{}, false
}

// ExtractEmail returns the first email match and its span.
func (e *ContactExtractor) ExtractEmail(message string) (string, Span, bool) {
	loc := emailPattern.FindStringIndex(message)
	if loc == nil {
		return "", Span{}, false
	}
	return message[loc[0]:loc[1]], Span{Start: loc[0], End: loc[1]}, true
}

// ExtractName removes the given spans, splits the remainder into whitespace
// segments, and asks Jev one Noul question per segment; segments with noul > 0.5
// are joined in order (punctuation trimmed from segment ends, periods preserved) and kept in the trace (single threshold site).
func (e *ContactExtractor) ExtractName(message string, spans []Span) (NameResult, error) {
	segments := strings.Fields(removeSpans(message, spans))
	if len(segments) == 0 {
		return NameResult{}, nil
	}

	questions := make(map[string]jev.JevQuestionInterface, len(segments))
	for i := range segments {
		questions[fmt.Sprintf("segment_%d", i)] = &jev.JevQuestionNoul{
			JevQuestion: jev.JevQuestion{
				Type:         jev.NoulQuestionType,
				Instructions: fmt.Sprintf("Is `segments[%d]` part of the person's name or reference in `message`?", i),
			},
			Criteria: jev.JevNoulCriteria{
				True:  "the segment is part of the person's name or reference",
				False: "the segment is a verb, article, preposition or other non-name word (e.g. 'salva', 'o', 'contato', 'do', 'telefone')",
			},
		}
	}

	resp, err := e.jev.MakeJevRequest(&jev.JevRequest{
		State: jev.JevState(map[string]any{
			"message":  message,
			"segments": segments,
		}),
		Questions: questions,
	})
	if err != nil {
		return NameResult{}, fmt.Errorf("%w: failed to extract name via jev: %w", ErrUpstream, err)
	}

	trace := make([]models.SegmentScore, 0, len(segments))
	for i, segment := range segments {
		answer, err := answerAsNoul(resp, fmt.Sprintf("segment_%d", i))
		if err != nil {
			return NameResult{}, fmt.Errorf("%w: %w", ErrUpstream, err)
		}
		trace = append(trace, models.SegmentScore{Text: segment, Score: answer.Noul, Included: answer.Noul > 0.5})
	}

	// Deterministic post-filter over Jev's per-segment verdicts. The Noul scores
	// for name particles ("do"/"da"/"de") hover around the 0.5 threshold and flip
	// between runs, so the AI verdict alone is not reproducible. Particles are
	// resolved by position, which is deterministic:
	//   1. Drop leading lowercase segments while a capitalized one follows - a
	//      PT-BR proper name never starts with a function word, so an included
	//      prefix like "do João da Silva" must become "João da Silva".
	//   2. Rescue a particle Jev excluded when it sits between two included
	//      capitalized segments - "José Carlos de Souza" keeps its "de", which is
	//      a surname particle in the middle even though the same word is noise at
	//      the start.
	firstCap := -1
	for i := range trace {
		if trace[i].Included && isCapitalized(trimSegment(trace[i].Text)) {
			firstCap = i
			break
		}
	}
	if firstCap > 0 {
		for i := 0; i < firstCap; i++ {
			trace[i].Included = false
		}
	}

	for i := 1; i+1 < len(trace); i++ {
		if trace[i].Included || !isNameParticle(trace[i].Text) {
			continue
		}
		if isIncludedCapitalized(trace, i-1) && isIncludedCapitalized(trace, i+1) {
			trace[i].Included = true
		}
	}

	nameParts := make([]string, 0, len(trace))
	for _, seg := range trace {
		if seg.Included {
			if trimmed := trimSegment(seg.Text); trimmed != "" {
				nameParts = append(nameParts, trimmed)
			}
		}
	}

	return NameResult{Name: strings.TrimSpace(strings.Join(nameParts, " ")), Segments: trace}, nil
}

// nameParticles are the PT-BR particles valid inside a surname. They are noise
// at the start of a name but legitimate between name words.
var nameParticles = map[string]bool{
	"de": true, "da": true, "do": true, "dos": true, "das": true,
}

// isNameParticle reports whether a segment is a PT-BR surname particle (ignoring
// surrounding punctuation and case).
func isNameParticle(segment string) bool {
	return nameParticles[strings.ToLower(trimSegment(segment))]
}

// isIncludedCapitalized reports whether trace[j] is an included capitalized segment.
func isIncludedCapitalized(trace []models.SegmentScore, j int) bool {
	return j >= 0 && j < len(trace) && trace[j].Included && isCapitalized(trimSegment(trace[j].Text))
}

// trimSegment strips the punctuation that can surround a segment.
func trimSegment(segment string) string {
	return strings.Trim(segment, trimPunctuation)
}

// isCapitalized reports whether s starts with an uppercase letter.
func isCapitalized(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

// removeSpans deletes the given spans from the message, preserving order.
func removeSpans(message string, spans []Span) string {
	if len(spans) == 0 {
		return message
	}

	sorted := append([]Span(nil), spans...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })

	var b strings.Builder
	last := 0
	for _, span := range sorted {
		if span.Start < last {
			continue
		}
		b.WriteString(message[last:span.Start])
		last = span.End
	}
	b.WriteString(message[last:])
	return b.String()
}

// normalizePhone strips formatting, drops optional country/trunk prefixes, and
// validates the result as a 10 (landline) or 11 (mobile) digit BR number.
func normalizePhone(raw string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, raw)

	for len(digits) > 11 && strings.HasPrefix(digits, "55") {
		digits = digits[2:]
	}
	for len(digits) > 10 && strings.HasPrefix(digits, "0") {
		digits = digits[1:]
	}

	if len(digits) != 10 && len(digits) != 11 {
		return ""
	}
	if digits[0] < '1' || digits[0] > '9' || digits[1] < '1' || digits[1] > '9' {
		return ""
	}
	if len(digits) == 11 && digits[2] != '9' {
		return ""
	}
	return digits
}

// answerAsNoul extracts a noul answer with a checked assertion (mirrors answerAsChoice).
func answerAsNoul(resp *jev.JevResponse, key string) (*jev.JevAnswerNoul, error) {
	answer, ok := resp.Answers[key]
	if !ok {
		return nil, fmt.Errorf("missing answer %q in jev response", key)
	}
	noul, ok := answer.(*jev.JevAnswerNoul)
	if !ok {
		return nil, fmt.Errorf("answer %q has type %T, want *jev.JevAnswerNoul", key, answer)
	}
	return noul, nil
}

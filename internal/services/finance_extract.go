package services

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"
)

// typeAnswerKey is the answer key of the transaction_type question.
const typeAnswerKey = "transaction_type"

// transactionTypeCriteria distinguishes a purchase (money spent on something) from
// settling a debt (money that leaves for a bill, a rent, a loan) and from money
// simply moving between accounts. The criteria say what the money IS, not how it
// was paid, so "paguei o aluguel no pix" is a payment and not a transfer.
var transactionTypeCriteria = map[string]string{
	models.TransactionTypePurchase: "the user spent money getting something — buying goods or food, or paying a shop for a product ('comprei arroz no supermercado', '50 reais de padaria')",
	models.TransactionTypeSale:     "the user received money for selling something they owned ('vendi o notebook por 800 reais')",
	models.TransactionTypePayment:  "the user paid to settle a debt, a bill or a service — rent, electricity, a loan installment, a subscription ('paguei o aluguel', 'paguei a fatura do cartão')",
	models.TransactionTypeReceipt:  "the user received money from a third party to settle what they were owed — a refund, a loan, a debt ('recebi o reembolso', 'o Fulano me pagou o que eu emprestei')",
	models.TransactionTypeTransfer: "money only moved between accounts, with no purchase and no debt settled ('fiz uma transferência para a conta', 'mandei um pix para outro banco')",
}

// partyCandidatePattern matches the words a PT-BR preposition introduces when the
// user names an establishment or a person: "no supermercado", "de Fulano", "no
// cartão". It is word-bounded and grabs at most partyWindowWords words, so each
// preposition is its own match instead of the first one eating the whole clause.
var partyCandidatePattern = regexp.MustCompile(`(?i)\b(?:no|na|nos|nas|ao|à|aos|às|do|da|dos|das|de|em|com|pra|pro|pelo|pela)\s+([\p{L}][\p{L}'’-]*(?:\s+[\p{L}][\p{L}'’-]*){0,2})`)

// partyStops end a candidate: past them the text is a new thought, not the same
// noun phrase ("na padaria e depois no mercado" is the padaria *and* the mercado).
// The genitives (do, da, de) are absent on purpose: "padaria do bairro" is one name.
var partyStops = map[string]bool{
	"e": true, "mas": true, "depois": true, "entao": true, "ai": true, "por": true,
	"em": true, "com": true, "no": true, "na": true, "nos": true, "nas": true,
	"hoje": true, "ontem": true, "amanha": true, "dia": true, "dias": true,
	"semana": true, "semanas": true, "mes": true, "meses": true, "ano": true, "anos": true,
	"passada": true, "passou": true, "proxima": true, "proximo": true, "essa": true, "esse": true,
}

// partyJunk are words that follow a preposition without naming a place or a
// person. "de 50 reais" is an amount, not a party.
var partyJunk = map[string]bool{
	"conta": true, "media": true, "real": true, "reais": true,
	"total": true, "valor": true, "valores": true,
}

// partySegmentQuestion asks whether each segment of the candidate names the shop
// or the person on the other side.
var partySegmentQuestion = segmentQuestion{
	prompt:        "Is `segments[%d]` part of the name of the establishment or of the person involved in the transaction in `message`?",
	trueCriteria:  "the segment names the shop, the establishment or the person involved ('supermercado', 'padaria', 'farmácia', 'Fulano')",
	falseCriteria: "the segment is a verb, an article, a preposition, an amount, or an abstract noun ('comprei', 'no', '50', 'reais', 'valor', 'conta')",
}

// partyCandidate is the span after a preposition: the raw text for the fallback
// and its words for the fan-out.
type partyCandidate struct {
	Raw      string
	Segments []string
}

// FinanceResult carries the extracted transaction type, the establishment or
// person involved, and the per-segment trace of the party fan-out.
type FinanceResult struct {
	Type     string
	Party    string
	Segments []models.SegmentScore
}

// FinanceExtractor asks Jev for the transaction type and, in the same request,
// which words of the establishment candidate are actually the shop or the person.
type FinanceExtractor struct {
	jev jevRequester
}

func NewFinanceExtractor(client jevRequester) *FinanceExtractor {
	return &FinanceExtractor{jev: client}
}

// ponytail: the party is a regex-anchored guess refined by one Jev fan-out, not a
// parse of the sentence. A candidate with no preposition ("padaria, 30 reais") yields
// no party at all, and a preposition that introduces something else ("paguei o valor
// de 50 reais") is filtered by a junk wordlist. A span-aware parse, or asking Jev to
// return the party as a quoted substring, is the way past it.

// Extract returns the transaction type and the establishment or person involved. An
// unknown type comes back as "compra" (the overwhelmingly common case) rather than
// as an error; a missing party is simply empty, since the transaction is still valid
// without it.
func (e *FinanceExtractor) Extract(message string) (FinanceResult, error) {
	candidate := partyCandidateFrom(message)
	extra := map[string]jev.JevQuestionInterface{typeAnswerKey: typeQuestion()}

	resp, trace, err := noulSegments(e.jev, message, candidate.Segments, partySegmentQuestion, extra)
	if err != nil && len(candidate.Segments) > 0 {
		// The party fan-out rides along with the type question, but a request mixing
		// both question kinds could be rejected. The type is what matters, so retry
		// without the fan-out and carry on without a party.
		resp, trace, err = noulSegments(e.jev, message, nil, partySegmentQuestion, extra)
	}
	if err != nil {
		return FinanceResult{}, err
	}

	answer, err := answerAsChoice(resp, typeAnswerKey)
	if err != nil {
		return FinanceResult{}, fmt.Errorf("%w: %w", ErrUpstream, err)
	}

	transactionType := answer.Choice
	if !models.IsTransactionType(transactionType) {
		transactionType = models.TransactionTypePurchase
	}

	party := joinIncluded(trace)
	if party == "" && len(trace) > 0 {
		// The fan-out ran and kept nothing: fall back to the candidate as it was
		// written. Without a trace there was no fan-out (no candidate, or the
		// degraded retry), so there is no raw candidate to fall back to.
		party = candidate.Raw
	}
	return FinanceResult{Type: transactionType, Party: party, Segments: trace}, nil
}

// typeQuestion asks which side of a financial movement the message describes.
func typeQuestion() jev.JevQuestionInterface {
	criteria := make(map[string]string, len(transactionTypeCriteria))
	for name, description := range transactionTypeCriteria {
		criteria[name] = description
	}
	return &jev.JevQuestionChoice{
		JevQuestion: jev.JevQuestion{
			Type:         jev.ChoiceQuestionType,
			Instructions: "Which type of transaction does the message in `message` describe?",
		},
		Criteria: criteria,
	}
}

// partyWindowWords bounds the candidate: a shop or person name is a short noun
// phrase, and without the bound "no mercado arroz, feijão" swallows the whole list.
const partyWindowWords = 3

// partyCandidateFrom returns the best establishment candidate in the message. The
// longest candidate wins, because a qualifier drags a longer noun phrase along
// ("na padaria do bairro" is the padaria, not the bairro); on a tie the last
// mention does, which is the one the user is talking about.
func partyCandidateFrom(message string) partyCandidate {
	best := partyCandidate{}
	for _, match := range partyCandidatePattern.FindAllStringSubmatch(message, -1) {
		candidate, ok := partyWindow(match[1])
		if ok && len(candidate.Segments) >= len(best.Segments) {
			best = candidate
		}
	}
	return best
}

// partyWindow keeps the words of a candidate up to the bound or the first stop word.
// A word carrying a digit disqualifies the span ("de 50 reais" is an amount), as does
// a span left with nothing but junk words.
func partyWindow(span string) (partyCandidate, bool) {
	candidate := partyCandidate{}
	for _, word := range strings.Fields(span) {
		if len(candidate.Segments) == partyWindowWords {
			break
		}
		trimmed := trimSegment(word)
		if trimmed == "" {
			continue
		}
		normalized := strings.ToLower(normalizeName(trimmed))
		if partyStops[normalized] {
			break
		}
		if containsDigit(trimmed) {
			return partyCandidate{}, false
		}
		candidate.Segments = append(candidate.Segments, trimmed)
	}
	if isJunkParty(candidate.Segments) {
		return partyCandidate{}, false
	}
	candidate.Raw = strings.Join(candidate.Segments, " ")
	return candidate, true
}

func containsDigit(text string) bool {
	return strings.ContainsFunc(text, unicode.IsDigit)
}

// isJunkParty reports whether every segment is a junk word (and an empty candidate
// counts as junk, so it never reaches the fan-out).
func isJunkParty(segments []string) bool {
	for _, segment := range segments {
		if !partyJunk[strings.ToLower(normalizeName(segment))] {
			return false
		}
	}
	return true
}

// joinIncluded joins the segments Jev kept, falling back to "" when it kept none.
func joinIncluded(trace []models.SegmentScore) string {
	parts := make([]string, 0, len(trace))
	for _, segment := range trace {
		if segment.Included {
			if trimmed := trimSegment(segment.Text); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
	}
	return strings.Join(parts, " ")
}

// Package jevq holds the Jev call helpers shared by every classification flow:
// the two client seams, the checked answer readers and the upstream error
// sentinel. It exists because the contact, notes and finance flows all build
// requests and read answers the same way, and the root services package depends
// on all three — so this layer has to sit below them.
package jevq

import (
	"errors"
	"fmt"

	"msg-classifier/pkg/jev"
)

// ErrUpstream marks Jev/TypeSafe API failures; controllers map it to HTTP 502.
var ErrUpstream = errors.New("upstream classification failed")

// Client is the template-file seam (classification.json, note.json), allowing
// unit tests without HTTP.
type Client interface {
	MakeJevRequestFromFile(state jev.JevState, fileName string) (*jev.JevResponse, error)
}

// Requester is the dynamic-request seam (a request built in code), used by the
// extractors that fan out one question per segment.
type Requester interface {
	MakeJevRequest(request *jev.JevRequest) (*jev.JevResponse, error)
}

// AnswerChoice extracts a choice answer with a checked assertion.
func AnswerChoice(resp *jev.JevResponse, key string) (*jev.JevAnswerChoice, error) {
	answer, ok := resp.Answers[key]
	if !ok {
		return nil, fmt.Errorf("missing answer %q in jev response", key)
	}
	choice, ok := answer.(*jev.JevAnswerChoice)
	if !ok {
		return nil, fmt.Errorf("answer %q has type %T, want *jev.JevAnswerChoice", key, answer)
	}
	return choice, nil
}

// AnswerNoul extracts a noul answer with a checked assertion (mirrors AnswerChoice).
func AnswerNoul(resp *jev.JevResponse, key string) (*jev.JevAnswerNoul, error) {
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
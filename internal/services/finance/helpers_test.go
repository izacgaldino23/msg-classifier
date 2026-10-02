package finance

import (
	"errors"

	"msg-classifier/pkg/jev"
)

// mockJevRequester fakes the dynamic-request seam the extractor uses.
type mockJevRequester struct {
	resp  *jev.JevResponse
	err   error
	got   *jev.JevRequest
	calls int
	// failFirst rejects the first call and serves resp to the next one, which is how
	// the finance extractor degrades to a type-only request.
	failFirst bool
}

func (m *mockJevRequester) MakeJevRequest(request *jev.JevRequest) (*jev.JevResponse, error) {
	m.got = request
	m.calls++
	if m.failFirst && m.calls == 1 {
		return nil, errors.New("jev unreachable")
	}
	return m.resp, m.err
}
package messages

// The error texts for the boundary. Services keep their English diagnostics for logs
// and error chains; a surface that recognizes the sentinel swaps in the text below,
// and an error it does not recognize is passed through unchanged.

// BadRequest is a malformed or unbindable request body.
func BadRequest() string { return T("error.bad_request") }

// BadKind is an unknown record kind on a data-screen route.
func BadKind() string { return T("error.bad_kind") }

// BadID is an unparseable or non-positive record id.
func BadID() string { return T("error.bad_id") }

// MissingFlow is a /prompts request without the flow parameter.
func MissingFlow() string { return T("error.missing_flow") }

// MessageRequired is an empty message on the classification route.
func MessageRequired() string { return T("error.message_required") }

// NotFound is repository.ErrNotFound reaching the surface.
func NotFound() string { return T("error.not_found") }

// InvalidFilter is services.ErrInvalidFilter reaching the surface.
func InvalidFilter() string { return T("error.invalid_filter") }

// InvalidData is services.ErrInvalidData reaching the surface.
func InvalidData() string { return T("error.invalid_data") }

// InvalidPrompt is services.ErrInvalidPrompt reaching the surface.
func InvalidPrompt() string { return T("error.invalid_prompt") }

// UpstreamUnavailable is jevq.ErrUpstream reaching the surface.
func UpstreamUnavailable() string { return T("error.upstream_unavailable") }
# CODE_STYLE

Conventions observed in this codebase. Follow these when writing new code.

## Naming Conventions

| Item | Convention | Examples |
|---|---|---|
| Files & directories | `snake_case` | `msg_handler.go`, `request_kind.json`, `web/templates/partial/` |
| Go packages | Single lowercase word | `handlers`, `config`, `jev`, `pkg` |
| Exported types | PascalCase, domain prefix | `JevRequest`, `JevAnswerChoice`, `MsgHandler`, `ReceiveMessageRequest` |
| Exported functions | PascalCase, `New*` constructors | `NewMsgHandler()`, `NewWebHandler()`, `MakeJevRequest()`, `GetEnv()` |
| Unexported functions | camelCase | `fromJevResponse()`, `checkMessageCategory()`, `validateJevRequest()` |
| Constants | Exported PascalCase, grouped in `const` blocks | `ChoiceQuestionType`, `ScoreQuestionType`, `SourcePath` |
| Variables | Short, lowercase, idiomatic Go | `c` (gin.Context), `h` (handler), `router`, `tmpl`, `jevResponse` |
| Struct fields | PascalCase with JSON tags | `UserID string \`json:"user_id" form:"user_id"\`` |
| JSON / form tags | `snake_case` | `json:"message"`, `form:"user_id"`, `json:"criteria"` |
| HTTP routes | lowercase | `/`, `/api/message` |
| Template names | lowercase, `:`-namespaced blocks | `base`, `page:title`, `page:content`, `resultado` |
| Env variables | `SCREAMING_SNAKE_CASE` | `TYPESAFE_API_URL`, `TYPESAFE_MODEL`, `TS_API_KEY` |
| Error strings | lowercase, wrapped with `%w` | `"failed to decode question %q in %q: %w"` |

## File Organization

- **`cmd/`** — executable entry points only (`cmd/api/main.go`). Thin bootstrap: router, middleware, route registration.
- **`internal/`** — private application code: `config/` (env singleton), `handlers/` (HTTP handlers).
- **`pkg/`** — reusable packages: `pkg/request.go` (response helper), `pkg/jev/` (API client + `requests/` JSON templates).
- **`web/templates/`** — HTML templates split into `layouts/`, `pages/`, `partial/`.
- Handlers are structs with `New*` constructors; methods take `c *gin.Context`.
- Request/response DTOs are grouped in a `type (...)` block at the top of the handler file.

## Import Style

- Standard library first, then third-party, then internal module imports, separated by blank lines:

```go
import (
	"fmt"
	"msg-classifier/pkg"
	"msg-classifier/pkg/jev"
	"net/http"

	"github.com/gin-gonic/gin"
)
```

- Internal packages imported via module path (`msg-classifier/internal/...`, `msg-classifier/pkg/...`), not relative paths.

## Code Patterns

### Handlers
```go
type MsgHandler struct{}

func NewMsgHandler() *MsgHandler {
	return &MsgHandler{}
}

func (h *MsgHandler) ReceiveMessage(c *gin.Context) {
	// bind → validate → call helpers → respond
}
```

### JSON responses
Use `pkg.ReturnJson(c, status, body)` — wraps 2xx in `{"data": ...}`, everything else in `{"error": ...}`.

### Jev domain types
All TypeSafe-related types are prefixed `Jev` and implement the `JevQuestionInterface` / `JevAnswer` interfaces with `GetType()` / `GetInstructions()` methods.

### Config access
Never read `os.Getenv` directly outside `internal/config/env.go` — use `config.GetEnv().TypesafeModel` etc.

### Template rendering
- Full page: `c.HTML(http.StatusOK, "base", nil)`
- HTMX partial: `c.HTML(http.StatusOK, "page:content", nil)` or `c.HTML(http.StatusOK, "resultado", data)`

## Error Handling

- Return errors up the stack; handlers convert them to HTTP responses via `pkg.ReturnJson`.
- Wrap errors with context: `fmt.Errorf("failed to decode question %q in %q: %w", name, finalPath, err)`.
- Error strings should be **lowercase** (Go convention).
- Known deviations to avoid in new code:
  - Capitalized error strings in `pkg/jev/jev.go` (`"Failed to marshal..."`, `"Failed to create request..."`, etc.) — do not repeat this pattern.
  - `panic(err)` on JSON marshal failure in `HttpResponseToJevResponse` (`jev.go:113`) — prefer returning the error.
  - Unchecked type assertions (`responseMap["model"].(string)`, `Answers["classification"].(*jev.JevAnswerChoice)`) — these panic on malformed data.

## Logging

- `log` package only (`log.Fatalf` in `init()`, `log.Printf` for API errors).
- No structured logging, no log levels, no logger abstraction.

## Testing

- **No tests exist yet.** Go convention applies: `*_test.go` files alongside source, `func TestXxx(t *testing.T)`.
- No test framework or mocking library is configured.

## Do's and Don'ts

**Do:**
- Use `snake_case` for files, `PascalCase` for exports, `camelCase` for unexported functions.
- Prefix TypeSafe domain types with `Jev`.
- Use `New*` constructors for handlers.
- Use `pkg.ReturnJson` for API responses.
- Access config through `config.GetEnv()`.
- Wrap errors with `%w` and lowercase messages.
- Add JSON tags (`snake_case`) to all struct fields that cross the wire.
- Use `omitempty` on optional JSON fields.

**Don't:**
- Don't capitalize error strings.
- Don't `panic` on parse/marshal failures — return errors.
- Don't use unchecked type assertions on API responses without guarding.
- Don't read env vars directly in handlers/packages — go through `internal/config`.
- Don't add new direct dependencies without updating `go.mod` (module is `msg-classifier`, Go 1.25.4).
- Don't commit secrets to `local.env` (e.g., `TS_API_KEY`).
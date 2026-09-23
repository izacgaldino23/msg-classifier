# CODE_STYLE

Conventions observed in this codebase. Follow these when writing new code.

## Naming Conventions

| Item | Convention | Examples |
|---|---|---|
| Files & directories | `snake_case` | `message_controller.go`, `classification.json`, `web/templates/partial/` |
| Go packages | Single lowercase word | `controllers`, `services`, `models`, `views`, `config`, `jev` |
| Exported types | PascalCase, domain prefix | `JevRequest`, `JevAnswerChoice`, `MessageController`, `ClassificationService`, `ReceiveMessageRequest` |
| Exported functions | PascalCase, `New*` constructors | `NewMessageController()`, `NewClassificationService()`, `NewClient()`, `GetEnv()` |
| Unexported functions | camelCase | `answerAsChoice()`, `makeRequest()`, `isHxRequest()`, `validateJevRequest()` |
| Constants | Exported PascalCase, grouped in `const` blocks | `ChoiceQuestionType`, `ScoreQuestionType`, `BaseTemplate`, `SourcePath` |
| Variables | Short, lowercase, idiomatic Go | `c` (gin.Context), `ctrl` (controller), `router`, `tmpl`, `env` |
| Struct fields | PascalCase with JSON tags | `UserID string \`json:"user_id" form:"user_id"\`` |
| JSON / form tags | `snake_case` | `json:"message"`, `form:"user_id"`, `json:"criteria"` |
| HTTP routes | lowercase | `/`, `/api/message` |
| Template names | lowercase, `:`-namespaced blocks | `base`, `page:title`, `page:content`, `resultado`, `error` |
| Env variables | `SCREAMING_SNAKE_CASE` | `TYPESAFE_API_URL`, `TYPESAFE_MODEL`, `TS_API_KEY` |
| Error strings | lowercase, wrapped with `%w` | `"failed to decode question %q in %q: %w"` |

## File Organization

- **`cmd/`** — executable entry points only (`cmd/api/main.go`). Composition root: dependency wiring + route registration.
- **`internal/`** — private application code, layered MVC:
  - `config/` — env singleton (single godotenv load site)
  - `controllers/` — HTTP concerns only (bind → service → render → status)
  - `models/` — DTOs and domain structs
  - `repository/` — persistence layer; structs with `New*` constructors holding `*gorm.DB`; methods return raw gorm errors; package exposes its own not-found sentinel (`ErrNotFound = gorm.ErrRecordNotFound`)
  - `services/` — business rules and orchestration
  - `views/` — template name constants + render helpers
- **`pkg/`** — reusable packages: `pkg/request.go` (response helper), `pkg/jev/` (API client + embedded `requests/` JSON templates).
- **`web/templates/`** — HTML templates split into `layouts/`, `pages/`, `partial/`.
- **`scripts/sql/`** — SQL seed files (`seed_prompts.sql` wipes and re-seeds `jev_prompts`).
- Controllers and services are structs with `New*` constructors; controller methods take `c *gin.Context`.
- DTOs live in `internal/models`, not in controller files.

## Import Style

- Standard library first, then internal module imports, then third-party, separated by blank lines:

```go
import (
	"errors"
	"fmt"

	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"

	"github.com/gin-gonic/gin"
)
```

- Internal packages imported via module path (`msg-classifier/internal/...`, `msg-classifier/pkg/...`), not relative paths.

## Code Patterns

### Controllers
```go
type MessageController struct {
	classifier *services.ClassificationService
}

func NewMessageController(classifier *services.ClassificationService) *MessageController {
	return &MessageController{classifier: classifier}
}

func (ctrl *MessageController) ReceiveMessage(c *gin.Context) {
	request := &models.ReceiveMessageRequest{}
	if err := c.Bind(request); err != nil {
		views.RenderError(c, http.StatusBadRequest, "invalid request")
		return
	}
	// call service → map errors → render via views
}
```

### Services
- Business logic lives in `internal/services`, never in controllers.
- Dependencies are injected via constructor; define a small interface at the service boundary for testability:

```go
type jevClient interface {
	MakeJevRequestFromFile(state jev.JevState, fileName string) (*jev.JevResponse, error)
}
```

- Use sentinel errors (e.g., `ErrUpstream`) wrapped with `%w` so controllers can map statuses with `errors.Is`.

### Category dispatch
- Category-specific use cases implement the `CategoryHandler` interface: `Handle(request, classification) (*models.UseCaseOutcome, error)`.
- A `Dispatcher` holds a `map[string]CategoryHandler` keyed by `Category.Choice`; misses fall back to an `ActionNone` outcome.
- Adding a category = new service implementing `CategoryHandler` + one wiring line in `main.go` — no dispatcher edits.
- Use cases return a `models.UseCaseOutcome` (`Classification` + `Action`); actions: `ActionNone`, `ActionContactAdd`, `ActionContactNoData`, `ActionContactFound`, `ActionContactNotFound`, `ActionContactDuplicate`. Future use-case results (extracted data, DB confirmation) flow back through the outcome without signature changes.

### Views
- Template names are constants in `internal/views/render.go` — never string literals at call sites.
- Render through `views.RenderPage` / `views.RenderResult` / `views.RenderError`, not raw `c.HTML`.
- Template name constants in `internal/views/render.go` include the harness partials: `prompt_table`, `evaluation_results`, `export_result`.

### Validation harness (/prompts)
- `PromptService` reuses the production Jev paths (`ClassificationService.Classify`, `ContactExtractor.ExtractName`) — no new request-building code.
- Per-prompt Jev failures are captured in the row (obtained = error string, match = false); evaluation continues.
- CSV exports go to `exports/` (created on demand) via `encoding/csv` (stdlib).
- Form DTOs (`PromptForm`, `EvaluateForm`) live in `internal/models`; `[]uint` form slices bind repeated `ids` values.

### JSON responses
Use `pkg.ReturnJson(c, status, body)` — wraps 2xx in `{"data": ...}`, everything else in `{"error": ...}`. Currently unused by htmx routes (they render HTML partials); kept for future JSON endpoints.

### Jev domain types
All TypeSafe-related types are prefixed `Jev` and implement the `JevQuestionInterface` / `JevAnswer` interfaces with `GetType()` / `GetInstructions()` methods.

### Config access
Never read `os.Getenv` directly outside `internal/config/env.go`. Config is read once at the composition root (`cmd/api/main.go`) and injected into constructors (`jev.NewClient(env.TypesafeApiUrl, env.TypesafeToken, env.TypesafeModel)`).

## Error Handling

- Return errors up the stack; controllers convert them to HTTP responses.
- Wrap errors with context: `fmt.Errorf("failed to decode question %q in %q: %w", name, fileName, err)`.
- Error strings should be **lowercase** (Go convention).
- Use sentinel errors for status mapping: `errors.Is(err, services.ErrUpstream)` → 502, anything else → 500, bind failure → 400.
- **Never panic in the request path.** Check every type assertion (`value, ok := ...`); missing or mistyped data becomes a descriptive error.
- All responses to htmx targets are HTML partials (result or error) — do not mix JSON errors into htmx routes.

## Logging

- `log` package only (`log.Printf` for API errors).
- No structured logging, no log levels, no logger abstraction.

## Testing

- Table-driven tests using **testify** (`assert`/`require`).
- Hand-written fakes for one-method Jev seams (`mockJevRequester`, `mockJevClient`).
- Real in-memory SQLite (`newTestDB`) for repository/service integration tests.
- Assert on sentinels with `errors.Is` (`ErrUpstream`, `repository.ErrNotFound`).

## Do's and Don'ts

**Do:**
- Use `snake_case` for files, `PascalCase` for exports, `camelCase` for unexported functions.
- Prefix TypeSafe domain types with `Jev`.
- Use `New*` constructors for controllers and services.
- Keep controllers thin: bind → service → render. No business logic, no Jev types, no template literals in controllers.
- Render through `internal/views` helpers.
- Access config through `config.GetEnv()` at the composition root and inject it.
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
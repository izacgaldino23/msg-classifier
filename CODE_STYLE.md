# CODE_STYLE

Conventions observed in this codebase. Follow these when writing new code.

## Naming Conventions

| Item | Convention | Examples |
|---|---|---|
| Files & directories | `snake_case` | `message_controller.go`, `classification.json`, `web/templates/partial/` |
| Go packages | Single lowercase word | `controllers`, `services`, `models`, `views`, `config`, `jev` |
| Exported types | PascalCase, domain prefix | `JevRequest`, `JevAnswerChoice`, `MessageController`, `ClassificationService`, `ReceiveMessageRequest` |
| Answers in `JevResponse` | `map[string]any`, typed values | `*JevAnswerChoice`, `*JevAnswerNoul` |
| Exported functions | PascalCase, `New*` constructors | `NewMessageController()`, `NewClassificationService()`, `NewClient()`, `GetEnv()` |
| Unexported functions | camelCase | `answerAsChoice()`, `isHxRequest()`, `validateJevRequest()` |
| Stateless helpers | package funcs, no struct to inject | `ParseDate()`, `ParseTime()` |
| Constants | Exported PascalCase, grouped in `const` blocks | `ChoiceQuestionType`, `NoulQuestionType`, `BaseTemplate`, `SourcePath` |
| Variables | Short, lowercase, idiomatic Go | `c` (gin.Context), `ctrl` (controller), `router`, `tmpl`, `env` |
| Struct fields | PascalCase with JSON tags | `UserID string \`json:"user_id" form:"user_id"\`` |
| JSON / form tags | `snake_case` | `json:"message"`, `form:"user_id"`, `json:"criteria"` |
| HTTP routes | lowercase | `/`, `/api/message`, `/data/:kind/:id` |
| Template names | lowercase, `:`-namespaced blocks | `base`, `page:title`, `page:content`, `resultado`, `error`, `contacts_table`, `data_detail` |
| Env variables | `SCREAMING_SNAKE_CASE` | `TYPESAFE_API_URL`, `TYPESAFE_MODEL`, `TS_API_KEY` |
| Error strings | lowercase, wrapped with `%w` | `"failed to decode question %q in %q: %w"` |

## File Organization

- **`cmd/`** — executable entry points only (`cmd/api/main.go`). Composition root: dependency wiring + route registration.
- **`internal/`** — private application code, layered MVC:
  - `config/` — env singleton (single godotenv load site)
  - `controllers/` — HTTP concerns only (bind → service → render → status)
  - `models/` — DTOs and domain structs
  - `repository/` — persistence layer; structs with `New*` constructors holding `*gorm.DB`; methods return raw gorm errors; package exposes its own not-found sentinel (`ErrNotFound = gorm.ErrRecordNotFound`)
  - `services/` — business rules and orchestration; a use case that grew past one screen splits like the contact flow did (`contact.go` + `contact_get.go`, `note.go` + `note_get.go`).
  - `views/` — template name constants + render helpers; page-scoped view models (`DataDetailData`) so controllers never hand entities to templates
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
- Use cases return a `models.UseCaseOutcome` (`Classification` + `Action` + the use-case payload: `Contact`/`Segments` for contacts, `Notes []*Note` for notes, `SearchTerm`); actions: `ActionNone`, `ActionContactAdd`, `ActionContactNoData`, `ActionContactFound`, `ActionContactNotFound`, `ActionContactDuplicate`, `ActionNoteAdd`, `ActionNoteNoData`, `ActionNoteFound`, `ActionNoteNotFound`.

### Notes use case
- A second Jev call is allowed per message: the classification call plus one category-specific extraction call (`note.json` for notes). Keep it conditional — the notes add path only asks for the sub-type when the message is actually a note.
- Deterministic text work (dates, times, list splitting, stopword removal) lives in `internal/services` as pure functions with no I/O, so it is testable without mocks; the Jev call is the only mocked seam.
- Dates are normalized to UTC midnight at parse time so the stored value and the query filter compare identically in SQLite; never compare a locally-built `time.Time` against a stored one.
- Normalize text with the existing `normalizeName` before matching PT-BR keywords or stopwords — the lists are written accent-free.
- Repository finders return `([]*models.Note, error)` and map an empty result to the package's `ErrNotFound`; "not found" is an outcome, not an error.
- Persist a note and its to-do items in one transaction (`Create(note, items)`); a partial write is never acceptable.

### Views
- Template names are constants in `internal/views/render.go` — never string literals at call sites.
- Render through `views.RenderPage` / `views.RenderResult` / `views.RenderError`, not raw `c.HTML`.
- Pages render through `views.RenderPage(c, page, content)`; `views.PagesRenderer` clones the shared layout set per page so page blocks never collide. Do not try to collapse this into one template set: Go templates have no inheritance and `{{ template }}` names must be string literals, so the block name cannot come from the view data.
- Template name constants in `internal/views/render.go` include the harness partials: `prompt_table`, `evaluation_results`.
- Template helpers are exposed via `views.FuncMap` (e.g., `label` for PT-BR badge text, `confidence` for the percent format) and registered on the shared template set in `cmd/api/main.go` (`template.New("").Funcs(views.FuncMap)`).
- htmx detection is `c.GetHeader("HX-Request") == "true"` in `views.isHxRequest`. There is no server-side htmx library or context middleware for it.

### Validation harness (/prompts)
- `PromptService` reuses the production Jev paths (`ClassificationService.Classify`, `ContactExtractor.ExtractName`) — no new request-building code.
- Per-prompt Jev failures are captured in the row (obtained = error string, match = false); evaluation continues.
- CSV exports go to `exports/` (created on demand) via `encoding/csv` (stdlib).
- Form DTOs (`PromptForm`, `EvaluateForm`) live in `internal/models`; `[]uint` form slices bind repeated `ids` values.

### Data screen (/data)
- A browse flow gets its own service composed from the existing repositories (`DataService`). Do **not** grow `ContactService`/`NotesService` with browse queries — those are the classification flow and their extraction logic has no meaning for browsing.
- Filter validation belongs to the service, not the repository. The repository falls back to `all` on an unknown filter on purpose so other callers stay safe; `DataService` returns `ErrInvalidFilter` and the controller maps it to 400.
- Register literal path segments **before** a `:param` route (`/data/table`, `/data/item-row` before `/data/:kind/:id`), or gin's router reads them as the parameter value.
- View and edit are one template, not two: toggle `disabled` on the form's `fieldset` and swap the buttons. Two branches drift out of sync.
- Repeat a row of markup with `{{ template "partial_name" . }}` and fetch a blank copy over htmx for "add" — never keep two copies of the same row.
- Nullable columns (`*string`, `*time.Time`) get a `Deref`/`DateBR` template helper; rendering the pointer leaks `0xc000…` into the page.
- Which tab/filter is selected is client-side state (`markFilterPill`). The server re-renders `#data-table` but never the pill list, so an `active` class rendered by the server cannot survive the re-render a delete triggers. Marking one pill `active` in the template is the initial state only — never the mechanism.
- Switching an outer tab must load that pane's default filter (`activateFilterPills`), or the previous pane's table stays on screen looking like the new one. Load it **unconditionally**: Bootstrap toggles the pane's own `active` class, so a `classList.contains('active')` early return skips the load precisely when you re-select the tab that is already showing.
- A pill that both boots a table on first paint and answers clicks carries `hx-trigger="load, click"`. An explicit `hx-trigger` **replaces** htmx's default `click` trigger, so a bare `hx-trigger="load"` turns the element into a dead button once the initial load fires — tabs would never re-load their pane's default filter, and direct clicks would do nothing.

### JSON responses
There are none. Every route renders an HTML partial; add a JSON helper on the day a JSON endpoint exists, not before.

### Jev domain types
All TypeSafe-related types are prefixed `Jev`. Questions implement `JevQuestionInterface` (`GetType()` / `GetInstructions()`), and `GetInstructions` is defined once on the embedded `JevQuestion` — it promotes to every question type, so do not repeat it. Answers are plain structs in `JevResponse.Answers map[string]any`; there is no answer interface.

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
- New to-do/note fixtures must use a fixed calendar date (`10/05/2026`) instead of "hoje"/"amanhã" so assertions never depend on the day the suite runs; relative forms are tested directly on `ParseDate` with an injected `now`.
- `gin.CreateTestContext` leaves `c.Request` nil, and anything that reads a header (`c.GetHeader`) panics on it. Set `c.Request = httptest.NewRequest(...)` in the test helper.
- Source files intentionally lack a trailing newline at EOF, so `gofmt -l` always lists them. Only a real diff counts — never bulk-reformat.
- Render tests assert on output with `strings.Contains`, and assert the absence of what must not leak: a prompt-page fragment in the data page, a pointer address in a table cell, a field that should be hidden.

## Do's and Don'ts

**Do:**
- Use `snake_case` for files, `PascalCase` for exports, `camelCase` for unexported functions.
- Prefix TypeSafe domain types with `Jev`.
- Use `New*` constructors for controllers and services.
- Keep controllers thin: bind → service → render. No business logic, no Jev types, no template literals in controllers.
- Render through `internal/views` helpers.
- Pass a view model from `internal/views` to a template, never an entity straight from the repository.
- Access config through `config.GetEnv()` at the composition root and inject it.
- Wrap errors with `%w` and lowercase messages.
- Add JSON tags (`snake_case`) to all struct fields that cross the wire.
- Use `omitempty` on optional JSON fields.

**Don't:**
- Don't capitalize error strings.
- Don't `panic` on parse/marshal failures — return errors.
- Don't use unchecked type assertions on API responses without guarding.
- Don't read env vars directly in handlers/packages — go through `internal/config`.
- Don't add a dependency for something the stdlib or the platform already does. The one htmx helper we needed was a header comparison; the library went away.
- Don't keep a type, interface or helper with no caller. If nothing uses it, delete it — "kept for a future endpoint" is not a reason.
- Don't add a constructor or a struct field for a value that never changes — a package function is enough.
- Don't add new direct dependencies without updating `go.mod` (module is `msg-classifier`, Go 1.25.4).
- Don't commit secrets to `local.env` (e.g., `TS_API_KEY`).
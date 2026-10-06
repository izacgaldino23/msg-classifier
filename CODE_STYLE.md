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

- **`cmd/`** — executable entry points only (`cmd/web/main.go`, `cmd/api/main.go`). Each registers its own routes; the dependency wiring lives in `internal/app`, and only the web entry point parses templates.
- **`internal/`** — private application code, layered MVC:
  - `api/` — the REST surface: JSON handlers + presenter; reads `models` and `services`, never `views`
  - `app/` — the composition root: config → Jev client → DB → repositories → services → dispatcher; every entry point calls `app.New`
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
- Adding a category = new service implementing `CategoryHandler` + one wiring line in `internal/app/app.go` — no dispatcher edits.
- Use cases return a `models.UseCaseOutcome` (`Classification` + `Action` + the use-case payload: `Contact`/`Segments` for contacts, `Notes []*Note` for notes, `SearchTerm`); actions: `ActionNone`, `ActionContactAdd`, `ActionContactNoData`, `ActionContactFound`, `ActionContactNotFound`, `ActionContactDuplicate`, `ActionNoteAdd`, `ActionNoteNoData`, `ActionNoteFound`, `ActionNoteNotFound`, `ActionTransactionAdd`, `ActionTransactionNoData`, `ActionTransactionFound`, `ActionTransactionNotFound`.

### Notes use case
- A second Jev call is allowed per message: the classification call plus one category-specific extraction call (`note.json` for notes). Keep it conditional — the notes add path only asks for the sub-type when the message is actually a note.
- Deterministic text work (dates, times, list splitting, stopword removal) lives in `internal/services` as pure functions with no I/O, so it is testable without mocks; the Jev call is the only mocked seam.
- Dates are normalized to UTC midnight at parse time so the stored value and the query filter compare identically in SQLite; never compare a locally-built `time.Time` against a stored one.
- Normalize text with the existing `normalizeName` before matching PT-BR keywords or stopwords — the lists are written accent-free.
- Repository finders return `([]*models.Note, error)` and map an empty result to the package's `ErrNotFound`; "not found" is an outcome, not an error.
- Persist a note and its to-do items in one transaction (`Create(note, items)`); a partial write is never acceptable.
- The to-do splitter is deterministic with a fixed precedence — newlines, then numbered markers, then the inline separators — and its two heuristic rules (drop a leading `Label:`, split on ` e `) only fire when the text already proved to be a list. Keep a heuristic gated behind proof: ungated splitting ("e" anywhere, or any `:`) mangles ordinary sentences.

### Finance use case
- The finance type and the party refinement travel in **one** request: a `transaction_type` choice plus the `segment_N` Noul fan-out. Reuse `noulSegments`/`segmentKey` from `extraction.go` rather than reimplementing the fan-out — the two flows ask the same shape of question.
- There is no `finance.json` template. The question count depends on the candidate, so the request is built in code and sent through `MakeJevRequest`. Do not invent a fixed template for it.
- If the mixed request fails and there were segments, retry with the type question alone and continue **without** the party. The type is what makes the message a transaction; the party is a bonus.
- Amounts are stored positive and the direction lives in the type. Never introduce negative amounts or a signed `Sum` — it would double the accounting.
- The require path composes one filter (period + type + term) instead of picking one dimension. "quanto gastei com mercado esse mês" is all three at once; an either/or filter would answer none of them.
- Any text the UI can render back into an edit field must be re-readable by the same parser: the amount field is prefilled with `money` (`R$ 1.234,56`), never `printf "%.2f"` (`50.00` parses back as 5000 because dots are thousands separators).
- Editing a transaction re-parses amount and date through `ParseAmount`/`ParseEventDate`, the same pair the message path uses, so a stored value can never be one the classifier would not have produced.

### Views
- Template names are constants in `internal/views/render.go` — never string literals at call sites.
- Render through `views.RenderPage` / `views.RenderResult` / `views.RenderError`, not raw `c.HTML`.
- Pages render through `views.RenderPage(c, page, content)`; `views.PagesRenderer` clones the shared layout set per page so page blocks never collide. Do not try to collapse this into one template set: Go templates have no inheritance and `{{ template }}` names must be string literals, so the block name cannot come from the view data.
- Template name constants in `internal/views/render.go` include the harness partials: `prompt_table`, `evaluation_results`.
- Template helpers are exposed via `views.FuncMap` (e.g., `label` for PT-BR badge text, `confidence` for the percent format, `money` for amounts) and registered on the shared template set in `cmd/web/main.go` (`template.New("").Funcs(views.FuncMap)`).
- htmx detection is `c.GetHeader("HX-Request") == "true"` in `views.isHxRequest`. There is no server-side htmx library or context middleware for it.
- **A page template must `{{ define "page:content" }}`** — the `<name>:<name>:content` name. `PagesRenderer` executes `page:content`, so a mismatched define name renders an **empty body with HTTP 200** and no log line.

### Frontend (Web Awesome)
- Layout and typography are **CSS utility classes** (`wa-stack`, `wa-cluster`, `wa-gap-*`, `wa-align-items-*`, `wa-justify-content-*`, `wa-heading-*`, `wa-body-*`, `wa-color-text-quiet`, `wa-list-plain`, `wa-form-control-label`, `wa-tabular-nums`). There are no `<wa-stack>` / `<wa-cluster>` **elements** — verify a class exists in the utilities stylesheet before using it.
- `<wa-button>` defaults `type` to `button`. Every submit button needs an explicit `type="submit"` or the form silently does nothing.
- Form controls are form-associated custom elements (`wa-input`, `wa-select`, `wa-textarea`, `wa-checkbox`), so `name` still binds and `c.ShouldBind` is unchanged.
- A `<wa-checkbox>` with a `name` sends **nothing** when unchecked. Any form that reads the checkbox state **positionally** (parallel slices, like the to-do `item_done`) must keep a synced hidden input — see `partial/data_item_row.html`.
- `<fieldset disabled>` still disables the Web Awesome controls inside it (`formDisabledCallback`), so view/edit is still one template toggling `disabled`.
- Loading feedback is htmx's, not the component's: `hx-indicator` + a `<wa-spinner>` marked `.htmx-indicator`, plus `hx-disabled-elt="this"`. Web Awesome's `loading` attribute cannot be toggled by htmx.
- **CSS custom properties in `app.css` must use the Web Awesome 3.x vocabulary, never Shoelace 2.x.** An undefined `var()` with no fallback invalidates the **whole declaration** at computed-value time — no error, no log, the property just computes to `unset`. So a typo'd token name renders as "the style was never written", which is indistinguishable from a missing stylesheet. The renames that bite: `--wa-radius-*` → `--wa-border-radius-*`, `--wa-font-family-mono` → `--wa-font-family-code`, `--wa-font-weight-semi` → `--wa-font-weight-semibold`, `--wa-color-text-muted` → `--wa-color-text-quiet`, `--wa-color-surface-hover` → `--wa-color-surface-lowered`, and numeric steps (`--wa-color-brand-500`) → the semantic `{fill|border|on}-{quiet|normal|loud}` tokens. `TestAppCSSUsesNoShoelaceTokens` fails on the old names; extend its denylist when you find another.
- A utility **class** that renders nothing is the same silent failure as a dead token: verify it exists in the shipped stylesheet (`wa-mobile-full` was Shoelace 2.x and is gone in 3.x). Note that `wa-stack`/`wa-cluster` are matched by `[class*='wa-stack']`, so they are absent from a literal `.wa-stack` grep — grep the substring.
- Every render branch of a partial needs a view test that asserts the **fields are present**. A wrong field path (`.ID` where the model has `PromptID`) does not fail loudly — `html/template` writes output as it goes, so the partial ships **truncated HTML with HTTP 200** (see `template: result.html:11: unexpected "="` style parse errors too: they surface as an empty body, not a 500).
- Tooling note: templates are indented with tabs in `layouts/`, `result.html` and `error.html`; with 4 spaces in `pages/` and `prompt_*`/`evaluation_results`.

### Validation harness (/prompts)
- `PromptService` reuses the production Jev paths (`ClassificationService.Classify`, `ContactExtractor.ExtractName`, `NoteExtractor.ExtractType`, `FinanceExtractor.Extract`) — no new request-building code.
- Per-prompt Jev failures are captured in the row (obtained = error string, match = false); evaluation continues.
- CSV exports go to `exports/` (created on demand) via `encoding/csv` (stdlib).
- Form DTOs (`PromptForm`, `EvaluateForm`) live in `internal/models`; `[]uint` form slices bind repeated `ids` values.

### Data screen (/data)
- A browse flow gets its own service composed from the existing repositories (`DataService`). Do **not** grow `ContactService`/`NotesService`/`FinanceService` with browse queries — those are the classification flow and their extraction logic has no meaning for browsing.
- Adding a third kind to the screen is a three-way branch, not a second one: `DataService` gains the list/get/update/delete methods, the controller's `Update` and `renderTable`/`renderDetail` switch gains a case, and the partial gains a branch. The `kind` literal is a constant (`services.DataKindTransaction`).
- A search box that posts into the same table partial must carry `kind` and `filter` in hidden inputs, and the pill that changes the filter must re-point them — otherwise a search silently filters a different type than the one on screen.
- Filter validation belongs to the service, not the repository. The repository falls back to `all` on an unknown filter on purpose so other callers stay safe; `DataService` returns `ErrInvalidFilter` and the controller maps it to 400.
- Register literal path segments **before** a `:param` route (`/data/table`, `/data/item-row` before `/data/:kind/:id`), or gin's router reads them as the parameter value.
- View and edit are one template, not two: toggle `disabled` on the form's `fieldset` and swap the buttons. Two branches drift out of sync.
- Repeat a row of markup with `{{ template "partial_name" . }}` and fetch a blank copy over htmx for "add" — never keep two copies of the same row.
- Nullable columns (`*string`, `*time.Time`) get a `Deref`/`DateBR` template helper; rendering the pointer leaks `0xc000…` into the page.
- Which tab/filter is selected is client-side state (`markFilterPill`). The server re-renders `#data-table` but never the pill list, so an `active` class rendered by the server cannot survive the re-render a delete triggers. Marking one pill `active` in the template is the initial state only — never the mechanism.
- Switching an outer tab must load that pane's default filter (`activateFilterPills`), or the previous pane's table stays on screen looking like the new one. Load it **unconditionally**: `<wa-tab-group>` shows/hides the pane itself, so a `classList.contains('active')` early return skips the load precisely when you re-select the tab that is already showing.
- A pill that both boots a table on first paint and answers clicks carries `hx-trigger="load, click"`. An explicit `hx-trigger` **replaces** htmx's default `click` trigger, so a bare `hx-trigger="load"` turns the element into a dead button once the initial load fires — tabs would never re-load their pane's default filter, and direct clicks would do nothing.

### JSON responses
- JSON lives in `internal/api`, never in `internal/controllers` and never mixed into an htmx route — the htmx routes keep rendering partials.
- A presenter is a `switch` over `models.Action`, one per surface: the `resultado` partial branches for the HTML, `api.Outcome` builds the JSON. That switch is the whole abstraction — no `Presenter` interface, no registry, no map. The surfaces share no type; all they have to agree on is the wording, so a new action needs a branch in each one.
- **`internal/api` never imports `internal/views`** (nor `internal/controllers`, which imports views). A JSON handler that reaches for a template helper drags the whole render layer — template set, `views.FuncMap`, `labelMap` — behind a surface that never renders HTML. So `api.noteKind` restates the note label and `api.StatusFor` restates the web's status map instead of sharing either.
- Failures answer `{"error":"..."}` with the status rules the web entry point already uses: 400 bind/invalid, 502 upstream, 500 anything else.

### Jev domain types
All TypeSafe-related types are prefixed `Jev`. Questions implement `JevQuestionInterface` (`GetType()` / `GetInstructions()`), and `GetInstructions` is defined once on the embedded `JevQuestion` — it promotes to every question type, so do not repeat it. Answers are plain structs in `JevResponse.Answers map[string]any`; there is no answer interface.

### Config access
Never read `os.Getenv` directly outside `internal/config/env.go`. Config is read once at the composition root (`internal/app/app.go`) and injected into constructors (`jev.NewClient(env.TypesafeApiUrl, env.TypesafeToken, env.TypesafeModel)`).

## Error Handling

- Return errors up the stack; controllers convert them to HTTP responses.
- Wrap errors with context: `fmt.Errorf("failed to decode question %q in %q: %w", name, fileName, err)`.
- Error strings should be **lowercase** (Go convention).
- Use sentinel errors for status mapping: `errors.Is(err, jevq.ErrUpstream)` → 502, anything else → 500, bind failure → 400.
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
- A status-200-only test does not test a presenter branch. A presenter that prints nothing still answers 200, which reads as a styling bug rather than a failure — so assert the exact sentence per branch (`TestSummaryPerAction`) and keep the nil payload under test (`TestSummaryNeverPanics`).

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
# ARCHITECTURE

## Overview

A Go web application that classifies user messages into categories (contact, finance, schedule, notes, other) and determines whether a message is a request to add or require something. Classification is performed by the **TypeSafe System One API** (Jev model) via one AI request per message (a single template with two choice questions). After classification, a **Dispatcher** routes the message to a category-specific use case; the contact flow implements both the **add** path (regex extraction of phone/email, Jev Noul name extraction, duplicate detection on save, and SQLite persistence via Gorm) and the **require** path (search by phone/email/name). The frontend is a server-rendered htmx page styled with Pico.css (no JS build step).

The codebase follows a **semantic MVC pattern** on an idiomatic Go layout:

- **Model** — `internal/models` (data structures) + `internal/services` (business rules)
- **View** — `web/templates` (HTML) + `internal/views` (render helpers)
- **Controller** — `internal/controllers` (HTTP concerns only)
- **Infrastructure** — `pkg/jev` (reusable TypeSafe API client)

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.25.4 |
| Web framework | [gin-gonic/gin](https://github.com/gin-gonic/gin) v1.12.0 |
| HTMX | [donseba/go-htmx](https://github.com/donseba/go-htmx) v1.13.1 + htmx.org 2.0.10 (CDN) + response-targets extension |
| Env loading | [joho/godotenv](https://github.com/joho/godotenv) v1.5.1 |
| AI API | TypeSafe System One (`https://api.typesafe.ai/v1/systemone`, model `jev-latest`) |
| Templating | Go `html/template` (ParseGlob) |
| CSS | Pico.css v2 (CDN, dark theme) |
| Database | SQLite via [glebarez/sqlite](https://github.com/glebarez/sqlite) (pure-Go, zero CGO) |
| ORM | [gorm.io/gorm](https://gorm.io) |

## Directory Structure

```
msg-classifier/
├── cmd/
│   └── api/
│       └── main.go              # Composition root: config → jev client → services → dispatcher → controllers → routes
├── internal/                    # Private application code (not importable externally)
│   ├── config/
│   │   └── env.go               # Env singleton (TYPESAFE_API_URL, TYPESAFE_MODEL, TS_API_KEY, DB_PATH) — single godotenv load site
│   ├── models/                  # M — data structures
│   │   ├── message.go           # DTOs + Classification domain struct + UseCaseOutcome/Action + ToResponse
│   │   ├── contact.go           # Contact entity + SegmentScore
│   │   └── prompt.go            # JevPrompt entity + Flow constants + EvaluationResult + form DTOs
│   ├── repository/              # Persistence — all gorm queries
│   │   ├── contact.go           # ContactRepository — all gorm queries
│   │   └── prompt.go            # PromptRepository — Create, ListByFlow
│   ├── services/                # M — business rules
│   │   ├── classification.go    # ClassificationService (single Jev call, checked answer mapping)
│   │   ├── dispatcher.go        # Dispatcher (category → handler registry) + CategoryHandler interface
│   │   ├── contact.go           # ContactService (add + duplicate check + NameNorm backfill; normalizeName folded in; require routing)
│   │   ├── contact_get.go       # ContactService.Get (require flow: phone → email → name search)
│   │   ├── extraction.go        # ContactExtractor (regex phone/email + Jev Noul name fan-out)
│   │   └── prompt.go            # PromptService (validation harness: Add, ListByFlow, Evaluate, ExportCSV)
│   ├── controllers/             # C — HTTP concerns only (bind → service → render → status)
│   │   ├── web_controller.go    # GET / handler (delegates page/partial switch to views)
│   │   ├── message_controller.go# POST /api/message handler (bind → Classify → Dispatch → render)
│   │   └── prompt_controller.go # /prompts routes (Page, Table, Add, Evaluate, Export)
│   └── views/                   # V — render helpers
│       └── render.go            # Template name constants + RenderPage/RenderResult/RenderError + prompt partial helpers + PagesRenderer (per-page template sets)
├── pkg/                         # Reusable packages
│   ├── request.go               # ReturnJson helper ({"data": ...} / {"error": ...})
│   └── jev/
│       ├── jev.go               # TypeSafe Jev API client (config-injected, panic-free, embedded templates)
│       └── requests/            # JSON prompt templates (embedded via go:embed)
│           └── classification.json  # two choice questions: "classification" + "adding_or_requiring"
├── web/
│   └── templates/
│       ├── layouts/base.html    # "base" layout (Pico.css + htmx CDN + response-targets + nav)
│       ├── pages/index.html     # "page:title" / "page:content" blocks (form)
│       ├── pages/prompts.html   # "Validação Jev" page (flow select + add form + swap targets)
│       └── partial/
│           ├── result.html      # "resultado" partial (classification + action-specific process trace)
│           ├── error.html       # "error" partial (error card for htmx swap targets)
│           ├── prompt_table.html        # checkbox table + Avaliar button + export checkbox
│           ├── evaluation_results.html  # expected vs obtained comparison + segment trace + CSV path
├── scripts/
│   └── sql/
│       └── seed_prompts.sql    # wipe + re-seed jev_prompts examples
├── go.mod / go.sum              # Module "msg-classifier", Go 1.25.4
├── local.env                    # Env vars (not committed secrets)
├── .vscode/launch.json          # Go debug config for cmd/api/main.go
└── README.md
```

## Core Components

### 1. Composition Root — `cmd/api/main.go`
- `main()` builds a `gin.Default()` router, parses `web/templates/**/*.html` (`template.Must`), installs an htmx middleware that stores an `*htmx.Handler` in the Gin context under key `"htmx"` (single htmx instance).
- Wires dependencies: `config.GetEnv()` (single godotenv load site) → `jev.NewClient(url, token, model)` → `services.NewClassificationService(client)` → `services.NewDispatcher` (registry: `"contact"` → `ContactService`) → controllers.
- SQLite pool is capped at one connection (`SetMaxOpenConns(1)` right after `gorm.Open`) — all DB access is serialized; the pure-Go driver (glebarez/modernc) is unstable with concurrent connections on Windows.
- Registers routes:
  - `GET /` → `webController.Home`
  - `POST /api/message` → `messageController.ReceiveMessage`
  - `GET /prompts` → `promptController.Page`
  - `GET /prompts/table` → `promptController.Table`
  - `POST /prompts` → `promptController.Add`
  - `POST /prompts/evaluate` → `promptController.Evaluate`
- Template render: shared set for `layouts/`+`partial/`, cloned per page (`index`, `prompts`) via `views.PagesRenderer`; partials render from the shared set.
- Server runs on `:8080`.

### 2. Config Singleton — `internal/config/env.go`
- `GetEnv()` lazily loads `local.env` and reads `TYPESAFE_API_URL`, `TYPESAFE_MODEL`, `TS_API_KEY`, and `DB_PATH` (default `contacts.db`) into an `Env` struct. Cached in a package-level `var env *Env`. This is the **only** place godotenv is loaded and `os.Getenv` is called.

### 3. Controllers — `internal/controllers/`
- `WebController.Home`: delegates to `views.RenderPage` — the htmx page/partial switch lives in the view layer.
- `MessageController.ReceiveMessage`: binds `ReceiveMessageRequest` (failure → 400 error partial) → calls `ClassificationService.Classify` → calls `Dispatcher.Dispatch` → maps errors via `renderServiceError` (`errors.Is(err, services.ErrUpstream)` → 502, else 500) → renders result or error partial. No business logic, no Jev types, no template name literals.

### 4. Classification Service — `internal/services/classification.go`
- `ClassificationService.Classify`: builds Jev state `{user, message}`, makes **one** Jev call (`classification.json`, which contains both the `classification` and the `adding_or_requiring` choice questions), extracts answers with checked assertions (`answerAsChoice`), returns a domain `Classification`.
- `ErrUpstream` sentinel marks failures originating from the Jev/TypeSafe API or its responses; controllers map it to HTTP 502.
- The `jevClient` interface (defined at the service boundary) makes the service unit-testable without HTTP.

### 5. Dispatcher — `internal/services/dispatcher.go`
- `CategoryHandler` interface: `Handle(request, classification) (*models.UseCaseOutcome, error)` — the seam for category-specific use cases.
- `Dispatcher` holds a `map[string]CategoryHandler` keyed by `Category.Choice`; `Dispatch` looks up the handler, falling back to an `ActionNone` outcome on a miss. Adding a category = new service implementing `CategoryHandler` + one wiring line in `main.go`.

### 6. Contact Service — `internal/services/contact.go` + `contact_get.go`
- `ContactService` implements `CategoryHandler` for the `contact` category. Constructor takes a `*ContactExtractor` and a `*repository.ContactRepository` — the repository owns all gorm queries, and the service wraps repo errors with the same context strings (`failed to check duplicate contact`, `failed to persist contact`, `failed to search contact`, `failed to load contacts for backfill`, `failed to backfill name_norm`). `Handle` routes `require` → `Get`, everything else → `Add`. `Add` extracts phone/email → neither found → `ActionContactNoData` outcome; a duplicate check (phone, then email) runs **before** name extraction — a match returns `ActionContactDuplicate` + the existing contact (no Jev call); otherwise name extraction runs and the repository inserts the contact (populating `NameNorm`) → `ActionContactAdd` outcome carrying the saved contact and the segment trace (`outcome.Segments`). `Get` (require flow) searches with phone → email → name priority; phone/email searches skip Jev entirely; name search uses `LIKE %term%` on `name_norm`; `SearchTerm` is set on the outcome and not-found is an outcome, not an error. `BackfillNameNorm()` recomputes `NameNorm` for pre-migration rows at startup.

### 6b. Contact Extractor — `internal/services/extraction.go`
- `ContactExtractor` extracts phone (BR regex, normalized to 10/11 digits) and email (first match + span) deterministically, and the name via one dynamic Jev request with a Noul question per whitespace segment (`segment_0..N`). `ExtractName` returns a `NameResult` — the joined name plus a per-segment trace (`SegmentScore`: text, noul score, `Included` = score > 0.5, the single threshold site the join and the UI both derive from). Depends on a minimal `jevRequester` interface (`MakeJevRequest`) so tests mock the Jev call.

### 6c. Contact Repository — `internal/repository/contact.go`
- `ContactRepository` owns all gorm queries for the `Contact` entity; `NewContactRepository(db *gorm.DB)` wraps the DB handle. Methods: `Create` (persist a new contact), `FindByPhone` / `FindByEmail` (case-insensitive) / `FindByName` (`name_norm LIKE %term%`), `ListNeedingNameNorm` (empty/NULL `name_norm`, pre-migration rows), and `Save` (backfill updates). `ErrNotFound = gorm.ErrRecordNotFound` is the not-found sentinel — services detect it with `errors.Is` without importing gorm; all other errors are returned raw and wrapped by the service with its context strings.

### 7. Models — `internal/models/message.go` + `contact.go`
- `message.go`: `ReceiveMessageRequest` / `ReceiveMessageResponse` DTOs; `Classification` domain struct (`CategoryFinding` / `KindFinding` with raw numeric confidences; `KindFinding.Choice` carries the kind: `"add"` / `"require"` / `"both"`); `UseCaseOutcome` (`Classification` + `Action` + `SearchTerm`) — the seam where use-case results (extracted contact, DB confirmation, search term) flow back without signature changes; actions are `ActionNone` / `ActionContactAdd` / `ActionContactNoData` / `ActionContactFound` / `ActionContactNotFound` / `ActionContactDuplicate`; `Classification.ToResponse()` formats confidences as `%.2f` percent strings for display (successor of the former `fromJevResponse`).
- `contact.go`: `Contact` Gorm entity (ID, Name, `NameNorm` column, Phone/Email nullable, timestamps) and `SegmentScore` (text, noul score, included flag); `UseCaseOutcome` carries `Contact` and `Segments` (nil unless name extraction ran).

### 8. Views — `internal/views/render.go`
- Template name constants (`base`, `page:content`, `resultado`, `error`) — no string literals at call sites.
- `RenderPage`: renders `page:content` for htmx requests, full `base` otherwise; falls back to the full page on missing/mistyped htmx context (no panic).
- `RenderResult` / `RenderError`: render the result or error partial; errors carry proper HTTP status so htmx swaps the error card into `#resultado`.

### 9. Jev API Client — `pkg/jev/jev.go`
- `Client` struct with `NewClient(apiURL, token, model)` — all configuration injected, no `internal/config` import (genuinely reusable).
- `MakeJevRequest`: sets model, validates, POSTs JSON with `Authorization: Bearer <token>`, parses response. 30s HTTP timeout; response body closed on all paths.
- `MakeJevRequestFromFile` / `LoadJevRequestFromFile`: load prompt templates from the embedded `requests/*.json` (via `go:embed` — no CWD-relative path dependency), inject state, delegate.
- `HttpResponseToJevResponse`: decodes answers by `type` into typed structs; **never panics** — every assertion is checked, unknown answer types return an explicit error.
- `validateJevRequest`: validates state, model, questions, and per-type criteria/true-false fields.

### 10. Jev Request Templates — `pkg/jev/requests/*.json`
- `classification.json`: two `choice` questions — `"classification"` with 5 criteria (contact, finance, schedule, notes, other) and `"adding_or_requiring"` with descriptive criteria (add, require, both).

### 11. HTML Templates — `web/templates/`
- `base.html`: `base` layout, loads Pico.css v2 (dark theme via `data-theme="dark"`) + htmx 2.0.10 + response-targets extension from CDNs; `hx-ext="response-targets"` + `hx-target-error="#resultado"` on `<body>` route 4xx/5xx responses into the result container.
- `index.html`: form posting via `hx-post="/api/message"` targeting `#resultado` with `hx-swap="innerHTML"`.
- `result.html`: `resultado` partial — a Pico `<article>` with the classification block in `<header>`, plus branches for add/found/not-found/duplicate/no-data and the per-segment extraction trace in `<footer><small>`.
- `error.html`: `error` partial rendering a Pico `<article>` error card (used for 400/502/500 responses).

### 12. Prompt Service (validation harness) — `internal/services/prompt.go` + `prompt_controller.go`
- `PromptService` orchestrates the Jev validation harness: `Add` (validates flow ∈ {classification, name} and non-empty fields, `ErrInvalidPrompt` → 400), `ListByFlow`, `Evaluate` (loads prompts by flow, filters to selected ids, runs the exact production paths — `ClassificationService.Classify` for classification, `ContactExtractor.ExtractName` for name — and compares expected vs obtained; a Jev failure for one prompt is captured in its row as the obtained result with match=false and evaluation continues), and `ExportCSV` (writes the given evaluation results to `exports/<flow>-<yyyyMMdd-HHmmss>.csv` via `encoding/csv`, folder created on demand — no re-run, the CSV mirrors the evaluation the user just saw).
- `PromptController` is thin: `Page` renders the page, `Table` renders the `prompt_table` partial, `Add` persists and re-renders the table, `Evaluate` renders `evaluation_results` (and, when the form's export checkbox is set, saves the CSV first). Bind failures → 400, invalid prompt → 400, service failures → 500 (existing `renderServiceError`).
- The harness reuses the exact production Jev paths — no new request-building code.

## Data Flow

```
Browser (htmx form)
  │  POST /api/message  (message, user_id)
  ▼
gin router ──► MessageController.ReceiveMessage
  │  Bind → ReceiveMessageRequest (fail → 400 error partial)
  ├─► ClassificationService.Classify
  │     ├─► jev client ──► POST api.typesafe.ai/v1/systemone (classification.json) ──► two choice answers
  │     ├─► checked extraction → Category + Kind (fail → ErrUpstream → 502 error partial)
  ├─► Dispatcher.Dispatch
  │     ├─► contact + add/both → ContactService.Add
  │     │     ├─► ExtractPhone / ExtractEmail (regex)
  │     │     ├─► neither → outcome ActionContactNoData (200, friendly partial)
  │     │     ├─► duplicate check → repo.FindByPhone / repo.FindByEmail (no Jev)
  │     │     ├─► remainder → segments → Jev call #2 (dynamic Noul fan-out)
  │     │     ├─► name = segments with noul > 0.5, joined in order (trace kept)
  │     │     ├─► repo.Create(contact) (SQLite)
  │     │     └─► outcome ActionContactAdd + saved contact + segment trace
  │     ├─► contact + require   → ContactService.Get
  │     │     ├─► ExtractPhone → repo.FindByPhone (no Jev)
  │     │     ├─► else ExtractEmail → repo.FindByEmail (no Jev)
  │     │     ├─► else ExtractName → normalizeName → repo.FindByName (name_norm LIKE %term%)
  │     │     ├─► found → outcome ActionContactFound + contact + SearchTerm
  │     │     ├─► not found → outcome ActionContactNotFound + SearchTerm
  │     │     └─► nothing extractable → outcome ActionContactNoData
  │     └─► other category      → outcome ActionNone
  ├─► outcome.Classification.ToResponse() → ReceiveMessageResponse
  └─► views.RenderResult ──► c.HTML(200, "resultado", ...) ──► htmx swaps #resultado innerHTML
```

Up to two TypeSafe API calls are made per request: the classification call, and (for contact add with extractable data, or a require-by-name search) the name fan-out call. All persistence goes through `ContactRepository` — `repo.Create` for saves, `repo.FindByPhone` / `repo.FindByEmail` / `repo.FindByName` for lookups, and `repo.ListNeedingNameNorm` + `repo.Save` for the startup `NameNorm` backfill. Contacts are persisted to a local SQLite database (`DB_PATH`, default `contacts.db`); the require flow searches by phone/email/name and duplicate saves are detected before name extraction. The result partial always shows the classification block and adds an action-specific block (saved contact with ID, fields, the per-segment extraction trace, found/not-found/duplicate messages, or the no-data message).

```
Browser (/prompts)
  │ select flow → hx-get /prompts/table?flow=classification
  ▼
PromptController.Table → PromptService.ListByFlow → prompt_table partial (checkboxes)
  │ check rows → "Avaliar" → hx-post /prompts/evaluate {flow, ids[], export?}
  ▼
PromptController.Evaluate → PromptService.Evaluate
  │   ├─ per id: ClassificationService.Classify  (or ContactExtractor.ExtractName)
  │   ├─ match = expected vs obtained (format per flow)
  │   ├─ per-row error capture (upstream → obtained=error, match=false)
  │   ├─ if export checkbox set → PromptService.ExportCSV(results) → exports/<flow>-<timestamp>.csv
  → evaluation_results partial (✓/✗ per row + CSV path when exported)
```

## Error Handling

| Failure | Status | Response |
|---|---|---|
| Request bind failure | 400 | `error` partial |
| Jev/TypeSafe API or response failure (`ErrUpstream`) | 502 | `error` partial |
| Any other service failure | 500 | `error` partial |
| No phone/email extractable | 200 | `resultado` partial (no-data branch) |
| Contact not found | 200 | `resultado` partial (not-found branch) |
| Duplicate contact on save | 200 | `resultado` partial (duplicate branch) |
| Database failure | 500 | `error` partial |
| Success | 200 | `resultado` partial |

All responses to htmx targets are HTML partials — no JSON on this route. The response-targets extension makes htmx swap 4xx/5xx responses into `#resultado`. No panics exist in the request path.

## External Integrations

| Service | Purpose | Config |
|---|---|---|
| TypeSafe System One API | Message classification (Jev model) | `TYPESAFE_API_URL`, `TYPESAFE_MODEL`, `TS_API_KEY` |
| htmx.org 2.0.10 (CDN) | Client-side partial page updates | — |
| htmx-ext-response-targets (CDN) | Swap 4xx/5xx responses into `#resultado` | — |
| Pico.css v2 (CDN, dark theme) | Styling | — |
| SQLite (glebarez/sqlite) | Contact persistence | DB_PATH |

## Configuration

| Variable | Source | Used by |
|---|---|---|
| `TYPESAFE_API_URL` | `local.env` | `cmd/api/main.go` → `jev.NewClient` (POST target) |
| `TYPESAFE_MODEL` | `local.env` | `cmd/api/main.go` → `jev.NewClient` (model field) |
| `TS_API_KEY` | environment (not in `local.env`) | `cmd/api/main.go` → `jev.NewClient` (Bearer token) |
| DB_PATH | local.env (default contacts.db) | cmd/api/main.go → gorm.Open |

> Note: `TS_API_KEY` is read by `internal/config/env.go` but not defined in `local.env` — it must be set in the environment or the Authorization header will be `Bearer ` (empty).

## Build & Deploy

```bash
# Run locally (loads local.env, serves on :8080)
go run ./cmd/api

# Build binary
go build ./cmd/api

# Debug (VS Code)
# .vscode/launch.json → "API Debug" runs cmd/api/main.go from workspace root
```

- No Dockerfile, Makefile, or CI pipeline exists yet.
- Tests exist for `pkg/jev`, `internal/models`, `internal/config`, `internal/services`, `internal/views`, and `internal/repository` (run with `go test ./...`).
- `.gitignore` ignores `**/*_bin.exe`, `thoughts/`, `*.db`, `.vscode`, and `exports/`.
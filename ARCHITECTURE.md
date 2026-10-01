# ARCHITECTURE

## Overview

A Go web application that classifies user messages into categories (contact, finance, schedule, notes, other) and determines whether a message is a request to add or require something. Classification is performed by the **TypeSafe System One API** (Jev model) via one AI request per message (a single template with two choice questions). After classification, a **Dispatcher** routes the message to a category-specific use case; the contact flow implements both the **add** path (regex extraction of phone/email, Jev Noul name extraction, duplicate detection on save, and SQLite persistence via Gorm) and the **require** path (search by phone/email/name). The frontend is a server-rendered htmx page styled with Bootstrap 5.3 (no JS build step).

The same `CategoryHandler` seam now also serves the **notes** flow (`NotesService`): a second Jev call (`note.json`, one `note_type` choice question) picks the sub-type (note / reminder / to-do list), a deterministic PT-BR parser extracts the reminder date and optional time and the to-do list is split into items, everything is persisted in `notes` + `todo_items` (SQLite via Gorm), and the **require** path searches by date, unfinished items or a content term.

A third surface, **`/data`**, browses and edits what those two flows persisted. `DataService` composes `ContactRepository` and `NotesRepository` directly — `ContactService`/`NotesService` belong to the classification flow and are not reused here — and `DataController` exposes it as htmx partials: filter pills, a per-record edit panel and bulk delete. No Jev call happens on this screen.

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
| HTMX | htmx.org 2.0.10 (CDN) + response-targets extension |
| Env loading | [joho/godotenv](https://github.com/joho/godotenv) v1.5.1 |
| AI API | TypeSafe System One (`https://api.typesafe.ai/v1/systemone`, model `jev-latest`) |
| Templating | Go `html/template` (ParseGlob) |
| CSS | Bootstrap 5.3 (CDN, default light theme) + Bootstrap Icons (CDN) + custom layer (web/static/css/app.css) |
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
│   │   ├── message.go           # ReceiveMessageRequest DTO + Classification domain struct + UseCaseOutcome/Action
│   │   ├── contact.go           # Contact entity + SegmentScore
│   │   ├── note.go              # Note + TodoItem entities + note type constants
│   │   ├── prompt.go            # JevPrompt entity + Flow constants + EvaluationResult + form DTOs
│   │   └── data.go              # DataForm + DataDeleteForm (data screen forms)
│   ├── repository/              # Persistence — all gorm queries
│   │   ├── contact.go           # ContactRepository — all gorm queries
│   │   ├── note.go              # NotesRepository — atomic Create(note, items), FindByDate, FindUnfinished, FindByTerm
│   │   └── prompt.go            # PromptRepository — Create, ListByFlow
│   ├── services/                # M — business rules
│   │   ├── classification.go    # ClassificationService (single Jev call, checked answer mapping)
│   │   ├── dispatcher.go        # Dispatcher (category → handler registry) + CategoryHandler interface
│   │   ├── contact.go           # ContactService (add + duplicate check + NameNorm backfill; normalizeName folded in; require routing)
│   │   ├── contact_get.go       # ContactService.Get (require flow: phone → email → name search)
│   │   ├── dateparse.go         # ParseDate / ParseTime (deterministic PT-BR date/time extraction)
│   │   ├── extraction.go        # ContactExtractor (regex phone/email + Jev Noul name fan-out)
│   │   ├── note.go              # NotesService (struct + Handle + Add: note / reminder / todo)
│   │   ├── note_extract.go      # NoteExtractor (Jev note_type choice call)
│   │   ├── note_get.go          # NotesService.Get (require flow: date → pending → term search)
│   │   ├── todo_split.go        # SplitTodoItems (newlines → numbered → commas/semicolons)
│   │   ├── prompt.go            # PromptService (validation harness: Add, ListByFlow, Evaluate, ExportCSV)
│   │   └── data.go              # DataService (browse/edit/delete over ContactRepository + NotesRepository)
│   ├── controllers/             # C — HTTP concerns only (bind → service → render → status)
│   │   ├── web_controller.go    # GET / handler (delegates page/partial switch to views)
│   │   ├── message_controller.go# POST /api/message handler (bind → Classify → Dispatch → render)
│   │   ├── prompt_controller.go # /prompts routes (Page, Table, Add, Evaluate, Export)
│   │   └── data_controller.go   # /data routes (Page, Table, Detail, Update, Delete, ItemRow)
│   └── views/                   # V — render helpers
│       ├── render.go            # Template name constants + RenderPage/RenderResult/RenderError + partial helpers + FuncMap
│       └── pages_renderer.go    # PagesRenderer (gin.HTMLRender with a template set per page)
├── pkg/                         # Reusable packages
│   └── jev/
│       ├── jev.go               # TypeSafe Jev API client (config-injected, panic-free, embedded templates)
│       └── requests/            # JSON prompt templates (embedded via go:embed)
│           ├── classification.json  # two choice questions: "classification" + "adding_or_requiring"
│           └── note.json            # one choice question: "note_type" (note / reminder / todo)
├── web/
│   ├── static/
│   │   └── css/app.css        # custom layer over Bootstrap 5.3 (tokens, badges, sidebar, spinner, tables)
│   └── templates/
│       ├── layouts/base.html    # "base" layout (pt-BR, Inter, favicon, sticky navbar, page:active block)
│       ├── pages/index.html     # home page (hero copy + form + spinner on Enviar)
│       ├── pages/prompts.html   # "Validação Jev" page (flow select + add form + spinner)
│       ├── pages/data.html      # "Dados" page (filter pills + table container + detail offcanvas)
│       └── partial/
│           ├── result.html      # "resultado" partial (badges + structured card + segment list)
│           ├── error.html       # "error" partial (red error card for htmx swap targets)
│           ├── prompt_table.html        # checkbox table + Avaliar button + export checkbox + spinner
│           ├── evaluation_results.html  # expected vs obtained comparison + match badges + segment rows + CSV path
│           ├── contacts_table.html      # contact rows + per-row "Ver" + bulk delete
│           ├── notes_table.html         # note rows (type badge, date/time, items) + per-row "Ver" + bulk delete
│           ├── data_detail.html         # offcanvas panel body: contact/note form in view + edit mode
│           ├── data_item_row.html       # one to-do item row (existing rows reuse it via {{ template }})
├── scripts/
│   └── sql/
│       └── seed_prompts.sql    # wipe + re-seed jev_prompts examples
├── go.mod / go.sum              # Module "msg-classifier", Go 1.25.4
├── local.env                    # Env vars (not committed secrets)
├── .vscode/launch.json          # Go debug config for cmd/api/main.go
├── README.md
└── docs/
    ├── decisions/DC-005.md     # notes, reminders, to-do lists
    └── todos.md                # backlog for the notes use case (relative dates, deadlines, accent-insensitive search)
```

## Core Components

### 1. Composition Root — `cmd/api/main.go`
- `main()` builds a `gin.Default()` router, serves `web/static` via `router.Static("/static", "./web/static")`, parses `web/templates/**/*.html` (`template.Must`) and installs `views.PagesRenderer` as `router.HTMLRender`.
- Wires dependencies: `config.GetEnv()` (single godotenv load site) → `jev.NewClient(url, token, model)` → `services.NewClassificationService(client)` → `services.NewNoteExtractor(client)` → `services.NewDispatcher` (registry: `"contact"` → `ContactService`, `"notes"` → `NotesService`) → controllers. The contact and notes repositories are built once and shared with the data screen.
- `db.AutoMigrate(&models.Contact{}, &models.JevPrompt{}, &models.Note{}, &models.TodoItem{})`.
- SQLite pool is capped at one connection (`SetMaxOpenConns(1)` right after `gorm.Open`) — all DB access is serialized; the pure-Go driver (glebarez/modernc) is unstable with concurrent connections on Windows.
- Registers routes:
  - `GET /` → `webController.Home`
  - `POST /api/message` → `messageController.ReceiveMessage`
  - `GET /prompts` → `promptController.Page`
  - `GET /prompts/table` → `promptController.Table`
  - `POST /prompts` → `promptController.Add`
  - `POST /prompts/evaluate` → `promptController.Evaluate`
  - `GET /data` → `dataController.Page`
  - `GET /data/table` → `dataController.Table`
  - `GET /data/item-row` → `dataController.ItemRow`
  - `GET /data/:kind/:id` → `dataController.Detail`
  - `POST /data/:kind/:id` → `dataController.Update`
  - `POST /data/delete` → `dataController.Delete`
- The three static `/data/*` segments are registered **before** `/data/:kind/:id` — gin's router would otherwise read `table` and `item-row` as a `:kind`.
- Template render: shared set for `layouts/`+`partial/`, cloned per page (`index`, `prompts`, `data`) via `views.PagesRenderer` — Go templates have no inheritance and `{{template}}` names must be literals, so each page needs its own set to keep `page:content` isolated; partials render from the shared set.
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
- `ContactExtractor` extracts phone (BR regex, normalized to 10/11 digits) and email (first match + span) deterministically, and the name via one dynamic Jev request with a Noul question per whitespace segment (`segment_0..N`). `ExtractName` returns a `NameResult` — the joined name plus a per-segment trace (`SegmentScore`: text, noul score, `Included`). A deterministic post-filter resolves name particles by position, since the Noul scores for `do`/`da`/`de` hover around the 0.5 threshold and flip between runs: (1) drop leading lowercase segments while a capitalized one follows (a proper name never starts with a function word, so `do João da Silva` → `João da Silva`); (2) rescue a particle Jev excluded when it sits between two included capitalized segments (`José Carlos de Souza` keeps its `de`). The trace's `Included` flags are updated to match. Depends on a minimal `jevRequester` interface (`MakeJevRequest`) so tests mock the Jev call.

### 6c. Contact Repository — `internal/repository/contact.go`
- `ContactRepository` owns all gorm queries for the `Contact` entity; `NewContactRepository(db *gorm.DB)` wraps the DB handle. Methods: `Create` (persist a new contact), `FindByPhone` / `FindByEmail` (case-insensitive) / `FindByName` (`name_norm LIKE %term%`), `ListNeedingNameNorm` (empty/NULL `name_norm`, pre-migration rows), and `Save` (backfill updates). `ErrNotFound = gorm.ErrRecordNotFound` is the not-found sentinel — services detect it with `errors.Is` without importing gorm; all other errors are returned raw and wrapped by the service with its context strings.
- The data screen added `List(filter)` / `FindByID` / `DeleteByIDs`. `List` orders `id DESC` (newest first) and falls back to all rows on an unknown filter instead of erroring, so `DataService` stays the gate that rejects one.

### 6d. Notes Service — `internal/services/note.go` + `note_get.go`
- `NotesService` implements `CategoryHandler` for the `notes` category. `Handle` routes `require` → `Get`, everything else → `Add`. Constructor takes the `*NoteExtractor` and the `*repository.NotesRepository`; repo errors are wrapped with `failed to persist note` / `failed to search notes`.
- `Add` asks Jev for the sub-type, then: **note** stores content only; **reminder** requires a date (optional time) and returns `ActionNoteNoData` when the date is unparseable; **todo** splits the content into ordered items. An unknown sub-type is also `ActionNoteNoData`. Success returns `ActionNoteAdd` carrying the saved note in `outcome.Notes`.
- `Get` (require) picks one filter from the message, in order: a parsed date (`FindByDate`, including `ontem`), an unfinished marker (`falta`/`faltam`/`pendente`/`pendentes`/`não fiz`/`ainda não` → `FindUnfinished`), otherwise a content term (`FindByTerm`, `LOWER(content) LIKE %term%`) built by stripping PT-BR stopwords from the accent-normalized message, with a retry on the last word when the phrase misses. Found → `ActionNoteFound` + the notes list; not found → `ActionNoteNotFound`; nothing extractable → `ActionNoteNoData`.
- Content is always the trimmed original message; the date, the time and the items live in their own columns.

### 6e. Note Extractor / Date Parser / Todo Splitter — `internal/services/note_extract.go`, `dateparse.go`, `todo_split.go`
- `NoteExtractor.ExtractType` makes the **second** Jev call of a notes message (`note.json`, one `note_type` choice answer) reusing the `jevClient` seam and `answerAsChoice`; the raw choice is validated by the service.
- `ParseDate` / `ParseTime` are deterministic and dependency-free package functions (no struct, nothing to inject): `dd/mm/aaaa`, `dd/mm` (current year), `dia N` (current month), `hoje`, `amanhã`, `ontem`; times as `14h`, `14h30`, `14:00`. Every date is normalized to UTC midnight so the equality filter on `notes.date` compares identically. `now` is a parameter, not `time.Now()`, to keep the relative forms testable.
- `SplitTodoItems` splits on newlines first, then numbered markers (`1. `, `2) `), then commas/semicolons, trimming punctuation and leading numbers/conjunctions; the service assigns `Position` from the slice order. Two more rules, both gated on the same "this is a list" signal (a `,`/`;` survived, or a label was dropped): a leading marker (`Comprar:`, `Tarefas:`) is dropped when a list follows it — a bare `Tarefas:` is left with nothing, which the service answers as no-data — and the `e` conjunction separates items only in an inline list. Line and numbered lists keep one item per line, and a single statement like `comprar pão e leite` stays one item.

### 7. Models — `internal/models/message.go` + `contact.go` + `note.go`
- `message.go`: `ReceiveMessageRequest` inbound DTO; `Classification` domain struct (`CategoryFinding` / `KindFinding` with raw numeric confidences; `KindFinding.Choice` carries the kind: `"add"` / `"require"` / `"both"`); `UseCaseOutcome` (`Classification` + `Action` + `SearchTerm`) — the seam where use-case results (extracted contact, DB confirmation, search term) flow back without signature changes; actions are `ActionNone` / `ActionContactAdd` / `ActionContactNoData` / `ActionContactFound` / `ActionContactNotFound` / `ActionContactDuplicate` / `ActionNoteAdd` / `ActionNoteNoData` / `ActionNoteFound` / `ActionNoteNotFound`. There is no response DTO: the result partial receives the `Classification` and formats it.
- `contact.go`: `Contact` Gorm entity (ID, Name, `NameNorm` column, Phone/Email nullable, timestamps) and `SegmentScore` (text, noul score, included flag); `UseCaseOutcome` carries `Contact` and `Segments` (nil unless name extraction ran).
- `note.go`: `Note` (ID, Type, Content, nullable `Date`/`Time`, `Items`, timestamps) and `TodoItem` (ID, `NoteID` FK with `ON DELETE CASCADE`, Text, `Done`, `Position`); indexes on `notes.type`, `notes.date` and `todo_items.done`. `UseCaseOutcome` also carries `Notes []*Note` (the add path returns one element); the contact constants are unchanged.
- `data.go`: `DataForm` and `DataDeleteForm`. Both contact and note edits bind to the same `DataForm` (`Name`/`Phone`/`Email` for contacts; `Content`/`Date`/`Time` plus parallel `ItemText []string` / `ItemDone []string` slices for a to-do). `DataDeleteForm` carries `Kind`, `Filter` and `IDs []uint` — a `[]uint` form field binds repeated `ids` values. `form` tags only; nothing here is part of a JSON API.

### 8. Views — `internal/views/render.go` + `pages_renderer.go`
- Template name constants (`base`, `page:content`, `resultado`, `error`, `contacts_table`, `notes_table`, `data_detail`, `data_item_row`) — no string literals at call sites.
- `RenderPage`: renders `page:content` when the `HX-Request` header is `true`, full `base` otherwise. The header is the whole htmx detection there is — no middleware, no server-side htmx dependency.
- `RenderResult` / `RenderError`: render the result or error partial; errors carry proper HTTP status so htmx swaps the error card into `#resultado`.
- The shared template set registers `views.FuncMap` (`label` for PT-BR badge text, `confidence` for the percent format, plus `dateBR`/`deref` for the data screen).
- The data screen passes templates page-scoped view models — `DataDetailData{Kind, Mode, Contact, Note}` and `ItemRowData` — rather than handing entities straight to a partial. `Mode` is `view` or `edit`, and the switch happens in the template through a single `disabled` attribute on the form's `fieldset` instead of two parallel branches.
- `DateBR(*time.Time)`, `Deref(*string)` and `Confidence(float64)` are template helpers, not entity methods: rendering a `*string` directly puts a pointer address (`0xc000…`) in the cell, and formatting a confidence is presentation.

### 9. Jev API Client — `pkg/jev/jev.go`
- `Client` struct with `NewClient(apiURL, token, model)` — all configuration injected, no `internal/config` import (genuinely reusable).
- `MakeJevRequest`: sets model, validates, POSTs JSON with `Authorization: Bearer <token>`, parses response. 30s HTTP timeout; response body closed on all paths.
- `MakeJevRequestFromFile` / `LoadJevRequestFromFile`: load prompt templates from the embedded `requests/*.json` (via `go:embed` — no CWD-relative path dependency), inject state, delegate.
- `HttpResponseToJevResponse`: decodes `answers` as `map[string]json.RawMessage`, reads each answer's `type` to pick the struct, then unmarshals once; answers land in `map[string]any` as `*JevAnswerChoice` / `*JevAnswerNoul` and callers type-assert. **Never panics** — unknown or missing answer types return an explicit error.
- Two question types exist because two templates exist: `choice` and `noul`. `validateJevRequest` validates state, model, questions, and per-type criteria/true-false fields.

### 10. Jev Request Templates — `pkg/jev/requests/*.json`
- `classification.json`: two `choice` questions — `"classification"` with 5 criteria (contact, finance, schedule, notes, other) and `"adding_or_requiring"` with descriptive criteria (add, require, both).
- `note.json`: one `choice` question — `"note_type"` with 3 criteria (note, reminder, todo). It is only called for messages classified as `notes`, and only on the add path.

### 11. HTML Templates — `web/templates/`
- `base.html`: `base` layout, `lang="pt-BR"`, loads Inter (Google Fonts) + Bootstrap 5.3 (default light theme) + Bootstrap Icons + `app.css` + htmx 2.0.10 + response-targets extension from CDNs; offcanvas-lg sidebar (fixed at lg+, drawer with hamburger below) with a `page:active` block for the active link; `hx-ext="response-targets"` + `hx-target-error="#resultado"` on `<body>` route 4xx/5xx responses into the result container.
- `index.html`: form posting via `hx-post="/api/message"` targeting `#resultado` with `hx-swap="innerHTML"`.
- `result.html`: `resultado` partial — a Bootstrap `<div class="card result-card">` with PT-BR category/kind badges in `card-header`, branches for add/found/not-found/duplicate/no-data (contact) and note_add/note_found/note_not_found/note_no_data (notes: type badge, content, date/time for reminders, item list with ✓/○ for to-dos), and the per-segment extraction trace as a list in `card-footer`.
- `error.html`: `error` partial rendering a Bootstrap `card border-danger` error card (used for 400/404/502/500 responses).
- `data.html`: `data` page — filter pills that each `hx-get` a table partial, an empty `#data-table` container that loads on first paint, and the `#dataPanel` offcanvas whose body is swapped per record. A `dataChanged` body event lets the delete endpoint re-render the table the user was on without a full page reload. Pill highlight state is client-side (`markFilterPill`), because the server re-renders `#data-table` but never the pill list — a server-rendered `active` class would be lost on every delete. Switching the outer tab calls `activateFilterPills`, which loads and selects that pane's default filter, so the other kind's table is never left on screen. The contacts `Todos` pill (and only it) carries `hx-trigger="load, click"` — it boots the table on first paint but must stay clickable, and an explicit `hx-trigger` replaces htmx's default `click` trigger, so `load` alone would make it a dead button. `activateFilterPills` clicks that default pill unconditionally, with no `active`-pane guard: Bootstrap toggles `active` itself, so such a guard would skip the reload exactly when the tab you re-select is the one already showing.
- `contacts_table.html` / `notes_table.html`: row tables plus a checkbox per row (`class="data-check"`) and a bulk delete button. Both carry `kind` and `filter` in hidden inputs so the delete POST knows which table to re-render.
- `data_detail.html`: the offcanvas panel body. One form serves both modes — `view` renders the `fieldset` disabled with an "Editar" button that fetches the same record with `mode=edit`. A reminder shows date + time, a to-do a `#todo-items` editor with add/remove rows, a plain note only the content.
- `data_item_row.html`: a single to-do row (text input + done checkbox + delete button). Existing rows render it with `{{ template }}` and "Adicionar item" fetches the same partial from `/data/item-row` — one row markup, not two.

### 12. Prompt Service (validation harness) — `internal/services/prompt.go` + `prompt_controller.go`
- `PromptService` orchestrates the Jev validation harness: `Add` (validates flow ∈ {classification, name, note} and non-empty fields, `ErrInvalidPrompt` → 400), `ListByFlow`, `Evaluate` (loads prompts by flow, filters to selected ids, runs the exact production paths — `ClassificationService.Classify` for classification, `ContactExtractor.ExtractName` for name (with the phone/email spans stripped first, exactly as `ContactService.Add` does, so the email/phone is never a name segment) — and `NoteExtractor.ExtractType` for note (expected/obtained are the sub-type, compared case-insensitively) — and compares expected vs obtained; a Jev failure for one prompt is captured in its row as the obtained result with match=false and evaluation continues), and `ExportCSV` (writes the given evaluation results to `exports/<flow>-<yyyyMMdd-HHmmss>.csv` via `encoding/csv`, folder created on demand — no re-run, the CSV mirrors the evaluation the user just saw).
- `PromptController` is thin: `Page` renders the page, `Table` renders the `prompt_table` partial, `Add` persists and re-renders the table, `Evaluate` renders `evaluation_results` (and, when the form's export checkbox is set, saves the CSV first). Bind failures → 400, invalid prompt → 400, service failures → 500 (existing `renderServiceError`).
- The harness reuses the exact production Jev paths — no new request-building code.

### 13. Data Screen — `internal/services/data.go` + `internal/controllers/data_controller.go`
- `DataService` is the browse/edit/delete use case for persisted contacts and notes. It composes `ContactRepository` and `NotesRepository` directly instead of going through `ContactService`/`NotesService`, which carry classification-flow extraction logic that means nothing here. It reuses the note flow's `ParseDate`/`ParseTime` so there is still one PT-BR date implementation.
- `DataService` is the filter gate: `ListContacts` takes `all`/`phone`/`email`/`name`, `ListNotes` takes `all`/`note`/`reminder`/`todo`, and anything else is `ErrInvalidFilter`. The repositories deliberately do not validate — `ContactRepository.List` falls back to all rows, and the notes side is a pair of methods (`List` / `ListByType`) the service chooses between.
- `UpdateContact` writes name/phone/email. `UpdateNote` writes the content plus the date and time, re-parsed from the form text through the same `parseDate`/`parseTime` pair the add path uses, and rebuilds the to-do items from the parallel `ItemText`/`ItemDone` slices in one transaction. Unparseable input is `ErrInvalidData`.
- `DataController` is HTTP-only: `Page` → `RenderPage`, while `Table`/`Detail`/`Update`/`Delete`/`ItemRow` bind, delegate and re-render. Path segments are validated before use (`path` and `renderTable` both reject an unknown `kind`); errors map to 400 (`ErrInvalidFilter`, `ErrInvalidData`, a malformed id), 404 (`repository.ErrNotFound`) or the shared 502/500 mapper.
- Update and delete re-render the record or the table they were called from, reading `kind`/`filter` back off the form — no redirect and no client-side state to keep in sync.
- No Jev call happens anywhere on this screen.

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
  │     ├─► notes + add/both     → NotesService.Add
  │     │     ├─► NoteExtractor.ExtractType → Jev call #2 (note.json)
  │     │     ├─► note → repo.Create(note) (content only)
  │     │     ├─► reminder → ParseDate (date required) + ParseTime (optional) → repo.Create(note)
  │     │     ├─► todo → SplitTodoItems → repo.Create(note, items) (transaction)
  │     │     └─► unparseable date / unknown type → outcome ActionNoteNoData
  │     ├─► notes + require     → NotesService.Get
  │     │     ├─► date in message → repo.FindByDate
  │     │     ├─► "falta/pendente/não fiz" → repo.FindUnfinished
  │     │     ├─► else term → repo.FindByTerm (LIKE, stopwords stripped)
  │     │     ├─► found → outcome ActionNoteFound + notes + SearchTerm
  │     │     └─► not found → outcome ActionNoteNotFound
  │     └─► other category      → outcome ActionNone
  ├─► outcome.Classification → views.ResultData
  └─► views.RenderResult ──► c.HTML(200, "resultado", ...) ──► htmx swaps #resultado innerHTML
```

Up to two TypeSafe API calls are made per request: the classification call, and (for contact add with extractable data, a require-by-name search, or a notes add) the second call — the name fan-out or the `note.json` sub-type call. All persistence goes through `ContactRepository` — `repo.Create` for saves, `repo.FindByPhone` / `repo.FindByEmail` / `repo.FindByName` for lookups, and `repo.ListNeedingNameNorm` + `repo.Save` for the startup `NameNorm` backfill. Contacts are persisted to a local SQLite database (`DB_PATH`, default `contacts.db`); the require flow searches by phone/email/name and duplicate saves are detected before name extraction. Notes are persisted the same way (notes + todo_items tables); the require flow searches by date, unfinished items or a content term. The result partial always shows the classification block and adds an action-specific block (saved contact with ID, fields, the per-segment extraction trace, found/not-found/duplicate messages, or the no-data message).

```
Browser (/prompts)
  │ select flow → hx-get /prompts/table?flow=classification
  ▼
PromptController.Table → PromptService.ListByFlow → prompt_table partial (checkboxes)
  │ check rows → "Avaliar" → hx-post /prompts/evaluate {flow, ids[], export?}
  ▼
PromptController.Evaluate → PromptService.Evaluate
  │   ├─ per id: ClassificationService.Classify  (or ContactExtractor.ExtractName, or NoteExtractor.ExtractType)
  │   ├─ match = expected vs obtained (format per flow)
  │   ├─ per-row error capture (upstream → obtained=error, match=false)
  │   ├─ if export checkbox set → PromptService.ExportCSV(results) → exports/<flow>-<timestamp>.csv
  → evaluation_results partial (✓/✗ per row + CSV path when exported)
```

```
Browser (/data)
  │ GET /data → page with filter pills, empty #data-table, #dataPanel offcanvas
  ▼
DataController.Page → views.RenderPage("data") → page boots → hx-get /data/table?kind&filter
  ▼
DataController.Table → DataService.ListContacts / ListNotes → contacts_table / notes_table partial
  │ row "Ver"  → hx-get /data/:kind/:id?mode=view  → offcanvas body, fieldset disabled
  │ "Editar"   → hx-get /data/:kind/:id?mode=edit  → same form, fieldset enabled
  │ "Salvar"   → hx-post /data/:kind/:id → DataService.UpdateContact / UpdateNote → detail back in view mode
  │ "Adicionar item" → hx-get /data/item-row → one blank row appended to #todo-items
  │ checkboxes + "Apagar selecionados" → hx-post /data/delete (hx-include #data-table-form)
  ▼
DataController.Delete → DataService.DeleteContacts / DeleteNotes → table re-rendered for kind+filter
```

Every data-screen response is an HTML partial; no action re-renders the page itself.
```

## Error Handling

| Failure | Status | Response |
|---|---|---|
| Request bind failure | 400 | `error` partial |
| Jev/TypeSafe API or response failure (`ErrUpstream`) | 502 | `error` partial |
| Any other service failure | 500 | `error` partial |
| No phone/email extractable | 200 | `resultado` partial (no-data branch) |
| Contact not found | 200 | `resultado` partial (not-found branch) |
| No date in a reminder / unknown note sub-type | 200 | `resultado` partial (no-data branch) |
| Note not found | 200 | `resultado` partial (note not-found branch) |
| Duplicate contact on save | 200 | `resultado` partial (duplicate branch) |
| Data-screen bad filter / kind / id / form field (`ErrInvalidFilter`, `ErrInvalidData`) | 400 | `error` partial |
| Data-screen record not found | 404 | `error` partial |
| Database failure | 500 | `error` partial |
| Success | 200 | `resultado` partial |

All responses to htmx targets are HTML partials — no JSON on this route. The response-targets extension makes htmx swap 4xx/5xx responses into `#resultado`. No panics exist in the request path.

## External Integrations

| Service | Purpose | Config |
|---|---|---|
| TypeSafe System One API | Message classification (Jev model) | `TYPESAFE_API_URL`, `TYPESAFE_MODEL`, `TS_API_KEY` |
| htmx.org 2.0.10 (CDN) | Client-side partial page updates | — |
| htmx-ext-response-targets (CDN) | Swap 4xx/5xx responses into `#resultado` | — |
| Bootstrap 5.3 (CDN, default light theme) + Bootstrap Icons (CDN) | Styling | — |
| SQLite (glebarez/sqlite) | Contact, note and to-do persistence | DB_PATH |

## Configuration

| Variable | Source | Used by |
|---|---|---|
| `TYPESAFE_API_URL` | `local.env` | `cmd/api/main.go` → `jev.NewClient` (POST target) |
| `TYPESAFE_MODEL` | `local.env` | `cmd/api/main.go` → `jev.NewClient` (model field) |
| `TS_API_KEY` | environment (not in `local.env`) | `cmd/api/main.go` → `jev.NewClient` (Bearer token) |
| DB_PATH | local.env (default contacts.db) | cmd/api/main.go → gorm.Open (contacts, notes, to-dos; also the data screen) |

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
- Tests exist for `pkg/jev`, `internal/models`, `internal/config`, `internal/services`, `internal/views`, and `internal/repository` (run with `go test ./...`). `internal/controllers` has no tests — its handlers are a thin bind/render shell over `DataService`.
- Go and template files uniformly lack a trailing newline at EOF, so `gofmt -l` always lists ~17 files. That is expected: read the real diff, not the list.
- `.gitignore` ignores `**/*_bin.exe`, `thoughts/`, `*.db`, `.vscode`, and `exports/`.
# ARCHITECTURE

## Overview

A Go web application that classifies user messages into categories (contact, finance, schedule, notes, other) and determines whether a message is a request to add or require something. Classification is performed by the **TypeSafe System One API** (Jev model) via one AI request per message (a single template with two choice questions). After classification, a **Dispatcher** routes the message to a category-specific use case; the contact flow implements both the **add** path (regex extraction of phone/email, Jev Noul name extraction, duplicate detection on save, and SQLite persistence via Gorm) and the **require** path (search by phone/email/name). The frontend is a server-rendered htmx page styled with Web Awesome 3.14.0 (no JS build step).

The same `CategoryHandler` seam now also serves the **notes** flow (`NotesService`): a second Jev call (`note.json`, one `note_type` choice question) picks the sub-type (note / reminder / to-do list), a deterministic PT-BR parser extracts the reminder date and optional time and the to-do list is split into items, everything is persisted in `notes` + `todo_items` (SQLite via Gorm), and the **require** path searches by date, unfinished items or a content term.

And the **finance** flow (`FinanceService`): a second Jev call asks for the transaction type (compra / venda / pagamento / recebimento / transferencia) and, in the same request, fans Noul questions over the words of the establishment candidate; `ParseAmount` and `ParseEventDate` supply the amount and the date, the transaction is persisted in a `transactions` table with the original message kept verbatim, and the **require** path composes a period + type + term filter, summing on demand when the message asks "quanto".

A third surface, **`/data`**, browses and edits what those flows persisted, one tab per kind. `DataService` composes `ContactRepository`, `NotesRepository` and `TransactionRepository` directly — `ContactService`/`NotesService`/`FinanceService` belong to the classification flow and are not reused here — and `DataController` exposes it as htmx partials: filter pills, a search box on the finance tab, a per-record edit panel and bulk delete. No Jev call happens on this screen.

Since DC-009 the same core is served by **three entrypoints** — `cmd/web` (the htmx app), `cmd/api` (a read-only JSON REST API with swagger) and `cmd/cli` (a terminal REPL) — all wired through `internal/app`. Each entrypoint brings its own presenter (`internal/views`, `internal/api`, `internal/cli`) and none of them re-implements business logic or PT-BR formatting.

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
| CSS | Web Awesome 3.14.0 (CDN: theme + native + utilities) + custom layer (web/static/css/app.css) |
| Database | SQLite via [glebarez/sqlite](https://github.com/glebarez/sqlite) (pure-Go, zero CGO) |
| ORM | [gorm.io/gorm](https://gorm.io) |
| API spec | [swaggo/gin-swagger](https://github.com/swaggo/gin-swagger) v1.6.1 + [swaggo/files](https://github.com/swaggo/files) v1.0.1 — swagger 2.0 UI at `/swagger` |

## Directory Structure

```
msg-classifier/
├── cmd/
│   ├── api/
│   │   └── main.go              # REST entrypoint (DC-009): JSON routes + swagger UI, nothing else
│   ├── cli/
│   │   └── main.go              # Terminal REPL entrypoint (DC-009): one line in, plain-text outcome out
│   └── web/
│       └── main.go              # Composition root is internal/app; this is templates + routes only
├── internal/                    # Private application code (not importable externally)
│   ├── api/                     # C — the JSON presenter surface over the same core (no views import)
│   │   ├── presenter.go         # Outcome() → MessageResponse + the per-action PT-BR summary
│   │   ├── errors.go            # ErrorResponse {"error":…} + StatusFor (upstream→502, filter/data→400, not found→404)
│   │   ├── message_controller.go# POST /api/v1/messages (bind → Classify → Dispatch → JSON)
│   │   └── data_controller.go   # read-only GET /contacts, /notes, /transactions over DataService
│   ├── app/
│   │   └── app.go               # composition root: config → jev → db → repos → services → dispatcher
│   ├── cli/                     # C — the terminal presenter over the same core (no views/api import)
│   │   ├── render.go            # outcome → plain-text lines (classification block + action block + trace)
│   │   └── repl.go              # msg> loop: scan a line → Classify → Dispatch → print; errors keep it alive
│   ├── config/
│   │   └── env.go               # Env singleton (TYPESAFE_API_URL, TYPESAFE_MODEL, TS_API_KEY, DB_PATH) — single godotenv load site
│   ├── models/                  # M — data structures
│   │   ├── message.go           # ReceiveMessageRequest DTO + Classification domain struct + UseCaseOutcome/Action
│   │   ├── contact.go           # Contact entity + SegmentScore
│   │   ├── note.go              # Note + TodoItem entities + note type constants
│   │   ├── transaction.go      # Transaction entity + transaction type constants
│   │   ├── prompt.go            # JevPrompt entity + Flow constants + EvaluationResult + form DTOs
│   │   └── data.go              # DataForm + DataDeleteForm (data screen forms)
│   ├── repository/              # Persistence — all gorm queries
│   │   ├── contact.go           # ContactRepository — all gorm queries
│   │   ├── note.go              # NotesRepository — atomic Create(note, items), FindByDate, FindUnfinished, FindByTerm
│   │   ├── transaction.go      # TransactionRepository — Create/Find/List/Sum/Save/DeleteByIDs
│   │   └── prompt.go            # PromptRepository — Create, ListByFlow
│   ├── ptbr/                         # Deterministic PT-BR text — pure functions, no Jev, no DB
│   │   ├── dateparse.go              # ParseDate / ParseTime / ParseRange / ParseEventDate + DateLayout + StartOfDay
│   │   ├── money.go                  # ParseAmount (PT-BR monetary forms)
│   │   ├── format.go                 # MoneyBRL + DateBR — the render side of the parsers, shared by web and cli
│   │   └── text.go                   # NormalizeName (accent fold) + TrimSegment / StripPunctuation / IsCapitalized
│   ├── jevq/                         # The Jev call layer shared by every flow
│   │   ├── jev.go                   # ErrUpstream + the Client/Requester seams + AnswerChoice / AnswerNoul
│   │   └── fanout.go                # Span + SegmentQuestion + NoulSegments + SegmentKey
│   ├── services/                     # M — business rules
│   │   ├── classification.go         # ClassificationService (single Jev call, checked answer mapping)
│   │   ├── dispatcher.go             # Dispatcher (category → handler registry) + CategoryHandler interface
│   │   ├── prompt.go                 # PromptService (validation harness: Add, ListByFlow, Evaluate, ExportCSV)
│   │   ├── data.go                   # DataService (browse/edit/delete over ContactRepository + NotesRepository + TransactionRepository)
│   │   ├── contact/                  # the contact use case, end to end
│   │   │   ├── contact.go            # ContactService (add + duplicate check + NameNorm backfill; require routing)
│   │   │   ├── contact_get.go        # ContactService.Get (require flow: phone → email → name search)
│   │   │   └── extract.go            # ContactExtractor (regex phone/email + Jev Noul name fan-out) + particle post-filter
│   │   ├── notes/                    # the notes use case, end to end
│   │   │   ├── note.go               # NotesService (struct + Handle + Add: note / reminder / todo)
│   │   │   ├── note_get.go           # NotesService.Get (require flow: date → pending → term search)
│   │   │   ├── note_extract.go       # NoteExtractor (Jev note_type choice call)
│   │   │   └── todo_split.go         # SplitTodoItems (newlines → numbered → commas/semicolons)
│   │   └── finance/                  # the finance use case, end to end
│   │       ├── finance.go            # FinanceService (struct + Handle + Add)
│   │       ├── finance_get.go        # FinanceService.Get (require flow: range → type → term, carries the total)
│   │       └── finance_extract.go    # FinanceExtractor (Jev transaction_type + party Noul fan-out)
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
│   │   └── css/app.css        # custom layer over Web Awesome (tokens, pills, sidebar, htmx indicator, tables)
│   └── templates/
│       ├── layouts/base.html    # "base" layout (pt-BR, wa-page shell, favicon, nav slot)
│       ├── pages/index.html     # home page (hero copy + form + spinner on Enviar)
│       ├── pages/prompts.html   # "Validação Jev" page (flow select + add form + spinner)
│       ├── pages/data.html      # "Dados" page (wa-tab-group + filter pills + table container + detail drawer)
│       └── partial/
│           ├── result.html      # "resultado" partial (badges + structured card + segment list)
│           ├── error.html       # "error" partial (red error card for htmx swap targets)
│           ├── prompt_table.html        # checkbox table + Avaliar button + export checkbox + spinner
│           ├── evaluation_results.html  # expected vs obtained comparison + match badges + segment rows + CSV path
│           ├── contacts_table.html      # contact rows + per-row "Ver" + bulk delete
│           ├── notes_table.html         # note rows (type badge, date/time, items) + per-row "Ver" + bulk delete
│           ├── transactions_table.html  # transaction rows (type badge, date, amount, party) + per-row "Ver" + bulk delete
│           ├── data_detail.html         # drawer panel body: contact/note/transaction form in view + edit mode
│           ├── data_item_row.html       # one to-do item row (existing rows reuse it via {{ template }})
├── scripts/
│   └── sql/
│       └── seed_prompts.sql    # wipe + re-seed jev_prompts examples
├── go.mod / go.sum              # Module "msg-classifier", Go 1.25.4
├── local.env                    # Env vars (not committed secrets)
├── .vscode/launch.json          # Go debug config for cmd/web/main.go
├── README.md
└── docs/
    ├── docs.go                 # generated by swag init; its init registers the spec with gin-swagger
    ├── swagger.json            # generated REST spec (committed so `go build ./...` works without the swag CLI)
    ├── swagger.yaml            # generated REST spec, YAML flavour
    ├── decisions/DC-005.md     # notes, reminders, to-do lists
    ├── decisions/DC-006.md     # the /data screen
    ├── decisions/DC-007.md     # finance transactions
    ├── decisions/DC-008.md     # duplicate-data validation (not implemented)
    ├── decisions/DC-009.md     # multi-entry: web, REST api, cli over one core
    └── todos.md                # backlog for the notes use case (relative dates, deadlines, accent-insensitive search)
```

## Core Components

### 1. Composition Root — `internal/app/app.go`
- `app.New(dbPath string) (*App, error)` is the one wiring point every entrypoint calls: `config.GetEnv()` (single godotenv load site) → `jev.NewClient(url, token, model)` → `gorm.Open(sqlite.Open(dsn(dbPath)))` → `db.DB()` → `sqlDB.SetMaxOpenConns(1)` → `db.AutoMigrate(&models.Contact{}, &models.JevPrompt{}, &models.Note{}, &models.TodoItem{}, &models.Transaction{})` → `services.NewClassificationService(client)` plus the `contact`/`notes`/`finance` extractors → the four repositories → `contact.NewService` → `contactService.BackfillNameNorm()` → `services.NewDispatcher` (registry: `"contact"` → `ContactService`, `"notes"` → `NotesService`, `"finance"` → `FinanceService`), along with `services.NewDataService` and `services.NewPromptService`. The contact, notes and transaction repositories are built once and shared with the data screen. `dbPath` is a parameter rather than a config read so a test can pass `:memory:`.
- It returns an `App` carrying `Classifier`, `Dispatcher`, `Data` and `Prompts`, and every failure comes back as a wrapped error (`failed to open database %q`, `failed to get database pool`, `failed to migrate database`, `failed to backfill name_norm`) instead of killing the process — `BackfillNameNorm` used to `log.Fatalf` here, so the entrypoint now owns that decision.
- `cmd/web/main.go` is the web entrypoint and nothing more: the `SourcePath = "web/templates"` const, `router.Static("/static", "./web/static")`, template parsing + `views.NewPagesRenderer` installed as `router.HTMLRender`, `app.New(config.GetEnv().DBPath)`, the four controllers, the route table below, and `router.Run(":8080")`.
- SQLite pool is capped at one connection (`SetMaxOpenConns(1)` right after `gorm.Open`) — DB access is serialized within the process; the pure-Go driver (glebarez/modernc) is unstable with concurrent connections on Windows. The cap is per-process, so it stays even though the file itself is now shared.
- A file-backed DSN also carries two pragmas — `file:<path>?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)` — because three processes write one SQLite file: `busy_timeout(5000)` turns a lock collision into a 5s wait instead of an error, and WAL lets a reader work while a writer commits. Without them the cost of three writers is an intermittent "database is locked" in production rather than in a test. An in-memory DSN is passed through untouched.
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
- `MessageController.ReceiveMessage`: binds `ReceiveMessageRequest` (failure → 400 error partial) → calls `ClassificationService.Classify` → calls `Dispatcher.Dispatch` → maps errors via `renderServiceError` (`errors.Is(err, jevq.ErrUpstream)` → 502, else 500) → renders result or error partial. No business logic, no Jev types, no template name literals.

### 4. Classification Service — `internal/services/classification.go`
- `ClassificationService.Classify`: builds Jev state `{user, message}`, makes **one** Jev call (`classification.json`, which contains both the `classification` and the `adding_or_requiring` choice questions), extracts answers with checked assertions (`jevq.AnswerChoice`), returns a domain `Classification`.
- `ErrUpstream` sentinel marks failures originating from the Jev/TypeSafe API or its responses; controllers map it to HTTP 502.
- The `jevClient` interface (defined at the service boundary) makes the service unit-testable without HTTP.

### 5. Dispatcher — `internal/services/dispatcher.go`
- `CategoryHandler` interface: `Handle(request, classification) (*models.UseCaseOutcome, error)` — the seam for category-specific use cases.
- `Dispatcher` holds a `map[string]CategoryHandler` keyed by `Category.Choice`; `Dispatch` looks up the handler, falling back to an `ActionNone` outcome on a miss. Adding a category = new service implementing `CategoryHandler` + one wiring line in `internal/app/app.go`.

### 6. Contact Service — `internal/services/contact/contact.go` + `contact_get.go`
- `ContactService` implements `CategoryHandler` for the `contact` category. Constructor takes a `*ContactExtractor` and a `*repository.ContactRepository` — the repository owns all gorm queries, and the service wraps repo errors with the same context strings (`failed to check duplicate contact`, `failed to persist contact`, `failed to search contact`, `failed to load contacts for backfill`, `failed to backfill name_norm`). `Handle` routes `require` → `Get`, everything else → `Add`. `Add` extracts phone/email → neither found → `ActionContactNoData` outcome; a duplicate check (phone, then email) runs **before** name extraction — a match returns `ActionContactDuplicate` + the existing contact (no Jev call); otherwise name extraction runs and the repository inserts the contact (populating `NameNorm`) → `ActionContactAdd` outcome carrying the saved contact and the segment trace (`outcome.Segments`). `Get` (require flow) searches with phone → email → name priority; phone/email searches skip Jev entirely; name search uses `LIKE %term%` on `name_norm`; `SearchTerm` is set on the outcome and not-found is an outcome, not an error. `BackfillNameNorm()` recomputes `NameNorm` for pre-migration rows at startup.

### 6b. Contact Extractor — `internal/services/contact/extract.go`
- `ContactExtractor` extracts phone (BR regex, normalized to 10/11 digits) and email (first match + span) deterministically, and the name via one dynamic Jev request with a Noul question per whitespace segment (`segment_0..N`). `ExtractName` returns a `NameResult` — the joined name plus a per-segment trace (`SegmentScore`: text, noul score, `Included`). A deterministic post-filter resolves name particles by position, since the Noul scores for `do`/`da`/`de` hover around the 0.5 threshold and flip between runs: (1) drop leading lowercase segments while a capitalized one follows (a proper name never starts with a function word, so `do João da Silva` → `João da Silva`); (2) rescue a particle Jev excluded when it sits between two included capitalized segments (`José Carlos de Souza` keeps its `de`). The trace's `Included` flags are updated to match. Depends on a minimal `jevRequester` interface (`MakeJevRequest`) so tests mock the Jev call.

### 6c. Contact Repository — `internal/repository/contact.go`
- `ContactRepository` owns all gorm queries for the `Contact` entity; `NewContactRepository(db *gorm.DB)` wraps the DB handle. Methods: `Create` (persist a new contact), `FindByPhone` / `FindByEmail` (case-insensitive) / `FindByName` (`name_norm LIKE %term%`), `ListNeedingNameNorm` (empty/NULL `name_norm`, pre-migration rows), and `Save` (backfill updates). `ErrNotFound = gorm.ErrRecordNotFound` is the not-found sentinel — services detect it with `errors.Is` without importing gorm; all other errors are returned raw and wrapped by the service with its context strings.
- The data screen added `List(filter)` / `FindByID` / `DeleteByIDs`. `List` orders `id DESC` (newest first) and falls back to all rows on an unknown filter instead of erroring, so `DataService` stays the gate that rejects one.

### 6d. Notes Service — `internal/services/notes/note.go` + `note_get.go`
- `NotesService` implements `CategoryHandler` for the `notes` category. `Handle` routes `require` → `Get`, everything else → `Add`. Constructor takes the `*NoteExtractor` and the `*repository.NotesRepository`; repo errors are wrapped with `failed to persist note` / `failed to search notes`.
- `Add` asks Jev for the sub-type, then: **note** stores content only; **reminder** requires a date (optional time) and returns `ActionNoteNoData` when the date is unparseable; **todo** splits the content into ordered items. An unknown sub-type is also `ActionNoteNoData`. Success returns `ActionNoteAdd` carrying the saved note in `outcome.Notes`.
- `Get` (require) picks one filter from the message, in order: a parsed date (`FindByDate`, including `ontem`), an unfinished marker (`falta`/`faltam`/`pendente`/`pendentes`/`não fiz`/`ainda não` → `FindUnfinished`), otherwise a content term (`FindByTerm`, `LOWER(content) LIKE %term%`) built by stripping PT-BR stopwords from the accent-normalized message, with a retry on the last word when the phrase misses. Found → `ActionNoteFound` + the notes list; not found → `ActionNoteNotFound`; nothing extractable → `ActionNoteNoData`.
- Content is always the trimmed original message; the date, the time and the items live in their own columns.

### 6e. Note Extractor / Date Parser / Todo Splitter — `internal/services/notes/note_extract.go`, `todo_split.go`, `internal/ptbr/dateparse.go`
- `NoteExtractor.ExtractType` makes the **second** Jev call of a notes message (`note.json`, one `note_type` choice answer) reusing the `jevq.Client` seam and `jevq.AnswerChoice`; the raw choice is validated by the service.
- `ParseDate` / `ParseTime` are deterministic and dependency-free package functions (no struct, nothing to inject): `dd/mm/aaaa`, `dd/mm` (current year), `dia N` (current month), `hoje`, `amanhã`, `ontem`; times as `14h`, `14h30`, `14:00`. Every date is normalized to UTC midnight so the equality filter on `notes.date` compares identically. `now` is a parameter, not `time.Now()`, to keep the relative forms testable.
- `SplitTodoItems` splits on newlines first, then numbered markers (`1. `, `2) `), then commas/semicolons, trimming punctuation and leading numbers/conjunctions; the service assigns `Position` from the slice order. Two more rules, both gated on the same "this is a list" signal (a `,`/`;` survived, or a label was dropped): a leading marker (`Comprar:`, `Tarefas:`) is dropped when a list follows it — a bare `Tarefas:` is left with nothing, which the service answers as no-data — and the `e` conjunction separates items only in an inline list. Line and numbered lists keep one item per line, and a single statement like `comprar pão e leite` stays one item.

### 6f. Finance Service — `internal/services/finance/finance.go` + `finance_get.go`
- `FinanceService` implements `CategoryHandler` for the `finance` category. `Handle` routes `require` → `Get`, everything else → `Add`. Constructor takes the `*FinanceExtractor` and the `*repository.TransactionRepository`; repo errors are wrapped with `failed to persist transaction` / `failed to search transactions`.
- `Add` asks Jev for the transaction type, parses the amount with `ParseAmount` (no amount → `ActionTransactionNoData` saying what is missing) and the date with `ParseEventDate`, defaulting to today when the message carries none. The repository inserts the transaction and the outcome carries it in `outcome.Transaction`.
- `Get` (require) builds one composed `repository.TransactionFilter` — the type named in the message (`financeTypeWords`: comprei/vendi/paguei/recebi/transferi…), the `ParseRange` window (`From`/`Until`, which is how "esse mês" and "semana passada" are answered), and the content term — because "quanto gastei com mercado esse mês" is all three at once. A multi-word term that misses is retried with the term dropped (grammar vs. name, see the ponytail note); a single-word miss stays a miss. Found → `ActionTransactionFound` + the list; not found → `ActionTransactionNotFound`; nothing extractable → `ActionTransactionNoData`. `SearchTerm` carries a human-readable `filterLabel` so the partial can show what was searched for.
- When the message asks how much (`quanto`, `total`, …) the outcome carries `outcome.Total` from `TransactionRepository.Sum(filter)` and the partial shows it.
- Amounts are always stored positive — the transaction type carries the direction, so a `recebimento` is not a negative number and `Sum` is a plain sum per filter rather than a signed one.

### 6g. Finance Extractor / Money Parser — `internal/services/finance/finance_extract.go`, `internal/ptbr/money.go`
- `FinanceExtractor.Extract` makes **one** dynamic Jev request carrying two question kinds: a `transaction_type` choice plus a `segment_N` Noul fan-out over the party candidate, so the party refines in the same round trip. The party candidate is regex-anchored on the preposition that introduces an establishment ("no"/"na"/"do"/"da"/"para o"…) and junk-filtered; `SegmentKey`/`NoulSegments` are shared with the contact name extraction rather than reimplemented. If the mixed request fails and there were segments to fan out, it retries with the type question alone and continues without a party — the type is what matters.
- `ParseAmount` matches the PT-BR monetary forms in order — `R$ 50,00`, `50 reais`, then a bare number carrying a decimal or thousands separator. Requiring one of those markers is what stops "3 vezes de 300 reais" reading the installment count as the amount and "10/10/2023" reading as money. Dots are thousands separators and the comma is the decimal mark, so the same parser accepts what the UI renders through `ptbr.MoneyBRL`.
- ponytail: the party is a regex-anchored guess refined by the fan-out, not a parse of the sentence — a candidate with no preposition yields no party, and a preposition introducing something else is caught by the junk wordlist. Asking Jev to return the party as a quoted substring is the way past it.

### 6h. Transaction Repository — `internal/repository/transaction.go`
- `TransactionRepository` owns every gorm query for `Transaction`. `Create`, `Find(filter)` and `List(filter)` over a `TransactionFilter{Type, Term, From, Until}` (type equality, a `From`-inclusive/`Until`-exclusive date window, `Term` as a `LIKE` on the content or the party, ordered newest first), `Sum(filter)` for the "quanto gastei" answer, `FindByID`, `Save` and `DeleteByIDs`. `ErrNotFound` is the shared `repository.ErrNotFound` sentinel, like the contact side.

### 7. Models — `internal/models/message.go` + `contact.go` + `note.go` + `transaction.go`
- `message.go`: `ReceiveMessageRequest` inbound DTO; `Classification` domain struct (`CategoryFinding` / `KindFinding` with raw numeric confidences; `KindFinding.Choice` carries the kind: `"add"` / `"require"` / `"both"`); `UseCaseOutcome` (`Classification` + `Action` + `SearchTerm`) — the seam where use-case results (extracted contact, DB confirmation, search term) flow back without signature changes; actions are `ActionNone` / `ActionContactAdd` / `ActionContactNoData` / `ActionContactFound` / `ActionContactNotFound` / `ActionContactDuplicate` / `ActionNoteAdd` / `ActionNoteNoData` / `ActionNoteFound` / `ActionNoteNotFound`. There is no response DTO: the result partial receives the `Classification` and formats it.
- `contact.go`: `Contact` Gorm entity (ID, Name, `NameNorm` column, Phone/Email nullable, timestamps) and `SegmentScore` (text, noul score, included flag); `UseCaseOutcome` carries `Contact` and `Segments` (nil unless name extraction ran).
- `note.go`: `Note` (ID, Type, Content, nullable `Date`/`Time`, `Items`, timestamps) and `TodoItem` (ID, `NoteID` FK with `ON DELETE CASCADE`, Text, `Done`, `Position`); indexes on `notes.type`, `notes.date` and `todo_items.done`. `UseCaseOutcome` also carries `Notes []*Note` (the add path returns one element); the contact constants are unchanged.
- `transaction.go`: `Transaction` (ID, `Type` indexed, `Amount`, `Date` indexed and normalized to UTC midnight, `Party` indexed — the establishment or person, `Content` holding the original message verbatim, timestamps) and the five type criteria (`compra`, `venda`, `pagamento`, `recebimento`, `transferencia`) plus `IsTransactionType`. `UseCaseOutcome` carries `Transactions []*Transaction` and `Total float64` (only when the message asked how much).
- `data.go`: `DataForm` and `DataDeleteForm`. Both contact and note edits bind to the same `DataForm` (`Name`/`Phone`/`Email` for contacts; `Content`/`Date`/`Time` plus parallel `ItemText []string` / `ItemDone []string` slices for a to-do). `DataDeleteForm` carries `Kind`, `Filter` and `IDs []uint` — a `[]uint` form field binds repeated `ids` values. `form` tags only; nothing here is part of a JSON API.

### 8. Views — `internal/views/render.go` + `pages_renderer.go`
- Template name constants (`base`, `page:content`, `resultado`, `error`, `contacts_table`, `notes_table`, `transactions_table`, `data_detail`, `data_item_row`) — no string literals at call sites.
- `RenderPage`: renders `page:content` when the `HX-Request` header is `true`, full `base` otherwise. The header is the whole htmx detection there is — no middleware, no server-side htmx dependency.
- `RenderResult(c, outcome)` hands the `*models.UseCaseOutcome` straight to the `resultado` partial — there is no result view model, because `UseCaseOutcome` already exposes every field the partial reads and a `ResultData` copy would be a second one to keep in sync. `RenderError` renders the error partial with the status it is given, so htmx swaps the error card into `#resultado`.
- The shared template set registers `views.FuncMap` (`label` for PT-BR badge text, `confidence` for the percent format, `money` for the amount, plus `dateBR`/`deref` for the data screen) — templates call these names, never the Go symbols, so a helper can move packages (as `money`/`dateBR` did, into `internal/ptbr`) without touching a single template.
- The data screen passes templates page-scoped view models — `DataDetailData{Kind, Mode, Contact, Note, Transaction}` and `ItemRowData` — rather than handing entities straight to a partial. `Mode` is `view` or `edit`, and the switch happens in the template through a single `disabled` attribute on the form's `fieldset` instead of two parallel branches.
- `ptbr.MoneyBRL` and `ptbr.DateBR` are the **render half of the parsers**: they live in `internal/ptbr/format.go` beside `ParseAmount`/`ParseDate` so whatever a surface prints is what the parser reads back — `views.FuncMap` wires them to the `money`/`dateBR` template names and `internal/cli` calls them directly. `MoneyBRL` emits `"R$ 1.234,56"`, which is why the transaction edit field is prefilled with `money` rather than `printf "%.2f"` (a bare `50.00` would parse back as 5000). `Deref(*string)`, `Confidence(float64)`, `Label` and `BadgeVariant` stay in `views`: rendering a `*string` directly puts a pointer address (`0xc000…`) in the cell, and label/confidence are screen presentation the CLI does not need. All of them are template helpers, not entity methods.

### 9. Jev API Client — `pkg/jev/jev.go`
- `Client` struct with `NewClient(apiURL, token, model)` — all configuration injected, no `internal/config` import (genuinely reusable).
- `MakeJevRequest`: sets model, validates, POSTs JSON with `Authorization: Bearer <token>`, parses response. 30s HTTP timeout; response body closed on all paths.
- `MakeJevRequestFromFile` / `LoadJevRequestFromFile`: load prompt templates from the embedded `requests/*.json` (via `go:embed` — no CWD-relative path dependency), inject state, delegate.
- `HttpResponseToJevResponse`: decodes `answers` as `map[string]json.RawMessage`, reads each answer's `type` to pick the struct, then unmarshals once; answers land in `map[string]any` as `*JevAnswerChoice` / `*JevAnswerNoul` and callers type-assert. **Never panics** — unknown or missing answer types return an explicit error.
- Two question types exist because two question kinds are in use: `choice` and `noul`. `validateJevRequest` validates state, model, questions, and per-type criteria/true-false fields.

### 10. Jev Request Templates — `pkg/jev/requests/*.json`
- `classification.json`: two `choice` questions — `"classification"` with 5 criteria (contact, finance, schedule, notes, other) and `"adding_or_requiring"` with descriptive criteria (add, require, both).
- `note.json`: one `choice` question — `"note_type"` with 3 criteria (note, reminder, todo). It is only called for messages classified as `notes`, and only on the add path.
- The finance flow has **no** JSON template: its question count depends on how many words the party candidate has, so `finance_extract.go` builds the request in code (`typeQuestion()` + one `segment_N` per candidate word) and sends it through `MakeJevRequest`.

### 11. HTML Templates — `web/templates/`
- `base.html`: `base` layout, `lang="pt-BR"`, Web Awesome 3.14.0 CSS from CDN (theme + native + utilities) + `app.css`, htmx 2.0.10 + response-targets extension from CDN, and the Web Awesome loader as a `type="module"` script. The body is a `<wa-page mobile-breakpoint="992px">` whose `slot="navigation"` carries the brand + nav; the page name marks the active link with `class="active"` (the `page:active` block is gone — `views.PageData.Page` is available directly). `hx-ext="response-targets"` + `hx-target-error="#resultado"` on `<body>` route 4xx/5xx responses into the result container.
- Layout and typography are Web Awesome **CSS utility classes** (`wa-stack`, `wa-cluster`, `wa-gap-*`, `wa-align-items-*`, `wa-justify-content-*`, `wa-heading-*`, `wa-body-*`, `wa-color-text-quiet`, `wa-list-plain`, `wa-form-control-label`, `wa-tabular-nums`) — there are no `wa-stack`/`wa-cluster` *elements*. Tables are plain `<table>` styled by `native.css`.
- Every `<wa-button>` that submits a form carries an explicit `type="submit"`: Web Awesome defaults `type` to `button`, so omitting it silently turns a submit into a no-op.
- Loading feedback stays htmx's: `hx-indicator` + `.htmx-indicator` (see app.css) with a `<wa-spinner>`; buttons also carry `hx-disabled-elt="this"`.
- `index.html`: form posting via `hx-post="/api/message"` targeting `#resultado` with `hx-swap="innerHTML"`.
- `result.html`: `resultado` partial — a `<wa-card>` with PT-BR category/kind `<wa-badge>`s in `slot="header"`, branches for add/found/not-found/duplicate/no-data (contact), note_add/note_found/note_not_found/note_no_data (notes: type badge, content, date/time for reminders, item list with ✓/○ for to-dos) and transaction_add/transaction_found/transaction_not_found/transaction_no_data (finance: type badge, amount through `money`, date, party, the original message, and a `Total:` line when the outcome carries one), and the per-segment extraction trace as a list in `slot="footer"`.
- `error.html`: `error` partial rendering a `<wa-callout variant="danger">` with a `wa-icon` (used for 400/404/502/500 responses).
- `data.html`: `data` page — a `<wa-tab-group>` with three tabs (Contatos / Notas / Finanças), filter pills that each `hx-get` a table partial, an empty `#data-table` container that loads on first paint, and the `#dataPanel` `<wa-drawer>` whose body is swapped per record. A `dataChanged` body event lets the delete endpoint re-render the table the user was on without a full page reload. Pill highlight state is client-side (`markFilterPill`), because the server re-renders `#data-table` but never the pill list — a server-rendered `active` class would be lost on every delete. Switching the outer tab is driven by the `<wa-tab-group>`'s `wa-tab-show` event, which calls `activateFilterPills` and loads that pane's default filter, so the other kind's table is never left on screen. The contacts `Todos` pill (and only it) carries `hx-trigger="load, click"` — it boots the table on first paint but must stay clickable, and an explicit `hx-trigger` replaces htmx's default `click` trigger, so `load` alone would make it a dead button. `activateFilterPills` clicks that default pill unconditionally, with no `active`-pane guard: `wa-tab-group` toggles the pane itself, so such a guard would skip the reload exactly when the tab you re-select is the one already showing. The finance pane adds `#txSearchForm`, a `hx-get` search box over the establishment/person/amount; it carries its own hidden `kind`/`filter` inputs and `markFilterPill` re-points `filter` on every pill click, so a search can never escape the type on screen.
- `contacts_table.html` / `notes_table.html` / `transactions_table.html`: row tables plus a checkbox per row (`class="data-check"`) and a bulk delete button. All three carry `kind` and `filter` in hidden inputs so the delete POST knows which table to re-render. The transaction rows show date, type badge, `money` amount, party and the original message.
- `data_detail.html`: the drawer panel body. One form serves both modes — `view` renders the `fieldset` disabled with an "Editar" button that fetches the same record with `mode=edit`. A reminder shows date + time, a to-do a `#todo-items` editor with add/remove rows, a plain note only the content, and a transaction a type `<wa-select>` + amount + date + party + original message.
- `data_item_row.html`: a single to-do row (text input + done checkbox + delete button). Existing rows render it with `{{ template }}` and "Adicionar item" fetches the same partial from `/data/item-row` — one row markup, not two.

### 12. Prompt Service (validation harness) — `internal/services/prompt.go` + `prompt_controller.go`
- `PromptService` orchestrates the Jev validation harness: `Add` (validates flow ∈ {classification, name, note, finance} and non-empty fields, `ErrInvalidPrompt` → 400), `ListByFlow`, `Evaluate` (loads prompts by flow, filters to selected ids, runs the exact production paths — `ClassificationService.Classify` for classification, `ContactExtractor.ExtractName` for name (with the phone/email spans stripped first, exactly as `ContactService.Add` does, so the email/phone is never a name segment), `NoteExtractor.ExtractType` for note (expected/obtained are the sub-type, compared case-insensitively) and `FinanceExtractor.Extract` for finance (expected/obtained are the transaction type, and the party fan-out's segment trace rides along in the row) — and compares expected vs obtained; a Jev failure for one prompt is captured in its row as the obtained result with match=false and evaluation continues), and `ExportCSV` (writes the given evaluation results to `exports/<flow>-<yyyyMMdd-HHmmss>.csv` via `encoding/csv`, folder created on demand — no re-run, the CSV mirrors the evaluation the user just saw).
- `PromptController` is thin: `Page` renders the page, `Table` renders the `prompt_table` partial, `Add` persists and re-renders the table, `Evaluate` renders `evaluation_results` (and, when the form's export checkbox is set, saves the CSV first). Bind failures → 400, invalid prompt → 400, service failures → 500 (existing `renderServiceError`).
- The harness reuses the exact production Jev paths — no new request-building code.

### 13. Data Screen — `internal/services/data.go` + `internal/controllers/data_controller.go`
- `DataService` is the browse/edit/delete use case for persisted contacts, notes and transactions. It composes `ContactRepository`, `NotesRepository` and `TransactionRepository` directly instead of going through `ContactService`/`NotesService`/`FinanceService`, which carry classification-flow extraction logic that means nothing here. It reuses the note flow's `ParseDate`/`ParseTime`, the finance flow's `ParseAmount`/`ParseEventDate`, so there is still one PT-BR date and money implementation.
- `DataService` is the filter gate: `ListContacts` takes `all`/`phone`/`email`/`name`, `ListNotes` takes `all`/`note`/`reminder`/`todo`, `ListTransactions` takes `all` or one of the five transaction types plus an optional search term, and anything else is `ErrInvalidFilter`. The repositories deliberately do not validate — `ContactRepository.List` falls back to all rows, the notes side is a pair of methods (`List` / `ListByType`) the service chooses between, and `TransactionRepository.List` applies a `TransactionFilter` as given.
- `UpdateContact` writes name/phone/email. `UpdateNote` writes the content plus the date and time, re-parsed from the form text through the same `parseDate`/`parseTime` pair the add path uses, and rebuilds the to-do items from the parallel `ItemText`/`ItemDone` slices in one transaction. `UpdateTransaction` writes type/amount/date/party with the amount and date re-parsed through the finance parsers, so an edit can never store a value the message flow would not have produced. Unparseable input is `ErrInvalidData`.
- `DataController` is HTTP-only: `Page` → `RenderPage`, while `Table`/`Detail`/`Update`/`Delete`/`ItemRow` bind, delegate and re-render. Path segments are validated before use (`path` and `renderTable` both reject an unknown `kind`); errors map to 400 (`ErrInvalidFilter`, `ErrInvalidData`, a malformed id), 404 (`repository.ErrNotFound`) or the shared 502/500 mapper.
- Update and delete re-render the record or the table they were called from, reading `kind`/`filter` off the form — no redirect and no client-side state to keep in sync.
- No Jev call happens anywhere on this screen.

### 14. REST API — `cmd/api/main.go` + `internal/api/`
- `cmd/api/main.go` is the REST entrypoint (DC-009) and nothing else: `config.GetEnv()` → `app.New(env.DBPath)` → a `gin.Default()` router with a `/api/v1` group and `GET /swagger/*any` → `router.Run(":" + env.ApiPort)`. It renders no template and imports neither `internal/views` nor `internal/controllers`, so the JSON binary carries no `html/template`.
- Routes — exactly what `docs/swagger.json` declares, no more:

  | Method | Path | Handler | Answers |
  |---|---|---|---|
  | POST | `/api/v1/messages` | `api.MessageController.ReceiveMessage` | `MessageResponse` |
  | GET | `/api/v1/contacts` | `api.DataController.ListContacts` | `[]models.Contact` |
  | GET | `/api/v1/notes` | `api.DataController.ListNotes` | `[]models.Note` |
  | GET | `/api/v1/transactions` | `api.DataController.ListTransactions` | `[]models.Transaction` |
  | GET | `/swagger/*any` | `ginSwagger.WrapHandler` | swagger 2.0 UI |

  The three browses take `?filter=` (`all|phone|email|name`, `all|note|reminder|todo`, `all|compra|venda|pagamento|recebimento|transferencia`) and the transaction one also `?search=`.
- The surface is **read-only** apart from the classification POST, which may persist whatever the flow saves — there are **no write endpoints** (the `/data` update/delete routes are HTML-only and stay in `cmd/web`) and **no `/prompts`** route: the validation harness is a UI, not an API.
- `api.Outcome` is the presenter — a `switch` over `models.Action` producing `MessageResponse`. `action` is the discriminant (`contact_add`, `note_found`, `transaction_not_found`, `none`, …): a client reads one field instead of inferring the outcome from the payload. Every payload field past `action`/`message`/`classification` is `omitempty`, so an absent one is missing rather than `null` — `models.Note.Items` is `json:"items,omitempty"` too, so a note with no to-do items omits `items` instead of carrying the `[]` the controllers emit for an empty list.
- Every non-2xx body is `{"error":"..."}` (`api.ErrorResponse`). `api.StatusFor` restates the web's mapping rather than importing it: `ErrInvalidFilter`/`ErrInvalidData` → 400, `repository.ErrNotFound` → 404, `jevq.ErrUpstream` → 502, else 500; an unbindable body or an empty `message` → 400.
- The spec is generated by `swag init` into `docs/` and **committed on purpose** — `docs/docs.go` registers it, so `go build ./...` and the tests work on a fresh clone without the `swag` CLI installed, and CI does not install it.

### 15. CLI — `cmd/cli/main.go` + `internal/cli/`

- `cmd/cli/main.go` is the terminal entrypoint (DC-009): `config.GetEnv()` → `app.New(dbPath)` → `cli.New(application.Classifier.Classify, application.Dispatcher.Dispatch, os.Stdin, os.Stdout)` → `runner.Run()`. It reads a line, classifies it through the **same core** as `cmd/web` and `cmd/api`, and prints the outcome as plain text. No template, no HTTP: the CLI binary carries no `html/template` because `internal/cli` imports neither `internal/views` nor `internal/api`.
- `internal/cli/render.go` turns a `*models.UseCaseOutcome` into terminal lines. It shares nothing with `views` but the outcome itself: a `switch` over `models.Action`, no `Presenter` interface, no registry. The **classification block prints raw choices** (`[contact/add]  categoria 94% · intenção 86%`) — the terminal is a developer surface and `contact/add` is more useful than `Contato/Adicionar` — while `ptbr.MoneyBRL`/`ptbr.DateBR` format amounts and dates identically to the web, so `R$ 1.234,56` and `10/05/2026` parse back through `ParseAmount`/`ParseDate`. The segment trace follows as `extração:` with each segment marked ✓/✗.
- `internal/cli/repl.go` owns the interactive loop: `msg> ` prompt, `bufio.Scanner`, one line → one message, `exit`/`sair`/`quit`/EOF to stop. A single failure prints `erro: ...` and the loop keeps going — one hiccup must not end the session. It sends `UserID: "cli"` in the Jev state because the terminal has no form field.
- Each entrypoint is a `cmd` (`web`, `api`, `cli`), and they share `internal/app/app.go` — build any one and the rest of the package stays unused rather than duplicated.

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
  │     ├─► finance + add/both   → FinanceService.Add
  │     │     ├─► party candidate (regex on the preposition) + segments
  │     │     ├─► Jev call #2 → transaction_type choice + segment_N Noul fan-out (one request)
  │     │     ├─► no amount → outcome ActionTransactionNoData
  │     │     ├─► ParseAmount (mandatory) + ParseEventDate (defaults to today)
  │     │     ├─► repo.Create(transaction) (SQLite, amount always positive)
  │     │     └─► outcome ActionTransactionAdd + saved transaction + segment trace
  │     ├─► finance + require   → FinanceService.Get
  │     │     ├─► composed filter: ParseRange window + type word + content term
  │     │     ├─► repo.Find(filter) (multi-word miss → retry without the term)
  │     │     ├─► "quanto/total" → repo.Sum(filter) → outcome.Total
  │     │     ├─► found → outcome ActionTransactionFound + transactions + SearchTerm
  │     │     └─► not found → outcome ActionTransactionNotFound
  │     └─► other category      → outcome ActionNone
  ├─► outcome (*models.UseCaseOutcome) → views.RenderResult (no view model copy)
  └─► views.RenderResult ──► c.HTML(200, "resultado", ...) ──► htmx swaps #resultado innerHTML
```

Up to two TypeSafe API calls are made per request: the classification call, and (for contact add with extractable data, a require-by-name search, a notes add or a finance add) the second call — the name/party fan-out or the `note.json` sub-type call. All persistence goes through `ContactRepository` — `repo.Create` for saves, `repo.FindByPhone` / `repo.FindByEmail` / `repo.FindByName` for lookups, and `repo.ListNeedingNameNorm` + `repo.Save` for the startup `NameNorm` backfill. Contacts are persisted to a local SQLite database (`DB_PATH`, default `contacts.db`); the require flow searches by phone/email/name and duplicate saves are detected before name extraction. Notes are persisted the same way (notes + todo_items tables); the require flow searches by date, unfinished items or a content term. Transactions are persisted in a third table; the require flow searches by period, type and content/party, and sums on demand. The result partial always shows the classification block and adds an action-specific block (saved contact with ID, fields, the per-segment extraction trace, found/not-found/duplicate messages, or the no-data message).

```
Browser (/prompts)
  │ select flow → hx-get /prompts/table?flow=classification
  ▼
PromptController.Table → PromptService.ListByFlow → prompt_table partial (checkboxes)
  │ check rows → "Avaliar" → hx-post /prompts/evaluate {flow, ids[], export?}
  ▼
PromptController.Evaluate → PromptService.Evaluate
  │   ├─ per id: ClassificationService.Classify  (or ContactExtractor.ExtractName, or NoteExtractor.ExtractType, or FinanceExtractor.Extract)
  │   ├─ match = expected vs obtained (format per flow)
  │   ├─ per-row error capture (upstream → obtained=error, match=false)
  │   ├─ if export checkbox set → PromptService.ExportCSV(results) → exports/<flow>-<timestamp>.csv
  → evaluation_results partial (✓/✗ per row + CSV path when exported)
```

```
Browser (/data)
  │ GET /data → page with filter pills, empty #data-table, #dataPanel drawer
  ▼
DataController.Page → views.RenderPage("data") → page boots → hx-get /data/table?kind&filter
  ▼
DataController.Table → DataService.ListContacts / ListNotes / ListTransactions → contacts_table / notes_table / transactions_table partial
  │ row "Ver"  → hx-get /data/:kind/:id?mode=view  → drawer body, fieldset disabled
  │ "Editar"   → hx-get /data/:kind/:id?mode=edit  → same form, fieldset enabled
  │ "Salvar"   → hx-post /data/:kind/:id → DataService.UpdateContact / UpdateNote / UpdateTransaction → detail back in view mode
  │ "Adicionar item" → hx-get /data/item-row → one blank row appended to #todo-items
  │ finance search box → hx-get /data/table {kind, filter, search} → transactions_table partial
  │ checkboxes + "Apagar selecionados" → hx-post /data/delete (hx-include #data-table-form)
  ▼
DataController.Delete → DataService.DeleteContacts / DeleteNotes / DeleteTransactions → table re-rendered for kind+filter
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
| No amount in a finance message | 200 | `resultado` partial (transaction no-data branch) |
| Nothing extractable to search for / transaction not found | 200 | `resultado` partial (transaction not-found branch) |
| Duplicate contact on save | 200 | `resultado` partial (duplicate branch) |
| Data-screen bad filter / kind / id / form field (`ErrInvalidFilter`, `ErrInvalidData`) | 400 | `error` partial |
| Data-screen record not found | 404 | `error` partial |
| Database failure | 500 | `error` partial |
| Success | 200 | `resultado` partial |

All responses to htmx targets are HTML partials — no JSON on this route. The response-targets extension makes htmx swap 4xx/5xx responses into `#resultado`. No panics exist in the request path.

The REST surface (`cmd/api`) answers JSON, and reuses the same status rules through `api.StatusFor`:

| Failure | Status | Response |
|---|---|---|
| Unbindable body / empty `message` on `POST /api/v1/messages` | 400 | `{"error":"..."}` |
| Bad `filter` / bad query (`ErrInvalidFilter`, `ErrInvalidData`) | 400 | `{"error":"..."}` |
| Jev/TypeSafe API or response failure (`ErrUpstream`) | 502 | `{"error":"..."}` |
| Record not found (`repository.ErrNotFound`) | 404 | `{"error":"..."}` |
| Any other service failure | 500 | `{"error":"..."}` |
| Success | 200 | `MessageResponse` or a JSON array |

## External Integrations

| Service | Purpose | Config |
|---|---|---|
| TypeSafe System One API | Message classification (Jev model) | `TYPESAFE_API_URL`, `TYPESAFE_MODEL`, `TS_API_KEY` |
| htmx.org 2.0.10 (CDN) | Client-side partial page updates | — |
| htmx-ext-response-targets (CDN) | Swap 4xx/5xx responses into `#resultado` | — |
| Web Awesome 3.14.0 (CDN) | Web components: `wa-page`, `wa-card`, `wa-badge`, `wa-button`, `wa-input`/`wa-select`/`wa-checkbox`/`wa-textarea`, `wa-tab-group`, `wa-drawer`, `wa-callout`, `wa-spinner`, `wa-icon`, `wa-option` | — |
| SQLite (glebarez/sqlite) | Contact, note, to-do and transaction persistence | DB_PATH |

## Configuration

| Variable | Source | Used by |
|---|---|---|
| `TYPESAFE_API_URL` | `local.env` | `internal/app/app.go` → `jev.NewClient` (POST target) |
| `TYPESAFE_MODEL` | `local.env` | `internal/app/app.go` → `jev.NewClient` (model field) |
| `TS_API_KEY` | environment (not in `local.env`) | `internal/app/app.go` → `jev.NewClient` (Bearer token) |
| DB_PATH | local.env (default contacts.db) | internal/app/app.go → gorm.Open (contacts, notes, to-dos, transactions; also the data screen; a file-backed DSN adds busy_timeout/WAL) |
| API_PORT | environment (default `8081`) | `cmd/api/main.go` → `router.Run` |

> Note: `TS_API_KEY` is read by `internal/config/env.go` but not defined in `local.env` — it must be set in the environment or the Authorization header will be `Bearer ` (empty).

## Build & Deploy

```bash
# One start target per entrypoint (Makefile)
make web    # loads local.env, serves on :8080
make api    # serves JSON on :8081 — API_PORT
make cli    # terminal REPL (msg> prompt)

# Or directly
go run ./cmd/web   # Web
go run ./cmd/api   # REST API
go run ./cmd/cli   # CLI

# Build binaries
go build ./cmd/web ./cmd/api ./cmd/cli

# Debug (VS Code)
# .vscode/launch.json → "API Debug" runs cmd/web/main.go from workspace root
```

- `cmd/web` and `cmd/api` can run at the same time: they share one SQLite file, which is why the DSN carries `busy_timeout(5000)` and WAL.
- The generated `docs/` swagger package is committed, so neither the build nor CI needs the `swag` CLI.
- The `Makefile` only starts entrypoints — no build/test targets; CI is `.github/workflows/ci.yml` (build + vet + test on push to `main`/`feat/**` and on every pull request, deliberately no gofmt gate). No Dockerfile exists.
- Tests exist for `pkg/jev`, `internal/models`, `internal/config`, `internal/ptbr`, `internal/services` (+ its `contact`, `notes` and `finance` subpackages), `internal/app`, `internal/views`, and `internal/repository` (run with `go test ./...`). `internal/controllers` has no tests — its handlers are a thin bind/render shell over `DataService`. `internal/jevq` has none either: it is answer decoding and request assembly over `pkg/jev`, exercised through the four flows.
- Go and template files uniformly lack a trailing newline at EOF, so `gofmt -l` lists 69 of the 72 tracked `.go` files. That is expected: read the real diff, not the list, and never bulk-reformat.
- `.gitignore` ignores `**/*_bin.exe`, `thoughts/`, `*.db`, `*.db-shm`, `*.db-wal`, `.vscode`, and `exports/`.
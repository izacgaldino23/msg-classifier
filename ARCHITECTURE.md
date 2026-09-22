# ARCHITECTURE

## Overview

A Go web application that classifies user messages into categories (contact, finance, schedule, notes, other) and determines whether a message is a request to add or require something. Classification is performed by the **TypeSafe System One API** (Jev model) via one AI request per message (a single template with a choice question and a score question). After classification, a **Dispatcher** routes the message to a category-specific use case; the contact **add** path is the first implemented flow (extraction and persistence are future work inside the use case). The frontend is a server-rendered htmx page (no JS build step).

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
| CSS | sakura.css (CDN) |

## Directory Structure

```
msg-classifier/
├── cmd/
│   └── api/
│       └── main.go              # Composition root: config → jev client → services → dispatcher → controllers → routes
├── internal/                    # Private application code (not importable externally)
│   ├── config/
│   │   └── env.go               # Env singleton (TYPESAFE_API_URL, TYPESAFE_MODEL, TS_API_KEY) — single godotenv load site
│   ├── controllers/             # C — HTTP concerns only (bind → service → render → status)
│   │   ├── web_controller.go    # GET / handler (delegates page/partial switch to views)
│   │   └── message_controller.go# POST /api/message handler (bind → Classify → Dispatch → render)
│   ├── models/                  # M — data structures
│   │   └── message.go           # ReceiveMessageRequest/Response DTOs + Classification domain struct + UseCaseOutcome/Action
│   ├── services/                # M — business rules
│   │   ├── classification.go    # ClassificationService (single Jev call, checked answer mapping, kind resolution)
│   │   ├── dispatcher.go        # Dispatcher (category → handler registry) + CategoryHandler interface
│   │   └── contact.go           # ContactService (contact add stub; get path TODO)
│   └── views/                   # V — render helpers
│       └── render.go            # Template name constants + RenderPage/RenderResult/RenderError
├── pkg/                         # Reusable packages
│   ├── request.go               # ReturnJson helper ({"data": ...} / {"error": ...})
│   └── jev/
│       ├── jev.go               # TypeSafe Jev API client (config-injected, panic-free, embedded templates)
│       └── requests/            # JSON prompt templates (embedded via go:embed)
│           └── classification.json  # choice "classification" + score "adding_or_requiring" questions
├── web/
│   └── templates/
│       ├── layouts/base.html    # "base" layout (sakura.css + htmx CDN + response-targets)
│       ├── pages/index.html     # "page:title" / "page:content" blocks (form)
│       └── partial/
│           ├── result.html      # "resultado" partial (classification output)
│           └── error.html       # "error" partial (error card for htmx swap targets)
├── go.mod / go.sum              # Module "msg-classifier", Go 1.25.4
├── local.env                    # Env vars (not committed secrets)
├── .vscode/launch.json          # Go debug config for cmd/api/main.go
└── README.md
```

## Core Components

### 1. Composition Root — `cmd/api/main.go`
- `main()` builds a `gin.Default()` router, parses `web/templates/**/*.html` (`template.Must`), installs an htmx middleware that stores an `*htmx.Handler` in the Gin context under key `"htmx"` (single htmx instance).
- Wires dependencies: `config.GetEnv()` (single godotenv load site) → `jev.NewClient(url, token, model)` → `services.NewClassificationService(client)` → `services.NewDispatcher` (registry: `"contact"` → `ContactService`) → controllers.
- Registers routes:
  - `GET /` → `webController.Home`
  - `POST /api/message` → `messageController.ReceiveMessage`
- Server runs on `:8080`.

### 2. Config Singleton — `internal/config/env.go`
- `GetEnv()` lazily loads `local.env` and reads `TYPESAFE_API_URL`, `TYPESAFE_MODEL`, `TS_API_KEY` into an `Env` struct. Cached in a package-level `var env *Env`. This is the **only** place godotenv is loaded and `os.Getenv` is called.

### 3. Controllers — `internal/controllers/`
- `WebController.Home`: delegates to `views.RenderPage` — the htmx page/partial switch lives in the view layer.
- `MessageController.ReceiveMessage`: binds `ReceiveMessageRequest` (failure → 400 error partial) → calls `ClassificationService.Classify` → calls `Dispatcher.Dispatch` → maps errors via `renderServiceError` (`errors.Is(err, services.ErrUpstream)` → 502, else 500) → renders result or error partial. No business logic, no Jev types, no template name literals.

### 4. Classification Service — `internal/services/classification.go`
- `ClassificationService.Classify`: builds Jev state `{user, message}`, makes **one** Jev call (`classification.json`, which contains both the `classification` choice question and the `adding_or_requiring` score question), extracts answers with checked assertions (`answerAsChoice` / `answerAsScore`), resolves the kind via `resolveKind` (round score → legend index → value), returns a domain `Classification`.
- `ErrUpstream` sentinel marks failures originating from the Jev/TypeSafe API or its responses; controllers map it to HTTP 502.
- The `jevClient` interface (defined at the service boundary) makes the service unit-testable without HTTP.

### 5. Dispatcher — `internal/services/dispatcher.go`
- `CategoryHandler` interface: `Handle(request, classification) (*models.UseCaseOutcome, error)` — the seam for category-specific use cases.
- `Dispatcher` holds a `map[string]CategoryHandler` keyed by `Category.Choice`; `Dispatch` looks up the handler, falling back to an `ActionNone` outcome on a miss. Adding a category = new service implementing `CategoryHandler` + one wiring line in `main.go`.

### 6. Contact Service — `internal/services/contact.go`
- `ContactService` implements `CategoryHandler` for the `contact` category.
- Kind `add`/`both` → `Add` stub returning an `ActionContactAdd` outcome (extraction/DB TODO lives here).
- Kind `require` → `ActionNone` outcome (get path TODO).

### 7. Models — `internal/models/message.go`
- `ReceiveMessageRequest` / `ReceiveMessageResponse` DTOs.
- `Classification` domain struct (`CategoryFinding` / `KindFinding` with raw numeric confidences; `KindFinding.Value` carries the resolved kind: `"add"` / `"require"` / `"both"`).
- `UseCaseOutcome` (`Classification` + `Action`) with `ActionNone` / `ActionContactAdd` constants — the seam where future use-case results (extracted contact, DB confirmation) flow back without signature changes.
- `Classification.ToResponse()` formats confidences as `%.2f` percent strings for display (successor of the former `fromJevResponse`).

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
- `classification.json`: one `choice` question `"classification"` with 5 criteria (contact, finance, schedule, notes, other) and one `score` question `"adding_or_requiring"` with criteria `["add", "require", "both"]` (the criteria order defines the legend indices used by kind resolution).

### 11. HTML Templates — `web/templates/`
- `base.html`: `base` layout, loads sakura.css + htmx 2.0.10 + response-targets extension from CDNs; `hx-ext="response-targets"` + `hx-target-error="#resultado"` on `<body>` route 4xx/5xx responses into the result container.
- `index.html`: form posting via `hx-post="/api/message"` targeting `#resultado` with `hx-swap="innerHTML"`.
- `result.html`: `resultado` partial rendering category/confidence/action.
- `error.html`: `error` partial rendering an error card (used for 400/502/500 responses).

## Data Flow

```
Browser (htmx form)
  │  POST /api/message  (message, user_id)
  ▼
gin router ──► MessageController.ReceiveMessage
  │  Bind → ReceiveMessageRequest (fail → 400 error partial)
  ├─► ClassificationService.Classify
  │     ├─► jev client ──► POST api.typesafe.ai/v1/systemone (classification.json) ──► choice + score answers
  │     ├─► checked extraction → Category + Kind (fail → ErrUpstream → 502 error partial)
  │     └─► resolveKind: round(Score) → Legend[index] → KindFinding.Value
  ├─► Dispatcher.Dispatch
  │     ├─► contact + add/both → ContactService.Add → outcome ActionContactAdd
  │     ├─► contact + require   → outcome ActionNone (get path TODO)
  │     └─► other category      → outcome ActionNone
  ├─► outcome.Classification.ToResponse() → ReceiveMessageResponse
  └─► views.RenderResult ──► c.HTML(200, "resultado", ...) ──► htmx swaps #resultado innerHTML
```

One TypeSafe API call is made per request (category + request kind in a single template). The server is **stateless** — no database is used yet (DB save/get is a TODO in the contact service).

## Error Handling

| Failure | Status | Response |
|---|---|---|
| Request bind failure | 400 | `error` partial |
| Jev/TypeSafe API or response failure (`ErrUpstream`), incl. kind resolution failure | 502 | `error` partial |
| Any other service failure | 500 | `error` partial |
| Success | 200 | `resultado` partial |

All responses to htmx targets are HTML partials — no JSON on this route. The response-targets extension makes htmx swap 4xx/5xx responses into `#resultado`. No panics exist in the request path.

## External Integrations

| Service | Purpose | Config |
|---|---|---|
| TypeSafe System One API | Message classification (Jev model) | `TYPESAFE_API_URL`, `TYPESAFE_MODEL`, `TS_API_KEY` |
| htmx.org 2.0.10 (CDN) | Client-side partial page updates | — |
| htmx-ext-response-targets (CDN) | Swap 4xx/5xx responses into `#resultado` | — |
| sakura.css (CDN) | Styling | — |

## Configuration

| Variable | Source | Used by |
|---|---|---|
| `TYPESAFE_API_URL` | `local.env` | `cmd/api/main.go` → `jev.NewClient` (POST target) |
| `TYPESAFE_MODEL` | `local.env` | `cmd/api/main.go` → `jev.NewClient` (model field) |
| `TS_API_KEY` | environment (not in `local.env`) | `cmd/api/main.go` → `jev.NewClient` (Bearer token) |

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
- No tests exist yet (no `_test.go` files).
- `.gitignore` ignores `**/*_bin.exe` and `thoughts/`.
# ARCHITECTURE

## Overview

A Go web application that classifies user messages into categories (contact, finance, schedule, notes, other) and determines whether a message is a request to add or require something. Classification is performed by the **TypeSafe System One API** (Jev model) via two AI requests per message. The frontend is a server-rendered htmx page (no JS build step).

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.25.4 |
| Web framework | [gin-gonic/gin](https://github.com/gin-gonic/gin) v1.12.0 |
| HTMX | [donseba/go-htmx](https://github.com/donseba/go-htmx) v1.13.1 + htmx.org 2.0.10 (CDN) |
| Env loading | [joho/godotenv](https://github.com/joho/godotenv) v1.5.1 |
| AI API | TypeSafe System One (`https://api.typesafe.ai/v1/systemone`, model `jev-latest`) |
| Templating | Go `html/template` (ParseGlob) |
| CSS | sakura.css (CDN) |

## Directory Structure

```
msg-classifier/
├── cmd/
│   └── api/
│       └── main.go              # Sole entry point: server bootstrap + route registration
├── internal/                    # Private application code (not importable externally)
│   ├── config/
│   │   └── env.go               # Env singleton (TYPESAFE_API_URL, TYPESAFE_MODEL, TS_API_KEY)
│   └── handlers/
│       ├── msg_handler.go       # POST /api/message handler + Jev orchestration
│       └── web_handler.go       # GET / handler (htmx-aware page render)
├── pkg/                         # Reusable packages
│   ├── request.go               # ReturnJson helper ({"data": ...} / {"error": ...})
│   └── jev/
│       ├── jev.go               # TypeSafe Jev API client + request/response types
│       └── requests/            # JSON prompt templates
│           ├── classification.json
│           └── request_kind.json
├── web/
│   └── templates/
│       ├── layouts/base.html    # "base" layout (sakura.css + htmx CDN)
│       ├── pages/index.html     # "page:title" / "page:content" blocks (form)
│       └── partial/result.html  # "resultado" partial (classification output)
├── go.mod / go.sum              # Module "msg-classifier", Go 1.25.4
├── local.env                    # Env vars (not committed secrets)
├── .vscode/launch.json          # Go debug config for cmd/api/main.go
└── README.md
```

## Core Components

### 1. Server Bootstrap — `cmd/api/main.go`
- `init()` loads `local.env` via godotenv (`log.Fatalf` on failure).
- `main()` builds a `gin.Default()` router, parses `web/templates/**/*.html` (`template.Must`), installs an htmx middleware that stores an `*htmx.Handler` in the Gin context under key `"htmx"`.
- `AddHandlers(router)` registers routes:
  - `GET /` → `webHandler.Home`
  - `POST /api/message` → `messageHandler.ReceiveMessage`
- Server runs on `:8080`.

### 2. Config Singleton — `internal/config/env.go`
- `GetEnv()` lazily loads `local.env` and reads `TYPESAFE_API_URL`, `TYPESAFE_MODEL`, `TS_API_KEY` into an `Env` struct. Cached in a package-level `var env *Env`.

### 3. Web Handler — `internal/handlers/web_handler.go`
- `Home`: reads the htmx handler from context. If `IsHxRequest()`, renders only the `page:content` block; otherwise renders the full `base` layout.

### 4. Message Handler — `internal/handlers/msg_handler.go`
- `ReceiveMessage`: binds JSON/form body into `ReceiveMessageRequest{Message, UserID}` → calls `checkMessageCategory` → calls `checkRequestKind` → type-asserts the two answers → renders the `resultado` template.
- `checkMessageCategory` / `checkRequestKind`: call `jev.MakeJevRequestFromFile` with state `{user, message}` and the respective JSON template.
- `fromJevResponse`: maps Jev answers to `ReceiveMessageResponse{Category, Kind}` (confidence formatted as `%.2f` percent string).
- TODO at `msg_handler.go:69-72`: DB save/get logic not yet implemented.

### 5. Jev API Client — `pkg/jev/jev.go`
- Defines the Jev type system: `JevRequest`, `JevState`, `JevQuestion*` (choice/score/noul), `JevAnswer*` (choice/score/noul), all prefixed `Jev`.
- `MakeJevRequest`: sets model from config, validates, POSTs JSON to `TYPESAFE_API_URL` with `Authorization: Bearer <TS_API_KEY>`, parses response.
- `MakeJevRequestFromFile`: loads a JSON template from `pkg/jev/requests/`, injects state, delegates to `MakeJevRequest`.
- `LoadJevRequestFromFile`: polymorphically decodes questions by `type` field.
- `HttpResponseToJevResponse`: decodes API answers by `type` into typed structs.
- `validateJevRequest`: validates state, model, questions, and per-type criteria/true-false fields.

### 6. Jev Request Templates — `pkg/jev/requests/*.json`
- `classification.json`: one `choice` question `"classification"` with 5 criteria (contact, finance, schedule, notes, other).
- `request_kind.json`: one `score` question `"adding_or_requiring"` with criteria `["add", "require", "both"]`.

### 7. HTML Templates — `web/templates/`
- `base.html`: `base` layout, loads sakura.css + htmx 2.0.10 from CDNs.
- `index.html`: form posting via `hx-post="/api/message"` targeting `#resultado` with `hx-swap="innerHTML"`.
- `result.html`: `resultado` partial rendering category/confidence/score/legend.

## Data Flow

```
Browser (htmx form)
  │  POST /api/message  (message, user_id)
  ▼
gin router ──► MsgHandler.ReceiveMessage
  │  Bind → ReceiveMessageRequest
  ├─► checkMessageCategory ──► MakeJevRequestFromFile(classification.json)
  │      └─► POST api.typesafe.ai/v1/systemone ──► JevResponse (choice answer)
  ├─► checkRequestKind ──► MakeJevRequestFromFile(request_kind.json)
  │      └─► POST api.typesafe.ai/v1/systemone ──► JevResponse (score answer)
  ├─► fromJevResponse → ReceiveMessageResponse{Category, Kind}
  └─► c.HTML(200, "resultado", ...) ──► htmx swaps #resultado innerHTML
```

Two sequential TypeSafe API calls are made per request (category + request kind). The server is **stateless** — no database is used yet (DB save/get is a TODO).

## External Integrations

| Service | Purpose | Config |
|---|---|---|
| TypeSafe System One API | Message classification (Jev model) | `TYPESAFE_API_URL`, `TYPESAFE_MODEL`, `TS_API_KEY` |
| htmx.org 2.0.10 (CDN) | Client-side partial page updates | — |
| sakura.css (CDN) | Styling | — |

## Configuration

| Variable | Source | Used by |
|---|---|---|
| `TYPESAFE_API_URL` | `local.env` | `pkg/jev/jev.go` (POST target) |
| `TYPESAFE_MODEL` | `local.env` | `pkg/jev/jev.go` (model field) |
| `TS_API_KEY` | environment (not in `local.env`) | `pkg/jev/jev.go` (Bearer token) |

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
- `.gitignore` only ignores `**/*_bin.exe`.
# Conference Tool

A web application for managing parliamentary-style conference meetings. It handles committees, meetings, agenda points, attendees, speakers lists, motions, and real-time session views — all driven by HTMX with no heavy JavaScript framework.

Built with Go, [HTMX](https://htmx.org/), [Templ](https://templ.guide/), [DaisyUI](https://daisyui.com/), and SQLite.

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [Architecture](#architecture)
- [Extending the Code](#extending-the-code)
- [Development Reference](#development-reference)
- [E2E Testing](#e2e-testing)

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| [Go](https://go.dev/dl/) | 1.25+ | Application runtime and all Go-based tooling |
| [Node.js](https://nodejs.org/) | 20+ | Tailwind CSS / DaisyUI build pipeline |
| [Task](https://taskfile.dev/) | any | Task runner — required to use the commands in this guide |

All other tools — `templ`, `air`, `sqlc`, `golangci-lint` — are declared as Go tool dependencies in `go.mod` and require no separate installation.

To verify your environment:

```bash
task deps:check
```

This checks Go, Node.js, and npm and prints a pass/fail summary.

---

## Getting Started

```bash
# 1. Clone
git clone <repository-url>
cd conference-tool

# 2. Check dependencies
task deps:check

# 3. Install Node modules, download Go modules, and generate all code
task setup

# 4. Copy the example env file and edit as needed
cp .env.example .env

# 5. Start the dev server with hot reload
task dev
```

For CSS hot-reload during development, run this in a second terminal:

```bash
task css:watch
```

The application is now available at `http://localhost:8080`.

---

## Project Structure

```
.
├── cmd/                          # Cobra CLI entry points
│   ├── root.go
│   └── serve.go                  # Wires all dependencies and starts the HTTP server
├── e2e/                          # Playwright browser E2E tests (build tag: e2e)
│   ├── main_test.go              # TestMain — Playwright driver init
│   ├── helpers_test.go           # testServer, DB seed helpers
│   └── browser_helpers_test.go   # newPage, adminLogin, userLogin
├── internal/
│   ├── assets/css/               # Tailwind CSS source (app.css)
│   ├── broker/                   # In-memory SSE event broker
│   ├── config/                   # Configuration loading (Viper + env vars)
│   ├── handlers/                 # HTTP request handlers
│   │   ├── handlers.go           # Handler struct + HandlerBuilder
│   │   ├── admin.go              # Admin panel handlers
│   │   ├── auth.go               # Login, logout, committee session handlers
│   │   ├── manage.go             # Meeting management (chairperson)
│   │   ├── moderate.go           # Live moderation view
│   │   └── ...                   # Other feature handlers
│   ├── locale/                   # i18n middleware and translation loader
│   │   └── locales/              # YAML translation catalogs (en.yaml, de.yaml)
│   ├── middleware/               # HTTP middleware registry
│   ├── pagination/               # Shared pagination helpers
│   ├── repository/               # Repository interface (repository.go)
│   │   └── sqlite/               # SQLite implementation
│   │       ├── client/           # SQLC-generated type-safe DB client (DO NOT EDIT)
│   │       ├── migrations/       # Numbered SQL migration files (.up.sql / .down.sql)
│   │       ├── queries/          # SQL query files read by SQLC
│   │       └── sqlc.yaml         # SQLC configuration
│   ├── routes/                   # Generated routing code (DO NOT EDIT)
│   │   ├── gen.go                # go:generate directive
│   │   ├── routes_gen.go         # Generated router + Handler interface
│   │   ├── paths/
│   │   │   └── paths_gen.go      # Generated type-safe URL builders (DO NOT EDIT)
│   │   └── static/               # Compiled CSS served as a static asset
│   ├── session/                  # Cookie-based session management
│   ├── storage/                  # File storage (attachments)
│   └── templates/                # Templ templates
│       ├── *.templ               # Template source files
│       └── *_templ.go            # Generated Go code (DO NOT EDIT)
├── tools/routing/                # Custom route code generator
│   ├── types.go                  # YAML config types
│   ├── parser.go                 # YAML parser and validator
│   └── generator.go              # Code generation logic
├── .air.toml                     # Hot reload config
├── .env.example                  # Example environment variables
├── go.mod / go.sum
├── package.json                  # Node.js devDependencies (Tailwind, DaisyUI)
├── routes.yaml                   # Route definitions — the source of truth
├── Taskfile.yaml                 # Task runner definitions
└── flake.nix                     # Nix development environment (optional)
```

---

## Architecture

### Request Lifecycle

```
HTTP Request
     │
     ▼
  Router (routes_gen.go)
     │  extracts path params
     │  applies middleware chain
     ▼
  Handler method
     │  reads DB, applies business logic
     │  returns (*TemplateInput, error)
     ▼
  Templ component
     │  renders HTML fragment or full page
     ▼
HTTP Response
```

For HTMX requests the handler returns a **partial** HTML fragment, which HTMX swaps into the page without a full reload.

---

### Type-Safe Routing

Routes are defined once in [`routes.yaml`](routes.yaml) and the code generator (`tools/routing/`) produces three artifacts:

1. **`routes_gen.go`** — the `Handler` interface every handler struct must satisfy, plus the `Router` that wires it all together.
2. **`paths_gen.go`** — locale-aware URL builder functions so templates and handlers never hardcode URL strings.
3. **`RouteParams`** — a struct carrying all path parameters for a given route.

Example route definition:

```yaml
- path: /committee/{slug}/meeting/{meeting_id}/manage
  methods:
    - verb: GET
      handler: CommitteeManageMeeting
      middleware:
        - session
        - auth
        - committee_access
      template:
        package: github.com/Y4shin/conference-tool/internal/templates
        type: MeetingManageTemplate
        input_type: MeetingManageInput
```

The generated handler signature becomes:

```go
CommitteeManageMeeting(
    ctx context.Context,
    r *http.Request,
    params routes.RouteParams,
) (*templates.MeetingManageInput, *routes.ResponseMeta, error)
```

Using the generated URL builder in a template:

```go
import "github.com/Y4shin/conference-tool/internal/routes/paths"

// No path params:
paths.Route.AdminDashboardGet(ctx, "")

// With path params:
paths.NewCommitteeSlugMeetingMeetingIdManageRoute(slug, meetingID).
    CommitteeManageMeetingGet(ctx, "")

// With query params:
paths.Route.CommitteePageGetWithQuery(ctx, "", paths.CommitteePageGetQueryParams{Page: "2"})
```

---

### Templates (Templ)

Templates live in `internal/templates/*.templ`. Each file defines:

- **Input types** (plain Go structs and methods)
- **`templ` components** — Go functions that render HTML

```templ
package templates

type MeetingManageInput struct {
    CommitteeSlug string
    Meeting       MeetingItem
    AgendaPoints  []AgendaPointItem
}

templ MeetingManageTemplate(input MeetingManageInput) {
    @Scaffold(ScaffoldInput{Title: input.Meeting.Name}) {
        <div id="agenda-point-list-container">
            @AgendaPointList(input)
        </div>
    }
}
```

Run `task generate:templates` (or `go generate ./internal/templates`) after editing `.templ` files. The `*_templ.go` output is committed but must not be edited by hand.

Templates use `i18n.T(ctx, "key")` for translations. Keys map to entries in `internal/locale/locales/en.yaml` (and other locale files).

---

### Database Layer

SQLite with SQLC for compile-time-checked queries:

- **Schema** — migrations in `internal/repository/sqlite/migrations/`, numbered `NNN_description.up.sql` / `.down.sql`. Migrations run automatically at startup.
- **Queries** — SQL files in `internal/repository/sqlite/queries/`. SQLC reads these and generates a type-safe Go client in `internal/repository/sqlite/client/`.
- **Interface** — `internal/repository/repository.go` declares the `Repository` interface. Handlers depend only on this interface, not on the SQLite implementation.

Run `task generate:db` after adding or modifying query files.

---

### Middleware

Middleware is registered by name in `internal/middleware/middleware.go` and referenced by name in `routes.yaml`:

```yaml
middleware:
  - session        # load/save session cookie
  - auth           # require authenticated user
  - committee_access  # verify user belongs to the committee
```

The middleware registry resolves names to `func(http.Handler) http.Handler` values and the router applies them in order.

---

### Authentication Providers

The app supports provider-gated authentication:

- Password login (`AUTH_PASSWORD_ENABLED=true|false`)
- OAuth/OIDC login (`AUTH_OAUTH_ENABLED=true|false`)

Startup validation enforces that at least one provider is enabled. If both are disabled, the server fails to start.

When OAuth is enabled, these values are required:

- `OAUTH_ISSUER_URL`
- `OAUTH_CLIENT_ID`
- `OAUTH_CLIENT_SECRET`
- `OAUTH_REDIRECT_URL`

Additional OAuth settings include:

- `OAUTH_SCOPES` (default `openid,profile,email`)
- `OAUTH_GROUPS_CLAIM` (default `groups`)
- `OAUTH_USERNAME_CLAIMS`
- `OAUTH_FULL_NAME_CLAIMS`
- `OAUTH_PROVISIONING_MODE` (`preprovisioned` or `auto_create`)
- `OAUTH_REQUIRED_GROUPS`
- `OAUTH_ADMIN_GROUP`
- `OAUTH_STATE_TTL_SECONDS`

If password login is disabled, password login submit endpoints return `404`.

Important compatibility note:

- Changing enabled auth providers for an already-populated database is intentionally undefined/unsupported in this phase.
- No migration/backfill guarantees are provided for converting existing password accounts to OAuth accounts (or vice versa).

Local interactive OIDC provider:

0. Generate a ready local `.env`: `task env:populate`.
1. Generate users file: `task oidc-dev:generate-users`.
2. Configure shared OAuth env vars in `.env` if needed (`OAUTH_ISSUER_URL`, `OAUTH_CLIENT_ID`, `OAUTH_CLIENT_SECRET`, `OAUTH_REDIRECT_URL`).
3. Configure `OIDC_DEV_USERS_FILE` if needed (default `dev/users.yaml`).
   Optional: set `OIDC_DEV_LISTEN_ADDR` if you need OIDC to bind a different address than the issuer host:port.
4. Start provider: `task oidc-dev:run`.
5. Start app with same `.env`: `task run` or `task dev`.

Full hot-reload SPA development with local OIDC:

- `task dev:spa:with-oidc`

That starts three processes together:

- local OIDC dev provider on `http://localhost:9096`
- Go backend with `air` on `http://localhost:8080`
- Vite SPA dev server on `http://localhost:5173`

The Vite dev server proxies the backend-facing routes (`/api`, `/oauth`, `/locale`, `/blobs`, `/docs/assets`) to the Go backend, and the OAuth redirect URL is adjusted to `http://localhost:5173/oauth/callback` for this flow. The OIDC dev server binds to `127.0.0.1` in this mode so it stays loopback-only while still being reachable at `localhost`.
The backend watcher uses a Vite-specific Air config that ignores `web/` so frontend file churn does not slow down Go restarts.
The frontend task also repairs missing platform-specific Rollup optional dependencies automatically before starting Vite, which helps when `node_modules` was previously installed on a different host platform.

Docker alternatives:

- Linux host networking: `task docker:up` (or `docker compose up --build app oidc`)
- Docker Desktop-compatible: `task docker:up:desktop` (or `docker compose -f docker-compose.desktop.yml up --build app oidc`)

Compose runs a one-shot `setup` service first (`populate-env` + `generate-users --force`), then starts the local OIDC provider and the SPA/Connect app server together.

The app service runs `conference-tool serve` and is considered healthy once `http://127.0.0.1:8080/login` responds.

By default, generated env uses `OAUTH_ADMIN_GROUP=ca-admin`, and generated users include Alice in `ca-admin` and `committee-a-chair`.

The local provider reads users (including groups) from YAML and exposes groups in the claim configured by `OIDC_DEV_GROUPS_CLAIM` (default `groups`).

---

### Internationalisation (i18n)

- Locale resolution order: URL prefix → `locale` cookie → `Accept-Language` header → default (`en`).
- Translation catalogs are YAML files rooted by locale code in `internal/locale/locales/`.
- Templates call `i18n.T(ctx, "key")` using the `github.com/invopop/ctxi18n/i18n` package.
- A language switcher component is available at `internal/templates/language_switcher.templ`.

---

### Real-Time Updates (SSE)

Routes marked `sse: true` in `routes.yaml` stream Server-Sent Events. Handlers subscribe to the in-memory broker and write events directly to the response writer. HTMX on the client uses `hx-ext="sse"` to consume the stream and swap content on incoming events.

---

## Extending the Code

### Adding a New Route

1. **Define the route** in `routes.yaml`:

   ```yaml
   - path: /committee/{slug}/my-feature
     methods:
       - verb: GET
         handler: CommitteeMyFeature
         middleware:
           - session
           - auth
           - committee_access
         template:
           package: github.com/Y4shin/conference-tool/internal/templates
           type: MyFeatureTemplate
           input_type: MyFeatureInput
   ```

2. **Create a Templ template** in `internal/templates/my_feature.templ`:

   ```templ
   package templates

   type MyFeatureInput struct {
       CommitteeSlug string
       // ... your data fields
   }

   templ MyFeatureTemplate(input MyFeatureInput) {
       @Scaffold(ScaffoldInput{Title: "My Feature"}) {
           <p>Hello from my feature!</p>
       }
   }
   ```

3. **Regenerate all code**:

   ```bash
   task generate
   ```

4. **Implement the handler** in a file under `internal/handlers/` (e.g., `my_feature.go`):

   ```go
   package handlers

   import (
       "context"
       "net/http"

       "github.com/Y4shin/conference-tool/internal/routes"
       "github.com/Y4shin/conference-tool/internal/templates"
   )

   func (h *Handler) CommitteeMyFeature(
       ctx context.Context,
       r *http.Request,
       params routes.RouteParams,
   ) (*templates.MyFeatureInput, *routes.ResponseMeta, error) {
       // fetch data, apply business logic...
       return &templates.MyFeatureInput{
           CommitteeSlug: params.Slug,
       }, nil, nil
   }
   ```

   The compiler will flag a missing method if you forget this step — the generated `Handler` interface is your checklist.

5. **Add an E2E test** in `e2e/` covering the happy path (see [E2E Testing](#e2e-testing)).

---

### Adding a New Database Query

1. Write a SQL query in `internal/repository/sqlite/queries/`, following the existing SQLC annotation style:

   ```sql
   -- name: GetMyThing :one
   SELECT * FROM my_table WHERE id = ? LIMIT 1;
   ```

2. Regenerate the DB client:

   ```bash
   task generate:db
   ```

3. Expose the query through the `Repository` interface in `internal/repository/repository.go` and implement it in `internal/repository/sqlite/repository.go` by delegating to the generated client method.

---

### Adding a New Database Migration

Create two numbered files in `internal/repository/sqlite/migrations/`:

```
NNN_my_change.up.sql
NNN_my_change.down.sql
```

Where `NNN` is the next sequential number. Migrations run automatically on startup via `repo.MigrateUp()`.

---

### Adding New Middleware

1. Implement a `func(http.Handler) http.Handler` in `internal/middleware/`.
2. Register it by name in `NewRegistry()` inside `internal/middleware/middleware.go`.
3. Reference the name in `routes.yaml`.

---

### Adding a Translation Key

Add the key to every locale file under `internal/locale/locales/`. Each file must be rooted by its locale code:

```yaml
en:
  my_feature:
    title: "My Feature"
    description: "Does something useful."
```

Then use it in a template:

```go
i18n.T(ctx, "my_feature.title")
```

---

## Development Reference

### Key Tasks

| Task | Description |
|------|-------------|
| `task deps:check` | Verify Go, Node.js, and npm are installed |
| `task setup` | First-time setup: check deps, install modules, generate code |
| `task dev` | Start the server with hot reload (`air`) |
| `task css:watch` | Rebuild CSS on file changes (run alongside `task dev`) |
| `task css:build` | Build CSS once |
| `task generate` | Regenerate all code (CSS, routes, templates, DB client) |
| `task generate:routes` | Regenerate routes only |
| `task generate:templates` | Regenerate templates only |
| `task generate:db` | Regenerate database client only |
| `task test` | Run unit tests |
| `task test:e2e` | Run Playwright browser tests |
| `task check` | Format, vet, and lint |
| `task ci` | Full CI gate (format check, vet, lint, test) |

Run `task --list` for the full list.

### Environment Variables

Copy `.env.example` to `.env` and adjust as needed:

| Variable | Default | Description |
|----------|---------|-------------|
| `ENVIRONMENT` | `development` | `development`, `staging`, `production` |
| `HOST` | `0.0.0.0` | Bind address |
| `PORT` | `8080` | HTTP port |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `text` | `text` or `json` |

### Code Generation Summary

After changing… | Run…
--- | ---
`routes.yaml` | `task generate:routes`
`*.templ` files | `task generate:templates`
SQL query files | `task generate:db`
Anything | `task generate` (does all of the above + CSS)

**Never edit generated files** — they have `_gen.go`, `_templ.go`, or live under `client/` (SQLC output).

---

## E2E Testing

Browser tests use [Playwright](https://playwright.dev/) and live in `e2e/`. They require the `e2e` build tag and are excluded from `task test`.

### First-Time Setup

```bash
task playwright:install    # Download Chromium (once per machine)
```

### Running Tests

```bash
task test:e2e                  # Headless
task test:e2e:headed           # With visible browser
task test:e2e:headed:slow      # Visible + slow-motion (useful for debugging)
```

### Test Helpers

- `newTestServer(t)` — starts a full app instance with an in-memory SQLite DB.
- `newPage(t)` — launches an isolated Chromium context. Skips the test automatically if browsers are not installed.
- `adminLogin(t, page, baseURL)` / `userLogin(t, page, baseURL, committee, username, password)` — reusable login flows.
- `ts.seedCommittee`, `ts.seedUser`, `ts.seedMeeting` — seed the DB directly without HTTP.

### HTMX Assertions

```go
// Confirm no full navigation occurred after an HTMX form submit
urlBefore := page.URL()
page.Locator("button:has-text('Save')").Click()
// ... wait for content swap ...
assert(page.URL() == urlBefore)

// Wait for swapped-in content
page.Locator("td:has-text('new value')").WaitFor()

// Wait for removed content
locator.WaitFor(playwright.LocatorWaitForOptions{
    State: playwright.WaitForSelectorStateDetached,
})

// Accept an hx-confirm dialog — register BEFORE clicking
page.OnDialog(func(d playwright.Dialog) { d.Accept() })
page.Locator("button:has-text('Delete')").Click()
```

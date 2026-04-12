# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A conference management tool built with Go, HTMX, and Templ. The project features a custom type-safe routing system using YAML-based route definitions with automatic code generation.

## Development Environment

The project uses Nix flakes with direnv for reproducible development environments. The flake provides:
- Go 1.26.1
- gopls, gotools, golangci-lint
- go-task for task running
- MCP servers (mcp-gopls, mcp-taskfile-server) for AI assistance

Alternatively, ensure Go 1.26.1+ is installed directly.

## Essential Commands

The project uses [Taskfile](https://taskfile.dev/) for common development tasks. Run `task` or `task --list` to see all available tasks.

### Running the Application

```bash
# Using Task (preferred)
task dev              # Start with hot reload (using air)
task run              # Run directly
task build            # Build the application

# Direct commands
go tool air           # Start with hot reload
go run . serve        # Run directly
go build -o tmp/conference-tool .
```

### Code Generation

**CRITICAL**: After any changes to routes, templates, or database queries, regenerate code:

```bash
# Using Task (preferred)
task generate              # Generate all code
task generate:routes       # Generate only routes
task generate:templates    # Generate only templates
task generate:db           # Generate only database client

# Direct command
go generate ./...
```

### Database Operations

Migrations run automatically at startup via `repo.MigrateUp()`. The migration files live in [internal/repository/sqlite/migrations/](internal/repository/sqlite/migrations/).

The Taskfile `db:migrate:*` tasks are stubs and not yet wired to the binary. To seed an initial admin user for local development:

```bash
task init:dev-db   # creates admin/admin — runs: go run . create-admin --username admin --password admin
```

### Testing

```bash
# Using Task (preferred)
task test                  # Run all tests (excludes E2E)
task test:verbose          # Run with verbose output
task test:coverage         # Generate coverage report
task test:watch            # Run tests in watch mode
task test:e2e              # Run browser-based E2E tests (requires Playwright browsers)

# Playwright browser setup (run once per machine)
task playwright:install    # Install Chromium and dependencies
task playwright:cli -- <args>  # Run Playwright CLI with arbitrary arguments

# Direct commands
go test ./...              # Run all tests
go test -v ./...           # Verbose output
go test ./internal/config/...           # Specific package
go test ./internal/config -run TestName # Specific test
go test -v -tags=e2e -timeout=120s ./e2e/...  # Run E2E tests directly
```

### Code Quality

```bash
# Using Task (preferred)
task check            # Run all checks (fmt, vet, lint)
task fmt              # Format code
task lint             # Run linter
task lint:fix         # Run linter with auto-fix
task vet              # Run go vet

# Direct commands
go fmt ./...
golangci-lint run
go vet ./...
```

### Development Workflows

```bash
task deps:check       # Verify Go, Node.js, and npm are installed
task setup            # Initial project setup (runs deps:check first)
task init:dev-db      # Seed a local admin user (admin/admin) after first setup
task fresh            # Clean, generate, build, and run
task ci               # Run all CI checks
task deps:tidy        # Tidy go.mod and go.sum
```

## Architecture Overview

### Internationalization (i18n)

The app supports locale-aware routing and translations:
- Locale resolution order: URL prefix -> `locale` cookie -> `Accept-Language` -> default (`en`)
- Locale middleware is in [internal/locale/](internal/locale/) and wired in [cmd/serve.go](cmd/serve.go)
- Translations are loaded via `locale.LoadTranslations()`
- Catalog files live in [internal/locale/locales/](internal/locale/locales/)
- Templates translate with `i18n.T(ctx, "key")` from `github.com/invopop/ctxi18n/i18n`
- Shared language switcher component: [internal/templates/language_switcher.templ](internal/templates/language_switcher.templ)

Catalog format requirement (`ctxi18n`): each file must be rooted by locale code, e.g.
```yaml
en:
  login:
    title: "Conference Tool Login"
```

### Type-Safe Routing System

The project uses a custom code generation system for type-safe routing:

1. **Route Definitions** ([routes.yaml](routes.yaml)): YAML-based route definitions with path parameters, HTTP methods, handlers, templates, and middleware
2. **Code Generator** ([tools/routing/](tools/routing/)): Parses YAML and generates type-safe Go code
3. **Generated Router** ([internal/routes/routes_gen.go](internal/routes/routes_gen.go)): Auto-generated router that wires handlers to routes

**Handler Flow**:
- Handler methods return template input data and an error: `(*TemplateInput, error)`
- Router extracts path parameters, calls handler, and passes result to template
- Templates render HTML directly to the response
- Errors return HTTP 500 automatically

**Adding a New Route**:
1. Add route definition to [routes.yaml](routes.yaml)
2. Create corresponding `.templ` template in [internal/templates/](internal/templates/)
3. Run `go generate ./...`
4. Implement handler method in [internal/handlers/handlers.go](internal/handlers/handlers.go)
5. Register any new middleware in [internal/middleware/middleware.go](internal/middleware/middleware.go)
6. Add or update E2E tests in [e2e/](e2e/) to cover the new/changed route

**Type-Safe URL Builders**:

The code generator also produces [internal/routes/paths/paths_gen.go](internal/routes/paths/paths_gen.go) with type-safe URL builder functions. **Templates and handlers must use these instead of hardcoding URL strings.**

- Routes without path parameters: `paths.Route.<MethodName>(ctx, "")` - e.g. `paths.Route.AdminDashboardGet(ctx, "")`
- Routes with path parameters: `paths.New<Route>(params...).MethodName(ctx, "")` - e.g. `paths.NewCommitteeSlugRoute(slug).CommitteePageGet(ctx, "")`
- Routes with query parameters: use the `WithQuery` variant - e.g. `paths.Route.AdminDashboardGetWithQuery(ctx, "", paths.AdminDashboardGetQueryParams{Page: "2"})`
- Static asset path methods remain parameterless (e.g. `paths.Route.HtmxMinJs()`)

Example usage in a Templ template:
```go
import "github.com/Y4shin/open-caucus/internal/routes/paths"

<form hx-post={ paths.NewCommitteeSlugMeetingCreateRoute(slug).CommitteeCreateMeetingPost(ctx, "") }>
<a href={ templ.URL(paths.Route.AdminDashboardGet(ctx, "")) }>Dashboard</a>
```

### Template System

Uses [Templ](https://templ.guide/) for type-safe HTML templating:
- Templates are in [internal/templates/](internal/templates/) as `.templ` files
- Running `go generate ./internal/templates` compiles them to `*_templ.go` files
- Templates are Go functions that accept typed input and render HTML
- Integrates seamlessly with HTMX for dynamic updates

### Server-Sent Events (SSE) Broker

The broker ([internal/broker/](internal/broker/)) manages real-time updates:
- `Broker` interface defines the contract for SSE event publishing
- `MemoryBroker` is a simple in-memory implementation
- Routes can be marked with `sse: true` in routes.yaml for SSE endpoints
- SSE handlers receive a channel from `broker.Subscribe()` and stream events to clients
- HTMX can consume SSE endpoints using `hx-ext="sse"`

### Database Layer

SQLite with SQLC for type-safe database access:
- **Schema**: Migrations in [internal/repository/sqlite/migrations/](internal/repository/sqlite/migrations/)
- **Queries**: SQL queries in [internal/repository/sqlite/queries/](internal/repository/sqlite/queries/)
- **Code Generation**: SQLC generates type-safe Go code from SQL ([sqlc.yaml](internal/repository/sqlite/sqlc.yaml))
- **Generated Code**: Go client code in [internal/repository/sqlite/client/](internal/repository/sqlite/client/)

**Database Schema** covers conference management:
- Committees, Users, Meetings
- Agenda Points, Attendees, Speakers List
- Motions, Binary Blobs, Agenda Attachments

### Configuration Management

Configuration system ([internal/config/](internal/config/)):
- Uses Viper for loading from environment variables and config files
- Environment-based configuration with validation
- `.env.example` shows available configuration options
- Configuration struct defined in [internal/config/config.go](internal/config/config.go)
- Validates constraints (e.g., port range, enum values) using custom validators

### Middleware System

Middleware registry pattern ([internal/middleware/middleware.go](internal/middleware/middleware.go)):
- `Registry` provides a centralized place to register and retrieve middleware
- Middleware can be applied via:
  - **Middleware Groups**: Prefix-based middleware in routes.yaml (e.g., all `/api` routes)
  - **Route-specific**: Per-route middleware in routes.yaml
- Middleware functions follow standard `func(http.Handler) http.Handler` signature

## Code Generation Tools

The project uses three `go:generate` directives:

1. **Route Generator** ([internal/routes/gen.go](internal/routes/gen.go)):
   - Reads [routes.yaml](routes.yaml)
   - Generates Handler interface and Router implementation
   - Creates locale-aware type-safe URL builders (`-locale-package ...`)

2. **Template Generator** ([internal/templates/gen.go](internal/templates/gen.go)):
   - Compiles `.templ` files to Go code using `templ generate`
   - Each template becomes a Go function

3. **Database Client** ([internal/repository/sqlite/gen.go](internal/repository/sqlite/gen.go)):
   - Reads SQL schema and queries
   - Generates type-safe database client using `sqlc generate`

**DO NOT manually edit generated files** - they have `_gen.go`, `_generated.go`, or `_templ.go` suffixes.

## Important Patterns

### SPA Architecture Rule: No Legacy HTML Proxying

**CRITICAL**: The SPA (`web/`) must never proxy or embed HTML fragments from the legacy
HTMX/Templ handler. All UI must be implemented natively in Svelte using Connect (gRPC-web)
API calls. Violations are forbidden even as a temporary workaround.

This means:
- The SPA must **never** fetch HTML from legacy routes (`/votes/partial`, `/agenda-point/{id}/edit-form`, etc.) and inject it with `{@html ...}`
- The SPA must **never** use `hx-get` to load HTML fragments from the legacy handler
- The SPA must **never** call legacy HTTP form endpoints (`postLegacyAttendeeAction` pattern)
- In `e2e/helpers_test.go`, `newTestServer()` must **never** proxy requests to `legacyRouter` for routes that the SPA should handle natively — `shouldServeLegacy*` proxy functions must be removed as routes are ported

The legacy HTMX/Templ handler exists solely for UI parity comparison tests. Once all parity
tests are replaced by native implementations, the legacy handler will be removed entirely.

When adding new features to the SPA:
1. Add a Connect RPC to the appropriate proto service
2. Run `buf generate` to generate Go + TypeScript bindings
3. Implement the RPC in the Go service and Connect handler
4. Call the RPC from the Svelte component

### HTMX Usage

**Rule**: Use HTMX for all dynamic interactions **in the legacy HTMX/Templ layer only**.
A page should never do a full reload unless the user explicitly navigates to a different
page (e.g., clicks a nav link or follows an anchor).

- Form submissions that create, update, or delete data must use `hx-post`/`hx-put`/`hx-delete` and swap only the affected region (list, row, section) — not the full page
- Use `hx-target` + `hx-swap` to update only the changed part of the DOM in response to a form submit
- Use `hx-confirm` for destructive actions (delete) to trigger a native browser confirmation dialog
- Avoid `hx-boost` on forms where partial-swap behavior is needed; use explicit `hx-post` attributes instead
- Handler responses for HTMX requests return a partial HTML fragment (a single Templ component), not a full page

### E2E Tests (Playwright)

Browser-based E2E tests live in [e2e/](e2e/) and are built with the `e2e` build tag so they are never included in `task test`.

**When to add/update E2E tests**:
- Add a test whenever a new route or user-facing feature is introduced
- Update existing tests when a route path, form field name, or page structure changes
- Tests must cover both the happy path and key role-based visibility rules (e.g., chairperson vs. member)

**Key helpers** (all in `e2e/helpers_test.go` / `e2e/browser_helpers_test.go`):
- `newTestServer(t)` — boots the full app with `:memory:` SQLite
- `newPage(t)` — launches isolated Chromium context; skips gracefully if browsers not installed
- `adminLogin(t, page, baseURL)` / `userLogin(t, page, baseURL, committee, username, password)` — shared login flows
- `ts.seedCommittee`, `ts.seedUser`, `ts.seedMeeting` — direct DB seeding without HTTP

**HTMX assertions in tests**:
- Capture `urlBefore := page.URL()` before an HTMX form submit and assert `page.URL() == urlBefore` after to confirm no full navigation occurred
- Use `page.Locator("td:has-text('value')").WaitFor()` to wait for swapped-in content
- Use `locator.WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateDetached})` to wait for removed content
- Register `page.OnDialog(func(d playwright.Dialog) { d.Accept() })` **before** clicking a button with `hx-confirm`

If Playwright browsers are not installed (`task playwright:install` has not been run), all E2E tests skip automatically via `t.Skip()`.

`newTestServer(t)` also loads i18n catalogs and wraps handlers with locale middleware. Keep this wiring when changing test server setup so templates do not render `!(MISSING ...)` in E2E runs.

### Handler Implementation

Handlers must implement the generated `Handler` interface from [internal/routes/routes_gen.go](internal/routes/routes_gen.go):
- Methods match route definitions in routes.yaml
- Return signature: `(*TemplateInputType, error)` for template routes
- Return signature: `error` for SSE routes (handlers manage their own response writing)
- Path parameters are passed via `RouteParams` struct
- HTTP request/response via standard `http.ResponseWriter` and `*http.Request`

### SSE Handler Pattern

For Server-Sent Events endpoints:
1. Mark route with `sse: true` in routes.yaml
2. Handler subscribes to broker channel: `ch := h.Broker.Subscribe(r.Context())`
3. Handler writes SSE format: `fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)`
4. Handler must flush after each write: `flusher.Flush()`
5. Handler returns when client disconnects: `<-r.Context().Done()`

### Error Handling in Handlers

- Return errors for issues like database failures, validation errors, not found
- Router automatically converts errors to HTTP 500 responses
- For custom status codes, write directly to ResponseWriter and return nil error

## Project Structure

```
.
├── cmd/                          # CLI commands (cobra)
│   ├── root.go                   # Root command
│   └── serve.go                  # HTTP server command
├── e2e/                          # Browser-based E2E tests (build tag: e2e)
│   ├── main_test.go              # TestMain: Playwright driver init
│   ├── helpers_test.go           # testServer, seed helpers
│   ├── browser_helpers_test.go   # newPage, adminLogin, userLogin
│   ├── admin_test.go             # Admin UI tests
│   └── committee_test.go         # Committee/chairperson UI tests
├── internal/
│   ├── broker/                   # SSE broker for real-time updates
│   ├── config/                   # Configuration management with Viper
│   ├── handlers/                 # HTTP request handlers
│   ├── middleware/               # HTTP middleware registry
│   ├── repository/sqlite/        # Database layer
│   │   ├── migrations/           # SQL migrations
│   │   ├── queries/              # SQLC queries
│   │   ├── client/               # Generated SQLC code
│   │   └── sqlc.yaml             # SQLC configuration
│   ├── routes/                   # Generated routing code
│   │   ├── gen.go                # go:generate directive
│   │   └── routes_gen.go         # Generated (DO NOT EDIT)
│   └── templates/                # Templ templates
│       ├── *.templ               # Template source files
│       └── *_templ.go            # Generated (DO NOT EDIT)
├── tools/routing/                # Custom route code generator
├── routes.yaml                   # Route definitions (source of truth)
├── .air.toml                     # Hot reload configuration
└── flake.nix                     # Nix development environment
```

## Configuration

Environment variables (see [.env.example](.env.example)):
- `ENVIRONMENT`: development, staging, production
- `HOST`: HTTP server bind address (default: 0.0.0.0)
- `PORT`: HTTP server port (default: 8080)
- `SERVICE_NAME`: Service identifier (default: conference-tool)
- `LOG_LEVEL`: debug, info, warn, error
- `LOG_FORMAT`: json, text

## Common Workflows

### Adding a New Feature with UI

1. Design the data model and add database migrations if needed
2. Add SQLC queries in `internal/repository/sqlite/queries/`
3. Define route in [routes.yaml](routes.yaml)
4. Create Templ template in `internal/templates/` — use HTMX attributes so the feature never causes a full page reload
5. Run `go generate ./...`
6. Implement handler method in [internal/handlers/handlers.go](internal/handlers/handlers.go)
7. Add business logic and database calls
8. Add E2E tests in [e2e/](e2e/) covering the new route and any role-based visibility
9. Run `task test:e2e` to verify

### Adding Real-Time Updates

1. Define SSE route with `sse: true` in routes.yaml
2. Implement SSE handler following the pattern in [internal/handlers/handlers.go](internal/handlers/handlers.go):Subscribe
3. Publish events through the broker when data changes
4. Use HTMX `hx-ext="sse"` in templates to consume events

### Modifying the Routing System

The route generator is in [tools/routing/](tools/routing/):
- [types.go](tools/routing/types.go): YAML structure definitions
- [parser.go](tools/routing/parser.go): YAML parsing and validation
- [generator.go](tools/routing/generator.go): Code generation logic

Changes to the generator require rebuilding: `go build ./tools/routing/cmd/route-codegen`

### Releasing a New Version

When tagging a new version:
1. Add an entry to the patch notes in `doc/content/07-patchnotes/index.{en,de}.md` covering all changes since the last tag.
2. Follow the existing format: `## vX.Y.Z — YYYY-MM-DD` with `### New Features`, `### Improvements`, and/or `### Bug Fixes` subsections as appropriate.

### Keeping Documentation Up to Date

When implementing new features or making user-visible changes:
1. Read all files under `doc/content/` to check whether any existing documentation needs updating.
2. Update both English (`.en.md`) and German (`.de.md`) variants of any affected doc pages.
3. Documentation lives in `doc/content/` and is embedded in the app — it must stay accurate.


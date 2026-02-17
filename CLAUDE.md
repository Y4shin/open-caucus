# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A conference management tool built with Go, HTMX, and Templ. The project features a custom type-safe routing system using YAML-based route definitions with automatic code generation.

## Development Environment

The project uses Nix flakes with direnv for reproducible development environments. The flake provides:
- Go 1.25.5
- gopls, gotools, golangci-lint
- go-task for task running
- MCP servers (mcp-gopls, mcp-taskfile-server) for AI assistance

Alternatively, ensure Go 1.25.5+ is installed directly.

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

Database migration commands are not yet implemented in the binary. The migrations exist in [internal/repository/sqlite/migrations/](internal/repository/sqlite/migrations/).

```bash
# Placeholder tasks (not yet implemented)
task db:migrate:up
task db:migrate:down
task db:migrate:create
task db:reset
```

### Testing

```bash
# Using Task (preferred)
task test                  # Run all tests
task test:verbose          # Run with verbose output
task test:coverage         # Generate coverage report
task test:watch            # Run tests in watch mode

# Direct commands
go test ./...              # Run all tests
go test -v ./...           # Verbose output
go test ./internal/config/...           # Specific package
go test ./internal/config -run TestName # Specific test
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
task setup            # Initial project setup
task fresh            # Clean, generate, build, and run
task ci               # Run all CI checks
task deps:tidy        # Tidy go.mod and go.sum
```

## Architecture Overview

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
   - Creates type-safe URL builders

2. **Template Generator** ([internal/templates/gen.go](internal/templates/gen.go)):
   - Compiles `.templ` files to Go code using `templ generate`
   - Each template becomes a Go function

3. **Database Client** ([internal/repository/sqlite/gen.go](internal/repository/sqlite/gen.go)):
   - Reads SQL schema and queries
   - Generates type-safe database client using `sqlc generate`

**DO NOT manually edit generated files** - they have `_gen.go`, `_generated.go`, or `_templ.go` suffixes.

## Important Patterns

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
4. Create Templ template in `internal/templates/`
5. Run `go generate ./...`
6. Implement handler method in [internal/handlers/handlers.go](internal/handlers/handlers.go)
7. Add business logic and database calls
8. Test the endpoint

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

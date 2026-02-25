# Conference Tool

A web application built with Go, HTMX, and Templ featuring type-safe routing through code generation.

## Features

- **Type-Safe Routing**: YAML-based route definitions with automatic code generation
- **HTMX Integration**: Dynamic, modern web UX without heavy JavaScript frameworks
- **Templ Templates**: Type-safe HTML templating with Go
- **Middleware Support**: Flexible middleware system with prefix-based and route-specific middleware
- **Server-Sent Events (SSE)**: Built-in support for real-time updates
- **Path Parameters**: Automatic extraction and type-safe handling

## Project Structure

```
.
├── cmd/
│   └── conference-tool/         # CLI application
├── internal/
│   ├── config/                  # Configuration management
│   ├── routes/                  # Generated routing code
│   │   ├── gen.go              # go:generate directive
│   │   └── routes_gen.go       # Generated router (DO NOT EDIT)
│   └── templates/              # Templ templates
│       ├── *.templ             # Template definitions
│       ├── *_templ.go          # Generated Go code from templates
│       └── gen.go              # go:generate directive for templ
├── tools/
│   └── routing/                # Route code generator
│       ├── cmd/route-codegen/  # CLI tool for code generation
│       ├── types.go            # YAML configuration types
│       ├── parser.go           # YAML parser and validator
│       └── generator.go        # Code generation logic
├── routes.yaml                 # Route definitions
├── go.mod
└── README.md
```

## Getting Started

### Prerequisites

- Go 1.25.5 or later
- Node.js 20+ (for Tailwind CSS/DaisyUI build)
- The project uses Go tools (templ, air, sqlc) which are automatically managed via `go.mod`
- (Optional) [Task](https://taskfile.dev/) for running common development tasks

### Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd conference-tool
   ```

2. Install dependencies and generate code:
   ```bash
   # Using Task (recommended)
   task setup

   # Or manually
   go mod download
   npm install
   go generate ./...
   ```

### Running the Application

```bash
# Using Task (recommended)
task dev              # Start with hot reload
task css:watch        # Run in a second terminal for CSS rebuilds
task run              # Run directly

# Or manually
go tool air           # Start with hot reload
npm run watch:css     # Run in a second terminal for CSS rebuilds
go run . serve        # Run directly
```

## Route Code Generation

This project uses a custom code generation tool to create type-safe routing from YAML definitions.

### Defining Routes

Routes are defined in [`routes.yaml`](routes.yaml):

```yaml
version: 1.0

# Middleware groups apply to all routes matching the prefix
middleware_groups:
  - prefix: /api
    middleware: [CORS, RateLimit]

routes:
  - path: /
    methods:
      - verb: GET
        handler: HomePage
        template:
          package: github.com/Y4shin/conference-tool/internal/templates
          type: HomePageTemplate
          input_type: HomePageInput
        middleware: [Logging]

  - path: /posts/{id}
    methods:
      - verb: GET
        handler: GetPost
        template:
          package: github.com/Y4shin/conference-tool/internal/templates
          type: PostDetailTemplate
          input_type: Post
```

### Route Configuration

Each route requires:
- **path**: URL path with optional `{param}` placeholders
- **verb**: HTTP method (GET, POST, PUT, DELETE, etc.)
- **handler**: Name of the handler method
- **template**:
  - `package`: Full import path to template package
  - `type`: Name of the templ component
  - `input_type`: Type of the input parameter for the template
- **middleware** (optional): Route-specific middleware
- **sse** (optional): Set to `true` for Server-Sent Events endpoints

### Middleware Groups

Middleware can be applied to all routes matching a prefix:

```yaml
middleware_groups:
  - prefix: /api
    middleware: [CORS, RateLimit]

  - prefix: /admin
    middleware: [RequireAuth, RequireAdmin]
```

Routes automatically inherit middleware from all matching prefixes. More specific prefixes are applied first.

### Generated Code

The code generator creates:

1. **Handler Interface**: Type-safe interface with methods returning template input data
   ```go
   type Handler interface {
       HomePage(w http.ResponseWriter, r *http.Request) (*templates.HomePageInput, error)
       GetPost(w http.ResponseWriter, r *http.Request, params RouteParams) (*templates.Post, error)
   }
   ```

2. **Router**: Registers all routes with the HTTP server
   ```go
   router := routes.NewRouter(handler, middleware)
   http.ListenAndServe(":8080", router.RegisterRoutes())
   ```

3. **RouteParams**: Struct containing all path parameters
   ```go
   type RouteParams struct {
       Id string
   }
   ```

4. **Type-safe URL Builders**: Functions for constructing URLs
   ```go
   routes.Route.HomePageGet()                    // "/"
   routes.PostsIdRoute{id: "123"}.Get()         // "/posts/123"
   ```

### Regenerating Routes

After modifying `routes.yaml`:

```bash
# Using Task (recommended)
task generate:routes    # Generate only routes
task generate           # Generate everything

# Or manually
go generate ./internal/routes
go generate ./...
```

## Templates

Templates are written using [Templ](https://templ.guide/), a type-safe HTML templating language.

### Creating Templates

Templates are defined in `internal/templates/*.templ`:

```templ
package templates

type Post struct {
    ID      string
    Title   string
    Content string
    Author  string
}

templ PostDetailTemplate(input Post) {
    <!DOCTYPE html>
    <html>
        <head>
            <title>{ input.Title }</title>
        </head>
        <body>
            <h1>{ input.Title }</h1>
            <p>By { input.Author }</p>
            <div>{ input.Content }</div>
        </body>
    </html>
}
```

### Generating Template Code

```bash
# Using Task (recommended)
task generate:templates

# Or manually
go generate ./internal/templates
```

This generates `*_templ.go` files with the compiled template functions.

## Implementing Handlers

Implement the generated `Handler` interface:

```go
package myapp

import (
    "net/http"
    "github.com/Y4shin/conference-tool/internal/routes"
    "github.com/Y4shin/conference-tool/internal/templates"
)

type MyHandler struct {
    db *sql.DB
}

func (h *MyHandler) HomePage(w http.ResponseWriter, r *http.Request) (*templates.HomePageInput, error) {
    return &templates.HomePageInput{
        Title: "Welcome",
    }, nil
}

func (h *MyHandler) GetPost(w http.ResponseWriter, r *http.Request, params routes.RouteParams) (*templates.Post, error) {
    post, err := h.db.GetPost(params.Id)
    if err != nil {
        return nil, err
    }

    return &templates.Post{
        ID:      post.ID,
        Title:   post.Title,
        Content: post.Content,
        Author:  post.Author,
    }, nil
}
```

## Middleware

Create a middleware registry:

```go
package middleware

import "net/http"

type Registry struct {
    middlewares map[string]func(http.Handler) http.Handler
}

func NewRegistry() *Registry {
    r := &Registry{
        middlewares: make(map[string]func(http.Handler) http.Handler),
    }

    r.Register("Logging", loggingMiddleware)
    r.Register("CORS", corsMiddleware)
    r.Register("RateLimit", rateLimitMiddleware)
    r.Register("RequireAuth", authMiddleware)

    return r
}

func (r *Registry) Get(name string) func(http.Handler) http.Handler {
    return r.middlewares[name]
}

func (r *Registry) Register(name string, mw func(http.Handler) http.Handler) {
    r.middlewares[name] = mw
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Log request
        next.ServeHTTP(w, r)
    })
}
```

## Configuration

Configuration is managed via environment variables and config files. See [`internal/config/`](internal/config) for details.

## Development

### Available Commands

The project uses [Taskfile](https://taskfile.dev/) for common development tasks. Run `task --list` to see all available commands.

Key tasks:
- `task dev` - Run with hot reload
- `task css:watch` - Watch and rebuild Tailwind + DaisyUI CSS
- `task css:build` - Build Tailwind + DaisyUI CSS once
- `task test` - Run tests
- `task generate` - Generate all code
- `task check` - Run all code quality checks
- `task ci` - Run CI checks

See [Taskfile.yaml](Taskfile.yaml) for the complete list of available tasks.

### Code Generation

The project uses `go generate` for code generation:

```bash
# Using Task (recommended)
task generate              # All code generation
task css:build             # Stylesheet only
task generate:routes       # Routes only
task generate:templates    # Templates only
task generate:db          # Database client only

# Or manually
npm run build:css                    # Stylesheet
go generate ./...                      # All
go generate ./internal/routes          # Routes
go generate ./internal/templates       # Templates
go generate ./internal/repository/sqlite  # Database
```

### Adding New Routes

1. Add route definition to `routes.yaml`
2. Create corresponding templ template in `internal/templates/`
3. Run `task generate` (or `go generate ./...`)
4. Implement the handler method
5. Register any new middleware if needed

### Building the Code Generator

```bash
cd tools/routing/cmd/route-codegen
go build
```

## Architecture

### Type-Safe Routing Flow

1. **Define** routes in YAML with template input types
2. **Generate** Handler interface and router code
3. **Implement** handlers that return template input data
4. **Router** automatically:
   - Extracts path parameters
   - Calls handler method
   - Handles errors (returns HTTP 500)
   - Passes input to template
   - Renders template to response

### Benefits

- **Compile-time Safety**: Invalid routes, missing handlers, or type mismatches caught at build time
- **No Manual Wiring**: Router generation eliminates boilerplate
- **Clear Separation**: Handlers prepare data, templates render HTML
- **Refactoring Support**: Rename handlers or change signatures with confidence
- **Documentation**: YAML serves as route documentation

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]

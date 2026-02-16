# Route Code Generator

A code generation tool for creating type-safe HTTP routing with Go, HTMX, and Templ.

## Features

- **Type-safe Handler Interface**: Generates an interface with methods for each route, ensuring all handlers are implemented with correct signatures
- **Automatic Route Registration**: Creates a router that automatically registers all routes with the HTTP server
- **Middleware Support**: Supports both prefix-based middleware groups and route-specific middleware
- **Path Parameter Extraction**: Automatically extracts and validates path parameters
- **SSE Support**: Special handling for Server-Sent Events endpoints
- **Type-safe URL Builders**: Generates builder types for constructing URLs safely at compile time

## Usage

### 1. Create a Route Configuration YAML

Create a `routes.yaml` file defining your application routes:

```yaml
version: 1.0

middleware_groups:
  - prefix: /api
    middleware: [CORS, RateLimit]

routes:
  - path: /
    methods:
      - verb: GET
        handler: HomePage
        template:
          package: github.com/user/blog/templates
          type: HomePageTemplate

  - path: /posts/{id}
    methods:
      - verb: GET
        handler: GetPost
        template:
          package: github.com/user/blog/templates
          type: PostDetailTemplate
```

### 2. Run the Code Generator

```bash
go run ./tools/routing/main.go \
  -config routes.yaml \
  -output internal/routes/routes_gen.go \
  -package routes
```

### 3. Implement the Handler Interface

```go
package myapp

import (
    "net/http"
    "github.com/user/blog/internal/routes"
    "github.com/user/blog/templates"
)

type MyHandler struct {
    db *sql.DB
}

func (h *MyHandler) HomePage(w http.ResponseWriter, r *http.Request) templates.HomePageTemplate {
    return templates.HomePageTemplate{
        Title: "Welcome",
    }
}

func (h *MyHandler) GetPost(w http.ResponseWriter, r *http.Request, params routes.RouteParams) templates.PostDetailTemplate {
    post := h.db.GetPost(params.ID)
    return templates.PostDetailTemplate{
        Post: post,
    }
}
```

### 4. Create a Middleware Registry

```go
package middleware

type Registry struct {
    middlewares map[string]func(http.Handler) http.Handler
}

func NewRegistry() *Registry {
    r := &Registry{
        middlewares: make(map[string]func(http.Handler) http.Handler),
    }

    r.Register("CORS", corsMiddleware)
    r.Register("RateLimit", rateLimitMiddleware)
    r.Register("Logging", loggingMiddleware)

    return r
}

func (r *Registry) Register(name string, mw func(http.Handler) http.Handler) {
    r.middlewares[name] = mw
}

func (r *Registry) Get(name string) func(http.Handler) http.Handler {
    return r.middlewares[name]
}

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        next.ServeHTTP(w, r)
    })
}

// ... other middleware implementations
```

### 5. Start the Server

```go
package main

import (
    "net/http"
    "github.com/user/blog/internal/routes"
    "github.com/user/blog/internal/middleware"
)

func main() {
    handler := &MyHandler{db: setupDB()}
    middlewareRegistry := middleware.NewRegistry()

    router := routes.NewRouter(handler, middlewareRegistry)

    http.ListenAndServe(":8080", router.RegisterRoutes())
}
```

## YAML Configuration Reference

### Route Definition

```yaml
- path: /posts/{id}          # Route path with optional {param} placeholders
  methods:
    - verb: GET                # HTTP method: GET, POST, PUT, DELETE, etc.
      handler: GetPost         # Handler method name (must be unique)
      template:
        package: github.com/user/blog/templates  # Full import path
        type: PostDetailTemplate                 # Templ component type
      middleware: [Auth]       # Optional route-specific middleware
      sse: false               # Optional: true for Server-Sent Events
```

### Middleware Groups

```yaml
middleware_groups:
  - prefix: /api              # All routes starting with /api
    middleware: [CORS, RateLimit]

  - prefix: /api/protected    # More specific prefixes take precedence
    middleware: [Auth]
```

Middleware from multiple matching prefixes will be combined. Route-specific middleware is applied after prefix middleware.

## Generated Code Structure

The generator creates:

1. **Handler Interface**: Interface with methods for each route
2. **RouteParams Struct**: Contains all path parameters across routes
3. **Router**: Main router struct with route registration logic
4. **MiddlewareRegistry Interface**: Interface for middleware lookup
5. **Route Builders**: Type-safe URL builder functions

## Example Generated Code

See `example-routes.yaml` for a complete example configuration.

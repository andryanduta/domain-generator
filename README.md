# domain-generator

[![CI](https://github.com/andryanduta/domain-generator/actions/workflows/ci.yml/badge.svg)](https://github.com/andryanduta/domain-generator/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/andryanduta/domain-generator.svg)](https://pkg.go.dev/github.com/andryanduta/domain-generator)

Scaffold a DDD-style Go domain package in one command, for people **and for AI coding agents**.

```console
$ domain-generator -name order -store -http
Created /src/shop/order (example.com/shop/order):
  AGENTS.md
  httpapi/handlers.go
  httpapi/handlers_test.go
  httpapi/respond.go
  httpapi/routes.go
  order.go
  service/service.go
  service/service_test.go
  sqlstore/sqlstore.go

Next: go get github.com/jmoiron/sqlx
```

## Why use a generator when you have an AI agent?

Agents are good at business logic and bad at consistency. Ask one for "a new domain" ten times
and you get ten slightly different layouts, and you pay tokens for boilerplate each time.
domain-generator handles the part that should never vary:

- **Deterministic.** The same flags always produce the same files. Every `.go` file is gofmt'ed,
  and CI checks that every layer combination compiles, passes `go vet`, and passes its own tests.
- **Agent-native.** Run it as an [MCP](https://modelcontextprotocol.io) server (`domain-generator mcp`)
  so Claude Code, Cursor, Codex and other agents can call `generate_domain` as a tool. There's also
  an [Agent Skill](skills/domain-generator/SKILL.md) and `-json` / `-dry-run` output for scripts.
- **Leaves a spec behind.** Each generated domain includes an `AGENTS.md` with its layer rules and
  table-driven tests, so the agent (or teammate) filling in the logic has guardrails, not just TODOs.
- **Safe.** It validates the name, never overwrites an existing directory, and writes atomically,
  so a failed run leaves nothing behind.

## Install

```console
$ go install github.com/andryanduta/domain-generator@latest
```

Requires Go 1.25 or later to install. Generated code needs Go 1.22+ (method-based `http.ServeMux` routes).

## Usage

Run it inside a Go module. The import path is detected from the nearest `go.mod`.

```console
$ domain-generator -name order                          # asks which layers to add
$ domain-generator -name order -dir internal -all       # recommended for applications
$ domain-generator -name order -all -dry-run -json      # plan only, machine-readable
```

| Flag | Meaning |
|---|---|
| `-name` | Domain name: a lowercase Go identifier such as `order` (required). Prefer one short word, as [Effective Go](https://go.dev/doc/effective_go#package-names) recommends; `user_profile` style names work and produce `UserProfile` types and `/user-profiles` routes |
| `-store`, `-cache`, `-http` | Add the SQL repository (sqlx), cache, or HTTP API layer |
| `-all` | Add every layer |
| `-dir` | Parent directory to create the domain in (default `.`). Use `internal` in applications so other modules can't import your domains |
| `-module` | Import path of `-dir`, if you're not inside a Go module |
| `-no-prompt` | Never ask; generate only the layers given by flags |
| `-dry-run` | Show what would be generated, write nothing |
| `-json` | Print the result as JSON (implies `-no-prompt`) |

## Generated layout

```text
order/
├── AGENTS.md               layer rules for people and agents
├── order.go                entities, domain errors, Service interface (no globals)
├── service/
│   ├── service.go          implements order.Service; declares Repository and Cache
│   └── service_test.go     table-driven tests with in-memory fakes
├── sqlstore/               -store
│   └── sqlstore.go         Repository with sqlx: context everywhere, WithTx, row ↔ domain mapping
├── cache/                  -cache
│   └── cache.go            in-memory Cache (swap in Redis behind the same interface)
└── httpapi/                -http
    ├── routes.go           RegisterRoutes(mux, logger, svc) with Go 1.22 method patterns
    ├── handlers.go         GET /orders/{id}, POST /orders
    ├── respond.go          JSON decode/encode, RFC 9457 errors, slog
    └── handlers_test.go    httptest table tests
```

The design follows well-known Go guidance rather than inventing its own:

| Choice | Why |
|---|---|
| Domain types and the `Service` interface in the root package, implementations in subpackages | Ben Johnson, [Standard Package Layout](https://www.gobeyond.dev/standard-package-layout/) |
| Interfaces (`Repository`, `Cache`) declared by the package that uses them | [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments#interfaces) |
| Adapter packages named after what they depend on (`sqlstore`, `httpapi`), not `store`/`handler` | Standard Package Layout; also avoids shadowing `net/http` |
| No package-level state; everything wired with constructors in `main` | Peter Bourgon, [A theory of modern Go](https://peter.bourgon.org/blog/2017/06/09/theory-of-modern-go.html) |
| `RegisterRoutes(mux, ...)`, handlers that return `http.Handler`, no global `DefaultServeMux` | Mat Ryer, [How I write HTTP services in Go after 13 years](https://grafana.com/blog/how-i-write-http-services-in-go-after-13-years/) |
| Transactions through `WithTx(ctx, fn)` instead of leaking Commit/Rollback into the service | Three Dots Labs, [Repository pattern in Go](https://threedots.tech/post/repository-pattern-in-go/) |
| No `pkg/`, optional `internal/` | [Organizing a Go module](https://go.dev/doc/modules/layout); Russ Cox on [project-layout](https://github.com/golang-standards/project-layout/issues/117) |

## Response format

Successful responses are the resource as plain JSON. Errors are
[RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) problem details:

```console
$ curl -i localhost:8080/orders/99
HTTP/1.1 404 Not Found
Content-Type: application/problem+json

{"type":"about:blank","title":"Not Found","status":404,"detail":"order: not found","instance":"/orders/99"}
```

Domain errors map to status codes in one place (`ErrNotFound` → 404, `ErrInvalid` → 400). Anything
else becomes a 500 that is logged but not shown to the client.

Need a JSON envelope instead (`header` / `status` / `data` / `errors`, as used at Tokopedia and many
other large codebases)? See [docs/response-formats.md](docs/response-formats.md) for the shape, the
trade-offs and a drop-in `respond.go`.

## Wiring it up

```go
db := sqlx.MustConnect("postgres", dsn)
svc := service.New(sqlstore.New(db), cache.NewMemory())

mux := http.NewServeMux()
httpapi.RegisterRoutes(mux, logger, svc)

srv := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
log.Fatal(srv.ListenAndServe())
```

## Use it from an AI agent

**MCP server.** Add it to your agent's MCP config. For example, with Claude Code:

```console
$ claude mcp add domain-generator -- domain-generator mcp
```

Or in any client that reads an `mcpServers` JSON config:

```json
{
  "mcpServers": {
    "domain-generator": { "command": "domain-generator", "args": ["mcp"] }
  }
}
```

The server exposes one tool, `generate_domain` (`name`, absolute `dir`, optional `module`,
`store`, `cache`, `http`, `dry_run`), and returns the created files and any `go get` steps as
structured output.

**Agent Skill.** Copy [`skills/domain-generator`](skills/domain-generator) into your agent's skills
directory (e.g. `~/.claude/skills/`) so the agent knows when to reach for the tool.

## Development

```console
$ go test ./...       # includes building and testing every generated layer combination
$ go vet ./...
```

See [AGENTS.md](AGENTS.md) for how the code is organized and the rules for changing templates.

## License

MIT. See [LICENSE](LICENSE).

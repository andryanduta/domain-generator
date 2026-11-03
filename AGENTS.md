# AGENTS.md

Instructions for AI coding agents (and humans) working on this repository.

## What this is

A Go CLI and library that scaffolds DDD-style domain packages. Three entry points share one core:

| Path | Role |
|---|---|
| `generator/generator.go` | Core: `ValidateName`, `DetectModule`, `Render` (pure, no I/O), `Generate` (atomic write) |
| `generator/template/` | `text/template` files, embedded with `go:embed` |
| `main.go` | CLI flags, interactive prompts, `-json` output |
| `mcp.go` | MCP server exposing the `generate_domain` tool (official Go SDK) |
| `skills/domain-generator/SKILL.md` | Agent Skill shipped to users |

## Commands

```console
go build ./...
go vet ./...
go test -race ./...                 # full suite; builds and tests 16 generated variants (needs module download access)
go test -short ./...                # skips building generated code (offline)
gofmt -l .                          # must print nothing
```

## Rules

1. **Generated code must compile and pass its own tests.** `TestGeneratedCodeCompiles` generates
   17 layer combinations (every store/cache/http mix, every queue, cron) for `order` and `user_profile`, then runs `go vet` and `go test` on
   them against the real `sqlx`. Keep it green. Generated code should also be clean under
   staticcheck.
2. **Templates use `text/template`** with `{{.Name}}` (package), `{{.Type}}` (entity, e.g.
   `UserProfile`), `{{.Var}}`, `{{.Route}}` (e.g. `user-profiles`), `{{.Table}}`, `{{.ImportPath}}`,
   `{{.Store}}`, `{{.Cache}}`, `{{.HTTP}}`, `{{.Queue}}`, `{{.QueuePkg}}`, `{{.Cron}}`. A literal `{{` in Go code breaks parsing, so avoid
   nested composite literals like `[]T{{...}}` in templates.
3. **Generated code follows Go community practice:** no package-level state, interfaces declared
   by their consumer, adapters named by dependency (`sqlstore`, `handler/httpapi`,
   `handler/kafkaconsumer`), inbound adapters under `handler/`, `context.Context` first, RFC 9457
   errors, idempotent consumers that separate permanent from retryable failures. Read the "design" table in `README.md` before changing the layout.
4. **Keep `Render` pure.** Disk access belongs in `Generate`, which writes to a temp directory and
   renames, never overwriting an existing domain.
5. **The library never prints or exits.** Return errors; `main.go` decides what to show.
6. **Adding a queue:** add an entry to `queues` in `generator.go`, write
   `template/handler/<pkg>/consumer.go.tmpl` and `consumer_test.go.tmpl` that call the shared
   `handle` from `template/handler/consumer/message.go.tmpl`, pin the client in `pinnedDeps` in the
   tests, and add a row to the README queue table.
7. **Adding a template:** add the file under `generator/template/`, register it in the `files`
   list with its `include` condition, update the expected file lists in the tests and the
   layout in `README.md`. If it imports a new package, add that name to `reservedNames`.
8. **The MCP tool schema is user-facing.** The `jsonschema` tags on `generateInput` are what agents
   read, so keep them precise. Keep `dir` absolute-only: the server's working directory is not the
   agent's.
9. Keep the minimum Go version in `go.mod` at the oldest supported release; check with
   `GOTOOLCHAIN=go1.25.0 go test ./...` before bumping dependencies.

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
go test -race ./...                 # full suite, ~2 s
go test -short ./...                # skips compiling generated code
gofmt -l .                          # must print nothing
```

## Rules

1. **Generated code must compile.** `TestGeneratedCodeCompiles` builds all 8 layer combinations
   against a stub `sqlx`. Any template change must keep it green. Generated code should also be
   clean under `go vet` and staticcheck.
2. **Templates use `text/template`** with `{{.Name}}`, `{{.ImportPath}}`, `{{.Store}}`, `{{.Cache}}`,
   `{{.HTTP}}`. A literal `{{` in Go code breaks parsing, so avoid nested composite literals
   like `[]T{{...}}` in templates.
3. **Keep `Render` pure.** Disk access belongs in `Generate`, which writes to a temp directory and
   renames, never overwriting an existing domain.
4. **The library never prints or exits.** Return errors; `main.go` decides what to show.
5. **Adding a template:** add the file under `generator/template/`, register it in the `files`
   list with its `include` condition, update the expected file lists in the tests and the
   layout in `README.md`.
6. **The MCP tool schema is user-facing.** The `jsonschema` tags on `generateInput` are what agents
   read, so keep them precise. Keep `dir` absolute-only: the server's working directory is not the
   agent's.
7. Keep the minimum Go version in `go.mod` at the oldest supported release; check with
   `GOTOOLCHAIN=go1.25.0 go test ./...` before bumping dependencies.

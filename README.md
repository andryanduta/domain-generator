# domain-generator

[![CI](https://github.com/andryanduta/domain-generator/actions/workflows/ci.yml/badge.svg)](https://github.com/andryanduta/domain-generator/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/andryanduta/domain-generator.svg)](https://pkg.go.dev/github.com/andryanduta/domain-generator)

Scaffold a DDD-style Go domain package in one command, for people **and for AI coding agents**.

```console
$ domain-generator -name order -store -http
Created /src/shop/order (example.com/shop/order):
  AGENTS.md
  handler/http/error.go
  handler/http/foo_handler.go
  handler/http/http.go
  handler/http/util.go
  order.go
  service/service.go
  store/store.go

Next: go get github.com/jmoiron/sqlx
```

## Why use a generator when you have an AI agent?

Agents are good at business logic and bad at consistency. Ask one for "a new domain" ten times
and you get ten slightly different layouts, and you pay tokens for boilerplate each time.
domain-generator handles the part that should never vary:

- **Deterministic.** The same flags always produce the same files. Every `.go` file is gofmt'ed,
  and CI checks that every layer combination compiles and passes `go vet`.
- **Agent-native.** Run it as an [MCP](https://modelcontextprotocol.io) server (`domain-generator mcp`)
  so Claude Code, Cursor, Codex and other agents can call `generate_domain` as a tool. There's also
  an [Agent Skill](skills/domain-generator/SKILL.md) and `-json` / `-dry-run` output for scripts.
- **Leaves instructions behind.** Each generated domain includes an `AGENTS.md` with its layer
  rules, so the agent (or teammate) filling in the TODOs keeps the architecture intact.
- **Safe.** It validates the name, never overwrites an existing directory, and writes atomically,
  so a failed run leaves nothing behind.

## Install

```console
$ go install github.com/andryanduta/domain-generator@latest
```

Requires Go 1.25 or later.

## Usage

Run it inside a Go module. The import path is detected from the nearest `go.mod`.

```console
$ domain-generator -name order                    # asks which layers to add
$ domain-generator -name order -store -cache -http
$ domain-generator -name order -all -dry-run -json # plan only, machine-readable
```

| Flag | Meaning |
|---|---|
| `-name` | Domain name: a lowercase Go identifier such as `order` or `user_profile` (required) |
| `-store`, `-cache`, `-http` | Add the SQL store (sqlx), cache store, or HTTP handler layer |
| `-all` | Add every layer |
| `-dir` | Parent directory to create the domain in (default `.`) |
| `-module` | Import path of `-dir`, if you're not inside a Go module |
| `-no-prompt` | Never ask; generate only the layers given by flags |
| `-dry-run` | Show what would be generated, write nothing |
| `-json` | Print the result as JSON (implies `-no-prompt`) |

## Generated layout

```text
order/
├── AGENTS.md             layer rules for people and agents
├── order.go              domain contract: entities + Service interface
├── service/service.go    business logic, implements order.Service
├── store/                optional
│   ├── store.go          SQL store (sqlx), implements service.Store
│   └── cachestore.go     cache store
└── handler/http/         optional
    ├── http.go           handler registration and scopes
    ├── foo_handler.go    example handler with request timeout
    ├── error.go          known errors
    └── util.go           JSON envelope helpers
```

Dependencies point inward (handler → domain ← service ← store), with compile-time
interface checks so drift breaks the build. `//go:generate` directives for
[mockgen](https://github.com/uber-go/mock) are included.

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
$ go test ./...       # includes compiling every generated layer combination
$ go vet ./...
```

See [AGENTS.md](AGENTS.md) for how the code is organized and the rules for changing templates.

## License

MIT. See [LICENSE](LICENSE).

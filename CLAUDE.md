# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What Is This

shard-cli is a pure Go terminal interface to Shard-Link's memory system. It communicates directly with a remote MCP server over HTTP — no LLM SDK, no Claude API tokens. The binary name is `shard`.

## Build & Run

```bash
# Initialize module (first time only)
go mod init github.com/izenberk/shard-cli
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest

# Build
go build -o shard .

# Install system-wide
sudo mv shard /usr/local/bin/shard

# Run tests
go test ./...

# Run a single package's tests
go test ./internal/client/
```

## Architecture

**Layered dependency chain:** `config → client → format → commands`

Each layer depends only on layers below it and is testable in isolation.

### Layers

| Layer | Package | Role |
|-------|---------|------|
| 1 — Config | `internal/config/` | Loads settings from CLI flags → env vars → `~/.shard/config.yaml` (priority order). Zero deps. |
| 2 — Client | `internal/client/` | `MCPClient` — HTTP client to the MCP server. Methods: `SearchAll`, `SaveMemory`, `GetCoreShards`, `GetShardByID`, `GetStatus`. Core of the entire CLI. |
| 3 — Format | `internal/format/` | Terminal output rendering. Two modes: human-readable (default) and `--json` (pipe-friendly). |
| 4 — Commands | `cmd/` | Cobra commands wiring the above layers. `root.go` defines global flags; each subcommand gets its own file. |

Entry point: `main.go` → `cmd.Execute()`.

### CLI Commands

- `shard status` — mesh health check (shards, bonds, communities, service status)
- `shard search "query"` — semantic memory search (`--limit`, `--category`, `--json`)
- `shard save "content"` — persist memory from args, stdin pipe, or `--file` (`--id`, `--category`)
- `shard get "id"` — fetch shard by ID or `--core` for core shards

### Config Precedence

1. CLI flags (`--hub-url`, `--api-key`)
2. Environment variables (`SHARD_HUB_URL`, `SHARD_API_KEY`)
3. Config file (`~/.shard/config.yaml`)

## Dependencies

Only two external dependencies — everything else uses standard library:

- **cobra** — CLI subcommand framework
- **viper** — config file + env var loading

## Shard Categories

When saving to Shard-Link (via CLI or `/shard` skill), always use the correct category:

| Category | Purpose |
|----------|---------|
| `core` | Identity, preferences, long-term facts about the user |
| `memory` | General knowledge, learnings, session notes (default for `shard save`) |
| `session` | Ephemeral context tied to a specific work session |
| `contract` | Hub-side change requests — specs for the hub agent to implement |

## Design Constraints

- No LLM/Claude SDK — pure HTTP to MCP endpoints
- No database drivers — MCP server handles persistence
- No TUI libraries — plain terminal output with simple formatting
- Pipe-friendly: `--json` flag on all read commands, stdin support on `save`

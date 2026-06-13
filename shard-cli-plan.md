# shard-cli — Project Plan

A terminal-first interface to Shard-Link. Zero Claude tokens. Pure Go.

---

## Project Structure

shard-cli/

├── cmd/  
│   ├── root.go          \# root command \+ global flags  
│   ├── search.go        \# shard search  
│   ├── save.go          \# shard save  
│   ├── status.go        \# shard status  
│   └── get.go           \# shard get \[id\]  
├── internal/  
│   ├── client/  
│   │   └── mcp.go       \# MCP HTTP client  
│   ├── config/  
│   │   └── config.go    \# load .env / config file  
│   └── format/  
│       └── output.go    \# terminal output formatting  
├── main.go  
├── go.mod  
└── .env.example

---

## Commands

### shard search

\# Basic search

shard search "rate limiting patterns"

\# Limit results

shard search "eviction logic" \--limit 5

\# Search specific category

shard search "auth decision" \--category arch

\# Output as JSON (for scripting)

shard search "vessel graph" \--json

### shard save

\# Basic save

shard save "decided to use MERGE for idempotent writes" \\

  \--id "decision-merge-idempotency-2026-06" \\

  \--category arch

\# Save from stdin (pipe friendly)

git log \--oneline \-5 | shard save \--id "git-log-$(date \+%Y-%m-%d)" \\

  \--category session

\# Save from file

shard save \--file ./notes.md \--id "notes-2026-06-01" \--category memory

### shard get

\# Get shard by ID

shard get "decision-merge-idempotency-2026-06"

\# Get core shards only

shard get \--core

### shard status

shard status

\# Output

\# MESH STATUS

\# ─────────────────────────────

\# Shards      : 142

\# Bonds       : 387

\# Communities : 8

\# ─────────────────────────────

\# SURVIVAL DISTRIBUTION

\# 0-20   ████░░░░░░  12

\# 21-50  ██░░░░░░░░   8

\# 51-80  ████████░░  34

\# 81-95  ██████████  88

\# ─────────────────────────────

\# Hub     : ✅ online

\# Neo4j   : ✅ online

\# Postgres: ✅ online

---

## Internal Design

### internal/client/mcp.go

Core of everything. Talks directly to your MCP server:

MCPClient

  → SearchAll(query, limit)     → \[\]Shard

  → SaveMemory(shard)           → string (id)

  → GetCoreShards()             → \[\]Shard

  → GetShardByID(id)            → Shard

  → GetStatus()                 → HealthResponse

  → GetRecentShards(limit, cat) → \[\]ShardMetadata

  → GetShardsByCategory(cat)    → \[\]ShardMetadata

  → GetAtRiskShards(limit, thr) → \[\]ShardMetadata

  → UpdateShard(input)          → string (confirmation)

  → DeleteShard(id, confirm)    → string (confirmation)

No LLM. No Claude. Just HTTP to your existing MCP endpoints.

### internal/config/config.go

Loads from three sources in priority order:

1\. CLI flags          \--hub-url, \--api-key

2\. Environment vars   SHARD\_HUB\_URL, SHARD\_API\_KEY

3\. Config file        \~/.shard/config.yaml

### internal/format/output.go

Two render modes:

Default  → human readable, colored terminal output

\--json   → raw JSON, pipe friendly for scripting

---

## Config File

\# \~/.shard/config.yaml

hub\_url: https://hub.izenberk.com

api\_key: shl\_live\_your\_key\_here

default\_limit: 10

default\_category: memory

---

## Build & Install

\# Phase 1 — scaffold

go mod init github.com/izenberk/shard-cli

go get github.com/spf13/cobra@latest

go get github.com/spf13/viper@latest    \# config management

\# Phase 2 — build order

1\. internal/config/config.go            \# load config first

2\. internal/client/mcp.go              \# MCP client second

3\. internal/format/output.go           \# formatter third

4\. cmd/root.go                         \# wire cobra root

5\. cmd/status.go                       \# simplest command first

6\. cmd/search.go                       \# core use case

7\. cmd/save.go                         \# core use case

8\. cmd/get.go                          \# utility

\# Phase 3 — install

go build \-o shard .

sudo mv shard /usr/local/bin/shard

---

## Build Order Rationale

config → client → formatter → commands

- config has zero dependencies — start here  
- client depends only on config — build next  
- formatter depends only on standard lib — build next  
- commands depend on all three — build last

Each layer is testable in isolation before the next layer touches it.

---

## Phase Milestones

Phase 1 — Working status command

  → can hit your live hub

  → prints mesh health

  → confirms config loading works

Phase 2 — Working search command

  → semantic search from terminal

  → formatted output

  → JSON flag works

Phase 3 — Working save command

  → save from args

  → save from stdin pipe

  → save from file

Phase 4 — Working get command

  → fetch by ID

  → fetch core shards

Phase 5 — Observation + CRUD commands

  → shard list (--recent, --category, --at-risk)

  → shard update (--content, --category, --file, --confirm-core)

  → shard delete (--confirm-core)

Phase 6 — Polish

  → shell completion (cobra built-in)

  → \--help is clean and useful

  → error messages are actionable

---

## The End State

shard status                                     \# morning check

shard search "what was I working on"             \# context reload

shard save "finished shard-cli phase 1" \\

  \--category session \\

  \--id "progress-shard-cli-2026-06-01"          \# log progress

shard search "why did I choose cobra"            \# memory query

shard get "decision-merge-idempotency-2026-06"  \# exact fetch

---

## Why Build This

- Query Shard-Link from terminal without opening browser or Claude.ai  
- Zero Claude tokens consumed — pure HTTP to your MCP server  
- Scriptable memory operations for dev workflow automation  
- Foundation for the agent CLI (Option B) — internal/client/mcp.go becomes the memory tool inside the agent loop

---

## Dependencies

| Package | Purpose |
| :---- | :---- |
| github.com/spf13/cobra | CLI subcommand structure |
| github.com/spf13/viper | Config file \+ env var loading |
| Standard lib only otherwise | No unnecessary dependencies |

---

*Plan drafted: 2026-06-01 | Phase 5 implemented: 2026-06-13 | Status: Phase 6 — Polish*  

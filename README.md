# shard-cli

A pure Go terminal interface to [Shard-Link](https://github.com/Izenberk/shard-link), a cognitive memory mesh for AI agents. Communicates directly with the MCP (Model Context Protocol) server over HTTP — no LLM SDK, no Claude API tokens.

## Prerequisites

- **Go 1.25+**
- **Shard-Link hub** running and accessible — this CLI is a client to the hub's MCP endpoint. Without a running Shard-Link instance, the CLI has nothing to connect to.

## Install

```bash
git clone https://github.com/Izenberk/shard-cli.git
cd shard-cli

# Option A: build and install to GOPATH/bin
go install .
ln -s ~/go/bin/shard-cli ~/go/bin/shard

# Option B: build and install system-wide
go build -o shard .
sudo mv shard /usr/local/bin/shard
```

Verify:

```bash
shard --help
```

## Configuration

Three-tier config chain (highest priority first):

### 1. CLI flags

```bash
shard status --hub-url https://your-hub.com/mcp --api-key your-key
```

### 2. Environment variables

```bash
export SHARD_HUB_URL=https://your-hub.com/mcp
export SHARD_API_KEY=your-key
```

### 3. Config file

```yaml
# ~/.shard/config.yaml
hub_url: https://your-hub.com/mcp
api_key: your-key
```

Create the config directory and file:

```bash
mkdir -p ~/.shard
cat > ~/.shard/config.yaml << 'EOF'
hub_url: https://your-hub.com/mcp
api_key: your-key
EOF
```

## Commands

### status

Check mesh health and service status.

```bash
shard status
shard status --json
```

```
MESH STATUS
─────────────────────────────
Shards      : 142
Bonds       : 387
Communities : 8
─────────────────────────────
Hub      : ✅ online
Neo4j    : ✅ online
Postgres : ✅ online
─────────────────────────────
SURVIVAL
  24h   ████░░░░░░░░░░░░░░░░ 14
  7d    ████░░░░░░░░░░░░░░░░ 14
  30d   ████████████████████ 61
  90d   ░░░░░░░░░░░░░░░░░░░░ 0
  older ░░░░░░░░░░░░░░░░░░░░ 0
```

### search

Semantic search across all memory engines (vector, text, graph mesh).

```bash
shard search "authentication flow"
shard search "MCP protocol" --limit 10
shard search "config" --category core
shard search "config" --json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--limit`, `-l` | 5 | Max results per engine |
| `--bias` | 0.7 | Cognitive bias (0.0=centroid, 1.0=query) |
| `--category`, `-c` | *(all)* | Filter by category (memory, session, core, contract) |

### save

Persist a memory shard. Accepts content from three sources: positional args, file, or stdin pipe.

```bash
# From argument
shard save "deployment notes for v2" --id "deploy-notes-v2" --category session

# From file
shard save --file ./notes.md --id "project-notes" --category memory

# From stdin pipe
git log --oneline -10 | shard save --id "recent-commits" --category session
```

| Flag | Default | Description |
|------|---------|-------------|
| `--id` | *(required)* | Unique shard identifier |
| `--category` | memory | Shard category |
| `--file`, `-f` | | Read content from file |

### get

Fetch a shard by exact ID, or retrieve all core identity shards.

```bash
# By ID
shard get "deploy-notes-v2"
shard get "deploy-notes-v2" --json

# All core shards
shard get --core
```

| Flag | Description |
|------|-------------|
| `--core` | Fetch all core identity shards |

### list

List shards using observation filters. Returns metadata only (no content — use `get` for full content).

```bash
# Recent shards (default)
shard list
shard list --limit 20

# By category
shard list --category session
shard list --category contract --limit 50

# At-risk shards (eviction candidates)
shard list --at-risk
shard list --at-risk --threshold 50

# Pipe-friendly
shard list --at-risk --json | jq '.[].id'
```

| Flag | Default | Description |
|------|---------|-------------|
| `--limit`, `-l` | 10 | Max results |
| `--category`, `-c` | | Filter by category |
| `--at-risk` | false | Show shards below survival threshold |
| `--threshold` | 30 | Survival score threshold (with `--at-risk`) |

### update

Update an existing shard's content and/or category. Content changes trigger automatic re-embedding on the server.

```bash
# Update content
shard update my-shard --content "updated notes"

# Update from file
shard update my-shard --file updated.md

# Change category
shard update my-shard --category session

# Update core shard (requires confirmation)
shard update core-identity --category memory --confirm-core
```

| Flag | Description |
|------|-------------|
| `--content` | New content (triggers re-embedding) |
| `--category` | New category |
| `--file`, `-f` | Read new content from file |
| `--confirm-core` | Required to update core shards |

### delete

Permanently remove a shard and all its relationships.

```bash
shard delete my-old-shard
shard delete core-identity --confirm-core
```

| Flag | Description |
|------|-------------|
| `--confirm-core` | Required to delete core shards |

## Global flags

These work with all commands:

| Flag | Description |
|------|-------------|
| `--hub-url` | MCP server URL |
| `--api-key` | API key for authentication |
| `--json` | Output as raw JSON (pipe-friendly) |

## Shell Completion

Generate completion scripts for your shell:

```bash
# Bash — add to ~/.bashrc or source directly
shard completion bash > ~/.local/share/bash-completion/completions/shard

# Zsh — add to your fpath (before compinit)
shard completion zsh > "${fpath[1]}/_shard"

# Fish
shard completion fish > ~/.config/fish/completions/shard.fish

# PowerShell
shard completion powershell | Out-String | Invoke-Expression
```

## Architecture

```
config → client → format → commands
```

| Layer | Package | Role |
|-------|---------|------|
| Config | `internal/config/` | Three-tier settings (flags > env > file) |
| Client | `internal/client/` | MCP HTTP client (JSON-RPC 2.0, SSE) |
| Format | `internal/format/` | Terminal output (human-readable + JSON) |
| Commands | `cmd/` | Cobra subcommands |

Entry point: `main.go` → `cmd.Execute()`

## What is Shard-Link?

Shard-Link is a cognitive memory mesh that gives AI agents persistent, searchable long-term memory. It uses a triple-engine architecture:

- **Vector search** — semantic similarity via embeddings
- **Text search** — keyword matching via SQL
- **Graph mesh** — relational context via Neo4j

The hub exposes these capabilities through an MCP (Model Context Protocol) endpoint. This CLI is one of several clients — AI agents like Claude can also connect directly via MCP.

## License

MIT

# shard-cli Phase 2 Build Guide

Phase 1 delivered the foundation (config, client, format, root command) and a working `search` command. Phase 2 adds the remaining three commands: `status`, `save`, and `get`.

**Build order:** Same principle as Phase 1 — bottom-up. For each command, we add the client method first, then the formatter method, then the command file. This keeps every layer compilable and testable before the next one touches it.

---

## Step 1: Add `GetStatus()` to `internal/client/mcp.go`

**Why this is different from SearchAll:** The `search_all` tool returns formatted text that needs regex parsing. The `get_status` tool (which you just shipped on the hub) returns structured JSON inside the content block. That means we can `json.Unmarshal` directly into a Go struct — no regex, no string splitting. This is the cleaner pattern.

**Why separate MeshStats and ServiceHealth structs:** They represent different concerns. Mesh stats are counts from Neo4j queries. Service health is from ping checks. Keeping them separate makes the formatter's job easier — it can render them as distinct sections with a divider between them, matching the planned terminal output.

Add these types to the domain types section (after `SearchResult`), and the method after `SearchAll`:

```go
// StatusResponse holds parsed output from get_status.
type StatusResponse struct {
	Mesh     MeshStats     `json:"mesh"`
	Services ServiceHealth `json:"services"`
}

type MeshStats struct {
	Shards      int `json:"shards"`
	Bonds       int `json:"bonds"`
	Communities int `json:"communities"`
}

type ServiceHealth struct {
	Hub      string `json:"hub"`
	Neo4j    string `json:"neo4j"`
	Postgres string `json:"postgres"`
}
```

```go
// GetStatus calls the get_status tool and returns mesh health data.
func (c *MCPClient) GetStatus() (*StatusResponse, error) {
	resp, err := c.sendRequest("tools/call", toolCallParams{
		Name: "get_status",
	})
	if err != nil {
		return nil, fmt.Errorf("get_status request failed: %w", err)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	var result toolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			return nil, fmt.Errorf("get_status error: %s", result.Content[0].Text)
		}
		return nil, fmt.Errorf("get_status returned an error")
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("get_status returned empty response")
	}

	// Unlike search_all, the text content IS valid JSON — unmarshal directly
	var status StatusResponse
	if err := json.Unmarshal([]byte(result.Content[0].Text), &status); err != nil {
		return nil, fmt.Errorf("failed to parse status JSON: %w", err)
	}

	return &status, nil
}
```

**Key difference from SearchAll:** Look at the last section. `SearchAll` calls `parseSearchResult()` to regex-parse text. `GetStatus` calls `json.Unmarshal` directly because the hub returns clean JSON. This is why we specified JSON in the health tool spec — it avoids the fragile text parsing pattern entirely.

---

## Step 2: Add `RenderStatus()` to `internal/format/output.go`

**Why a status emoji helper:** The plan shows ✅ for online and a marker for offline. A simple helper function keeps the rendering clean and gives you one place to change the indicator style later.

**Why `renderStatusJSON` takes `interface{}` instead of the struct:** The `renderJSON` helper for search already exists but is typed to `*client.SearchResult`. Rather than making a generic one right now, we just use `json.MarshalIndent` directly in each render method. Same pattern, no unnecessary abstraction.

Add these methods to `output.go`:

```go
// RenderStatus formats status response for display.
func (f *Formatter) RenderStatus(status *client.StatusResponse) string {
	if f.AsJSON {
		data, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return fmt.Sprintf(`{"error": "%s"}`, err)
		}
		return string(data) + "\n"
	}

	var b strings.Builder

	b.WriteString("MESH STATUS\n")
	b.WriteString("─────────────────────────────\n")
	b.WriteString(fmt.Sprintf("Shards      : %d\n", status.Mesh.Shards))
	b.WriteString(fmt.Sprintf("Bonds       : %d\n", status.Mesh.Bonds))
	b.WriteString(fmt.Sprintf("Communities : %d\n", status.Mesh.Communities))
	b.WriteString("─────────────────────────────\n")
	b.WriteString(fmt.Sprintf("Hub      : %s\n", statusIcon(status.Services.Hub)))
	b.WriteString(fmt.Sprintf("Neo4j    : %s\n", statusIcon(status.Services.Neo4j)))
	b.WriteString(fmt.Sprintf("Postgres : %s\n", statusIcon(status.Services.Postgres)))

	return b.String()
}

func statusIcon(s string) string {
	if s == "online" {
		return "✅ online"
	}
	return "❌ " + s
}
```

---

## Step 3: `cmd/status.go` — The Simplest Command

**Why `cobra.NoArgs`:** The status command doesn't take any positional arguments — it just hits the health endpoint and prints the result. `NoArgs` makes Cobra reject `shard status foo` with a clean error instead of silently ignoring the extra argument.

**Why this is the simplest command:** No flags to parse (beyond the globals), no input to read, no positional args. Just: create client → call GetStatus → render → print. It's the minimal proof that the full stack works end-to-end, which is why the original plan had it as the first command milestone.

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/izenberk/shard-cli/internal/client"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show mesh health",
	Long:  "Display Shard-Link mesh statistics and service health.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		mcpClient, err := client.NewMCPClient(cfg.HubURL, cfg.APIKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		status, err := mcpClient.GetStatus()
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		fmt.Print(fmtr.RenderStatus(status))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
```

### Verify status works

```bash
go build -o shard .
./shard status
./shard status --json
```

Expected output:
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
```

---

## Step 4: Add `SaveMemory()` to `internal/client/mcp.go`

**Why no struct for the response:** The `save_memory` tool returns a confirmation message as text — something like "Memory saved: shard-id-here". There's no structured data to parse. We just return the confirmation string so the command can print it.

**Why the method takes a struct instead of individual params:** `SaveMemory` needs three required fields (id, content, category). Passing them as a struct keeps the signature clean and makes it obvious what each field is at the call site. If you added optional fields later (like `vector`), you'd just add them to the struct without changing the method signature.

Add the input type to the domain types section, and the method after `GetStatus`:

```go
// SaveInput holds the parameters for saving a memory shard.
type SaveInput struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Category string `json:"category"`
}
```

```go
// SaveMemory calls the save_memory tool and returns the confirmation message.
func (c *MCPClient) SaveMemory(input SaveInput) (string, error) {
	args := map[string]interface{}{
		"id":       input.ID,
		"content":  input.Content,
		"category": input.Category,
	}

	resp, err := c.sendRequest("tools/call", toolCallParams{
		Name:      "save_memory",
		Arguments: args,
	})
	if err != nil {
		return "", fmt.Errorf("save_memory request failed: %w", err)
	}
	if resp.Error != nil {
		return "", resp.Error
	}

	var result toolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("failed to parse tool result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			return "", fmt.Errorf("save_memory error: %s", result.Content[0].Text)
		}
		return "", fmt.Errorf("save_memory returned an error")
	}

	if len(result.Content) == 0 {
		return "saved (no confirmation from server)", nil
	}

	return result.Content[0].Text, nil
}
```

---

## Step 5: Add `RenderSave()` to `internal/format/output.go`

**Why this is so short:** Save is a write operation — the user just needs confirmation that it worked. No tables, no sections, no truncation. Just the server's confirmation message. The JSON mode wraps it in a JSON object so scripts can parse the result.

```go
// RenderSave formats save confirmation for display.
func (f *Formatter) RenderSave(message string) string {
	if f.AsJSON {
		data, _ := json.Marshal(map[string]string{"status": message})
		return string(data) + "\n"
	}
	return message + "\n"
}
```

---

## Step 6: `cmd/save.go` — Three Input Modes

**Why three input sources:** The plan calls for saving from (1) positional args, (2) stdin pipe, and (3) a file. This makes `shard save` useful in scripts (`git log | shard save --id ...`), in automation (`--file ./notes.md`), and interactively (`shard save "my decision" --id ...`).

**How stdin detection works:** `os.Stdin.Stat()` returns file info about stdin. If it's a pipe (not a terminal), the `ModeCharDevice` bit is NOT set. That's how you distinguish `echo "text" | shard save` from `shard save` with no input. This is the standard Go pattern for detecting piped input.

**Why `--id` and `--category` are required:** The MCP `save_memory` tool requires both. Rather than generating random IDs (which would make memories hard to find), we require the user to provide a meaningful ID. Cobra's `MarkFlagRequired` enforces this before `RunE` fires.

**Precedence: args > file > stdin.** If you provide a positional argument, that's the content — file and stdin are ignored. If no argument but `--file` is set, read the file. If neither, check if stdin has piped data. If none of the three, Cobra errors because there's nothing to save.

```go
package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/izenberk/shard-cli/internal/client"
)

var saveCmd = &cobra.Command{
	Use:   "save [content]",
	Short: "Save to Shard-Link memory",
	Long:  "Persist a memory shard from args, stdin pipe, or file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		category, _ := cmd.Flags().GetString("category")
		filePath, _ := cmd.Flags().GetString("file")

		// Resolve content from three sources: args > file > stdin
		var content string

		switch {
		case len(args) > 0:
			content = args[0]

		case filePath != "":
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			content = string(data)

		default:
			// Check if stdin has piped data
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read stdin: %w", err)
				}
				content = string(data)
			}
		}

		if content == "" {
			return fmt.Errorf("no content provided — use args, --file, or pipe to stdin")
		}

		mcpClient, err := client.NewMCPClient(cfg.HubURL, cfg.APIKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		message, err := mcpClient.SaveMemory(client.SaveInput{
			ID:       id,
			Content:  content,
			Category: category,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		fmt.Print(fmtr.RenderSave(message))
		return nil
	},
}

func init() {
	saveCmd.Flags().String("id", "", "Unique shard identifier (required)")
	saveCmd.Flags().String("category", "memory", "Shard category (memory, session, core)")
	saveCmd.Flags().StringP("file", "f", "", "Read content from file")

	saveCmd.MarkFlagRequired("id")

	rootCmd.AddCommand(saveCmd)
}
```

### Verify save works

```bash
go build -o shard .

# Save from args
./shard save "test shard from cli" --id "cli-test-$(date +%Y-%m-%d)" --category session

# Save from file
echo "some notes" > /tmp/test-note.md
./shard save --file /tmp/test-note.md --id "cli-file-test" --category memory

# Save from stdin pipe
echo "piped content" | ./shard save --id "cli-pipe-test" --category session

# Verify it landed
./shard search "cli-test" --limit 1
```

---

## Step 7: `cmd/get.go` — Fetch by ID

**Blocker check:** The `get` command needs two MCP tools on the hub side:
- A tool to fetch a single shard by its exact ID (e.g. `get_shard`)
- A tool to fetch all core-category shards (e.g. `get_core_shards`)

**These tools may not exist on the hub yet.** Check your MCP server's registered tools before building this command. If they don't exist, you'll need to add them on the shard-link side first — same process as `get_status`.

If the tools are available, the pattern is identical to status: call `tools/call` with the tool name, unmarshal the JSON response, render it. Here's the shape to aim for:

### Client method signatures (for mcp.go)

```go
// GetShardByID calls the get_shard tool and returns a single shard.
func (c *MCPClient) GetShardByID(id string) (*Shard, error)

// GetCoreShards calls the get_core_shards tool and returns all core shards.
func (c *MCPClient) GetCoreShards() ([]Shard, error)
```

### Command structure (for cmd/get.go)

```go
// shard get "some-id"       → fetch by ID (positional arg)
// shard get --core           → fetch all core shards
//
// Flags:
//   --core    Fetch core shards instead of by ID
//
// Args: ExactArgs(1) when no --core flag, NoArgs when --core is set
```

### Formatter (for output.go)

```go
// RenderShard — single shard, full content (no truncation)
// RenderShards — list of shards (same as search but without scores/bonds)
```

**Don't build this until the hub tools exist.** Write the hub spec first (like you did for `get_status`), implement it on the shard-link side, then come back and wire it up here.

---

## Build Order Summary

| Step | File | What |
|------|------|------|
| 1 | `internal/client/mcp.go` | Add `StatusResponse` types + `GetStatus()` method |
| 2 | `internal/format/output.go` | Add `RenderStatus()` + `statusIcon()` |
| 3 | `cmd/status.go` | Wire up the status command |
| 4 | `internal/client/mcp.go` | Add `SaveInput` type + `SaveMemory()` method |
| 5 | `internal/format/output.go` | Add `RenderSave()` |
| 6 | `cmd/save.go` | Wire up save with args/file/stdin |
| 7 | `cmd/get.go` | **Blocked** — needs hub-side tools first |

Same pattern as Phase 1: client → format → command, for each feature. Compile and test after each step.

---

## Verification — Full Phase 2

After Steps 1-6 are done:

```bash
go build -o shard .

# Status
./shard status
./shard status --json

# Save from args
./shard save "phase 2 complete" --id "cli-phase2-test" --category session

# Save from pipe
echo "piped test" | ./shard save --id "cli-pipe-test" --category session

# Save from file
./shard save --file ./CLAUDE.md --id "cli-file-test" --category memory

# Confirm saves landed
./shard search "cli-phase2" --limit 1

# Error cases
./shard save --id "missing-content"          # should error: no content
./shard save "no id provided"                # should error: --id required
./shard status --hub-url https://bad.url/mcp # should error: connection failed
```

package format

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/izenberk/shard-cli/internal/client"
)

// Formatter handles rendering output to the terminal.
type Formatter struct {
	AsJSON bool
}

// RenderSearch formats search results for display.
func (f *Formatter) RenderSearch(result *client.SearchResult) string {
	if f.AsJSON {
		return f.renderJSON(result)
	}
	return f.renderHuman(result)
}

// RenderSave formats save confirmation for display.
func (f *Formatter) RenderSave(message string) string {
	if f.AsJSON {
		data, _ := json.Marshal(map[string]string{"status": message})
		return string(data) + "\n"
	}
	return message + "\n"
}

// RenderError formats an error for stderr with actionable hints.
func (f *Formatter) RenderError(err error) string {
	msg := err.Error()
	hint := errorHint(msg)
	if hint != "" {
		return fmt.Sprintf("error: %s\n  → %s", msg, hint)
	}
	return fmt.Sprintf("error: %s", msg)
}

// errorHint returns an actionable suggestion based on common error patterns.
func errorHint(msg string) string {
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "dial tcp") ||
		strings.Contains(lower, "no such host"):
		return "Is the Shard-Link hub running? Check hub_url in ~/.shard/config.yaml"

	case strings.Contains(lower, "http 401") ||
		strings.Contains(lower, "http 403") ||
		strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "forbidden"):
		return "Check your API key (--api-key, SHARD_API_KEY, or ~/.shard/config.yaml)"

	case strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "deadline exceeded") ||
		strings.Contains(lower, "context deadline"):
		return "Hub is not responding. Check network or increase timeout"

	case strings.Contains(lower, "http 404"):
		return "Endpoint not found. Verify hub_url points to the MCP server path (e.g. /mcp)"

	case strings.Contains(lower, "http 5"):
		return "Hub returned a server error. Check hub logs for details"
	}

	return ""
}

func (f *Formatter) renderJSON(result *client.SearchResult) string {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err)
	}
	return string(data) + "\n"
}

func (f *Formatter) renderHuman(result *client.SearchResult) string {
	var b strings.Builder

	shardCount := len(result.Shards)
	bondCount := len(result.Bonds)

	b.WriteString(fmt.Sprintf("SEARCH RESULTS (%d shards, %d bonds)\n", shardCount, bondCount))
	b.WriteString("─────────────────────────────────────\n\n")

	if shardCount == 0 {
		b.WriteString("No results found.\n")
		return b.String()
	}

	for _, shard := range result.Shards {
		b.WriteString(fmt.Sprintf("[%s] Score: %.2f\n", shard.ID, shard.Score))
		b.WriteString(truncate(shard.Content, 200))
		b.WriteString("\n\n")
	}

	if bondCount > 0 {
		b.WriteString("─────────────────────────────────────\n")
		b.WriteString("BONDS\n")
		for _, bond := range result.Bonds {
			b.WriteString(fmt.Sprintf("  %s <-> %s  (%.2f)\n", bond.From, bond.To, bond.Strength))
		}
	}
	return b.String()
}

// truncate shortens text to maxLen characters, appending "..." if cut.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

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

	// Survival distribution histogram
	s := status.Survival
	if s.Day24h+s.Day7d+s.Day30d+s.Day90d+s.Older > 0 {
		b.WriteString("─────────────────────────────\n")
		b.WriteString("SURVIVAL\n")

		buckets := []struct {
			label string
			count int
		}{
			{"24h  ", s.Day24h},
			{"7d   ", s.Day7d},
			{"30d  ", s.Day30d},
			{"90d  ", s.Day90d},
			{"older", s.Older},
		}

		// Find max for scaling bars
		max := 0
		for _, b := range buckets {
			if b.count > max {
				max = b.count
			}
		}

		barWidth := 20
		for _, bucket := range buckets {
			width := 0
			if max > 0 {
				width = (bucket.count * barWidth) / max
			}
			bar := strings.Repeat("█", width) + strings.Repeat("░", barWidth-width)
			b.WriteString(fmt.Sprintf("  %s %s %d\n", bucket.label, bar, bucket.count))
		}
	}

	return b.String()
}

// RenderShard formats a single shard for display (full content, no truncation).
func (f *Formatter) RenderShard(shard *client.ShardDetail) string {
	if f.AsJSON {
		data, err := json.MarshalIndent(shard, "", "  ")
		if err != nil {
			return fmt.Sprintf(`{"error": "%s"}`, err)
		}
		return string(data) + "\n"
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("[%s] (%s)\n", shard.ID, shard.Category))
	b.WriteString("─────────────────────────────────────\n")
	b.WriteString(shard.Content)
	b.WriteString("\n─────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("Created : %s\n", shard.CreatedAt))
	b.WriteString(fmt.Sprintf("Updated : %s\n", shard.UpdatedAt))

	return b.String()
}

// RenderShards formats a list of shards for display.
func (f *Formatter) RenderShards(shards []client.ShardDetail) string {
	if f.AsJSON {
		data, err := json.MarshalIndent(shards, "", "  ")
		if err != nil {
			return fmt.Sprintf(`{"error": "%s"}`, err)
		}
		return string(data) + "\n"
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("CORE SHARDS (%d)\n", len(shards)))
	b.WriteString("─────────────────────────────────────\n\n")

	if len(shards) == 0 {
		b.WriteString("No core shards found.\n")
		return b.String()
	}

	for _, shard := range shards {
		b.WriteString(fmt.Sprintf("[%s]\n", shard.ID))
		b.WriteString(truncate(shard.Content, 200))
		b.WriteString("\n\n")
	}

	return b.String()
}

func statusIcon(s string) string {
	if s == "online" {
		return "✅ online"
	}
	return "❌ " + s
}
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

// RenderError formats an error for stderr.
func (f *Formatter) RenderError(err error) string {
	return fmt.Sprintf("error: %s", err)
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
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/izenberk/shard-cli/internal/client"
)

var searchCmd = &cobra.Command{
	Use:			"search [query]",
	Short:		"Search Shard-Link memory",
	Long: 		"Semantic search across all memory engines (vector, text, graph mesh).",
	Example: `  shard search "golang error handling"
  shard search "docker networking" --limit 10
  shard search "MCP protocol" --bias 0.9
  shard search "kubernetes pods" --json | jq '.shards[].id'`,
	Args:			cobra.ExactArgs(1),
	RunE: 		func(cmd *cobra.Command, args []string) error {
		query 	:= args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		bias, _ := cmd.Flags().GetFloat64("bias")
		category, _ := cmd.Flags().GetString("category")

		// Create MCP client — performs the handshake
		mcpClient, err := client.NewMCPClient(cfg.HubURL, cfg.APIKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		// Call search_all
		result, err := mcpClient.SearchAll(query, limit, bias, category)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		// Render and print
		fmt.Print(fmtr.RenderSearch(result))
		return nil
	},
}

func init() {
	searchCmd.Flags().IntP("limit", "l", 5, "Max results per engine")
	searchCmd.Flags().Float64("bias", 0.7, "Cognitive bias (0.0=centroid, 1.0=query)")
	searchCmd.Flags().StringP("category", "c", "", "Filter by category (memory, session, core, contract)")

	rootCmd.AddCommand(searchCmd)
}
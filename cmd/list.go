package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/izenberk/shard-cli/internal/client"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List shards by filter",
	Long: `List shards using observation filters. Returns metadata only (no content).

Filters (mutually exclusive):
  --recent       Most recently updated shards (default)
  --category     All shards in a specific category
  --at-risk      Shards below a survival score threshold`,
	Example: `  shard list                              # recent shards (default)
  shard list --limit 20                   # last 20 updated
  shard list --category session           # all session shards
  shard list --at-risk                    # eviction candidates
  shard list --at-risk --threshold 50     # shards scoring below 50
  shard list --json                       # pipe-friendly output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		category, _ := cmd.Flags().GetString("category")
		atRisk, _ := cmd.Flags().GetBool("at-risk")
		threshold, _ := cmd.Flags().GetFloat64("threshold")

		mcpClient, err := client.NewMCPClient(cfg.HubURL, cfg.APIKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		var shards []client.ShardMetadata

		switch {
		case atRisk:
			shards, err = mcpClient.GetAtRiskShards(limit, threshold)
		case category != "":
			shards, err = mcpClient.GetShardsByCategory(category, limit)
		default:
			shards, err = mcpClient.GetRecentShards(limit, category)
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		fmt.Print(fmtr.RenderMetadataList(shards, listTitle(atRisk, category)))
		return nil
	},
}

func listTitle(atRisk bool, category string) string {
	switch {
	case atRisk:
		return "AT-RISK SHARDS"
	case category != "":
		return fmt.Sprintf("SHARDS [%s]", category)
	default:
		return "RECENT SHARDS"
	}
}

func init() {
	listCmd.Flags().IntP("limit", "l", 10, "Max results")
	listCmd.Flags().StringP("category", "c", "", "Filter by category")
	listCmd.Flags().Bool("at-risk", false, "Show shards below survival threshold")
	listCmd.Flags().Float64("threshold", 30, "Survival score threshold (used with --at-risk)")

	rootCmd.AddCommand(listCmd)
}

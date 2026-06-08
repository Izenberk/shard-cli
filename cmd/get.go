package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/izenberk/shard-cli/internal/client"
)

var getCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Fetch a shard by ID",
	Long:  "Retrieve a single shard by exact ID, or all core shards with --core.",
	Example: `  shard get my-shard-id
  shard get my-shard-id --json
  shard get --core`,
	RunE: func(cmd *cobra.Command, args []string) error {
		core, _ := cmd.Flags().GetBool("core")

		mcpClient, err := client.NewMCPClient(cfg.HubURL, cfg.APIKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		if core {
			shards, err := mcpClient.GetCoreShards()
			if err != nil {
				fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
				return err
			}

			fmt.Print(fmtr.RenderShards(shards))
			return nil
		}

		// Require an ID when --core is not set
		if len(args) == 0 {
			return fmt.Errorf("provide a shard ID or use --core")
		}

		shard, err := mcpClient.GetShardByID(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		fmt.Print(fmtr.RenderShard(shard))
		return nil
	},
}

func init() {
	getCmd.Flags().Bool("core", false, "Fetch all core identity shards")

	rootCmd.AddCommand(getCmd)
}

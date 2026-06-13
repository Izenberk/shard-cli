package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/izenberk/shard-cli/internal/client"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a shard permanently",
	Long: `Permanently remove a shard and all its relationships from every backend.

Core shards require --confirm-core. System-managed comm-summary-* shards
cannot be deleted.`,
	Example: `  shard delete my-old-shard
  shard delete core-identity --confirm-core
  shard delete temp-note --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		confirmCore, _ := cmd.Flags().GetBool("confirm-core")

		mcpClient, err := client.NewMCPClient(cfg.HubURL, cfg.APIKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		message, err := mcpClient.DeleteShard(id, confirmCore)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		fmt.Print(fmtr.RenderMutation(message))
		return nil
	},
}

func init() {
	deleteCmd.Flags().Bool("confirm-core", false, "Required to delete core shards")

	rootCmd.AddCommand(deleteCmd)
}

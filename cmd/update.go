package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/izenberk/shard-cli/internal/client"
)

var updateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update an existing shard",
	Long: `Update a shard's content and/or category. If content changes, the vector
is re-embedded automatically on the server side.

Core shards require --confirm-core. System-managed comm-summary-* shards
cannot be updated.`,
	Example: `  shard update my-shard --content "new content"
  shard update my-shard --category session
  shard update my-shard --file updated.md
  shard update core-identity --category memory --confirm-core`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		content, _ := cmd.Flags().GetString("content")
		category, _ := cmd.Flags().GetString("category")
		filePath, _ := cmd.Flags().GetString("file")
		confirmCore, _ := cmd.Flags().GetBool("confirm-core")

		if filePath != "" && content == "" {
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			content = string(data)
		}

		if content == "" && filePath == "" {
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read stdin: %w", err)
				}
				content = string(data)
			}
		}

		if content == "" && category == "" {
			return fmt.Errorf("provide --content, --category, --file, or pipe to stdin")
		}

		mcpClient, err := client.NewMCPClient(cfg.HubURL, cfg.APIKey)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		message, err := mcpClient.UpdateShard(client.UpdateInput{
			ID:          id,
			Content:     content,
			Category:    category,
			ConfirmCore: confirmCore,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, fmtr.RenderError(err))
			return err
		}

		fmt.Print(fmtr.RenderMutation(message))
		return nil
	},
}

func init() {
	updateCmd.Flags().String("content", "", "New content (triggers re-embedding)")
	updateCmd.Flags().String("category", "", "New category")
	updateCmd.Flags().StringP("file", "f", "", "Read new content from file")
	updateCmd.Flags().Bool("confirm-core", false, "Required to update core shards")

	rootCmd.AddCommand(updateCmd)
}

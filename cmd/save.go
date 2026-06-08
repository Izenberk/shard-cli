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

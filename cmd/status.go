package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/izenberk/shard-cli/internal/client"
)

var statusCmd = &cobra.Command{
	Use:		"status",
	Short:	"Show mesh health",
	Long:		"Display Shard-Link mesh statistics and service health.",
	Args: 	cobra.NoArgs,
	RunE: 	func(cmd *cobra.Command, args []string) error {
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
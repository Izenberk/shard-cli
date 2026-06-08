package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/izenberk/shard-cli/internal/config"
	"github.com/izenberk/shard-cli/internal/format"
)

var (
	cfg 	*config.Config
	fmtr 	*format.Formatter
)

var rootCmd = &cobra.Command{
	Use: 		"shard",
	Short: 	"Terminal interface to Shard-Link",
	Long: `Query and manage your Shard-Link knowledge mesh from the command line.

Configuration is resolved in order: CLI flags → environment variables → ~/.shard/config.yaml.

Examples:
  shard status                          # mesh health check
  shard search "golang concurrency"     # semantic search
  shard save "TIL: channels" --id til-1 # save a memory shard
  shard get til-1                       # fetch shard by ID
  shard get --core                      # list core identity shards`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load config — this is where the three-tier chain resolves.
		// Flags are already parsed by Cobra at this point.
		var err error
		cfg, err = config.Load()
		if err != nil {
			return err
		}

		// Initialize formatter with the resolved --json flag
		fmtr = &format.Formatter{AsJSON: cfg.JSON}
		return nil
	},
}

// Execute is the single entry point called from main.go
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Register global persistent flags — available to all subcommands
	rootCmd.PersistentFlags().String("hub-url", "", "MCP server URL")
	rootCmd.PersistentFlags().String("api-key", "", "API key for authentication")
	rootCmd.PersistentFlags().Bool("json", false, "Output as raw JSON")

	// Bind flags to Viper keys — this registers the mapping, not the value.
	// Viper reads the actual flag value lazily during config.Load()
	viper.BindPFlag("hub_url", rootCmd.PersistentFlags().Lookup("hub-url"))
	viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json"))
}
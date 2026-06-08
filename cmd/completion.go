package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for shard.

The output is written to stdout so you can pipe it to your shell config.

Bash:
  # Add to ~/.bashrc or source directly:
  shard completion bash > ~/.local/share/bash-completion/completions/shard

Zsh:
  # Add to your fpath (before compinit):
  shard completion zsh > "${fpath[1]}/_shard"

Fish:
  shard completion fish > ~/.config/fish/completions/shard.fish

PowerShell:
  shard completion powershell | Out-String | Invoke-Expression`,
	Args:      cobra.ExactValidArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletion(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

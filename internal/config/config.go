package config

import (
	"strings"
	"github.com/spf13/viper"
)

// Config holds all runtime settings resolved from flags, env, and file.
type Config struct {
	HubURL	string
	APIKey	string
	Limit		int
	JSON		bool
}

// Load reads config from the three-tier chain.
// Called once from cmd/root.go PersistentPreRunE — after flags are parsed
// but before any subcommand runs.
func Load() (*Config, error) {
	// Tier 4: hardcoded defaults
	viper.SetDefault("hub_url", "https://hub.izenberk.com/mcp")
	viper.SetDefault("limit", 5)
	viper.SetDefault("json", false)

	// Tier 3: config file (~/.shard/config.yaml)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.shard")

	// ReadInConfig is non-fatal if the file doesn't exist yet.
	// But if the file exists and has syntax error, surface that.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// Tier 2: environment variables (SHARD_HUB_URL, SHARD_API_KEY)
	viper.SetEnvPrefix("SHARD")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Tier 1: CLI flags are bound in cmd/root.go init() via viper.BindPFlag.
	// Viper checks them automatically — highest priority.

	return &Config{
		HubURL: viper.GetString("hub_url"),
		APIKey: viper.GetString("api_key"),
		Limit: 	viper.GetInt("limit"),
		JSON:		viper.GetBool("json"),
	}, nil
}
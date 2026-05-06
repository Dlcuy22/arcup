// Module: internal/config
// Purpose: Defines the application configuration struct and provides
// loading from viper (which merges YAML file, env vars, and CLI flags).
//
// Key Components:
//   - Config: Top-level configuration struct
//   - RetentionConfig: Retention policy sub-config
//   - Load(): Reads viper state into a typed Config struct
//   - Validate(): Checks required fields and value constraints
//
// Dependencies:
//   - github.com/spf13/viper: Reads merged config values
//
// Error Types:
//   - ErrMissingSources: No source paths provided
//   - ErrMissingRemote: No rclone remote path specified
//   - ErrInvalidAlgo: Unsupported compression algorithm
//
// Example:
//
//	cfg, err := config.Load()
//	if err != nil { log.Fatal().Err(err).Msg("config") }
package config

import (
	"errors"
	"os"

	"github.com/spf13/viper"
)

var (
	ErrMissingSources = errors.New("at least one --source/-s is required")
	ErrMissingRemote  = errors.New("--remote/-r is required")
	ErrInvalidAlgo    = errors.New("--algo/-a must be one of: zstd, gz, zip")
)

type Config struct {
	Name      string          `mapstructure:"name"`
	Sources   []string        `mapstructure:"sources"`
	Algo      string          `mapstructure:"algo"`
	AlgoArgs  string          `mapstructure:"algo-args"`
	UploadTo  string          `mapstructure:"upload-to"`
	Remote    string          `mapstructure:"remote"`
	OutputDir string          `mapstructure:"output-dir"`
	Keep      int             `mapstructure:"keep"`
	DryRun    bool            `mapstructure:"dry-run"`
	Watch     bool            `mapstructure:"watch"`
	Interval  string          `mapstructure:"interval"`
	Retention RetentionConfig `mapstructure:"retention"`
	Retry     RetryConfig     `mapstructure:"retry"`
	Webhook   WebhookConfig   `mapstructure:"webhook"`
}

type RetryConfig struct {
	MaxAttempts int    `mapstructure:"max-attempts"`
	Delay       string `mapstructure:"delay"`
}

type WebhookConfig struct {
	Discord   string `mapstructure:"discord"`
	CustomURL string `mapstructure:"custom-url"`
}

type RetentionConfig struct {
	KeepDays int `mapstructure:"keep-days"`
	KeepLast int `mapstructure:"keep-last"`
}

/*
Load reads the merged viper configuration into a typed Config struct.
Applies defaults for fields not set by flags, env, or config file.

	returns:
	      *Config: populated configuration
	      error: viper unmarshal failure
*/
func Load() (*Config, error) {
	cfg := &Config{
		Algo:     "zstd",
		UploadTo: "rclone",
		OutputDir: "/tmp/arcup",
		Retry: RetryConfig{
			MaxAttempts: 3,
			Delay:       "1m",
		},
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	if cfg.Name == "" {
		hostname, _ := os.Hostname()
		cfg.Name = hostname
	}

	return cfg, nil
}

/*
Validate checks that the loaded config has all required fields
and that values are within acceptable ranges.

	returns:
	      error: first validation failure encountered
*/
func (c *Config) Validate() error {
	if len(c.Sources) == 0 {
		return ErrMissingSources
	}
	if c.Remote == "" {
		return ErrMissingRemote
	}
	switch c.Algo {
	case "zstd", "gz", "zip":
	default:
		return ErrInvalidAlgo
	}
	return nil
}

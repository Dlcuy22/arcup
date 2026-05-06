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
	ErrMissingSources = errors.New("at least one source path is required")
	ErrMissingRemote  = errors.New("remote path is required")
	ErrInvalidAlgo    = errors.New("algo must be one of: zstd, gz, zip")
)

type Config struct {
	Name      string          `mapstructure:"name"`
	Sources   []string        `mapstructure:"sources"`
	Algo      string          `mapstructure:"algo"`
	AlgoArgs  string          `mapstructure:"algo-args"`
	UploadTo  string          `mapstructure:"upload-to"`
	Remote    string          `mapstructure:"remote"`
	Interval  string          `mapstructure:"interval"`
	OutputDir string          `mapstructure:"output-dir"`
	Keep      int             `mapstructure:"keep"`
	DryRun    bool            `mapstructure:"dry-run"`
	Retention RetentionConfig `mapstructure:"retention"`
}

type RetentionConfig struct {
	KeepDays    int    `mapstructure:"keep-days"`
	KeepLast    int    `mapstructure:"keep-last"`
	Interval    string `mapstructure:"interval"`
	AfterBackup bool   `mapstructure:"after-backup"`
	DryRun      bool   `mapstructure:"dry-run"`
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
		Algo:      "zstd",
		UploadTo:  "rclone",
		OutputDir: "/tmp/arcup",
		Retention: RetentionConfig{
			KeepDays: 7,
			KeepLast: 5,
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

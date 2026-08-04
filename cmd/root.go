// Module: cmd/root
// Purpose: Defines the root cobra command, global persistent flags,
// and viper config file binding. All subcommands attach here.
//
// Key Components:
//   - rootCmd: Top-level cobra command with global flags
//   - Execute(): Entry point called from main.go
//   - initConfig(): Loads config file via viper and binds flags
//   - installDefaultConfig(): Copies embedded default config to ~/.arcup.yaml
//
// Dependencies:
//   - github.com/dlcuy22/arcup/assets: Embedded default config
//   - github.com/spf13/cobra: CLI framework
//   - github.com/spf13/viper: Config file + flag binding
//   - github.com/rs/zerolog: Structured logging
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dlcuy22/arcup/assets"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile      string
	installFlag  bool
	arcupVersion = "0.5.4"
)

var rootCmd = &cobra.Command{
	Use:     "arcup",
	Short:   "Archive and upload backups",
	Version: arcupVersion,
	Long: `arcup is a portable CLI backup tool that archives files
using tar with configurable compression (zstd, gzip, zip)
and uploads them via rclone to any supported backend.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		log.Info().Str("version", arcupVersion).Msg("running arcup")
		if installFlag {
			if err := installDefaultConfig(); err != nil {
				return err
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if installFlag {
			return nil
		}
		return cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: ~/.arcup.yaml)")
	rootCmd.PersistentFlags().BoolP("dry-run", "n", false, "simulate without writing or uploading")
	rootCmd.PersistentFlags().StringP("remote", "r", "", "rclone remote path (e.g. s3:bucket/backups)")
	rootCmd.Flags().BoolVar(&installFlag, "install", false, "install default configuration to ~/.arcup.yaml")

	viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run"))
	viper.BindPFlag("remote", rootCmd.PersistentFlags().Lookup("remote"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to find home directory")
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".arcup")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Warn().Err(err).Msg("failed to read config file")
		}
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

/*
installDefaultConfig copies the embedded default config to ~/.arcup.yaml.

    returns:
          error: error if writing default config fails
*/
func installDefaultConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find home directory: %w", err)
	}

	targetPath := filepath.Join(home, ".arcup.yaml")
	if _, err := os.Stat(targetPath); err == nil {
		log.Info().Str("path", targetPath).Msg("configuration file already exists, skipping installation")
		return nil
	}

	if err := os.WriteFile(targetPath, assets.DefaultConfig, 0644); err != nil {
		return fmt.Errorf("write default config to %s: %w", targetPath, err)
	}

	log.Info().Str("path", targetPath).Msg("successfully installed default configuration")
	return nil
}


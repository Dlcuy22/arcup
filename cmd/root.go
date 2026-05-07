// Module: cmd/root
// Purpose: Defines the root cobra command, global persistent flags,
// and viper config file binding. All subcommands attach here.
//
// Key Components:
//   - rootCmd: Top-level cobra command with global flags
//   - Execute(): Entry point called from main.go
//   - initConfig(): Loads config file via viper and binds flags
//
// Dependencies:
//   - github.com/spf13/cobra: CLI framework
//   - github.com/spf13/viper: Config file + flag binding
//   - github.com/rs/zerolog: Structured logging
package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile      string
	arcupVersion = "0.5.1"
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
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.Info().Str("version", arcupVersion).Msg("running arcup")
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

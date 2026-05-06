// Module: cmd/validate
// Purpose: Implements the "validate" subcommand that checks
// configuration validity, source paths, and rclone remote
// accessibility before running a real backup.
//
// Key Components:
//   - validateCmd: Cobra command for "arcup validate"
//   - executeValidate(): Runs all validation checks
//
// Dependencies:
//   - internal/config: Configuration loading
//   - internal/upload: Remote connectivity check
package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/dlcuy22/arcup/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration and remote connectivity",
	Long:  `Checks that sources exist, compression tools are available, and the rclone remote is reachable.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeValidate()
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func executeValidate() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	log.Info().Msg("Validating configuration...")
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	log.Info().Msg("Validating sources...")
	for _, src := range cfg.Sources {
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("source missing: %s", src)
		}
	}

	log.Info().Msg("Checking dependencies...")
	deps := []string{"tar", cfg.Algo, cfg.UploadTo}
	for _, dep := range deps {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("dependency missing in $PATH: %s", dep)
		}
	}

	log.Info().Msgf("Checking remote %q...", cfg.Remote)
	cmd := exec.Command(cfg.UploadTo, "lsd", cfg.Remote)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("remote check failed: %s (%v)", string(output), err)
	}

	log.Info().Msg("All checks passed! arcup is ready.")
	return nil
}

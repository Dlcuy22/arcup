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

import "github.com/spf13/cobra"

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
	// TODO: implement validation
	// 1. Load config
	// 2. Check source paths exist
	// 3. Check compression binary on $PATH (zstd, gzip, zip)
	// 4. Check tar on $PATH
	// 5. Check rclone on $PATH
	// 6. rclone lsd remote → verify accessibility
	// 7. Print summary of checks
	return nil
}

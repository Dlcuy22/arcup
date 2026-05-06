// Module: cmd/run
// Purpose: Implements the one-shot "run" subcommand that performs
// a single backup cycle: archive sources, generate metadata sidecar,
// upload both to the remote, and clean up local files.
//
// Key Components:
//   - runCmd: Cobra command for "arcup run"
//   - executeRun(): Core backup logic orchestrator
//
// Dependencies:
//   - internal/config: Resolved configuration struct
//   - internal/archive: Archiver factory and compression
//   - internal/upload: Rclone upload executor
//   - internal/meta: Sidecar metadata generation
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a one-shot backup",
	Long:  `Archives the configured sources, uploads to the remote, and exits.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeRun()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringSliceP("source", "s", nil, "source file or directory (repeatable)")
	runCmd.Flags().StringP("algo", "a", "zstd", "compression algorithm: zstd, gz, zip")
	runCmd.Flags().StringP("algo-args", "A", "", "extra arguments for the compression tool")
	runCmd.Flags().StringP("upload-to", "u", "rclone", "upload backend")
	runCmd.Flags().StringP("remote", "r", "", "rclone remote path (e.g. s3:bucket/backups)")
	runCmd.Flags().StringP("output-dir", "o", "/tmp/arcup", "local staging directory for archives")
	runCmd.Flags().IntP("keep", "k", 0, "keep N local copies (0 = delete after upload)")
	runCmd.Flags().StringP("name", "N", "", "archive name prefix (default: hostname)")

	viper.BindPFlag("sources", runCmd.Flags().Lookup("source"))
	viper.BindPFlag("algo", runCmd.Flags().Lookup("algo"))
	viper.BindPFlag("algo-args", runCmd.Flags().Lookup("algo-args"))
	viper.BindPFlag("upload-to", runCmd.Flags().Lookup("upload-to"))
	viper.BindPFlag("remote", runCmd.Flags().Lookup("remote"))
	viper.BindPFlag("output-dir", runCmd.Flags().Lookup("output-dir"))
	viper.BindPFlag("keep", runCmd.Flags().Lookup("keep"))
	viper.BindPFlag("name", runCmd.Flags().Lookup("name"))
}

func executeRun() error {
	// TODO: implement backup cycle
	// 1. Load and validate config
	// 2. Validate sources exist
	// 3. Build archive name ({name}_{hostname}_{timestamp}.tar.{ext})
	// 4. archive.New(algo) → Archiver
	// 5. archiver.Archive(sources, outputPath, algoArgs)
	// 6. Compute SHA-256 checksum of the archive
	// 7. Write sidecar .meta.json
	// 8. uploader.Upload(archive, remote)
	// 9. uploader.Upload(sidecar, remote)
	// 10. Cleanup local files if keep=0
	return nil
}

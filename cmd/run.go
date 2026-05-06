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
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dlcuy22/arcup/internal/archive"
	"github.com/dlcuy22/arcup/internal/config"
	"github.com/dlcuy22/arcup/internal/meta"
	"github.com/dlcuy22/arcup/internal/upload"
)

const arcupVersion = "0.1.0"

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
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)

	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("missing or invalid arguments: %v", err)
	}

	// Validate source paths exist
	for _, src := range cfg.Sources {
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("source path does not exist: %s", src)
		}
	}
	log.Info().Strs("sources", cfg.Sources).Msg("sources validated")

	// Ensure output directory exists
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// Build archiver
	archiver, err := archive.New(cfg.Algo)
	if err != nil {
		return err
	}

	// Build archive filename
	hostname, _ := os.Hostname()
	timestamp := meta.FormatTimestamp()
	archiveName := fmt.Sprintf("%s_%s_%s.tar.%s", cfg.Name, hostname, timestamp, archiver.Extension())
	archivePath := filepath.Join(cfg.OutputDir, archiveName)

	log.Info().
		Str("algo", cfg.Algo).
		Str("output", archivePath).
		Msg("archiving")

	if cfg.DryRun {
		log.Info().Msg("dry run — skipping archive, upload, and cleanup")
		return nil
	}

	// Create archive
	if err := archiver.Archive(cfg.Sources, archivePath, cfg.AlgoArgs); err != nil {
		return fmt.Errorf("archive: %w", err)
	}

	// Get archive size
	archiveInfo, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("stat archive: %w", err)
	}
	log.Info().
		Str("file", archiveName).
		Int64("size_bytes", archiveInfo.Size()).
		Msg("archive created")

	// Compute checksum
	checksum, err := meta.ComputeSHA256(archivePath)
	if err != nil {
		return fmt.Errorf("checksum: %w", err)
	}
	log.Info().Str("sha256", checksum[:16]+"...").Msg("checksum computed")

	// Write sidecar metadata
	sidecarName := fmt.Sprintf("%s_%s_%s.meta.json", cfg.Name, hostname, timestamp)
	sidecarPath := filepath.Join(cfg.OutputDir, sidecarName)

	m := &meta.Meta{
		Name:      cfg.Name,
		Hostname:  hostname,
		Timestamp: timestamp,
		Sources:   cfg.Sources,
		Algo:      cfg.Algo,
		AlgoArgs:  cfg.AlgoArgs,
		Archive:   archiveName,
		SizeBytes: archiveInfo.Size(),
		Checksum: meta.Checksum{
			Algo:  "sha256",
			Value: checksum,
		},
		ArcupVersion: arcupVersion,
	}

	if err := m.Write(sidecarPath); err != nil {
		return fmt.Errorf("write sidecar: %w", err)
	}
	log.Info().Str("file", sidecarName).Msg("sidecar written")

	// Upload archive and sidecar
	uploader, err := upload.New(cfg.UploadTo)
	if err != nil {
		return err
	}

	log.Info().Str("remote", cfg.Remote).Msg("uploading archive")
	if err := uploader.Upload(archivePath, cfg.Remote); err != nil {
		return fmt.Errorf("upload archive: %w", err)
	}

	log.Info().Str("remote", cfg.Remote).Msg("uploading sidecar")
	if err := uploader.Upload(sidecarPath, cfg.Remote); err != nil {
		return fmt.Errorf("upload sidecar: %w", err)
	}

	log.Info().Msg("upload complete")

	// Cleanup local files
	if cfg.Keep == 0 {
		os.Remove(archivePath)
		os.Remove(sidecarPath)
		log.Info().Msg("local files cleaned up")
	} else {
		log.Info().Int("keep", cfg.Keep).Msg("keeping local copies")
	}

	log.Info().Str("archive", archiveName).Msg("backup complete")
	return nil
}

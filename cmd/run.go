// Module: cmd/run
// Purpose: Implements the "run" subcommand that performs a backup
// cycle: archive sources, generate metadata sidecar, upload both
// to the remote, and clean up local files. Supports watch mode
// (-w) to repeat on a schedule, and optional retention cleanup
// after each successful backup.
//
// Key Components:
//   - runCmd: Cobra command for "arcup run"
//   - executeRun(): Core backup logic orchestrator
//   - runBackupCycle(): Single backup + optional retention pass
//
// Dependencies:
//   - internal/config: Resolved configuration struct
//   - internal/archive: Archiver factory and compression
//   - internal/upload: Rclone upload executor
//   - internal/meta: Sidecar metadata generation
//   - internal/scheduler: Cron/duration scheduling for watch mode
//   - internal/retention: Policy evaluation for cleanup
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dlcuy22/arcup/internal/archive"
	"github.com/dlcuy22/arcup/internal/config"
	"github.com/dlcuy22/arcup/internal/meta"
	"github.com/dlcuy22/arcup/internal/retention"
	"github.com/dlcuy22/arcup/internal/retry"
	"github.com/dlcuy22/arcup/internal/scheduler"
	"github.com/dlcuy22/arcup/internal/upload"
)

const arcupVersion = "0.1.0"

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a backup (one-shot or scheduled with -w)",
	Long: `Archives the configured sources, uploads to the remote, and exits.
Use -w/--watch with -i/--interval to repeat on a schedule.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeRun(cmd)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringSliceP("source", "s", nil, "source file or directory (repeatable)")
	runCmd.Flags().StringP("algo", "a", "zstd", "compression algorithm: zstd, gz, zip")
	runCmd.Flags().StringP("algo-args", "A", "", "extra arguments for the compression tool")
	runCmd.Flags().StringP("upload-to", "u", "rclone", "upload backend")
	runCmd.Flags().StringP("output-dir", "o", "/tmp/arcup", "local staging directory for archives")
	runCmd.Flags().IntP("keep", "k", 0, "keep N local copies (0 = delete after upload)")
	runCmd.Flags().StringP("name", "N", "", "archive name prefix (default: hostname)")

	// Watch mode
	runCmd.Flags().BoolP("watch", "w", false, "run on a schedule instead of one-shot")
	runCmd.Flags().StringP("interval", "i", "", `schedule: cron expr or duration (e.g. "@daily", "6h")`)

	// Retention
	runCmd.Flags().IntP("keep-days", "d", 0, "delete remote backups older than N days (0 = disabled)")
	runCmd.Flags().IntP("keep-last", "K", 0, "always keep at least N most recent (0 = disabled)")

	// Retry
	runCmd.Flags().Int("retry-attempts", 3, "max attempts for archive/upload")
	runCmd.Flags().String("retry-delay", "1m", "delay between retries")

	viper.BindPFlag("sources", runCmd.Flags().Lookup("source"))
	viper.BindPFlag("algo", runCmd.Flags().Lookup("algo"))
	viper.BindPFlag("algo-args", runCmd.Flags().Lookup("algo-args"))
	viper.BindPFlag("upload-to", runCmd.Flags().Lookup("upload-to"))
	viper.BindPFlag("output-dir", runCmd.Flags().Lookup("output-dir"))
	viper.BindPFlag("keep", runCmd.Flags().Lookup("keep"))
	viper.BindPFlag("name", runCmd.Flags().Lookup("name"))
	viper.BindPFlag("watch", runCmd.Flags().Lookup("watch"))
	viper.BindPFlag("interval", runCmd.Flags().Lookup("interval"))
	viper.BindPFlag("retention.keep-days", runCmd.Flags().Lookup("keep-days"))
	viper.BindPFlag("retention.keep-last", runCmd.Flags().Lookup("keep-last"))
	viper.BindPFlag("retry.max-attempts", runCmd.Flags().Lookup("retry-attempts"))
	viper.BindPFlag("retry.delay", runCmd.Flags().Lookup("retry-delay"))
}

func executeRun(cmd *cobra.Command) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		cmd.Help()
		fmt.Fprintf(os.Stderr, "\n  error: %v\n", err)
		return nil
	}

	if cfg.Watch {
		if cfg.Interval == "" {
			cmd.Help()
			fmt.Fprintf(os.Stderr, "\n  error: --watch requires --interval (e.g. -i 6h)\n")
			return nil
		}

		runner, err := scheduler.Parse(cfg.Interval)
		if err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		log.Info().Str("interval", cfg.Interval).Msg("watch mode started, press Ctrl+C to stop")

		return runner.Start(ctx, func() {
			log.Info().Msg("starting backup cycle")
			if err := runBackupCycle(cfg); err != nil {
				log.Error().Err(err).Msg("backup cycle failed")
			}
		})
	}

	return runBackupCycle(cfg)
}

func runBackupCycle(cfg *config.Config) error {
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
		log.Info().Msg("dry run, skipping archive, upload, and cleanup")
		return nil
	}

	// Create archive with retries
	err = retry.Do("archive", cfg.Retry.MaxAttempts, cfg.Retry.Delay, func() error {
		return archiver.Archive(cfg.Sources, archivePath, cfg.AlgoArgs)
	})
	if err != nil {
		return fmt.Errorf("archive failed: %w", err)
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
	err = retry.Do("upload_archive", cfg.Retry.MaxAttempts, cfg.Retry.Delay, func() error {
		return uploader.Upload(archivePath, cfg.Remote)
	})
	if err != nil {
		return fmt.Errorf("upload archive: %w", err)
	}

	log.Info().Str("remote", cfg.Remote).Msg("uploading sidecar")
	err = retry.Do("upload_sidecar", cfg.Retry.MaxAttempts, cfg.Retry.Delay, func() error {
		return uploader.Upload(sidecarPath, cfg.Remote)
	})
	if err != nil {
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

	// Run retention if configured
	if cfg.Retention.KeepDays > 0 || cfg.Retention.KeepLast > 0 {
		log.Info().
			Int("keep_days", cfg.Retention.KeepDays).
			Int("keep_last", cfg.Retention.KeepLast).
			Msg("running retention policy")

		entries, err := buildRetentionEntries(uploader, cfg.Remote)
		if err != nil {
			log.Warn().Err(err).Msg("retention skipped: could not list remote")
		} else {
			policy := retention.Policy{
				KeepDays: cfg.Retention.KeepDays,
				KeepLast: cfg.Retention.KeepLast,
			}
			results := retention.Evaluate(policy, entries)

			deleted := 0
			for _, r := range results {
				if r.Action == retention.Delete {
					if cfg.DryRun {
						log.Info().Str("archive", r.Entry.ArchivePath).Msg("dry run: would delete")
						continue
					}
					if err := uploader.Delete(r.Entry.ArchivePath); err != nil {
						log.Warn().Err(err).Str("file", r.Entry.ArchivePath).Msg("failed to delete archive")
						continue
					}
					if err := uploader.Delete(r.Entry.SidecarPath); err != nil {
						log.Warn().Err(err).Str("file", r.Entry.SidecarPath).Msg("failed to delete sidecar")
					}
					deleted++
				}
			}
			log.Info().Int("deleted", deleted).Int("retained", len(results)-deleted).Msg("retention complete")
		}
	}

	log.Info().Str("archive", archiveName).Msg("backup complete")
	return nil
}

func buildRetentionEntries(uploader upload.Uploader, remote string) ([]retention.Entry, error) {
	remoteEntries, err := uploader.List(remote)
	if err != nil {
		return nil, err
	}

	var entries []retention.Entry
	for _, re := range remoteEntries {
		if !strings.HasSuffix(re.Name, ".meta.json") {
			continue
		}

		ts := parseTimestampFromName(re.Name)
		archiveName := strings.TrimSuffix(re.Name, ".meta.json")

		found := false
		for _, candidate := range remoteEntries {
			if strings.HasPrefix(candidate.Name, archiveName+".tar.") {
				entries = append(entries, retention.Entry{
					ArchivePath: remoteJoin(remote, candidate.Name),
					SidecarPath: remoteJoin(remote, re.Name),
					Timestamp:   ts,
					SizeBytes:   candidate.Size,
				})
				found = true
				break
			}
		}
		if !found {
			entries = append(entries, retention.Entry{
				ArchivePath: "",
				SidecarPath: remoteJoin(remote, re.Name),
				Timestamp:   ts,
				SizeBytes:   0,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})
	return entries, nil
}

func parseTimestampFromName(name string) time.Time {
	// name format: {prefix}_{host}_{timestamp}.meta.json or .tar.ext
	// timestamp format: 2006-01-02T15-04-05
	parts := strings.Split(name, "_")
	if len(parts) >= 3 {
		tsPart := parts[len(parts)-1]
		tsPart = strings.TrimSuffix(tsPart, ".meta.json")
		if idx := strings.Index(tsPart, ".tar."); idx != -1 {
			tsPart = tsPart[:idx]
		}
		if t, err := time.ParseInLocation("2006-01-02T15-04-05", tsPart, time.Local); err == nil {
			return t
		}
	}
	return time.Time{}
}

func remoteJoin(remote, name string) string {
	if !strings.HasSuffix(remote, "/") {
		return remote + "/" + name
	}
	return remote + name
}

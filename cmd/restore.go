// Module: cmd/restore
// Purpose: Implements the "restore" subcommand that lists available
// backups on the remote, allows selection by index, downloads the
// archive + sidecar, verifies checksum, and extracts to a destination.
//
// Key Components:
//   - restoreCmd: Cobra command for "arcup restore"
//   - executeRestore(): Full restore orchestration
//   - listBackups(): Fetches and displays remote backups
//
// Dependencies:
//   - internal/upload: Rclone listing and download
//   - internal/meta: Sidecar metadata parsing and checksum verification
//   - internal/archive: Extraction by algo type
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/dlcuy22/arcup/internal/archive"
	"github.com/dlcuy22/arcup/internal/config"
	"github.com/dlcuy22/arcup/internal/meta"
	"github.com/dlcuy22/arcup/internal/upload"
)

type backupEntry struct {
	Meta        *meta.Meta
	SidecarName string
	Age         string
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "List and restore backups from a remote",
	Long:  `Fetches the backup list from the remote, displays recent entries, and restores a selected archive.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeRestore(cmd)
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().StringP("to", "t", ".", "local destination for extraction")
	restoreCmd.Flags().IntP("select", "S", 0, "skip prompt, pick backup by index")
	restoreCmd.Flags().BoolP("list", "l", false, "list only, don't download or extract")
	restoreCmd.Flags().IntP("limit", "L", 5, "number of backups to show")
	restoreCmd.Flags().BoolP("verify", "v", true, "checksum verify after download")
}

func executeRestore(cmd *cobra.Command) error {
	cfg, err := config.Load()
	if err != nil || cfg.Remote == "" {
		cmd.Help()
		return nil
	}

	listOnly, _ := cmd.Flags().GetBool("list")
	limit, _ := cmd.Flags().GetInt("limit")
	selectIdx, _ := cmd.Flags().GetInt("select")
	destDir, _ := cmd.Flags().GetString("to")
	verify, _ := cmd.Flags().GetBool("verify")

	uploader, err := upload.New(cfg.UploadTo)
	if err != nil {
		return err
	}

	// List .meta.json files from remote
	log.Info().Str("remote", cfg.Remote).Msg("listing backups")
	entries, err := listBackups(uploader, cfg.Remote, limit)
	if err != nil {
		return fmt.Errorf("list backups: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("no backups found")
		return nil
	}

	// Display backups
	fmt.Printf("\n  %-5s %-40s %-12s %s\n", "INDEX", "ARCHIVE", "SIZE", "AGE")
	fmt.Printf("  %-5s %-40s %-12s %s\n", "-----", "-------", "----", "---")
	for i, e := range entries {
		size := formatSize(e.Meta.SizeBytes)
		fmt.Printf("  %-5d %-40s %-12s %s\n", i+1, e.Meta.Archive, size, e.Age)
	}
	fmt.Println()

	if listOnly {
		return nil
	}

	// Select backup
	idx := selectIdx
	if idx == 0 {
		fmt.Print("  select backup to restore (index): ")
		if _, err := fmt.Scan(&idx); err != nil {
			return fmt.Errorf("invalid selection: %w", err)
		}
	}
	if idx < 1 || idx > len(entries) {
		return fmt.Errorf("index %d out of range (1-%d)", idx, len(entries))
	}

	selected := entries[idx-1]
	log.Info().Str("archive", selected.Meta.Archive).Msg("selected for restore")

	// Create temp download directory
	tmpDir, err := os.MkdirTemp("", "arcup-restore-")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download archive
	archiveRemote := remoteJoin(cfg.Remote, selected.Meta.Archive)
	log.Info().Str("file", selected.Meta.Archive).Msg("downloading archive")
	if err := uploader.Download(archiveRemote, tmpDir); err != nil {
		return fmt.Errorf("download archive: %w", err)
	}

	localArchive := filepath.Join(tmpDir, selected.Meta.Archive)

	// Verify checksum
	if verify {
		log.Info().Msg("verifying checksum")
		computed, err := meta.ComputeSHA256(localArchive)
		if err != nil {
			return fmt.Errorf("checksum compute: %w", err)
		}
		if computed != selected.Meta.Checksum.Value {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", selected.Meta.Checksum.Value[:16]+"...", computed[:16]+"...")
		}
		log.Info().Msg("checksum verified")
	}

	// Extract archive
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}

	archiver, err := archive.New(selected.Meta.Algo)
	if err != nil {
		return fmt.Errorf("unsupported algo %q: %w", selected.Meta.Algo, err)
	}

	log.Info().Str("to", destDir).Msg("extracting")
	if err := archiver.Extract(localArchive, destDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	log.Info().Str("archive", selected.Meta.Archive).Str("destination", destDir).Msg("restore complete")
	return nil
}

func listBackups(uploader upload.Uploader, remote string, limit int) ([]backupEntry, error) {
	// First, download all .meta.json files at once to speed up listing
	tmpDir, err := os.MkdirTemp("", "arcup-meta-")
	if err != nil {
		return nil, fmt.Errorf("create meta temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := uploader.DownloadFiltered(remote, tmpDir, "*.meta.json"); err != nil {
		return nil, fmt.Errorf("download meta files: %w", err)
	}

	// Read downloaded files
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("read meta temp dir: %w", err)
	}

	var backups []backupEntry
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".meta.json") || f.IsDir() {
			continue
		}

		localSidecar := filepath.Join(tmpDir, f.Name())
		data, err := os.ReadFile(localSidecar)
		if err != nil {
			continue
		}

		var m meta.Meta
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}

		ts, _ := time.ParseInLocation("2006-01-02T15-04-05", m.Timestamp, time.Local)
		age := formatAge(time.Since(ts))

		backups = append(backups, backupEntry{
			Meta:        &m,
			SidecarName: f.Name(),
			Age:         age,
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Meta.Timestamp > backups[j].Meta.Timestamp
	})

	if limit > 0 && len(backups) > limit {
		backups = backups[:limit]
	}
	return backups, nil
}

func formatAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func remoteJoin(remote, name string) string {
	if !strings.HasSuffix(remote, "/") {
		return remote + "/" + name
	}
	return remote + name
}

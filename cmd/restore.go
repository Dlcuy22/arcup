// Module: cmd/restore
// Purpose: Implements the "restore" subcommand that lists available
// backups on the remote, allows interactive selection, downloads,
// verifies checksum, and extracts the archive.
//
// Key Components:
//   - restoreCmd: Cobra command for "arcup restore"
//   - executeRestore(): Lists, selects, downloads, and extracts
//
// Dependencies:
//   - internal/upload: Rclone listing and download
//   - internal/meta: Sidecar metadata parsing
//   - internal/archive: Extraction
package cmd

import "github.com/spf13/cobra"

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "List and restore backups from a remote",
	Long:  `Fetches the backup list from the remote, displays recent entries, and restores a selected archive.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeRestore()
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().StringP("remote", "r", "", "rclone remote path to list from")
	restoreCmd.Flags().StringP("to", "t", ".", "local destination for extraction")
	restoreCmd.Flags().IntP("select", "S", 0, "skip prompt, pick backup by index")
	restoreCmd.Flags().BoolP("list", "l", false, "list only, don't download or extract")
	restoreCmd.Flags().IntP("limit", "L", 5, "number of backups to show")
	restoreCmd.Flags().BoolP("verify", "v", true, "checksum verify after download")
}

func executeRestore() error {
	// TODO: implement restore
	// 1. rclone lsjson remote → list .meta.json files
	// 2. Parse timestamps, sort descending
	// 3. Display top N with index, size, and age
	// 4. If --list, exit after display
	// 5. Prompt user for selection (or use --select)
	// 6. rclone copy archive to temp dir
	// 7. Verify checksum against .meta.json
	// 8. Extract archive to --to destination
	return nil
}

// Module: cmd/retention
// Purpose: Implements the "retention" subcommand that applies
// retention policies to remote backups, deleting expired archives
// while respecting keep-last safety guards.
//
// Key Components:
//   - retentionCmd: Cobra command for "arcup retention"
//   - executeRetention(): Evaluates policy and deletes expired entries
//
// Dependencies:
//   - internal/retention: Policy evaluation logic
//   - internal/upload: Remote listing and deletion
//   - internal/meta: Sidecar parsing
package cmd

import "github.com/spf13/cobra"

var retentionCmd = &cobra.Command{
	Use:   "retention",
	Short: "Apply retention policy to remote backups",
	Long:  `Lists remote backups and deletes those that exceed the configured retention policy.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeRetention()
	},
}

func init() {
	rootCmd.AddCommand(retentionCmd)

	retentionCmd.Flags().StringP("remote", "r", "", "rclone remote path")
	retentionCmd.Flags().IntP("keep-days", "d", 7, "delete backups older than N days")
	retentionCmd.Flags().IntP("keep-last", "K", 5, "always keep at least N most recent")
}

func executeRetention() error {
	// TODO: implement retention
	// 1. rclone lsjson remote → list .meta.json files
	// 2. Parse timestamps, sort descending (newest first)
	// 3. Mark top N as protected (keep-last)
	// 4. Mark anything older than cutoff as expired (keep-days)
	// 5. Candidates = expired AND NOT protected
	// 6. If dry-run, log what would be deleted and exit
	// 7. rclone deletefile archive + sidecar for each candidate
	// 8. Log summary: X deleted, Y retained
	return nil
}

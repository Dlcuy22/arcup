// Module: cmd/watch
// Purpose: Implements the "watch" subcommand that runs backup
// and optionally retention cycles on a schedule. Blocks until
// interrupted via signal.
//
// Key Components:
//   - watchCmd: Cobra command for "arcup watch"
//   - executeWatch(): Sets up scheduler loops and blocks
//
// Dependencies:
//   - internal/scheduler: Cron/duration scheduler
//   - internal/config: Resolved configuration
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Run scheduled backups (daemon mode)",
	Long:  `Runs backup cycles on the configured interval. Blocks until interrupted.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeWatch()
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().StringP("interval", "i", "", `schedule: cron expr or duration (e.g. "@daily", "6h")`)
	viper.BindPFlag("interval", watchCmd.Flags().Lookup("interval"))
}

func executeWatch() error {
	// TODO: implement watch mode
	// 1. Load config
	// 2. Parse interval via scheduler.Parse()
	// 3. Register backup cycle callback (executeRun)
	// 4. If retention.interval is set, register retention cycle
	// 5. Start scheduler and block until SIGINT/SIGTERM
	return nil
}

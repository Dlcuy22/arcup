// Module: internal/scheduler
// Purpose: Parses interval strings (cron expressions or Go durations)
// and provides a unified runner that executes a callback on schedule.
//
// Key Components:
//   - Runner: Interface for scheduled execution
//   - TickRunner: Duration-based runner using time.Ticker
//   - CronRunner: Cron expression runner using robfig/cron
//   - Parse(): Factory that returns the correct runner type
//
// Dependencies:
//   - github.com/robfig/cron/v3: Cron expression scheduling
//
// Example:
//
//	r, err := scheduler.Parse("6h")
//	r.Start(ctx, func() { fmt.Println("tick") })
package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

type Runner interface {
	Start(ctx context.Context, fn func()) error
}

type TickRunner struct {
	Interval time.Duration
}

type CronRunner struct {
	Expr string
}

/*
Parse takes an interval string and returns the appropriate Runner.
Tries Go duration first ("6h", "30m"), falls back to cron expression
("@daily", "0 2 * * *").

	params:
	      interval: duration string or cron expression
	returns:
	      Runner: scheduled executor
	      error: if neither format parses
*/
func Parse(interval string) (Runner, error) {
	if d, err := time.ParseDuration(interval); err == nil {
		return &TickRunner{Interval: d}, nil
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(interval); err != nil {
		return nil, fmt.Errorf("invalid interval %q: not a duration or cron expression", interval)
	}
	return &CronRunner{Expr: interval}, nil
}

func (t *TickRunner) Start(ctx context.Context, fn func()) error {
	ticker := time.NewTicker(t.Interval)
	defer ticker.Stop()

	// Run immediately on start, then on each tick
	fn()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			fn()
		}
	}
}

func (c *CronRunner) Start(ctx context.Context, fn func()) error {
	// Run immediately on start, then on schedule
	fn()

	cr := cron.New()
	_, err := cr.AddFunc(c.Expr, fn)
	if err != nil {
		return fmt.Errorf("cron add: %w", err)
	}

	cr.Start()
	<-ctx.Done()
	cr.Stop()
	return ctx.Err()
}

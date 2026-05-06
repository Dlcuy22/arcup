// Module: internal/retention
// Purpose: Evaluates retention policies against a list of backup
// entries and determines which should be deleted. Applies both
// keep-days (age-based) and keep-last (count-based) rules.
//
// Key Components:
//   - Policy: Retention policy configuration
//   - Entry: Represents a backup with timestamp for evaluation
//   - Result: Outcome for a single entry (retain/delete + reason)
//   - Evaluate(): Applies policy to a sorted list of entries
//
// Example:
//
//	results := retention.Evaluate(policy, entries)
//	for _, r := range results { ... }
package retention

import "time"

type Policy struct {
	KeepDays int
	KeepLast int
}

type Entry struct {
	ArchivePath string
	SidecarPath string
	Timestamp   time.Time
	SizeBytes   int64
}

type Action int

const (
	Retain Action = iota
	Delete
)

type Result struct {
	Entry  Entry
	Action Action
	Reason string
}

/*
Evaluate applies the retention policy to a list of entries.
Entries MUST be pre-sorted descending by timestamp (newest first).

The logic:
  - Top N entries (KeepLast) are always protected
  - Entries older than KeepDays are marked expired
  - Only entries that are expired AND NOT protected are deleted

	params:
	      policy: retention policy with KeepDays and KeepLast
	      entries: sorted backup entries (newest first)
	returns:
	      []Result: action and reason for each entry
*/
func Evaluate(policy Policy, entries []Entry) []Result {
	cutoff := time.Now().Add(-time.Duration(policy.KeepDays) * 24 * time.Hour)
	results := make([]Result, len(entries))

	for i, entry := range entries {
		protected := i < policy.KeepLast
		expired := entry.Timestamp.Before(cutoff)

		switch {
		case protected && !expired:
			results[i] = Result{Entry: entry, Action: Retain, Reason: "keep-last"}
		case protected && expired:
			results[i] = Result{Entry: entry, Action: Retain, Reason: "keep-last (expired but protected)"}
		case !protected && !expired:
			results[i] = Result{Entry: entry, Action: Retain, Reason: "keep-days"}
		default:
			results[i] = Result{Entry: entry, Action: Delete, Reason: "expired"}
		}
	}

	return results
}

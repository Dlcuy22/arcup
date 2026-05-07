// Module: internal/retention
// Purpose: Evaluates retention policies against a list of backup
// entries and determines which should be deleted. Applies both
// keep-within (duration/age-based) and keep-last (count-based) rules.
// Provides an Execute function to orchestrate listing, evaluation, and deletion.
//
// Key Components:
//   - Policy: Retention policy configuration
//   - Entry: Represents a backup with timestamp for evaluation
//   - Result: Outcome for a single entry (retain/delete + reason)
//   - Evaluate(): Applies policy to a sorted list of entries
//   - Execute(): Evaluates and deletes expired entries via Uploader
//
// Dependencies:
//   - github.com/dlcuy22/arcup/internal/upload: remote file operations
//
// Example:
//
//	deleted, err := retention.Execute(policy, uploader, remote, false)
package retention

import (
	"sort"
	"strings"
	"time"

	"github.com/dlcuy22/arcup/internal/upload"
	"github.com/rs/zerolog/log"
)

type Policy struct {
	KeepWithin time.Duration
	KeepLast   int
}

type Entry struct {
	ArchivePath string
	SidecarPath string
	ArchiveName string
	SidecarName string
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
  - Entries older than KeepWithin are marked expired
  - Only entries that are expired AND NOT protected are deleted

	params:
	      policy: retention policy with KeepWithin and KeepLast
	      entries: sorted backup entries (newest first)
	returns:
	      []Result: action and reason for each entry
*/
func Evaluate(policy Policy, entries []Entry) []Result {
	cutoff := time.Now().Add(-policy.KeepWithin)
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
			results[i] = Result{Entry: entry, Action: Retain, Reason: "keep-within"}
		default:
			results[i] = Result{Entry: entry, Action: Delete, Reason: "expired"}
		}
	}

	return results
}

/*
Execute runs the complete retention cycle: builds remote entries,
evaluates the policy, and deletes expired files via the given uploader.

	params:
	      policy: retention policy to apply
	      uploader: upload client used to list and delete files
	      remote: remote path to list
	      dryRun: if true, skips actual deletion
	returns:
	      int: number of entries deleted
	      error: if building entries fails
*/
func Execute(policy Policy, uploader upload.Uploader, remote string, dryRun bool) (int, error) {
	entries, err := buildEntries(uploader, remote)
	if err != nil {
		return 0, err
	}

	results := Evaluate(policy, entries)
	deleted := 0
	var toDelete []string

	for _, r := range results {
		if r.Action == Delete {
			if dryRun {
				log.Info().Str("archive", r.Entry.ArchivePath).Msg("dry run: would delete")
				continue
			}
			if r.Entry.ArchiveName != "" {
				toDelete = append(toDelete, r.Entry.ArchiveName)
			}
			toDelete = append(toDelete, r.Entry.SidecarName)
			deleted++
		}
	}

	if len(toDelete) > 0 {
		log.Info().Int("count", len(toDelete)).Msg("executing batch delete")
		if err := uploader.DeleteBatch(remote, toDelete); err != nil {
			log.Warn().Err(err).Msg("batch delete failed partially or entirely")
		}
	}

	return deleted, nil
}

func buildEntries(uploader upload.Uploader, remote string) ([]Entry, error) {
	remoteEntries, err := uploader.List(remote)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, re := range remoteEntries {
		if !strings.HasSuffix(re.Name, ".meta.json") {
			continue
		}

		ts := parseTimestampFromName(re.Name)
		archiveName := strings.TrimSuffix(re.Name, ".meta.json")

		found := false
		for _, candidate := range remoteEntries {
			if strings.HasPrefix(candidate.Name, archiveName+".tar.") {
				entries = append(entries, Entry{
					ArchivePath: remoteJoin(remote, candidate.Name),
					SidecarPath: remoteJoin(remote, re.Name),
					ArchiveName: candidate.Path,
					SidecarName: re.Path,
					Timestamp:   ts,
					SizeBytes:   candidate.Size,
				})
				found = true
				break
			}
		}
		if !found {
			entries = append(entries, Entry{
				ArchivePath: "",
				SidecarPath: remoteJoin(remote, re.Name),
				ArchiveName: "",
				SidecarName: re.Path,
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

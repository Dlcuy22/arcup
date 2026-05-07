// Module: test/retention_test.go
// Purpose: Unit tests for retention policy evaluation, covering keep-last
// protection, keep-days expiration, mixed policies, and edge cases like
// empty input and all-expired scenarios.
package test

import (
	"testing"
	"time"

	"github.com/dlcuy22/arcup/internal/retention"
)

func makeEntries(ages ...int) []retention.Entry {
	entries := make([]retention.Entry, len(ages))
	for i, daysOld := range ages {
		entries[i] = retention.Entry{
			ArchivePath: "remote:backup/archive-" + time.Now().AddDate(0, 0, -daysOld).Format("2006-01-02") + ".tar.zst",
			SidecarPath: "remote:backup/archive-" + time.Now().AddDate(0, 0, -daysOld).Format("2006-01-02") + ".meta.json",
			Timestamp:   time.Now().AddDate(0, 0, -daysOld),
			SizeBytes:   1024,
		}
	}
	return entries
}

func TestRetention_KeepLast_ProtectsNewest(t *testing.T) {
	entries := makeEntries(0, 1, 2, 30, 60)
	policy := retention.Policy{KeepWithin: 7 * 24 * time.Hour, KeepLast: 3}

	results := retention.Evaluate(policy, entries)

	for i := 0; i < 3; i++ {
		if results[i].Action != retention.Retain {
			t.Fatalf("entry %d should be retained (keep-last), got delete", i)
		}
	}
	for i := 3; i < 5; i++ {
		if results[i].Action != retention.Delete {
			t.Fatalf("entry %d should be deleted (expired), got retain", i)
		}
	}
}

func TestRetention_KeepWithin_RetainsRecent(t *testing.T) {
	entries := makeEntries(1, 3, 5, 10, 20)
	policy := retention.Policy{KeepWithin: 7 * 24 * time.Hour, KeepLast: 0}

	results := retention.Evaluate(policy, entries)

	for _, r := range results[:3] {
		if r.Action != retention.Retain {
			t.Fatalf("entry within 7 days should be retained, got delete: %s", r.Entry.ArchivePath)
		}
	}
	for _, r := range results[3:] {
		if r.Action != retention.Delete {
			t.Fatalf("entry older than 7 days should be deleted, got retain: %s", r.Entry.ArchivePath)
		}
	}
}

func TestRetention_KeepLast_ProtectsExpired(t *testing.T) {
	entries := makeEntries(30, 60, 90)
	policy := retention.Policy{KeepWithin: 7 * 24 * time.Hour, KeepLast: 2}

	results := retention.Evaluate(policy, entries)

	if results[0].Action != retention.Retain {
		t.Fatal("first entry should be protected by keep-last even though expired")
	}
	if results[1].Action != retention.Retain {
		t.Fatal("second entry should be protected by keep-last even though expired")
	}
	if results[2].Action != retention.Delete {
		t.Fatal("third entry should be deleted (expired and not protected)")
	}
}

func TestRetention_EmptyEntries(t *testing.T) {
	results := retention.Evaluate(retention.Policy{KeepWithin: 7 * 24 * time.Hour, KeepLast: 3}, nil)
	if len(results) != 0 {
		t.Fatalf("expected 0 results for nil entries, got %d", len(results))
	}
}

func TestRetention_AllRetained(t *testing.T) {
	entries := makeEntries(0, 1, 2)
	policy := retention.Policy{KeepWithin: 30 * 24 * time.Hour, KeepLast: 5}

	results := retention.Evaluate(policy, entries)

	for i, r := range results {
		if r.Action != retention.Retain {
			t.Fatalf("entry %d should be retained, got delete", i)
		}
	}
}

func TestRetention_AllExpiredButProtected(t *testing.T) {
	entries := makeEntries(100, 200, 300)
	policy := retention.Policy{KeepWithin: 24 * time.Hour, KeepLast: 10}

	results := retention.Evaluate(policy, entries)

	for i, r := range results {
		if r.Action != retention.Retain {
			t.Fatalf("entry %d should be protected by keep-last, got delete", i)
		}
	}
}

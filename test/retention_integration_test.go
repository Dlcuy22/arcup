package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dlcuy22/arcup/internal/retention"
	"github.com/dlcuy22/arcup/internal/upload"
)

// To run this test, ensure rclone is configured with a 'gd:' remote.
// Run: go test ./test/... -run TestRetentionBatchIntegration
func TestRetentionBatchIntegration(t *testing.T) {
	remote := "gd:/tmp/arcup-test-batch/"

	uploader, err := upload.New("rclone")
	if err != nil {
		t.Fatalf("Failed to initialize uploader: %v", err)
	}

	t.Logf("Cleaning remote: %s", remote)
	cleanupRemote(t, remote)
	defer cleanupRemote(t, remote)
	localTmp, err := os.MkdirTemp("", "arcup-batch-test-")
	if err != nil {
		t.Fatalf("Failed to create local temp dir: %v", err)
	}
	defer os.RemoveAll(localTmp)

	t.Logf("Creating 10 dummy backup pairs in %s", localTmp)
	for i := 0; i < 10; i++ {
		ts := time.Now().Add(-48 * time.Hour).Add(time.Duration(i) * time.Minute)
		timestampStr := ts.Format("2006-01-02T15-04-05")

		metaName := fmt.Sprintf("batchtest_localhost_%s.meta.json", timestampStr)
		archiveName := fmt.Sprintf("batchtest_localhost_%s.tar.zst", timestampStr)

		if err := os.WriteFile(filepath.Join(localTmp, metaName), []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to write dummy meta: %v", err)
		}
		if err := os.WriteFile(filepath.Join(localTmp, archiveName), []byte("dummy archive"), 0644); err != nil {
			t.Fatalf("Failed to write dummy archive: %v", err)
		}
	}

	t.Logf("Uploading dummy files to %s", remote)
	cmd := exec.Command("rclone", "copy", localTmp, remote)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to upload dummy files: %v\nOutput: %s", err, output)
	}
	entries, err := uploader.List(remote)
	if err != nil {
		t.Fatalf("Failed to list remote: %v", err)
	}
	if len(entries) != 20 {
		t.Fatalf("Expected 20 files on remote, found %d", len(entries))
	}

	t.Log("Executing batch retention...")
	policy := retention.Policy{
		KeepWithin: 24 * time.Hour,
		KeepLast:   0,
	}

	deletedCount, err := retention.Execute(policy, uploader, remote, false)
	if err != nil {
		t.Fatalf("Retention Execute failed: %v", err)
	}
	if deletedCount != 10 {
		t.Fatalf("Expected 10 entries deleted, got %d", deletedCount)
	}

	t.Log("Verifying remote is empty...")
	entriesAfter, err := uploader.List(remote)
	if err != nil {
		t.Fatalf("Failed to list remote after deletion: %v", err)
	}
	if len(entriesAfter) > 0 {
		t.Fatalf("Expected remote to be empty, but found %d files left", len(entriesAfter))
	}
	t.Log("Integration test completed successfully!")
}

func cleanupRemote(t *testing.T, remote string) {
	cmd := exec.Command("rclone", "purge", remote)
	_ = cmd.Run()
}

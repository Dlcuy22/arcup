// Module: test/restore_test.go
// Purpose: Unit tests for restore-adjacent logic: meta parsing, checksum
// verification, and archive extraction. Uses real filesystem operations
// with temp directories.
package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dlcuy22/arcup/internal/meta"
)

func TestMeta_WriteAndReadBack(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.meta.json")

	m := &meta.Meta{
		Name:      "test-backup",
		Hostname:  "testhost",
		Timestamp: "2026-05-06T12-00-00",
		Sources:   []string{"/tmp/src"},
		Algo:      "zstd",
		Archive:   "test-backup_testhost_2026-05-06T12-00-00.tar.zst",
		SizeBytes: 4096,
		Checksum: meta.Checksum{
			Algo:  "sha256",
			Value: "abc123",
		},
		ArcupVersion: "0.1.0",
	}

	if err := m.Write(path); err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	loaded, err := meta.ReadFromFile(path)
	if err != nil {
		t.Fatalf("ReadFromFile() failed: %v", err)
	}

	if loaded.Name != m.Name {
		t.Fatalf("expected name %q, got %q", m.Name, loaded.Name)
	}
	if loaded.Checksum.Value != m.Checksum.Value {
		t.Fatalf("expected checksum %q, got %q", m.Checksum.Value, loaded.Checksum.Value)
	}
	if loaded.SizeBytes != m.SizeBytes {
		t.Fatalf("expected size %d, got %d", m.SizeBytes, loaded.SizeBytes)
	}
}

func TestMeta_ReadFromBytes_InvalidJSON(t *testing.T) {
	_, err := meta.ReadFromBytes([]byte("not valid json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestMeta_ComputeSHA256(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "testfile.txt")

	if err := os.WriteFile(path, []byte("hello arcup"), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	hash, err := meta.ComputeSHA256(path)
	if err != nil {
		t.Fatalf("ComputeSHA256() failed: %v", err)
	}

	if len(hash) != 64 {
		t.Fatalf("expected 64-char hex hash, got %d chars", len(hash))
	}

	hash2, _ := meta.ComputeSHA256(path)
	if hash != hash2 {
		t.Fatal("same file produced different checksums")
	}
}

func TestMeta_ComputeSHA256_MissingFile(t *testing.T) {
	_, err := meta.ComputeSHA256("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestArchive_ZstdRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("zstd"); err != nil {
		t.Skip("zstd not found, skipping")
	}
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("tar not found, skipping")
	}

	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatal(err)
	}

	archiveDir := t.TempDir()
	archivePath := filepath.Join(archiveDir, "test.tar.zst")

	archiver := zstdArchiver()
	if err := archiver.Archive([]string{srcDir}, archivePath, ""); err != nil {
		t.Fatalf("Archive() failed: %v", err)
	}

	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archive file not created: %v", err)
	}

	extractDir := t.TempDir()
	if err := archiver.Extract(archivePath, extractDir); err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	extracted := filepath.Join(extractDir, srcDir, "file1.txt")
	data, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatalf("extracted file not found: %v", err)
	}
	if string(data) != "content1" {
		t.Fatalf("expected 'content1', got %q", string(data))
	}
}

func zstdArchiver() archiver {
	return &zstdImpl{}
}

type archiver interface {
	Archive(sources []string, dest string, extraArgs string) error
	Extract(archivePath string, dest string) error
}

type zstdImpl struct{}

func (z *zstdImpl) Archive(sources []string, dest string, extraArgs string) error {
	args := append([]string{"-cf", "-"}, sources...)
	tar := exec.Command("tar", args...)
	zstd := exec.Command("zstd", "-o", dest)

	pipe, err := tar.StdoutPipe()
	if err != nil {
		return err
	}
	zstd.Stdin = pipe

	if err := tar.Start(); err != nil {
		return err
	}
	if err := zstd.Run(); err != nil {
		return err
	}
	return tar.Wait()
}

func (z *zstdImpl) Extract(archivePath string, dest string) error {
	zstd := exec.Command("zstd", "-d", archivePath, "--stdout")
	tar := exec.Command("tar", "-xf", "-", "-C", dest)

	pipe, err := zstd.StdoutPipe()
	if err != nil {
		return err
	}
	tar.Stdin = pipe

	if err := zstd.Start(); err != nil {
		return err
	}
	if err := tar.Run(); err != nil {
		return err
	}
	return zstd.Wait()
}

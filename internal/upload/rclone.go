// Module: internal/upload (rclone)
// Purpose: Implements the Uploader interface by shelling out
// to rclone. Inherits all rclone config (remotes, auth, etc).
//
// Key Components:
//   - RcloneUploader: Concrete uploader using rclone CLI
//
// Dependencies:
//   - os/exec: Shells out to rclone binary
//   - encoding/json: Parses rclone lsjson output
package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type RcloneUploader struct{}

func (r *RcloneUploader) Upload(localPath, remotePath string) error {
	// rclone copy wants a directory as source; use the parent dir
	// with --include to target only the specific file
	dir := filepath.Dir(localPath)
	name := filepath.Base(localPath)
	cmd := exec.Command("rclone", "copy", dir, remotePath,
		"--include", name, "--progress")
	var stderrBytes bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBytes)
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone copy: %w: %s", err, stderrBytes.String())
	}
	return nil
}

func (r *RcloneUploader) Download(remotePath, localDir string) error {
	cmd := exec.Command("rclone", "copy", remotePath, localDir, "--progress")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone download: %w", err)
	}
	return nil
}

func (r *RcloneUploader) DownloadFiltered(remotePath, localDir, includePattern string) error {
	cmd := exec.Command("rclone", "copy", remotePath, localDir,
		"--include", includePattern, "--progress")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone filtered download: %w", err)
	}
	return nil
}

func (r *RcloneUploader) List(remotePath string) ([]RemoteEntry, error) {
	cmd := exec.Command("rclone", "lsjson", remotePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rclone lsjson: %w", err)
	}

	var entries []RemoteEntry
	if err := json.Unmarshal(output, &entries); err != nil {
		return nil, fmt.Errorf("parse rclone output: %w", err)
	}
	return entries, nil
}

func (r *RcloneUploader) Delete(remotePath string) error {
	cmd := exec.Command("rclone", "deletefile", remotePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rclone deletefile: %w: %s", err, output)
	}
	return nil
}

func (r *RcloneUploader) DeleteBatch(remotePath string, relativePaths []string) error {
	if len(relativePaths) == 0 {
		return nil
	}

	tmpFile, err := os.CreateTemp("", "arcup-delete-*.txt")
	if err != nil {
		return fmt.Errorf("create temp file for batch delete: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	content := strings.Join(relativePaths, "\n") + "\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write to temp file: %w", err)
	}
	tmpFile.Close()

	cmd := exec.Command("rclone", "delete", remotePath, "--files-from", tmpFile.Name())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rclone delete batch: %w: %s", err, output)
	}
	return nil
}

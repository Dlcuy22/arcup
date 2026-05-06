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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type RcloneUploader struct{}

func (r *RcloneUploader) Upload(localPath, remotePath string) error {
	cmd := exec.Command("rclone", "copy", localPath, remotePath, "--progress")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone copy: %w", err)
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

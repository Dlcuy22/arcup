// Module: internal/upload
// Purpose: Defines the Uploader interface for transferring files
// to and from remote storage backends.
//
// Key Components:
//   - Uploader: Interface for upload, download, list, and delete
//   - RemoteEntry: Represents a file on the remote
//   - New(): Factory that returns an uploader by backend name
//
// Example:
//
//	u, err := upload.New("rclone")
//	err = u.Upload("/tmp/backup.tar.zst", "s3:bucket/backups")
package upload

import "fmt"

type RemoteEntry struct {
	Path    string
	Name    string
	Size    int64
	ModTime string
	IsDir   bool
}

type Uploader interface {
	Upload(localPath, remotePath string) error
	Download(remotePath, localDir string) error
	List(remotePath string) ([]RemoteEntry, error)
	Delete(remotePath string) error
}

/*
New returns an Uploader implementation for the given backend name.

	params:
	      backend: upload backend name (currently only "rclone")
	returns:
	      Uploader: concrete implementation
	      error: if backend is not recognized
*/
func New(backend string) (Uploader, error) {
	switch backend {
	case "rclone":
		return &RcloneUploader{}, nil
	default:
		return nil, fmt.Errorf("unknown upload backend: %s", backend)
	}
}

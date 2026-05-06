// Module: internal/archive (gz)
// Purpose: Implements the Archiver interface using tar piped
// into gzip for compression and the reverse for extraction.
//
// Key Components:
//   - GzArchiver: Produces .tar.gz files
//
// Dependencies:
//   - os/exec: Shells out to tar and gzip binaries
package archive

import (
	"fmt"
	"os/exec"
)

type GzArchiver struct{}

func (g *GzArchiver) Archive(sources []string, dest string, extraArgs string) error {
	// tar -czf <dest> <sources...>
	// gzip doesn't support arbitrary extra args well via pipe,
	// so we use tar's built-in -z flag for simplicity
	tarArgs := append([]string{"-czf", dest}, sources...)
	cmd := exec.Command("tar", tarArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar+gz: %w: %s", err, output)
	}
	return nil
}

func (g *GzArchiver) Extract(archivePath string, dest string) error {
	cmd := exec.Command("tar", "-xzf", archivePath, "-C", dest)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar+gz extract: %w: %s", err, output)
	}
	return nil
}

func (g *GzArchiver) Extension() string {
	return "gz"
}

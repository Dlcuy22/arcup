// Module: internal/archive (zstd)
// Purpose: Implements the Archiver interface using tar piped
// into zstd for compression and the reverse for extraction.
//
// Key Components:
//   - ZstdArchiver: Produces .tar.zst files
//
// Dependencies:
//   - os/exec: Shells out to tar and zstd binaries
package archive

import (
	"fmt"
	"os/exec"
)

type ZstdArchiver struct{}

func (z *ZstdArchiver) Archive(sources []string, dest string, extraArgs string) error {
	compressCmd := "zstd"
	if extraArgs != "" {
		compressCmd = "zstd " + extraArgs
	}

	tarArgs := []string{"-I", compressCmd, "-cf", dest}
	tarArgs = append(tarArgs, sources...)

	cmd := exec.Command("tar", tarArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar+zstd: %w: %s", err, output)
	}
	return nil
}

func (z *ZstdArchiver) Extract(archivePath string, dest string) error {
	cmd := exec.Command("tar", "-I", "zstd -d", "-xf", archivePath, "-C", dest)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar+zstd extract: %w: %s", err, output)
	}
	return nil
}

func (z *ZstdArchiver) Extension() string {
	return "zst"
}

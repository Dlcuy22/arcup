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
	"strings"
)

type ZstdArchiver struct{}

func (z *ZstdArchiver) Archive(sources []string, dest string, extraArgs string) error {
	tarArgs := append([]string{"-cf", "-"}, sources...)
	tar := exec.Command("tar", tarArgs...)

	zstdArgs := []string{}
	if extraArgs != "" {
		zstdArgs = append(zstdArgs, strings.Fields(extraArgs)...)
	}
	zstdArgs = append(zstdArgs, "-o", dest)
	zstd := exec.Command("zstd", zstdArgs...)

	pipe, err := tar.StdoutPipe()
	if err != nil {
		return fmt.Errorf("tar stdout pipe: %w", err)
	}
	zstd.Stdin = pipe

	if err := tar.Start(); err != nil {
		return fmt.Errorf("tar start: %w", err)
	}
	if err := zstd.Run(); err != nil {
		return fmt.Errorf("zstd: %w", err)
	}
	return tar.Wait()
}

func (z *ZstdArchiver) Extract(archivePath string, dest string) error {
	zstd := exec.Command("zstd", "-d", archivePath, "--stdout")
	tar := exec.Command("tar", "-xf", "-", "-C", dest)

	pipe, err := zstd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("zstd stdout pipe: %w", err)
	}
	tar.Stdin = pipe

	if err := zstd.Start(); err != nil {
		return fmt.Errorf("zstd start: %w", err)
	}
	if err := tar.Run(); err != nil {
		return fmt.Errorf("tar extract: %w", err)
	}
	return zstd.Wait()
}

func (z *ZstdArchiver) Extension() string {
	return "zst"
}

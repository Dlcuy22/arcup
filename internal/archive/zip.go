// Module: internal/archive (zip)
// Purpose: Implements the Archiver interface by first creating
// a tar archive, then compressing it with zip.
//
// Key Components:
//   - ZipArchiver: Produces .tar.zip files
//
// Dependencies:
//   - os/exec: Shells out to tar and zip binaries
package archive

import (
	"fmt"
	"os"
	"os/exec"
)

type ZipArchiver struct{}

func (z *ZipArchiver) Archive(sources []string, dest string, extraArgs string) error {
	// Two-step: tar first, then zip the tarball
	tarDest := dest + ".tar"
	tarArgs := append([]string{"-cf", tarDest}, sources...)
	tarCmd := exec.Command("tar", tarArgs...)

	if output, err := tarCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tar: %w: %s", err, output)
	}

	zipCmd := exec.Command("zip", "-j", dest, tarDest)
	if output, err := zipCmd.CombinedOutput(); err != nil {
		os.Remove(tarDest)
		return fmt.Errorf("zip: %w: %s", err, output)
	}

	// Clean up intermediate tarball
	os.Remove(tarDest)
	return nil
}

func (z *ZipArchiver) Extract(archivePath string, dest string) error {
	// unzip then untar
	unzipCmd := exec.Command("unzip", "-o", archivePath, "-d", dest)
	if output, err := unzipCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("unzip: %w: %s", err, output)
	}

	// Find the .tar inside and extract it
	// Convention: the tar inside the zip has the same base name
	tarPath := archivePath + ".tar"
	tarCmd := exec.Command("tar", "-xf", tarPath, "-C", dest)
	if output, err := tarCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tar extract: %w: %s", err, output)
	}

	os.Remove(tarPath)
	return nil
}

func (z *ZipArchiver) Extension() string {
	return "zip"
}

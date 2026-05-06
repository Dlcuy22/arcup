// Module: internal/archive
// Purpose: Defines the Archiver interface and a factory function
// to instantiate the correct archiver based on the chosen algorithm.
//
// Key Components:
//   - Archiver: Interface for archive creation and extraction
//   - New(): Factory that returns a concrete archiver by algo name
//
// Error Types:
//   - Returns error when algo string doesn't match any known archiver
//
// Example:
//
//	a, err := archive.New("zstd")
//	err = a.Archive([]string{"/home/user/docs"}, "/tmp/backup.tar.zst", "")
package archive

import "fmt"

type Archiver interface {
	Archive(sources []string, dest string, extraArgs string) error
	Extract(archivePath string, dest string) error
	Extension() string
}

/*
New returns an Archiver implementation for the given algorithm name.

	params:
	      algo: one of "zstd", "gz", "zip"
	returns:
	      Archiver: concrete implementation
	      error: if algo is not recognized
*/
func New(algo string) (Archiver, error) {
	switch algo {
	case "zstd":
		return &ZstdArchiver{}, nil
	case "gz":
		return &GzArchiver{}, nil
	case "zip":
		return &ZipArchiver{}, nil
	default:
		return nil, fmt.Errorf("unknown algo: %s", algo)
	}
}

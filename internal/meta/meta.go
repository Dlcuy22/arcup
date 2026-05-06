// Module: internal/meta
// Purpose: Handles creation and parsing of sidecar .meta.json files
// that accompany each backup archive. These metadata files enable
// listing, verification, and restore without downloading archives.
//
// Key Components:
//   - Meta: Struct representing a backup's metadata
//   - Checksum: Sub-struct for hash algorithm and value
//   - Write(): Serializes and writes meta to disk
//   - ReadFromFile(): Parses a local .meta.json file
//   - ReadFromBytes(): Parses raw JSON bytes into Meta
//
// Example:
//
//	m := &meta.Meta{Name: "my-backup", ...}
//	err := m.Write("/tmp/arcup/my-backup.meta.json")
package meta

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type Meta struct {
	Name         string   `json:"name"`
	Hostname     string   `json:"hostname"`
	Timestamp    string   `json:"timestamp"`
	Sources      []string `json:"sources"`
	Algo         string   `json:"algo"`
	AlgoArgs     string   `json:"algo_args,omitempty"`
	Archive      string   `json:"archive"`
	SizeBytes    int64    `json:"size_bytes"`
	Checksum     Checksum `json:"checksum"`
	ArcupVersion string   `json:"arcup_version"`
}

type Checksum struct {
	Algo  string `json:"algo"`
	Value string `json:"value"`
}

/*
Write serializes the Meta struct to JSON and writes it to the given path.

	params:
	      path: destination file path for the .meta.json
	returns:
	      error: file creation or marshal failure
*/
func (m *Meta) Write(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

/*
ReadFromFile loads and parses a .meta.json file from disk.

	params:
	      path: path to the .meta.json file
	returns:
	      *Meta: parsed metadata
	      error: read or parse failure
*/
func ReadFromFile(path string) (*Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read meta: %w", err)
	}
	return ReadFromBytes(data)
}

/*
ReadFromBytes parses raw JSON bytes into a Meta struct.

	params:
	      data: JSON byte slice
	returns:
	      *Meta: parsed metadata
	      error: parse failure
*/
func ReadFromBytes(data []byte) (*Meta, error) {
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse meta: %w", err)
	}
	return &m, nil
}

/*
ComputeSHA256 calculates the SHA-256 checksum of a file.

	params:
	      path: file to hash
	returns:
	      string: hex-encoded SHA-256 hash
	      error: file open or read failure
*/
func ComputeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

/*
FormatTimestamp returns a filename-safe ISO timestamp string.

	returns:
	      string: timestamp in format "2006-01-02T15-04-05"
*/
func FormatTimestamp() string {
	return time.Now().Format("2006-01-02T15-04-05")
}

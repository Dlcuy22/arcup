// Package assets embeds default configuration files and assets.
//
// Key Components:
//   - DefaultConfig: Embedded bytes of default .arcup.yaml configuration
//
// Dependencies:
//   - embed: Go standard library embed package
package assets

import _ "embed"

// DefaultConfig contains the embedded default .arcup.yaml file contents.
//go:embed .arcup.yaml
var DefaultConfig []byte

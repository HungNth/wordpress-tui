// Package wptui exposes compile-time application metadata.
package wptui

import _ "embed"

// ManifestJSON holds the root application manifest embedded at compile time.
//
//go:embed manifest.json
var ManifestJSON []byte

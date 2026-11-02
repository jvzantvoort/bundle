// Package metadata provides types and functions for managing bundle metadata.
//
// Metadata includes human-readable information about a bundle such as title,
// author, creation timestamp, and the deterministic bundle checksum. All
// metadata is stored in .bundle/META.json in JSON format.
//
// Example usage:
//
//	// Create metadata
//	meta := &metadata.Metadata{
//	    Title:          "Vacation Photos",
//	    CreatedAt:      time.Now(),
//	    BundleChecksum: "abc123...",
//	    Author:         "username",
//	    Version:        metadata.CurrentVersion,
//	}
//
//	// Save to .bundle/META.json
//	err := meta.Save("/path/to/bundle")
//
//	// Load from .bundle/META.json
//	meta, err := metadata.Load("/path/to/bundle")
//
//	// Validate metadata
//	err = meta.Validate()
package metadata

import "time"

// Metadata schema versions. The version selects how BundleChecksum is computed.
const (
	// VersionLegacy bundles hash file checksums only, ignoring file names.
	VersionLegacy = 1

	// CurrentVersion bundles hash the manifest, covering names and contents.
	CurrentVersion = 2
)

// Metadata represents the bundle metadata stored in .bundle/META.json.
//
// It contains immutable information about the bundle that is set at creation
// time and should not be modified (except Title which can be updated).
//
// Fields:
//   - Title: human-readable name for the bundle (mutable)
//   - CreatedAt: ISO 8601 timestamp of bundle creation
//   - BundleChecksum: SHA256 of the file manifest (64 hex chars)
//   - Author: system username that created the bundle
//   - Version: metadata schema version (see CurrentVersion)
//
// Example JSON:
//
//	{
//	  "title": "Vacation Photos",
//	  "created_at": "2024-01-15T10:30:00Z",
//	  "bundle_checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
//	  "author": "username",
//	  "version": 2
//	}
type Metadata struct {
	Title          string    `json:"title"`            // Human-readable name
	CreatedAt      time.Time `json:"created_at"`       // ISO 8601 timestamp
	BundleChecksum string    `json:"bundle_checksum"`  // SHA256 of the file manifest
	Author         string    `json:"author"`           // System username
	Version        int       `json:"version"`          // Metadata schema version
	Ignore         []string  `json:"ignore,omitempty"` // Patterns of files that are not part of the bundle
}

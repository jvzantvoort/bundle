package metadata

import "time"

// Metadata represents the bundle metadata stored in .bundle/META.json
type Metadata struct {
	Title          string    `json:"title"`            // Human-readable name
	CreatedAt      time.Time `json:"created_at"`       // ISO 8601 timestamp
	BundleChecksum string    `json:"bundle_checksum"`  // SHA256 of sorted file checksums
	Author         string    `json:"author"`           // System username
	Version        int       `json:"version"`          // Metadata version (starts at 1)
}

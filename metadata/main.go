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
//	    Version:        1,
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

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// Load reads metadata from .bundle/META.json.
//
// It parses the JSON file and returns a Metadata struct. The file must
// exist and contain valid JSON matching the Metadata structure.
//
// Example:
//
//	meta, err := metadata.Load("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Title: %s\n", meta.Title)
//	fmt.Printf("Author: %s\n", meta.Author)
//	fmt.Printf("Created: %s\n", meta.CreatedAt.Format(time.RFC3339))
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - *Metadata: parsed metadata
//   - error: if file cannot be read or JSON is invalid
func Load(bundlePath string) (*Metadata, error) {
	metaFile := filepath.Join(bundlePath, ".bundle", "META.json")
	data, err := os.ReadFile(metaFile)
	if err != nil {
		return nil, err
	}

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// Save writes metadata to .bundle/META.json.
//
// It serializes the metadata to JSON with indentation for readability and
// writes it to .bundle/META.json. The file is created with 0644 permissions.
//
// Example:
//
//	meta := &metadata.Metadata{
//	    Title:          "My Bundle",
//	    CreatedAt:      time.Now(),
//	    BundleChecksum: "abc123...",
//	    Author:         "username",
//	    Version:        1,
//	}
//	err := meta.Save("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - error: if file cannot be created, written, or JSON cannot be serialized
func (m *Metadata) Save(bundlePath string) error {
	metaFile := filepath.Join(bundlePath, ".bundle", "META.json")

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metaFile, data, 0644)
}

// Validate checks metadata fields against validation rules.
//
// It validates:
//   - BundleChecksum is exactly 64 lowercase hex characters
//   - Version is >= 1
//   - Author is not empty
//   - CreatedAt is not zero
//
// Example:
//
//	meta := &metadata.Metadata{
//	    Title:          "My Bundle",
//	    CreatedAt:      time.Now(),
//	    BundleChecksum: "abc123...",
//	    Author:         "username",
//	    Version:        1,
//	}
//	if err := meta.Validate(); err != nil {
//	    log.Fatal("Invalid metadata:", err)
//	}
//
// Returns:
//   - error: describing the first validation failure, or nil if valid
func (m *Metadata) Validate() error {
	// Check bundle checksum format (64 hex characters)
	if len(m.BundleChecksum) != 64 {
		return fmt.Errorf("invalid bundle checksum length: %d, want 64", len(m.BundleChecksum))
	}

	hexPattern := regexp.MustCompile("^[a-f0-9]{64}$")
	if !hexPattern.MatchString(m.BundleChecksum) {
		return fmt.Errorf("invalid bundle checksum format: must be 64 hex characters")
	}

	// Check version
	if m.Version < 1 {
		return fmt.Errorf("invalid version: %d, must be >= 1", m.Version)
	}

	// Check required fields
	if m.Author == "" {
		return fmt.Errorf("author cannot be empty")
	}

	if m.CreatedAt.IsZero() {
		return fmt.Errorf("created_at cannot be zero")
	}

	return nil
}

// UpdateTitle updates the title field and saves the metadata.
//
// This is a convenience function that loads the metadata, updates the title,
// and saves it back to disk in a single operation.
//
// Example:
//
//	err := metadata.UpdateTitle("/path/to/bundle", "New Title")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//   - newTitle: new title to set
//
// Returns:
//   - error: if metadata cannot be loaded or saved
func UpdateTitle(bundlePath string, newTitle string) error {
	// Load existing metadata
	meta, err := Load(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	// Update title
	meta.Title = newTitle

	// Save back to disk
	if err := meta.Save(bundlePath); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	return nil
}

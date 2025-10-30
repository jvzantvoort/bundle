package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// Load reads metadata from .bundle/META.json
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

// Save writes metadata to .bundle/META.json
func (m *Metadata) Save(bundlePath string) error {
	metaFile := filepath.Join(bundlePath, ".bundle", "META.json")

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metaFile, data, 0644)
}

// Validate checks metadata fields against validation rules
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

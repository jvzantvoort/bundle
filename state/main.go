// Package state provides types and functions for managing bundle operational state.
//
// State includes mutable operational information such as verification status,
// last check timestamp, known replicas, and total size. State is stored in
// .bundle/STATE.json and is updated as the bundle is verified or replicated.
//
// Example usage:
//
//	// Load state
//	st, err := state.Load("/path/to/bundle")
//
//	// Update verification status
//	st.MarkVerified(true, time.Now())
//
//	// Add replica location
//	st.AddReplica("s3://bucket/path")
//
//	// Update size
//	st.UpdateSize(1024000)
//
//	// Save changes
//	err = st.Save("/path/to/bundle")
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// State represents the bundle operational state stored in .bundle/STATE.json.
//
// Unlike metadata, state is mutable and updated during bundle operations such
// as verification and replication. It tracks the current operational status
// of the bundle.
//
// Fields:
//   - Verified: true if last integrity check passed
//   - LastChecked: timestamp of last verification
//   - Replicas: URIs of known bundle replicas
//   - SizeBytes: total size of all files (excluding .bundle/)
//
// Example JSON:
//
//	{
//	  "verified": true,
//	  "last_checked": "2024-01-15T10:30:00Z",
//	  "replicas": ["s3://bucket/path", "/mnt/backup/bundle"],
//	  "size_bytes": 1024000
//	}
type State struct {
	Verified    bool      `json:"verified"`     // Last verification result
	LastChecked time.Time `json:"last_checked"` // Last verification timestamp
	Replicas    []string  `json:"replicas"`     // Known replica locations
	SizeBytes   int64     `json:"size_bytes"`   // Total bundle size (excluding .bundle/)
}

// Load reads state from .bundle/STATE.json.
//
// It parses the JSON file and returns a State struct. The file must exist
// and contain valid JSON matching the State structure.
//
// Example:
//
//	st, err := state.Load("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Verified: %v\n", st.Verified)
//	fmt.Printf("Last checked: %s\n", st.LastChecked.Format(time.RFC3339))
//	fmt.Printf("Size: %d bytes\n", st.SizeBytes)
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - *State: parsed state
//   - error: if file cannot be read or JSON is invalid
func Load(bundlePath string) (*State, error) {
	stateFile := filepath.Join(bundlePath, ".bundle", "STATE.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// Save writes state to .bundle/STATE.json.
//
// It serializes the state to JSON with indentation for readability and
// writes it to .bundle/STATE.json. The file is created with 0644 permissions.
//
// Example:
//
//	st := &state.State{
//	    Verified:    true,
//	    LastChecked: time.Now(),
//	    Replicas:    []string{"s3://bucket/path"},
//	    SizeBytes:   1024000,
//	}
//	err := st.Save("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - error: if file cannot be created, written, or JSON cannot be serialized
func (s *State) Save(bundlePath string) error {
	stateFile := filepath.Join(bundlePath, ".bundle", "STATE.json")

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0644)
}

// MarkVerified updates verification status and timestamp.
//
// It sets the Verified field and updates LastChecked to the provided timestamp.
// Call Save() to persist the changes to disk.
//
// Example:
//
//	st, _ := state.Load("/path/to/bundle")
//	st.MarkVerified(true, time.Now())
//	st.Save("/path/to/bundle")
//
// Parameters:
//   - verified: true if integrity check passed, false otherwise
//   - timestamp: time of the verification check
func (s *State) MarkVerified(verified bool, timestamp time.Time) {
	s.Verified = verified
	s.LastChecked = timestamp
}

// UpdateSize sets the total bundle size.
//
// The size should be the sum of all file sizes in the bundle, excluding
// the .bundle/ directory. Call Save() to persist the changes to disk.
//
// Example:
//
//	st, _ := state.Load("/path/to/bundle")
//	st.UpdateSize(2048000)
//	st.Save("/path/to/bundle")
//
// Parameters:
//   - bytes: total size in bytes
func (s *State) UpdateSize(bytes int64) {
	s.SizeBytes = bytes
}

// AddReplica appends a replica location if not already present.
//
// Replica URIs can be any location identifier (file paths, S3 URIs, etc.).
// Duplicates are automatically ignored. Call Save() to persist the changes.
//
// Example:
//
//	st, _ := state.Load("/path/to/bundle")
//	st.AddReplica("s3://bucket/backup/bundle")
//	st.AddReplica("/mnt/backup/bundle")
//	st.AddReplica("s3://bucket/backup/bundle")  // Ignored (duplicate)
//	st.Save("/path/to/bundle")
//	// st.Replicas = ["s3://bucket/backup/bundle", "/mnt/backup/bundle"]
//
// Parameters:
//   - uri: location identifier for the replica
func (s *State) AddReplica(uri string) {
	for _, existing := range s.Replicas {
		if existing == uri {
			return
		}
	}
	s.Replicas = append(s.Replicas, uri)
}

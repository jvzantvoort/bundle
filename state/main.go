package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// State represents the bundle operational state stored in .bundle/STATE.json
type State struct {
	Verified    bool      `json:"verified"`      // Last verification result
	LastChecked time.Time `json:"last_checked"`  // Last verification timestamp
	Replicas    []string  `json:"replicas"`      // Known replica locations
	SizeBytes   int64     `json:"size_bytes"`    // Total bundle size (excluding .bundle/)
}

// Load reads state from .bundle/STATE.json
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

// Save writes state to .bundle/STATE.json
func (s *State) Save(bundlePath string) error {
	stateFile := filepath.Join(bundlePath, ".bundle", "STATE.json")

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0644)
}

// MarkVerified updates verification status and timestamp
func (s *State) MarkVerified(verified bool, timestamp time.Time) {
	s.Verified = verified
	s.LastChecked = timestamp
}

// UpdateSize sets the total bundle size
func (s *State) UpdateSize(bytes int64) {
	s.SizeBytes = bytes
}

// AddReplica appends a replica location if not already present
func (s *State) AddReplica(uri string) {
	for _, existing := range s.Replicas {
		if existing == uri {
			return
		}
	}
	s.Replicas = append(s.Replicas, uri)
}

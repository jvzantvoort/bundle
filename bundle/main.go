package bundle

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/jvzantvoort/bundle/checksum"
	"github.com/jvzantvoort/bundle/lock"
	"github.com/jvzantvoort/bundle/metadata"
	"github.com/jvzantvoort/bundle/state"
	"github.com/jvzantvoort/bundle/tag"
	log "github.com/sirupsen/logrus"
)

// Bundle represents a complete bundle with all metadata and state
type Bundle struct {
	Path     string                 // Absolute path to bundle directory
	Metadata *metadata.Metadata     // Loaded from META.json
	State    *state.State           // Loaded from STATE.json
	Tags     *tag.Tags              // Loaded from TAGS.txt
	Files    *checksum.ChecksumFile // Loaded from SHA256SUM.txt
}

// Create initializes a new bundle from a directory
func Create(path string, title string) (*Bundle, error) {
	log.Debugf("Creating bundle at path: %s with title: %s", path, title)
	defer log.Debugf("Bundle creation completed for path: %s", path)
	
	// Acquire lock
	bundleLock, err := lock.AcquireLock(path)
	if err != nil {
		return nil, err
	}
	defer bundleLock.Release()

	// Create .bundle directory
	bundleDir := filepath.Join(path, ".bundle")
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		return nil, err
	}

	// Scan and compute checksums
	files := &checksum.ChecksumFile{}
	if err := files.Compute(path); err != nil {
		return nil, err
	}

	// Compute bundle checksum
	var checksums []string
	for _, record := range files.Records {
		checksums = append(checksums, record.Checksum)
	}
	bundleChecksum := checksum.ComputeBundleChecksum(checksums)

	// Get author from system user
	currentUser, _ := user.Current()
	author := "unknown"
	if currentUser != nil {
		author = currentUser.Username
	}

	// Create metadata
	meta := &metadata.Metadata{
		Title:          title,
		CreatedAt:      time.Now(),
		BundleChecksum: bundleChecksum,
		Author:         author,
		Version:        1,
	}

	// Calculate total size
	var totalSize int64
	for _, record := range files.Records {
		filePath := filepath.Join(path, record.FilePath)
		info, err := os.Stat(filePath)
		if err == nil {
			totalSize += info.Size()
		}
	}

	// Create state
	bundleState := &state.State{
		Verified:    true,
		LastChecked: time.Now(),
		Replicas:    []string{},
		SizeBytes:   totalSize,
	}

	// Create empty tags
	bundleTags := &tag.Tags{Tags: []string{}}

	// Save all metadata
	if err := meta.Save(path); err != nil {
		return nil, err
	}
	if err := files.Save(path); err != nil {
		return nil, err
	}
	if err := bundleState.Save(path); err != nil {
		return nil, err
	}
	if err := bundleTags.Save(path); err != nil {
		return nil, err
	}

	return &Bundle{
		Path:     path,
		Metadata: meta,
		State:    bundleState,
		Tags:     bundleTags,
		Files:    files,
	}, nil
}

// Verify checks bundle integrity by recomputing checksums
func Verify(path string) (bool, []string, error) {
	// Load checksums
	files := &checksum.ChecksumFile{}
	if err := files.Load(path); err != nil {
		return false, nil, err
	}

	// Verify
	corrupted, err := files.Verify(path)
	if err != nil {
		return false, nil, err
	}

	// Update state
	bundleState, err := state.Load(path)
	if err != nil {
		// If state doesn't exist, create it
		bundleState = &state.State{}
	}

	verified := len(corrupted) == 0
	bundleState.MarkVerified(verified, time.Now())
	bundleState.Save(path)

	return verified, corrupted, nil
}

// Load reads all bundle metadata
func Load(path string) (*Bundle, error) {
	// Check if .bundle exists
	bundleDir := filepath.Join(path, ".bundle")
	if _, err := os.Stat(bundleDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory is not a bundle (missing .bundle/)")
	}

	// Load all components
	meta, err := metadata.Load(path)
	if err != nil {
		return nil, err
	}

	bundleState, err := state.Load(path)
	if err != nil {
		return nil, err
	}

	bundleTags, err := tag.Load(path)
	if err != nil {
		return nil, err
	}

	files := &checksum.ChecksumFile{}
	if err := files.Load(path); err != nil {
		return nil, err
	}

	return &Bundle{
		Path:     path,
		Metadata: meta,
		State:    bundleState,
		Tags:     bundleTags,
		Files:    files,
	}, nil
}

// Package bundle provides high-level operations for managing content-addressable
// file bundles with SHA256-based integrity verification.
//
// A bundle is a directory containing files with associated metadata stored in a
// .bundle/ subdirectory. Each bundle is uniquely identified by the SHA256 checksum
// of its contents, ensuring data integrity and enabling deduplication.
//
// Example usage:
//
//	// Create a new bundle
//	b, err := bundle.Create("/path/to/files", "My Photos")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Created bundle: %s\n", b.Metadata.BundleChecksum)
//
//	// Load an existing bundle
//	b, err = bundle.Load("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Verify bundle integrity
//	verified, corrupted, err := bundle.Verify("/path/to/bundle")
//	if !verified {
//	    fmt.Printf("Corrupted files: %v\n", corrupted)
//	}
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

// Bundle represents a complete bundle with all metadata and state.
//
// A Bundle contains:
//   - Path: absolute path to the bundle directory
//   - Metadata: title, author, creation time, bundle checksum
//   - State: verification status, replicas, size
//   - Tags: searchable labels
//   - Files: checksum records for all files
//
// Example:
//
//	b, _ := bundle.Load("/path/to/bundle")
//	fmt.Printf("Bundle: %s\n", b.Metadata.Title)
//	fmt.Printf("Files: %d\n", len(b.Files.Records))
//	fmt.Printf("Checksum: %s\n", b.Metadata.BundleChecksum)
type Bundle struct {
	Path     string                 // Absolute path to bundle directory
	Metadata *metadata.Metadata     // Loaded from META.json
	State    *state.State           // Loaded from STATE.json
	Tags     *tag.Tags              // Loaded from TAGS.txt
	Files    *checksum.ChecksumFile // Loaded from SHA256SUM.txt
}

// Create initializes a new bundle from a directory.
//
// It scans all files in the directory (excluding .bundle/), computes SHA256
// checksums, generates a deterministic bundle checksum, and creates metadata
// files in the .bundle/ subdirectory.
//
// The function acquires an exclusive lock during creation to prevent concurrent
// modifications. If another process holds a lock, Create returns an error.
//
// Example:
//
//	bundle, err := bundle.Create("/path/to/photos", "Vacation 2024")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Created bundle with %d files\n", len(bundle.Files.Records))
//	fmt.Printf("Bundle checksum: %s\n", bundle.Metadata.BundleChecksum)
//
// Parameters:
//   - path: absolute or relative path to the directory to bundle
//   - title: human-readable bundle title
//
// Returns:
//   - *Bundle: the created bundle with all metadata loaded
//   - error: lock errors, I/O errors, or checksum computation errors
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
		return nil, fmt.Errorf("failed to compute checksums: %w", err)
	}

	// Compute bundle checksum - pre-allocate slice for better performance
	checksums := make([]string, len(files.Records))
	for i, record := range files.Records {
		checksums[i] = record.Checksum
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

	// Create state with size already computed during checksum scan
	bundleState := &state.State{
		Verified:    true,
		LastChecked: time.Now(),
		Replicas:    []string{},
		SizeBytes:   files.TotalSize,
	}

	// Create empty tags
	bundleTags := &tag.Tags{Tags: []string{}}

	// Save all metadata
	if err := meta.Save(path); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}
	if err := files.Save(path); err != nil {
		return nil, fmt.Errorf("failed to save checksums: %w", err)
	}
	if err := bundleState.Save(path); err != nil {
		return nil, fmt.Errorf("failed to save state: %w", err)
	}
	if err := bundleTags.Save(path); err != nil {
		return nil, fmt.Errorf("failed to save tags: %w", err)
	}

	return &Bundle{
		Path:     path,
		Metadata: meta,
		State:    bundleState,
		Tags:     bundleTags,
		Files:    files,
	}, nil
}

// Verify checks bundle integrity by recomputing checksums.
//
// It recomputes SHA256 checksums for all files and compares them against the
// stored checksums in .bundle/SHA256SUM.txt. Updates the bundle state with
// verification results and timestamp.
//
// Example:
//
//	verified, corrupted, err := bundle.Verify("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if !verified {
//	    fmt.Printf("Corrupted files: %v\n", corrupted)
//	} else {
//	    fmt.Println("Bundle integrity verified")
//	}
//
// Parameters:
//   - path: absolute or relative path to the bundle directory
//
// Returns:
//   - bool: true if all checksums match, false if any files are corrupted
//   - []string: list of relative paths to corrupted or missing files
//   - error: I/O errors or missing bundle metadata
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

// Load reads all bundle metadata from disk.
//
// It loads metadata, state, tags, and checksums from the .bundle/ directory.
// Returns an error if the directory is not a bundle (missing .bundle/) or if
// any required metadata files cannot be read.
//
// Example:
//
//	bundle, err := bundle.Load("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Title: %s\n", bundle.Metadata.Title)
//	fmt.Printf("Author: %s\n", bundle.Metadata.Author)
//	fmt.Printf("Files: %d\n", len(bundle.Files.Records))
//	fmt.Printf("Tags: %v\n", bundle.Tags.List())
//
// Parameters:
//   - path: absolute or relative path to the bundle directory
//
// Returns:
//   - *Bundle: the loaded bundle with all metadata
//   - error: if path is not a bundle or metadata files cannot be read
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

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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/jvzantvoort/bundle/checksum"
	"github.com/jvzantvoort/bundle/lock"
	"github.com/jvzantvoort/bundle/metadata"
	"github.com/jvzantvoort/bundle/state"
	"github.com/jvzantvoort/bundle/tag"
	"github.com/jvzantvoort/bundle/utils"
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
// The directory must already exist and must not already be a bundle, so an
// existing title and tags are never overwritten.
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
//   - error: lock errors, I/O errors, checksum computation errors,
//     utils.ErrInvalidPath if path is not a directory, or
//     utils.ErrBundleExists if path is already a bundle
func Create(path string, title string, ignore ...string) (*Bundle, error) {
	log.Debugf("Creating bundle at path: %s with title: %s", path, title)
	defer log.Debugf("Bundle creation completed for path: %s", path)

	// Fail before anything is written
	ignore, err := checksum.NormalizeIgnore(ignore)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", utils.ErrInvalidPath, err)
	}

	// The directory must exist: creating it would turn a mistyped path into an
	// empty bundle
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: not a directory: %s", utils.ErrInvalidPath, path)
	}

	// Acquire lock
	bundleLock, err := lock.AcquireLock(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := bundleLock.Release(); err != nil {
			log.Errorf("failed to release lock: %v", err)
		}
	}()

	// Refuse to overwrite an existing bundle
	if _, err := os.Stat(filepath.Join(path, ".bundle", "META.json")); err == nil {
		return nil, fmt.Errorf("%w: %s", utils.ErrBundleExists, path)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// Create .bundle directory
	bundleDir := filepath.Join(path, ".bundle")
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		return nil, err
	}

	// Scan and compute checksums
	files := &checksum.ChecksumFile{}
	if err := files.Compute(path, ignore...); err != nil {
		return nil, fmt.Errorf("failed to compute checksums: %w", err)
	}

	// Compute bundle checksum
	bundleChecksum := checksum.ComputeBundleChecksum(files.Records)

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
		Version:        metadata.CurrentVersion,
		Ignore:         ignore,
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

	// Save all metadata; META.json goes last, because its presence is what marks
	// the bundle as complete
	if err := files.Save(path); err != nil {
		return nil, fmt.Errorf("failed to save checksums: %w", err)
	}
	if err := bundleState.Save(path); err != nil {
		return nil, fmt.Errorf("failed to save state: %w", err)
	}
	if err := bundleTags.Save(path); err != nil {
		return nil, fmt.Errorf("failed to save tags: %w", err)
	}
	if err := meta.Save(path); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
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
// stored checksums in .bundle/SHA256SUM.txt, reports files the manifest does not
// list, and checks the manifest against the bundle checksum in META.json. Updates
// the bundle state with verification results and timestamp.
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
//   - bool: true if the bundle matches its manifest and bundle checksum
//   - []string: relative paths of corrupted, missing or unlisted files;
//     .bundle/SHA256SUM.txt is listed if it does not match the bundle checksum
//   - error: utils.ErrNotABundle, utils.ErrIncompleteBundle,
//     utils.ErrCorruptedBundle for unreadable metadata, or an I/O error
func Verify(path string, ignore ...string) (bool, []string, error) {
	// Check if .bundle exists
	if _, err := os.Stat(filepath.Join(path, ".bundle")); os.IsNotExist(err) {
		return false, nil, utils.ErrNotABundle
	}

	// Load metadata
	meta, err := metadata.Load(path)
	if err != nil {
		return false, nil, classifyLoadError(err)
	}
	if err := meta.Validate(); err != nil {
		return false, nil, fmt.Errorf("%w: META.json: %v", utils.ErrCorruptedBundle, err)
	}

	// Load checksums
	files := &checksum.ChecksumFile{}
	if err := files.Load(path); err != nil {
		return false, nil, classifyLoadError(err)
	}

	// Verify
	corrupted, err := files.Verify(path, append(meta.Ignore, ignore...)...)
	if err != nil {
		return false, nil, err
	}
	if expectedChecksum(meta, files) != meta.BundleChecksum {
		corrupted = append(corrupted, ".bundle/SHA256SUM.txt")
	}

	// Update state
	bundleState, err := state.Load(path)
	if err != nil {
		// If state doesn't exist, create it
		bundleState = &state.State{}
	}

	verified := len(corrupted) == 0
	bundleState.MarkVerified(verified, time.Now())
	if err := bundleState.Save(path); err != nil {
		log.Warnf("failed to save verification state: %v", err)
	}

	return verified, corrupted, nil
}

// Migrate upgrades a version 1 bundle to the current checksum algorithm.
//
// The files are verified against the existing manifest first, so a corrupted
// bundle is never given a new, valid looking checksum. Title, author, creation
// time and tags are preserved. Migrating an up-to-date bundle changes nothing.
//
// Version 1 skipped every file whose path contained ".bundle". Those files are
// added to the manifest; any other unlisted file makes the migration fail.
//
// Parameters:
//   - path: absolute or relative path to the bundle directory
//
// Returns:
//   - string: the bundle checksum after migration
//   - bool: true if the bundle was changed
//   - error: lock or I/O errors, or utils.ErrCorruptedBundle if verification fails
func Migrate(path string) (string, bool, error) {
	if _, err := os.Stat(filepath.Join(path, ".bundle")); os.IsNotExist(err) {
		return "", false, utils.ErrNotABundle
	}

	bundleLock, err := lock.AcquireLock(path)
	if err != nil {
		return "", false, err
	}
	defer func() {
		if err := bundleLock.Release(); err != nil {
			log.Errorf("failed to release lock: %v", err)
		}
	}()

	meta, err := metadata.Load(path)
	if err != nil {
		return "", false, classifyLoadError(err)
	}
	if err := meta.Validate(); err != nil {
		return "", false, fmt.Errorf("%w: META.json: %v", utils.ErrCorruptedBundle, err)
	}
	if meta.Version >= metadata.CurrentVersion {
		return meta.BundleChecksum, false, nil
	}

	files := &checksum.ChecksumFile{}
	if err := files.Load(path); err != nil {
		return "", false, classifyLoadError(err)
	}

	// Only migrate a bundle that is intact under its current version
	if expectedChecksum(meta, files) != meta.BundleChecksum {
		return "", false, fmt.Errorf("%w: SHA256SUM.txt does not match the bundle checksum", utils.ErrCorruptedBundle)
	}
	corrupted, err := files.VerifyRecords(path)
	if err != nil {
		return "", false, err
	}
	if len(corrupted) > 0 {
		return "", false, fmt.Errorf("%w: %s", utils.ErrCorruptedBundle, strings.Join(corrupted, ", "))
	}

	// Pick up files that version 1 wrongly skipped
	untracked, err := files.Untracked(path)
	if err != nil {
		return "", false, err
	}
	var addedSize int64
	for _, relPath := range untracked {
		filePath := filepath.Join(path, filepath.FromSlash(relPath))
		if !strings.Contains(filePath, ".bundle") {
			return "", false, fmt.Errorf("%w: unlisted file: %s", utils.ErrCorruptedBundle, relPath)
		}
		sum, err := checksum.ComputeFileSHA256(filePath)
		if err != nil {
			return "", false, err
		}
		info, err := os.Stat(filePath)
		if err != nil {
			return "", false, err
		}
		files.Records = append(files.Records, checksum.ChecksumRecord{Checksum: sum, FilePath: relPath})
		addedSize += info.Size()
		log.Infof("Added file skipped by version 1: %s", relPath)
	}

	if addedSize > 0 {
		bundleState, err := state.Load(path)
		if err != nil {
			return "", false, classifyLoadError(err)
		}
		bundleState.UpdateSize(bundleState.SizeBytes + addedSize)
		if err := bundleState.Save(path); err != nil {
			return "", false, fmt.Errorf("failed to save state: %w", err)
		}
	}

	meta.Version = metadata.CurrentVersion
	meta.BundleChecksum = checksum.ComputeBundleChecksum(files.Records)
	if err := files.Save(path); err != nil {
		return "", false, fmt.Errorf("failed to save checksums: %w", err)
	}
	if err := meta.Save(path); err != nil {
		return "", false, fmt.Errorf("failed to save metadata: %w", err)
	}

	return meta.BundleChecksum, true, nil
}

// expectedChecksum computes the bundle checksum for the metadata's version.
func expectedChecksum(meta *metadata.Metadata, files *checksum.ChecksumFile) string {
	if meta.Version == metadata.VersionLegacy {
		checksums := make([]string, len(files.Records))
		for i, record := range files.Records {
			checksums[i] = record.Checksum
		}
		return checksum.ComputeLegacyBundleChecksum(checksums)
	}
	return checksum.ComputeBundleChecksum(files.Records)
}

// classifyLoadError maps a metadata load failure to the matching bundle error.
// I/O errors such as permission denied are returned unchanged.
func classifyLoadError(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %v", utils.ErrIncompleteBundle, err)
	}
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return err
	}
	return fmt.Errorf("%w: %v", utils.ErrCorruptedBundle, err)
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
		return nil, utils.ErrNotABundle
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

// Package utils provides utility functions for CLI operations, error handling,
// and output formatting.
//
// It includes helpers for:
//   - JSON and table output formatting
//   - Error message formatting
//   - Exit code mapping
//   - File path operations
//   - Bundle directory detection
//
// Example usage:
//
//	// Check if directory is a bundle
//	if utils.IsBundleDir("/path/to/dir") {
//	    fmt.Println("Is a bundle")
//	}
//
//	// Get bundle metadata directory
//	metaDir := utils.GetBundleMetadataDir("/path/to/bundle")
//	// metaDir = "/path/to/bundle/.bundle"
//
//	// Normalize path
//	absPath, err := utils.NormalizePath("../relative/path")
package utils

import (
	"os"
	"path/filepath"
	"strings"
)

const bundleMetadataDir = ".bundle"

// IsBundleDir checks if a path contains a .bundle subdirectory.
//
// It verifies that the .bundle/ directory exists, indicating the path is
// a valid bundle directory.
//
// Example:
//
//	if utils.IsBundleDir("/path/to/dir") {
//	    fmt.Println("Directory is a bundle")
//	} else {
//	    fmt.Println("Not a bundle")
//	}
//
// Parameters:
//   - path: absolute or relative path to check
//
// Returns:
//   - bool: true if .bundle/ exists, false otherwise
func IsBundleDir(path string) bool {
	bundlePath := filepath.Join(path, bundleMetadataDir)
	_, err := os.Stat(bundlePath)
	return err == nil
}

// GetBundleMetadataDir returns the absolute path to the .bundle subdirectory.
//
// It constructs the path to the .bundle/ directory but does not verify
// its existence.
//
// Example:
//
//	metaDir := utils.GetBundleMetadataDir("/path/to/bundle")
//	// metaDir = "/path/to/bundle/.bundle"
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - string: path to .bundle/ subdirectory
func GetBundleMetadataDir(bundlePath string) string {
	return filepath.Join(bundlePath, bundleMetadataDir)
}

// ShouldExclude checks if a path should be excluded from bundle operations.
//
// It returns true for any path containing ".bundle", which excludes the
// .bundle/ metadata directory itself.
//
// Example:
//
//	if utils.ShouldExclude("/path/to/bundle/.bundle/META.json") {
//	    fmt.Println("Excluded")  // true
//	}
//
//	if utils.ShouldExclude("/path/to/bundle/file.txt") {
//	    fmt.Println("Excluded")  // false
//	}
//
// Parameters:
//   - path: file path to check
//
// Returns:
//   - bool: true if path contains .bundle, false otherwise
func ShouldExclude(path string) bool {
	return strings.Contains(path, bundleMetadataDir)
}

// NormalizePath cleans and returns the absolute path.
//
// It converts relative paths to absolute and cleans them (removes . and ..
// elements, multiple slashes, etc.).
//
// Example:
//
//	absPath, err := utils.NormalizePath("../photos")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(absPath)  // /home/user/photos (cleaned absolute path)
//
// Parameters:
//   - path: relative or absolute path to normalize
//
// Returns:
//   - string: cleaned absolute path
//   - error: if absolute path cannot be determined
func NormalizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

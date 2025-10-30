package utils

import (
	"os"
	"path/filepath"
	"strings"
)

const bundleMetadataDir = ".bundle"

// IsBundleDir checks if a path contains a .bundle subdirectory
func IsBundleDir(path string) bool {
	bundlePath := filepath.Join(path, bundleMetadataDir)
	_, err := os.Stat(bundlePath)
	return err == nil
}

// GetBundleMetadataDir returns the absolute path to the .bundle subdirectory
func GetBundleMetadataDir(bundlePath string) string {
	return filepath.Join(bundlePath, bundleMetadataDir)
}

// ShouldExclude checks if a path should be excluded from bundle operations
// Excludes the .bundle/ directory itself
func ShouldExclude(path string) bool {
	return strings.Contains(path, bundleMetadataDir)
}

// NormalizePath cleans and returns the absolute path
func NormalizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

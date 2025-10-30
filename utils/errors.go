// Package utils provides utility functions for CLI operations, error handling,
// and output formatting.
//
// Standard error types for bundle operations. These errors map to CLI exit codes:
//   - User errors (exit code 1): invalid usage, missing resources, corrupted bundle
//   - System errors (exit code 2): I/O failures, permissions, system resources
//
// Example usage:
//
//	if !utils.IsBundleDir(path) {
//	    return utils.ErrNotABundle
//	}
//
//	// In CLI:
//	if err != nil {
//	    code := utils.ExitCodeFromError(err)
//	    os.Exit(code)
//	}
package utils

import "errors"

// User errors (exit code 1) - invalid usage, missing resources
var (
	// ErrNotABundle indicates the directory is not a valid bundle (missing .bundle/ subdirectory)
	ErrNotABundle = errors.New("directory is not a bundle (missing .bundle/)")

	// ErrInvalidPath indicates an invalid or non-existent path was provided
	ErrInvalidPath = errors.New("invalid path provided")

	// ErrBundleLocked indicates another process holds a lock on the bundle
	ErrBundleLocked = errors.New("bundle is locked by another process")

	// ErrCorruptedBundle indicates bundle integrity check failed
	ErrCorruptedBundle = errors.New("bundle integrity check failed")

	// ErrIncompleteBundle indicates bundle is missing required metadata files
	ErrIncompleteBundle = errors.New("bundle is incomplete (missing required files)")
)

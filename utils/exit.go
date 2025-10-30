// Package utils provides utility functions for CLI operations, error handling,
// and output formatting.
//
// Exit code mapping for consistent CLI behavior:
//   - 0: Success
//   - 1: User error (invalid usage, missing resources, corrupted bundle)
//   - 2: System error (I/O failure, permissions, system resources)
//
// Example usage:
//
//	err := bundle.Create(path, title)
//	code := utils.ExitCodeFromError(err)
//	os.Exit(code)
package utils

import (
	"errors"
	"os"
)

// ExitCodeFromError maps errors to CLI exit codes following the constitution.
//
// It examines the error type and returns the appropriate exit code:
//   - 0: Success (err == nil)
//   - 1: User error (invalid usage, missing resources, corrupted bundle)
//   - 2: System error (I/O failure, permissions, system resources)
//
// Example:
//
//	err := bundle.Verify(path)
//	if err != nil {
//	    code := utils.ExitCodeFromError(err)
//	    if code == 1 {
//	        fmt.Println("User error:", err)
//	    } else {
//	        fmt.Println("System error:", err)
//	    }
//	    os.Exit(code)
//	}
//
// Parameters:
//   - err: error to map to exit code
//
// Returns:
//   - int: 0 for success, 1 for user errors, 2 for system errors
func ExitCodeFromError(err error) int {
	if err == nil {
		return 0
	}

	// User errors (exit code 1)
	if errors.Is(err, ErrNotABundle) ||
		errors.Is(err, ErrInvalidPath) ||
		errors.Is(err, ErrBundleLocked) ||
		errors.Is(err, ErrCorruptedBundle) ||
		errors.Is(err, ErrIncompleteBundle) {
		return 1
	}

	// System errors (exit code 2)
	if errors.Is(err, os.ErrPermission) ||
		errors.Is(err, os.ErrNotExist) ||
		errors.Is(err, os.ErrExist) ||
		errors.Is(err, os.ErrClosed) ||
		errors.Is(err, os.ErrNoDeadline) {
		return 2
	}

	// Default to system error for unknown errors
	return 2
}

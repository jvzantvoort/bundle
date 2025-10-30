package utils

import (
	"errors"
	"os"
)

// ExitCodeFromError maps errors to CLI exit codes following the constitution:
// - 0: Success
// - 1: User error (invalid usage, missing resources, corrupted bundle)
// - 2: System error (I/O failure, permissions, system resources)
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

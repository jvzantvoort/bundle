// Package lock provides file-based locking for concurrent bundle operations.
//
// It implements exclusive locking to prevent multiple processes from modifying
// a bundle simultaneously. Locks are fail-fast and atomic using OS-level file
// creation primitives.
//
// Example usage:
//
//	// Acquire exclusive lock
//	lock, err := lock.AcquireLock("/path/to/bundle")
//	if err != nil {
//	    log.Fatal("Bundle is locked:", err)
//	}
//	defer lock.Release()
//
//	// Perform write operations...
//	bundle.Create("/path/to/bundle", "My Bundle")
package lock

import (
	"fmt"
	"os"
	"path/filepath"
)

// Lock represents a bundle lock.
//
// It holds a reference to the lock file and path for cleanup. Locks should
// always be released using Release() when operations are complete.
//
// Example:
//
//	lock, err := lock.AcquireLock("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer lock.Release()
type Lock struct {
	lockPath string
	lockFile *os.File
}

// AcquireLock attempts to acquire a lock on the bundle (fail-fast).
//
// It creates a lock file at .bundle/.lock atomically. If the lock file already
// exists (another process holds the lock), it returns an error immediately
// without waiting.
//
// The lock file contains the PID of the process holding the lock for debugging.
//
// Example:
//
//	lock, err := lock.AcquireLock("/path/to/bundle")
//	if err != nil {
//	    if strings.Contains(err.Error(), "locked by another process") {
//	        log.Fatal("Bundle is currently in use")
//	    }
//	    log.Fatal(err)
//	}
//	defer lock.Release()
//
//	// Perform bundle modifications...
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - *Lock: lock handle for Release()
//   - error: if lock is held by another process or .bundle/ cannot be created
func AcquireLock(bundlePath string) (*Lock, error) {
	lockPath := filepath.Join(bundlePath, ".bundle", ".lock")

	// Ensure the .bundle directory exists so OpenFile can create the lock
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, err
	}

	// Atomic create-if-not-exists
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("bundle is locked by another process")
		}
		return nil, err
	}

	// Write PID for debugging
	fmt.Fprintf(lockFile, "PID: %d\n", os.Getpid())

	return &Lock{
		lockPath: lockPath,
		lockFile: lockFile,
	}, nil
}

// Release removes the lock.
//
// It closes the lock file handle and deletes .bundle/.lock. Should always be
// called when operations are complete, typically via defer.
//
// Example:
//
//	lock, _ := lock.AcquireLock("/path/to/bundle")
//	defer lock.Release()
//
//	// Perform operations...
//
// Returns:
//   - error: if lock file cannot be deleted
func (l *Lock) Release() error {
	if l.lockFile != nil {
		l.lockFile.Close()
	}
	return os.Remove(l.lockPath)
}

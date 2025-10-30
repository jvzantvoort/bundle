package lock

import (
	"fmt"
	"os"
	"path/filepath"
)

// Lock represents a bundle lock
type Lock struct {
	lockPath string
	lockFile *os.File
}

// AcquireLock attempts to acquire a lock on the bundle (fail-fast)
func AcquireLock(bundlePath string) (*Lock, error) {
	lockPath := filepath.Join(bundlePath, ".bundle", ".lock")

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

// Release removes the lock
func (l *Lock) Release() error {
	if l.lockFile != nil {
		l.lockFile.Close()
	}
	return os.Remove(l.lockPath)
}

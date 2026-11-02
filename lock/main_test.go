package lock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireLock_IsExclusive(t *testing.T) {
	dir := t.TempDir()

	first, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}
	if _, err := AcquireLock(dir); err == nil {
		t.Fatalf("second AcquireLock succeeded while locked")
	}

	if err := first.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".bundle", ".lock")); !os.IsNotExist(err) {
		t.Errorf("lock file still present after Release")
	}

	second, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("AcquireLock after Release failed: %v", err)
	}
	if err := second.Release(); err != nil {
		t.Errorf("Release failed: %v", err)
	}
}

// A move deletes the bundle while it is locked. On Windows this fails if the lock
// file is still open.
func TestLock_BundleCanBeRemovedWhileLocked(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bundle")

	l, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}
	if err := l.Release(); err != nil {
		t.Errorf("Release after removal failed: %v", err)
	}
}

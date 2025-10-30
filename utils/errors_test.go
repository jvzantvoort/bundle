package utils

import (
	"errors"
	"testing"
)

func TestCustomErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"ErrNotABundle", ErrNotABundle, "directory is not a bundle (missing .bundle/)"},
		{"ErrInvalidPath", ErrInvalidPath, "invalid path provided"},
		{"ErrBundleLocked", ErrBundleLocked, "bundle is locked by another process"},
		{"ErrCorruptedBundle", ErrCorruptedBundle, "bundle integrity check failed"},
		{"ErrIncompleteBundle", ErrIncompleteBundle, "bundle is incomplete (missing required files)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	if errors.Is(ErrNotABundle, ErrInvalidPath) {
		t.Error("ErrNotABundle should not match ErrInvalidPath")
	}
	if errors.Is(ErrBundleLocked, ErrCorruptedBundle) {
		t.Error("ErrBundleLocked should not match ErrCorruptedBundle")
	}
}

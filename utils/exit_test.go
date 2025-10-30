package utils

import (
	"testing"
)

func TestExitCodeFromError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"nil error", nil, 0},
		{"user error - not a bundle", ErrNotABundle, 1},
		{"user error - invalid path", ErrInvalidPath, 1},
		{"user error - bundle locked", ErrBundleLocked, 1},
		{"user error - corrupted", ErrCorruptedBundle, 1},
		{"user error - incomplete", ErrIncompleteBundle, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExitCodeFromError(tt.err)
			if got != tt.wantCode {
				t.Errorf("ExitCodeFromError() = %v, want %v", got, tt.wantCode)
			}
		})
	}
}

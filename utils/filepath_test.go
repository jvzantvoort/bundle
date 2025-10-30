package utils

import (
	"path/filepath"
	"testing"
)

func TestShouldExclude(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"bundle metadata dir", ".bundle", true},
		{"file in bundle dir", ".bundle/META.json", true},
		{"normal file", "file.txt", false},
		{"normal dir", "subdir/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldExclude(tt.path); got != tt.want {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestGetBundleMetadataDir(t *testing.T) {
	bundlePath := "/path/to/bundle"
	want := filepath.Join(bundlePath, ".bundle")
	got := GetBundleMetadataDir(bundlePath)
	if got != want {
		t.Errorf("GetBundleMetadataDir(%q) = %q, want %q", bundlePath, got, want)
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"relative path", "./test", false},
		{"absolute path", "/tmp/test", false},
		{"current dir", ".", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !filepath.IsAbs(got) {
				t.Errorf("NormalizePath() = %v, want absolute path", got)
			}
		})
	}
}

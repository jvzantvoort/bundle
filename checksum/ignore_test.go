package checksum

import "testing"

func TestIgnored(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"SHA256SUM.txt", "SHA256SUM.txt", true},
		{"SHA256SUM.txt", "a/b/SHA256SUM.txt", true},
		{"SHA256SUM.txt", "SHA256SUM.txt.bak", false},
		{"*.tmp", "a/b.tmp", true},
		{"*.tmp", "b.txt", false},
		{"build", "build/x/y", true},
		{"build", "a/build/y", true},
		{"build/", "build/x", true},
		{"build", "rebuild/x", false},
		{"docs/draft.txt", "docs/draft.txt", true},
		{"docs/draft.txt", "a/docs/draft.txt", false},
		{"docs/*", "docs/x/y", true},
		{"/top.txt", "top.txt", true},
		{"/top.txt", "a/top.txt", false},
		{"./top.txt", "top.txt", true},
		{"./top.txt", "a/top.txt", false},
	}
	for _, tt := range tests {
		patterns, err := NormalizeIgnore([]string{tt.pattern})
		if err != nil {
			t.Fatalf("%q: %v", tt.pattern, err)
		}
		if got := Ignored(patterns, tt.path); got != tt.want {
			t.Errorf("Ignored(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}

func TestNormalizeIgnoreRefusesBadPatterns(t *testing.T) {
	for _, pattern := range []string{"", "/", ".", "./", "[", "a/["} {
		if _, err := NormalizeIgnore([]string{pattern}); err == nil {
			t.Errorf("NormalizeIgnore(%q) = nil error", pattern)
		}
	}
}

func TestMalformedPatternMatchesNothing(t *testing.T) {
	if Ignored([]string{"["}, "[") {
		t.Error("a malformed pattern must not hide a file")
	}
}

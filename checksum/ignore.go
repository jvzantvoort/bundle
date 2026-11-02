package checksum

import (
	"fmt"
	"path"
	"strings"
)

// NormalizeIgnore checks ignore patterns and returns them in the form Ignored expects.
//
// A pattern uses the syntax of path.Match and always uses forward slashes:
//
//	SHA256SUM.txt   a file or directory with this name, at any depth
//	*.tmp           any name that matches, at any depth
//	docs/draft.txt  this path, relative to the bundle root
//	/SHA256SUM.txt  this name in the bundle root only
//
// A trailing slash is dropped, and ignoring a directory ignores all files below it.
//
// Parameters:
//   - patterns: the patterns as given by the user
//
// Returns:
//   - []string: the normalized patterns, nil if there are none
//   - error: if a pattern is empty or malformed
func NormalizeIgnore(patterns []string) ([]string, error) {
	var normalized []string
	for _, pattern := range patterns {
		clean := strings.TrimRight(pattern, "/")
		if rest, ok := strings.CutPrefix(clean, "./"); ok {
			clean = "/" + rest
		}
		if bare := strings.Trim(clean, "/"); bare == "" || bare == "." {
			return nil, fmt.Errorf("invalid ignore pattern %q", pattern)
		}
		if _, err := path.Match(strings.TrimPrefix(clean, "/"), ""); err != nil {
			return nil, fmt.Errorf("invalid ignore pattern %q: %w", pattern, err)
		}
		normalized = append(normalized, clean)
	}
	return normalized, nil
}

// Ignored reports whether a path, or a directory above it, matches a pattern.
//
// A malformed pattern matches nothing, so that a damaged list cannot hide a file.
//
// Parameters:
//   - patterns: patterns from NormalizeIgnore
//   - relPath: slash-separated path relative to the bundle root
//
// Returns:
//   - bool: true if the path is ignored
func Ignored(patterns []string, relPath string) bool {
	parts := strings.Split(relPath, "/")
	for _, pattern := range patterns {
		anchored := strings.Contains(pattern, "/")
		pattern = strings.TrimPrefix(pattern, "/")
		for i, part := range parts {
			candidate := part
			if anchored {
				candidate = strings.Join(parts[:i+1], "/")
			}
			if matched, err := path.Match(pattern, candidate); err == nil && matched {
				return true
			}
		}
	}
	return false
}

// Package tag provides types and functions for managing bundle tags.
//
// Tags are searchable labels that can be attached to bundles for categorization
// and discovery. Tags are stored in .bundle/TAGS.txt with one tag per line.
//
// Tag validation rules:
//   - Converted to lowercase for case-insensitive matching
//   - Must match pattern: ^[a-z0-9._-]{1,64}$
//   - Automatically deduplicated
//
// Example usage:
//
//	// Load tags
//	tags, err := tag.Load("/path/to/bundle")
//
//	// Add tags
//	tags.Add("vacation", "europe", "2024")
//
//	// Remove tags
//	tags.Remove("2024")
//
//	// Get sorted list
//	tagList := tags.List()
//
//	// Save changes
//	err = tags.Save("/path/to/bundle")
package tag

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var tagPattern = regexp.MustCompile(`^[a-z0-9._-]{1,64}$`)

// normalizeTag trims whitespace, lowercases and validates a tag.
//
// It converts tags to lowercase for case-insensitive matching and validates
// against allowed characters and length constraints.
//
// Validation rules:
//   - Must be 1-64 characters
//   - Only lowercase letters, digits, dots, underscores, hyphens
//   - No whitespace allowed
//
// Example:
//
//	tag, ok := normalizeTag("Vacation")
//	// tag = "vacation", ok = true
//
//	tag, ok = normalizeTag("my tag")
//	// tag = "", ok = false (contains space)
//
//	tag, ok = normalizeTag("")
//	// tag = "", ok = false (empty)
//
// Parameters:
//   - s: raw tag string
//
// Returns:
//   - string: normalized tag (lowercase, trimmed)
//   - bool: true if valid, false if invalid
func normalizeTag(s string) (string, bool) {
	t := strings.TrimSpace(s)
	if t == "" {
		return "", false
	}
	// Normalize to lowercase to make tags case-insensitive
	t = strings.ToLower(t)
	// Disallow whitespace inside tag
	if strings.ContainsAny(t, " \t\n\r") {
		return "", false
	}
	// Validate allowed characters and max length
	if !tagPattern.MatchString(t) {
		return "", false
	}
	return t, true
}

// Tags represents the collection of tags associated with a bundle.
//
// Tags are stored as unique, normalized strings (lowercase, alphanumeric with
// dots, underscores, and hyphens). Duplicates are automatically removed.
//
// Example:
//
//	tags := &tag.Tags{
//	    Tags: []string{"travel", "photos", "2024"},
//	}
type Tags struct {
	Tags []string // Unique, case-sensitive tag names
}

// Load reads tags from .bundle/TAGS.txt.
//
// It reads one tag per line, normalizes them to lowercase, and removes duplicates.
// If the file doesn't exist, it returns an empty Tags struct without error.
//
// Example TAGS.txt:
//
//	travel
//	photos
//	vacation
//	europe
//
// Example usage:
//
//	tags, err := tag.Load("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Tags: %v\n", tags.List())
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - *Tags: parsed tags (empty if file doesn't exist)
//   - error: if file cannot be read (but not if it doesn't exist)
func Load(bundlePath string) (*Tags, error) {
	tagsFile := filepath.Join(bundlePath, ".bundle", "TAGS.txt")
	data, err := os.ReadFile(tagsFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty tags if file doesn't exist
			return &Tags{Tags: []string{}}, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	tags := []string{}
	tagSet := make(map[string]bool)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if nt, ok := normalizeTag(trimmed); ok {
			if !tagSet[nt] {
				tags = append(tags, nt)
				tagSet[nt] = true
			}
		}
	}

	return &Tags{Tags: tags}, nil
}

// Save writes tags to .bundle/TAGS.txt in sorted order.
//
// Tags are written one per line in alphabetical order for deterministic output.
// The file is created with 0644 permissions.
//
// Example:
//
//	tags := &tag.Tags{Tags: []string{"travel", "photos", "vacation"}}
//	err := tags.Save("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Output file:
//
//	photos
//	travel
//	vacation
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - error: if file cannot be created or written
func (t *Tags) Save(bundlePath string) error {
	tagsFile := filepath.Join(bundlePath, ".bundle", "TAGS.txt")

	// Sort tags
	sort.Strings(t.Tags)

	file, err := os.Create(tagsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, tag := range t.Tags {
		if _, err := writer.WriteString(tag + "\n"); err != nil {
			return fmt.Errorf("failed to write tag: %w", err)
		}
	}
	return writer.Flush()
}

// Add appends tags (deduplicates automatically).
//
// Tags are normalized to lowercase and validated. Invalid tags are silently
// ignored. Duplicates are automatically filtered out. Call Save() to persist.
//
// Example:
//
//	tags := &tag.Tags{Tags: []string{"travel"}}
//	tags.Add("Photos", "VACATION", "travel")  // "Photos" and "VACATION" normalized
//	tags.Save("/path/to/bundle")
//	// tags.Tags = ["travel", "photos", "vacation"]
//
// Parameters:
//   - newTags: one or more tag strings to add
func (t *Tags) Add(newTags ...string) {
	// Use struct{} for sets - more memory efficient than bool
	tagSet := make(map[string]struct{}, len(t.Tags))
	for _, tag := range t.Tags {
		tagSet[tag] = struct{}{}
	}

	for _, tag := range newTags {
		if nt, ok := normalizeTag(tag); ok {
			if _, exists := tagSet[nt]; !exists {
				t.Tags = append(t.Tags, nt)
				tagSet[nt] = struct{}{}
			}
		}
		// invalid tags are ignored silently; callers (CLI) should validate if they need stricter handling
	}
}

// Remove filters out specified tags.
//
// Tags are normalized before removal. Tags not in the collection are ignored.
// Call Save() to persist the changes.
//
// Example:
//
//	tags := &tag.Tags{Tags: []string{"travel", "photos", "vacation"}}
//	tags.Remove("Photos", "VACATION")  // Normalized to lowercase
//	tags.Save("/path/to/bundle")
//	// tags.Tags = ["travel"]
//
// Parameters:
//   - removeTags: one or more tag strings to remove
func (t *Tags) Remove(removeTags ...string) {
	// Use struct{} for sets - more memory efficient
	removeSet := make(map[string]struct{}, len(removeTags))
	for _, tag := range removeTags {
		if nt, ok := normalizeTag(tag); ok {
			removeSet[nt] = struct{}{}
		}
	}

	// Pre-allocate with capacity hint to avoid reallocations
	filtered := make([]string, 0, len(t.Tags))
	for _, tag := range t.Tags {
		if _, shouldRemove := removeSet[tag]; !shouldRemove {
			filtered = append(filtered, tag)
		}
	}
	t.Tags = filtered
}

// List returns sorted tag list.
//
// It returns a new slice sorted alphabetically. The original Tags slice
// is not modified.
//
// Example:
//
//	tags := &tag.Tags{Tags: []string{"vacation", "photos", "travel"}}
//	sorted := tags.List()
//	// sorted = ["photos", "travel", "vacation"]
//
// Returns:
//   - []string: alphabetically sorted copy of tags
func (t *Tags) List() []string {
	sorted := make([]string, len(t.Tags))
	copy(sorted, t.Tags)
	sort.Strings(sorted)
	return sorted
}

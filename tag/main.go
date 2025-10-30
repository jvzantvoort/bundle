package tag

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var tagPattern = regexp.MustCompile(`^[a-z0-9._-]{1,64}$`)

// normalizeTag trims whitespace, lowercases and validates a tag.
// Returns normalized tag and ok=true when valid.
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

// Tags represents the collection of tags associated with a bundle
type Tags struct {
	Tags []string // Unique, case-sensitive tag names
}

// Load reads tags from .bundle/TAGS.txt
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

// Save writes tags to .bundle/TAGS.txt in sorted order
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
		writer.WriteString(tag + "\n")
	}
	return writer.Flush()
}

// Add appends tags (deduplicates automatically)
func (t *Tags) Add(newTags ...string) {
	tagSet := make(map[string]bool)
	for _, tag := range t.Tags {
		tagSet[tag] = true
	}

	for _, tag := range newTags {
		if nt, ok := normalizeTag(tag); ok {
			if !tagSet[nt] {
				t.Tags = append(t.Tags, nt)
				tagSet[nt] = true
			}
		}
		// invalid tags are ignored silently; callers (CLI) should validate if they need stricter handling
	}
}

// Remove filters out specified tags
func (t *Tags) Remove(removeTags ...string) {
	removeSet := make(map[string]bool)
	for _, tag := range removeTags {
		if nt, ok := normalizeTag(tag); ok {
			removeSet[nt] = true
		}
	}

	filtered := []string{}
	for _, tag := range t.Tags {
		if !removeSet[tag] {
			filtered = append(filtered, tag)
		}
	}
	t.Tags = filtered
}

// List returns sorted tag list
func (t *Tags) List() []string {
	sorted := make([]string, len(t.Tags))
	copy(sorted, t.Tags)
	sort.Strings(sorted)
	return sorted
}

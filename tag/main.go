package tag

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			tags = append(tags, trimmed)
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
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" && !tagSet[trimmed] {
			t.Tags = append(t.Tags, trimmed)
			tagSet[trimmed] = true
		}
	}
}

// Remove filters out specified tags
func (t *Tags) Remove(removeTags ...string) {
	removeSet := make(map[string]bool)
	for _, tag := range removeTags {
		removeSet[strings.TrimSpace(tag)] = true
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

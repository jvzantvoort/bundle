package tag

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestNormalizeTag(t *testing.T) {
    cases := []struct{
        in string
        want string
        ok bool
    }{
        {"Photos", "photos", true},
        {"  travel  ", "travel", true},
        {"UPPER_case", "upper_case", true},
        {"with space", "", false},
        {"", "", false},
        {"toolongtoolongtoolongtoolongtoolongtoolongtoolongtoolongtoolongtoolong", "", false},
        {"good-tag.123", "good-tag.123", true},
    }

    for _, c := range cases {
        got, ok := normalizeTag(c.in)
        if ok != c.ok {
            t.Fatalf("normalizeTag(%q) ok = %v, want %v", c.in, ok, c.ok)
        }
        if got != c.want {
            t.Fatalf("normalizeTag(%q) = %q, want %q", c.in, got, c.want)
        }
    }
}

func TestAddListRemove(t *testing.T) {
    tgs := &Tags{Tags: []string{}}

    tgs.Add("Photos", "photos", " travel ", "INVALID TAG", "UPPER")

    got := tgs.List()
    // Expect normalized, deduped and sorted: photos, travel, upper
    want := []string{"photos", "travel", "upper"}
    if len(got) != len(want) {
        t.Fatalf("List length = %d, want %d; got=%v", len(got), len(want), got)
    }
    for i := range want {
        if got[i] != want[i] {
            t.Fatalf("List[%d]=%q, want %q", i, got[i], want[i])
        }
    }

    // Remove with different case and whitespace
    tgs.Remove(" PHOTOS ")
    got2 := tgs.List()
    want2 := []string{"travel", "upper"}
    if len(got2) != len(want2) {
        t.Fatalf("After remove, len=%d want %d got=%v", len(got2), len(want2), got2)
    }
    for i := range want2 {
        if got2[i] != want2[i] {
            t.Fatalf("After remove, List[%d]=%q, want %q", i, got2[i], want2[i])
        }
    }
}

func TestLoadSaveRoundtrip(t *testing.T) {
    dir := t.TempDir()
    bundleDir := filepath.Join(dir, ".bundle")
    if err := os.MkdirAll(bundleDir, 0755); err != nil {
        t.Fatalf("mkdir .bundle: %v", err)
    }

    // write a TAGS.txt with mixed content
    data := "Photos\n travel \nINVALID TAG\nphotos\n"
    tagsFile := filepath.Join(bundleDir, "TAGS.txt")
    if err := os.WriteFile(tagsFile, []byte(data), 0644); err != nil {
        t.Fatalf("write tags: %v", err)
    }

    tgs, err := Load(dir)
    if err != nil {
        t.Fatalf("Load failed: %v", err)
    }

    // Expect photos and travel only
    got := tgs.List()
    want := []string{"photos", "travel"}
    if len(got) != len(want) {
        t.Fatalf("Load List len=%d want=%d got=%v", len(got), len(want), got)
    }
    for i := range want {
        if got[i] != want[i] {
            t.Fatalf("Load List[%d]=%q want=%q", i, got[i], want[i])
        }
    }

    // Save and re-read file to ensure persisted values are normalized and sorted
    if err := tgs.Save(dir); err != nil {
        t.Fatalf("Save failed: %v", err)
    }
    b, err := os.ReadFile(tagsFile)
    if err != nil {
        t.Fatalf("Read saved tags: %v", err)
    }
    lines := strings.Split(strings.TrimSpace(string(b)), "\n")
    if len(lines) != len(want) {
        t.Fatalf("Saved lines len=%d want=%d, lines=%v", len(lines), len(want), lines)
    }
    for i := range want {
        if lines[i] != want[i] {
            t.Fatalf("Saved line[%d]=%q want=%q", i, lines[i], want[i])
        }
    }
}

func TestEdgeCases(t *testing.T) {
    // Very long tag
    long := strings.Repeat("a", 100)
    if _, ok := normalizeTag(long); ok {
        t.Fatalf("expected very long tag to be invalid")
    }

    // Unicode tag should be invalid under current ASCII-only policy
    if _, ok := normalizeTag("café"); ok {
        t.Fatalf("expected unicode tag to be invalid")
    }
    if _, ok := normalizeTag("🔥"); ok {
        t.Fatalf("expected emoji tag to be invalid")
    }

    // Add mixture of valid and invalid tags
    tgs := &Tags{Tags: []string{}}
    tgs.Add("GoodTag", long, "café", "another")
    got := tgs.List()
    expected := []string{"another", "goodtag"}
    if len(got) != len(expected) {
        t.Fatalf("Add edge mixture: got=%v expected=%v", got, expected)
    }
    for i := range expected {
        if got[i] != expected[i] {
            t.Fatalf("Add edge mixture: got[%d]=%q expected=%q", i, got[i], expected[i])
        }
    }

    // Removing an invalid tag should be a no-op
    tgs.Remove("café")
    after := tgs.List()
    if len(after) != len(expected) {
        t.Fatalf("Remove invalid changed tags: got=%v expected=%v", after, expected)
    }
}

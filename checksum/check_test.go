package checksum

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCheck_ReportsEachReason(t *testing.T) {
	dir := t.TempDir()
	write := func(name, data string) {
		t.Helper()
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("same.txt", "same")
	write("changed.txt", "before")
	write("gone.txt", "gone")

	cf := &ChecksumFile{}
	if err := cf.Compute(dir); err != nil {
		t.Fatal(err)
	}
	expected := make(map[string]string)
	for _, record := range cf.Records {
		expected[record.FilePath] = record.Checksum
	}

	write("changed.txt", "after")
	if err := os.Remove(filepath.Join(dir, "gone.txt")); err != nil {
		t.Fatal(err)
	}
	write("sub/new.txt", "new")
	found, err := ComputeFileSHA256(filepath.Join(dir, "changed.txt"))
	if err != nil {
		t.Fatal(err)
	}

	problems, err := cf.Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []Problem{
		{Path: "changed.txt", Reason: ReasonChecksumMismatch, Expected: expected["changed.txt"], Found: found},
		{Path: "gone.txt", Reason: ReasonMissing, Expected: expected["gone.txt"]},
		{Path: "sub/new.txt", Reason: ReasonUnlisted},
	}
	if !reflect.DeepEqual(problems, want) {
		t.Fatalf("Check =\n%+v\nwant\n%+v", problems, want)
	}

	// Verify reports the same files, and ignore patterns apply to unlisted files only
	paths, err := cf.Verify(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paths, Paths(problems)) {
		t.Errorf("Verify = %v, want %v", paths, Paths(problems))
	}
	ignored, err := cf.Check(dir, "sub/new.txt", "changed.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(ignored) != 2 || ignored[0].Path != "changed.txt" || ignored[1].Path != "gone.txt" {
		t.Errorf("Check with ignore = %+v, want changed.txt and gone.txt", ignored)
	}
}

func TestCheck_CleanBundleHasNoProblems(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	cf := &ChecksumFile{}
	if err := cf.Compute(dir); err != nil {
		t.Fatal(err)
	}
	problems, err := cf.Check(dir)
	if err != nil || problems == nil || len(problems) != 0 {
		t.Fatalf("Check = %v, %v; want an empty, non-nil list", problems, err)
	}
	if paths := Paths(problems); paths == nil || len(paths) != 0 {
		t.Fatalf("Paths = %v, want an empty, non-nil list", paths)
	}
}

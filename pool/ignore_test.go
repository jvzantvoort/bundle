package pool

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/utils"
)

func writeTestFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, data := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestImportLeavesOutFilesIgnoredAtCreate(t *testing.T) {
	dir := t.TempDir()
	writeTestFiles(t, dir, map[string]string{
		"keep.txt":          "keep",
		"SHA256SUM.txt":     "ignored",
		"sub/SHA256SUM.txt": "ignored",
		"cache/a/b":         "ignored",
	})
	b, err := bundle.Create(dir, "T", "SHA256SUM.txt", "cache")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Files.Records) != 1 {
		t.Fatalf("records = %v", b.Files.Records)
	}
	if ok, corrupted, err := bundle.Verify(dir); err != nil || !ok {
		t.Fatalf("verify = %v, %v, %v", ok, corrupted, err)
	}

	p := &Pool{Root: t.TempDir()}
	if err := p.Import(dir, false); err != nil {
		t.Fatal(err)
	}
	dest := p.GetBundlePath(b.Metadata.BundleChecksum)
	if !exists(filepath.Join(dest, "keep.txt")) {
		t.Error("keep.txt was not copied")
	}
	for _, name := range []string{"SHA256SUM.txt", "sub/SHA256SUM.txt", "cache"} {
		if exists(filepath.Join(dest, filepath.FromSlash(name))) {
			t.Errorf("%s was copied", name)
		}
	}
	if ok, corrupted, err := bundle.Verify(dest); err != nil || !ok {
		t.Errorf("verify of the copy = %v, %v, %v", ok, corrupted, err)
	}
}

func TestIgnoreDoesNotApplyToBundleMetadata(t *testing.T) {
	dir := t.TempDir()
	writeTestFiles(t, dir, map[string]string{"keep.txt": "keep"})
	// These match every metadata file by name, and a catch-all matches .bundle itself
	b, err := bundle.Create(dir, "T", "SHA256SUM.txt", "META.json", "/.bundle", ".*.json")
	if err != nil {
		t.Fatal(err)
	}
	p := &Pool{Root: t.TempDir()}
	if err := p.Import(dir, false); err != nil {
		t.Fatal(err)
	}
	if ok, corrupted, err := bundle.Verify(p.GetBundlePath(b.Metadata.BundleChecksum)); err != nil || !ok {
		t.Errorf("verify of the copy = %v, %v, %v", ok, corrupted, err)
	}
}

func TestIgnoreCannotHideChangesToListedFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFiles(t, dir, map[string]string{"keep.txt": "keep"})
	if _, err := bundle.Create(dir, "T", "*.tmp"); err != nil {
		t.Fatal(err)
	}
	writeTestFiles(t, dir, map[string]string{"keep.txt": "changed", "new.tmp": "x", "new.txt": "y"})

	ok, corrupted, err := bundle.Verify(dir)
	if err != nil || ok {
		t.Fatalf("verify = %v, %v", ok, err)
	}
	if len(corrupted) != 2 || corrupted[0] != "keep.txt" || corrupted[1] != "new.txt" {
		t.Errorf("corrupted = %v", corrupted)
	}
}

func TestImportIgnoreAcceptsUnlistedFiles(t *testing.T) {
	dir, b := newBundle(t, map[string]string{"keep.txt": "keep"})
	writeTestFiles(t, dir, map[string]string{"SHA256SUM.txt": "added later"})
	p := &Pool{Root: t.TempDir()}

	if err := p.Import(dir, false); !errors.Is(err, utils.ErrCorruptedBundle) {
		t.Fatalf("import without --ignore: err = %v", err)
	}
	if err := p.Import(dir, false, "SHA256SUM.txt"); err != nil {
		t.Fatal(err)
	}
	dest := p.GetBundlePath(b.Metadata.BundleChecksum)
	if exists(filepath.Join(dest, "SHA256SUM.txt")) {
		t.Error("ignored file was copied")
	}
	if ok, corrupted, err := bundle.Verify(dest); err != nil || !ok {
		t.Errorf("verify of the copy = %v, %v, %v", ok, corrupted, err)
	}
}

func TestImportIgnoreRefusesListedFile(t *testing.T) {
	dir, _ := newBundle(t, map[string]string{"keep.txt": "keep"})
	p := &Pool{Root: t.TempDir()}

	if err := p.Import(dir, false, "keep.txt"); !errors.Is(err, utils.ErrInvalidPath) {
		t.Errorf("err = %v, want ErrInvalidPath", err)
	}
	if err := p.Import(dir, false, "["); !errors.Is(err, utils.ErrInvalidPath) {
		t.Errorf("malformed pattern: err = %v, want ErrInvalidPath", err)
	}
	if entries := poolEntries(t, p); len(entries) != 0 {
		t.Errorf("pool = %v", entries)
	}
}

func TestImportMoveKeepsIgnoredFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFiles(t, dir, map[string]string{"keep.txt": "keep", "note.tmp": "precious"})
	b, err := bundle.Create(dir, "T", "*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	p := &Pool{Root: t.TempDir()}

	if err := p.Import(dir, true); !errors.Is(err, utils.ErrInvalidPath) {
		t.Fatalf("err = %v, want ErrInvalidPath", err)
	}
	if !exists(filepath.Join(dir, "note.tmp")) || !exists(filepath.Join(dir, "keep.txt")) {
		t.Error("refused move changed the source")
	}
	if exists(filepath.Join(dir, ".bundle", ".lock")) {
		t.Error("refused move left the lock behind")
	}
	if exists(p.GetBundlePath(b.Metadata.BundleChecksum)) {
		t.Error("refused move reached the pool")
	}

	// Once the file is gone there is nothing left to lose
	if err := os.Remove(filepath.Join(dir, "note.tmp")); err != nil {
		t.Fatal(err)
	}
	if err := p.Import(dir, true); err != nil {
		t.Fatal(err)
	}
	if exists(dir) {
		t.Error("source still exists after the move")
	}
}

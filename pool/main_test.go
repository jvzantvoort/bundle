package pool

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/checksum"
	"github.com/jvzantvoort/bundle/metadata"
	"github.com/jvzantvoort/bundle/utils"
)

// newBundle creates a bundle with the given file contents.
func newBundle(t *testing.T, files map[string]string) (string, *bundle.Bundle) {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := bundle.Create(dir, "T")
	if err != nil {
		t.Fatal(err)
	}
	return dir, b
}

// makeLegacy rewrites a bundle as a version 1 bundle and returns its checksum.
func makeLegacy(t *testing.T, dir string, b *bundle.Bundle) string {
	t.Helper()
	checksums := []string{}
	for _, record := range b.Files.Records {
		checksums = append(checksums, record.Checksum)
	}
	b.Metadata.Version = metadata.VersionLegacy
	b.Metadata.BundleChecksum = checksum.ComputeLegacyBundleChecksum(checksums)
	if err := b.Metadata.Save(dir); err != nil {
		t.Fatal(err)
	}
	return b.Metadata.BundleChecksum
}

// poolEntries returns the names in the pool root.
func poolEntries(t *testing.T, p *Pool) []string {
	t.Helper()
	entries, err := os.ReadDir(p.Root)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func TestImport_CopyAndMove(t *testing.T) {
	for _, move := range []bool{false, true} {
		src, b := newBundle(t, map[string]string{"my file.txt": "x", "sub/y": "y"})
		p := &Pool{Root: filepath.Join(t.TempDir(), "pool")}

		if err := p.Import(src, move); err != nil {
			t.Fatalf("Import(move=%v) failed: %v", move, err)
		}

		dest := p.GetBundlePath(b.Metadata.BundleChecksum)
		if ok, corrupted, err := bundle.Verify(dest); err != nil || !ok {
			t.Errorf("pool copy invalid: %v %v %v", ok, corrupted, err)
		}
		if _, err := os.Stat(filepath.Join(dest, ".bundle", ".lock")); !os.IsNotExist(err) {
			t.Errorf("lock copied to pool (move=%v)", move)
		}
		if names := poolEntries(t, p); len(names) != 1 {
			t.Errorf("pool entries = %v, want only the bundle", names)
		}

		_, err := os.Stat(src)
		if move && !os.IsNotExist(err) {
			t.Errorf("move kept the source")
		}
		if !move && err != nil {
			t.Errorf("copy removed the source: %v", err)
		}
	}
}

func TestImport_RefusesCorruptedBundleAndKeepsSource(t *testing.T) {
	for _, move := range []bool{false, true} {
		src, _ := newBundle(t, map[string]string{"file": "x"})
		if err := os.WriteFile(filepath.Join(src, "file"), []byte("changed"), 0644); err != nil {
			t.Fatal(err)
		}
		p := &Pool{Root: t.TempDir()}

		if err := p.Import(src, move); !errors.Is(err, utils.ErrCorruptedBundle) {
			t.Fatalf("err = %v, want ErrCorruptedBundle", err)
		}
		if names := poolEntries(t, p); len(names) != 0 {
			t.Errorf("pool entries = %v, want none", names)
		}
		if _, err := os.Stat(filepath.Join(src, "file")); err != nil {
			t.Errorf("source damaged: %v", err)
		}
		if _, err := os.Stat(filepath.Join(src, ".bundle", ".lock")); !os.IsNotExist(err) {
			t.Errorf("lock left on source (move=%v)", move)
		}
	}
}

func TestImport_RefusesPathTraversal(t *testing.T) {
	src, b := newBundle(t, map[string]string{"file": "x"})
	b.Metadata.BundleChecksum = "../../escaped"
	if err := b.Metadata.Save(src); err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	p := &Pool{Root: filepath.Join(base, "a", "pool")}

	if err := p.Import(src, false); !errors.Is(err, utils.ErrCorruptedBundle) {
		t.Fatalf("err = %v, want ErrCorruptedBundle", err)
	}
	if _, err := os.Stat(filepath.Join(base, "escaped")); !os.IsNotExist(err) {
		t.Errorf("import wrote outside the pool")
	}
}

func TestImport_RefusesDuplicateLegacyAndNonBundle(t *testing.T) {
	src, _ := newBundle(t, map[string]string{"file": "x"})
	p := &Pool{Root: t.TempDir()}
	if err := p.Import(src, false); err != nil {
		t.Fatal(err)
	}
	if err := p.Import(src, true); !errors.Is(err, utils.ErrBundleExists) {
		t.Errorf("duplicate: err = %v, want ErrBundleExists", err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("refused move removed the source: %v", err)
	}

	legacySrc, legacy := newBundle(t, map[string]string{"old": "y"})
	makeLegacy(t, legacySrc, legacy)
	if err := p.Import(legacySrc, false); !errors.Is(err, utils.ErrOutdatedBundle) {
		t.Errorf("legacy: err = %v, want ErrOutdatedBundle", err)
	}

	if err := p.Import(t.TempDir(), false); !errors.Is(err, utils.ErrNotABundle) {
		t.Errorf("non-bundle: err = %v, want ErrNotABundle", err)
	}
}

func TestImport_RefusesPoolInsideBundle(t *testing.T) {
	src, _ := newBundle(t, map[string]string{"file": "x"})
	p := &Pool{Root: filepath.Join(src, "pool")}

	if err := p.Import(src, true); !errors.Is(err, utils.ErrInvalidPath) {
		t.Fatalf("err = %v, want ErrInvalidPath", err)
	}
	if _, err := os.Stat(filepath.Join(src, "file")); err != nil {
		t.Errorf("source damaged: %v", err)
	}
}

func TestImport_SameContentDifferentNamesBothStored(t *testing.T) {
	one, _ := newBundle(t, map[string]string{"one": "same"})
	two, _ := newBundle(t, map[string]string{"two": "same"})
	p := &Pool{Root: t.TempDir()}

	if err := p.Import(one, false); err != nil {
		t.Fatal(err)
	}
	if err := p.Import(two, false); err != nil {
		t.Fatalf("second bundle refused: %v", err)
	}
	if names := poolEntries(t, p); len(names) != 2 {
		t.Errorf("pool entries = %v, want 2", names)
	}
}

func TestListBundles_SkipsInvalidEntries(t *testing.T) {
	src, b := newBundle(t, map[string]string{"file": "x"})
	p := &Pool{Root: t.TempDir()}
	if err := p.Import(src, false); err != nil {
		t.Fatal(err)
	}

	// A short checksum used to panic
	short := filepath.Join(p.Root, "short", ".bundle")
	if err := os.MkdirAll(short, 0755); err != nil {
		t.Fatal(err)
	}
	meta := `{"title":"x","created_at":"2024-01-01T00:00:00Z","bundle_checksum":"abc","author":"a","version":1}`
	if err := os.WriteFile(filepath.Join(short, "META.json"), []byte(meta), 0644); err != nil {
		t.Fatal(err)
	}

	// An abandoned import must not be listed
	if err := copyDir(p.GetBundlePath(b.Metadata.BundleChecksum), filepath.Join(p.Root, stagingPrefix+"1")); err != nil {
		t.Fatal(err)
	}

	bundles, err := p.ListBundles()
	if err != nil {
		t.Fatalf("ListBundles failed: %v", err)
	}
	if len(bundles) != 1 || bundles[0].BundleChecksum != b.Metadata.BundleChecksum {
		t.Errorf("bundles = %+v, want only the imported bundle", bundles)
	}
}

func TestMigrate(t *testing.T) {
	p := &Pool{Root: t.TempDir()}

	// Add a version 1 bundle stored under its legacy checksum
	src, b := newBundle(t, map[string]string{"file": "x"})
	newChecksum := b.Metadata.BundleChecksum
	legacy := makeLegacy(t, src, b)
	if err := copyDir(src, p.GetBundlePath(legacy)); err != nil {
		t.Fatal(err)
	}

	// Add a corrupted one, which must be reported and left alone
	badSrc, bad := newBundle(t, map[string]string{"other": "y"})
	badLegacy := makeLegacy(t, badSrc, bad)
	if err := os.WriteFile(filepath.Join(badSrc, "other"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := copyDir(badSrc, p.GetBundlePath(badLegacy)); err != nil {
		t.Fatal(err)
	}

	results, err := p.Migrate()
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %+v, want 2", results)
	}
	for _, result := range results {
		switch result.Name {
		case legacy:
			if result.Err != nil || result.NewChecksum != newChecksum {
				t.Errorf("good bundle result = %+v", result)
			}
		case badLegacy:
			if !errors.Is(result.Err, utils.ErrCorruptedBundle) {
				t.Errorf("bad bundle result = %+v", result)
			}
		}
	}

	if ok, corrupted, err := bundle.Verify(p.GetBundlePath(newChecksum)); err != nil || !ok {
		t.Errorf("migrated bundle invalid: %v %v %v", ok, corrupted, err)
	}
	if _, err := os.Stat(p.GetBundlePath(legacy)); !os.IsNotExist(err) {
		t.Errorf("old directory still present")
	}
	if _, err := os.Stat(p.GetBundlePath(badLegacy)); err != nil {
		t.Errorf("corrupted bundle moved: %v", err)
	}
}

func TestMigrate_KeepsEntryWhenNewChecksumExists(t *testing.T) {
	p := &Pool{Root: t.TempDir()}

	// The same bundle exists at its new name and as a version 1 entry
	current, b := newBundle(t, map[string]string{"file": "x"})
	if err := p.Import(current, false); err != nil {
		t.Fatal(err)
	}
	legacy := makeLegacy(t, current, b)
	if err := copyDir(current, p.GetBundlePath(legacy)); err != nil {
		t.Fatal(err)
	}

	results, err := p.Migrate()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !errors.Is(results[0].Err, utils.ErrBundleExists) {
		t.Fatalf("results = %+v, want ErrBundleExists", results)
	}
	if _, err := os.Stat(p.GetBundlePath(legacy)); err != nil {
		t.Errorf("entry was deleted: %v", err)
	}
}

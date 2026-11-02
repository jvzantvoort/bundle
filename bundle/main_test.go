package bundle

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jvzantvoort/bundle/checksum"
	"github.com/jvzantvoort/bundle/metadata"
	"github.com/jvzantvoort/bundle/utils"
)

// TestCreateLoadVerify performs an end-to-end create, load, verify and corruption detection
func TestCreateLoadVerify(t *testing.T) {
	dir := t.TempDir()

	// Create some test files
	files := []struct {
		name string
		data string
	}{
		{"a.txt", "hello"},
		{"sub/b.txt", "world"},
	}

	for _, f := range files {
		p := filepath.Join(dir, f.name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte(f.data), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	// Create bundle
	b, err := Create(dir, "Test Bundle")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if b == nil {
		t.Fatalf("Create returned nil bundle")
	}

	// Load bundle
	lb, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if lb.Metadata.Title != "Test Bundle" {
		t.Fatalf("unexpected title: %s", lb.Metadata.Title)
	}

	// Verify should succeed
	ok, corrupted, err := Verify(dir)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if !ok || len(corrupted) != 0 {
		t.Fatalf("expected verify ok, got corrupted=%v", corrupted)
	}

	// Corrupt a file
	f0 := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f0, []byte("corrupt"), 0644); err != nil {
		t.Fatalf("corrupt write failed: %v", err)
	}

	ok, corrupted, err = Verify(dir)
	if err != nil {
		t.Fatalf("Verify after corrupt error: %v", err)
	}
	if ok {
		t.Fatalf("expected verify to detect corruption")
	}
	if len(corrupted) == 0 {
		t.Fatalf("expected corrupted list, got none")
	}
}

// TestLoadNonBundle ensures Load returns error for non-bundle directory
func TestLoadNonBundle(t *testing.T) {
	dir := t.TempDir()
	// Ensure no .bundle exists
	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected error loading non-bundle dir")
	}
}

// writeFiles creates files below dir, keyed by relative path.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, data := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte(data), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
}

// makeLegacy rewrites a bundle as a version 1 bundle.
func makeLegacy(t *testing.T, dir string, b *Bundle) string {
	t.Helper()
	checksums := []string{}
	for _, record := range b.Files.Records {
		checksums = append(checksums, record.Checksum)
	}
	b.Metadata.Version = metadata.VersionLegacy
	b.Metadata.BundleChecksum = checksum.ComputeLegacyBundleChecksum(checksums)
	if err := b.Metadata.Save(dir); err != nil {
		t.Fatalf("save metadata: %v", err)
	}
	return b.Metadata.BundleChecksum
}

func TestCreate_IncludesNamesContainingDotBundle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "photos.bundle")
	writeFiles(t, root, map[string]string{"my file.txt": "a", "my.bundle.txt": "b", "x.bundle/c": "c"})

	b, err := Create(root, "T")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if len(b.Files.Records) != 3 {
		t.Fatalf("got %d records, want 3: %v", len(b.Files.Records), b.Files.Records)
	}

	ok, corrupted, err := Verify(root)
	if err != nil || !ok {
		t.Fatalf("Verify = %v %v %v, want valid", ok, corrupted, err)
	}
}

func TestCreate_ChecksumDependsOnFileNames(t *testing.T) {
	one, two := t.TempDir(), t.TempDir()
	writeFiles(t, one, map[string]string{"one": "same"})
	writeFiles(t, two, map[string]string{"two": "same"})

	b1, err := Create(one, "")
	if err != nil {
		t.Fatal(err)
	}
	b2, err := Create(two, "")
	if err != nil {
		t.Fatal(err)
	}
	if b1.Metadata.BundleChecksum == b2.Metadata.BundleChecksum {
		t.Errorf("different file names produced the same checksum")
	}
	if b1.Metadata.Version != metadata.CurrentVersion {
		t.Errorf("version = %d, want %d", b1.Metadata.Version, metadata.CurrentVersion)
	}
}

func TestCreate_RefusesMissingPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "typo")
	if _, err := Create(missing, ""); !os.IsNotExist(err) {
		t.Fatalf("err = %v, want not exist", err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Errorf("Create made the missing directory")
	}
}

func TestCreate_RefusesFile(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"file": "x"})
	if _, err := Create(filepath.Join(dir, "file"), ""); !errors.Is(err, utils.ErrInvalidPath) {
		t.Fatalf("err = %v, want ErrInvalidPath", err)
	}
}

func TestCreate_RefusesExistingBundle(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"file": "x"})
	b, err := Create(dir, "Original")
	if err != nil {
		t.Fatal(err)
	}
	b.Tags.Add("keep")
	if err := b.Tags.Save(dir); err != nil {
		t.Fatal(err)
	}

	if _, err := Create(dir, "Other"); !errors.Is(err, utils.ErrBundleExists) {
		t.Fatalf("err = %v, want ErrBundleExists", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Metadata.Title != "Original" || !slices.Contains(loaded.Tags.List(), "keep") {
		t.Errorf("existing bundle changed: title=%q tags=%v", loaded.Metadata.Title, loaded.Tags.List())
	}
	if _, err := os.Stat(filepath.Join(dir, ".bundle", ".lock")); !os.IsNotExist(err) {
		t.Errorf("lock left behind")
	}
}

func TestVerify_DetectsTampering(t *testing.T) {
	tests := []struct {
		name   string
		tamper func(t *testing.T, dir string)
		want   string
	}{
		{"added file", func(t *testing.T, dir string) {
			writeFiles(t, dir, map[string]string{"extra": "z"})
		}, "extra"},
		{"removed file", func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, "file")); err != nil {
				t.Fatal(err)
			}
		}, "file"},
		{"manifest with matching rename", func(t *testing.T, dir string) {
			if err := os.Rename(filepath.Join(dir, "file"), filepath.Join(dir, "renamed")); err != nil {
				t.Fatal(err)
			}
			cf := &checksum.ChecksumFile{}
			if err := cf.Compute(dir); err != nil {
				t.Fatal(err)
			}
			if err := cf.Save(dir); err != nil {
				t.Fatal(err)
			}
		}, ".bundle/SHA256SUM.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"file": "x"})
			if _, err := Create(dir, ""); err != nil {
				t.Fatal(err)
			}
			tt.tamper(t, dir)

			ok, corrupted, err := Verify(dir)
			if err != nil {
				t.Fatalf("Verify error: %v", err)
			}
			if ok || !slices.Contains(corrupted, tt.want) {
				t.Errorf("Verify = %v %q, want %q reported", ok, corrupted, tt.want)
			}
		})
	}
}

func TestVerify_Errors(t *testing.T) {
	t.Run("not a bundle", func(t *testing.T) {
		if _, _, err := Verify(t.TempDir()); !errors.Is(err, utils.ErrNotABundle) {
			t.Errorf("err = %v, want ErrNotABundle", err)
		}
	})
	t.Run("missing manifest", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := Create(dir, ""); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(dir, ".bundle", "SHA256SUM.txt")); err != nil {
			t.Fatal(err)
		}
		if _, _, err := Verify(dir); !errors.Is(err, utils.ErrIncompleteBundle) {
			t.Errorf("err = %v, want ErrIncompleteBundle", err)
		}
	})
	t.Run("invalid checksum in metadata", func(t *testing.T) {
		dir := t.TempDir()
		b, err := Create(dir, "")
		if err != nil {
			t.Fatal(err)
		}
		b.Metadata.BundleChecksum = "../../escaped"
		if err := b.Metadata.Save(dir); err != nil {
			t.Fatal(err)
		}
		if _, _, err := Verify(dir); !errors.Is(err, utils.ErrCorruptedBundle) {
			t.Errorf("err = %v, want ErrCorruptedBundle", err)
		}
	})
}

func TestVerify_AcceptsLegacyBundle(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"a file": "x"})
	b, err := Create(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	makeLegacy(t, dir, b)

	ok, corrupted, err := Verify(dir)
	if err != nil || !ok {
		t.Fatalf("Verify = %v %v %v, want valid", ok, corrupted, err)
	}
}

func TestMigrate(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"file one": "x", "two": "y"})
	b, err := Create(dir, "Title")
	if err != nil {
		t.Fatal(err)
	}
	b.Tags.Add("keep")
	if err := b.Tags.Save(dir); err != nil {
		t.Fatal(err)
	}
	wantChecksum := b.Metadata.BundleChecksum
	created := b.Metadata.CreatedAt
	makeLegacy(t, dir, b)

	got, changed, err := Migrate(dir)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	if !changed || got != wantChecksum {
		t.Errorf("Migrate = %s %v, want %s true", got, changed, wantChecksum)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Metadata.Version != metadata.CurrentVersion || loaded.Metadata.BundleChecksum != wantChecksum {
		t.Errorf("metadata not upgraded: %+v", loaded.Metadata)
	}
	if loaded.Metadata.Title != "Title" || !loaded.Metadata.CreatedAt.Equal(created) || !slices.Contains(loaded.Tags.List(), "keep") {
		t.Errorf("migration lost data: %+v tags=%v", loaded.Metadata, loaded.Tags.List())
	}
	if ok, corrupted, err := Verify(dir); err != nil || !ok {
		t.Errorf("Verify after migrate = %v %v %v", ok, corrupted, err)
	}

	// A second run changes nothing
	got, changed, err = Migrate(dir)
	if err != nil || changed || got != wantChecksum {
		t.Errorf("second Migrate = %s %v %v", got, changed, err)
	}
}

func TestMigrate_AddsFilesSkippedByVersion1(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"keep": "x"})
	b, err := Create(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	makeLegacy(t, dir, b)
	writeFiles(t, dir, map[string]string{"my.bundle.txt": "skipped"})

	if _, _, err := Migrate(dir); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Files.Records) != 2 {
		t.Errorf("records = %v, want 2", loaded.Files.Records)
	}
	if want := int64(len("x") + len("skipped")); loaded.State.SizeBytes != want {
		t.Errorf("size = %d, want %d", loaded.State.SizeBytes, want)
	}
	if ok, corrupted, err := Verify(dir); err != nil || !ok {
		t.Errorf("Verify after migrate = %v %v %v", ok, corrupted, err)
	}
}

func TestMigrate_RefusesCorruptedBundles(t *testing.T) {
	tests := []struct {
		name   string
		tamper func(t *testing.T, dir string)
	}{
		{"modified file", func(t *testing.T, dir string) {
			writeFiles(t, dir, map[string]string{"file": "changed"})
		}},
		{"unlisted file", func(t *testing.T, dir string) {
			writeFiles(t, dir, map[string]string{"extra": "z"})
		}},
		{"edited manifest", func(t *testing.T, dir string) {
			writeFiles(t, dir, map[string]string{"extra": "z"})
			cf := &checksum.ChecksumFile{}
			if err := cf.Compute(dir); err != nil {
				t.Fatal(err)
			}
			if err := cf.Save(dir); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"file": "x"})
			b, err := Create(dir, "")
			if err != nil {
				t.Fatal(err)
			}
			legacy := makeLegacy(t, dir, b)
			tt.tamper(t, dir)

			if _, _, err := Migrate(dir); !errors.Is(err, utils.ErrCorruptedBundle) {
				t.Fatalf("err = %v, want ErrCorruptedBundle", err)
			}
			meta, err := metadata.Load(dir)
			if err != nil {
				t.Fatal(err)
			}
			if meta.Version != metadata.VersionLegacy || meta.BundleChecksum != legacy {
				t.Errorf("refused migration changed metadata: %+v", meta)
			}
		})
	}
}

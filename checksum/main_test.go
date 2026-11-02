package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestComputeFileSHA256(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	checksum, err := ComputeFileSHA256(testFile)
	if err != nil {
		t.Fatalf("ComputeFileSHA256() error = %v", err)
	}

	if len(checksum) != 64 {
		t.Errorf("checksum length = %d, want 64", len(checksum))
	}
}

func TestComputeBundleChecksum_Deterministic(t *testing.T) {
	// Run 100 times with shuffled order
	records := []ChecksumRecord{
		{Checksum: strings.Repeat("a", 64), FilePath: "a.txt"},
		{Checksum: strings.Repeat("b", 64), FilePath: "dir/b.txt"},
		{Checksum: strings.Repeat("c", 64), FilePath: "c d.txt"},
	}

	var firstResult string
	for i := 0; i < 100; i++ {
		// Shuffle
		shuffled := make([]ChecksumRecord, len(records))
		copy(shuffled, records)
		rand.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		result := ComputeBundleChecksum(shuffled)

		if i == 0 {
			firstResult = result
		} else if result != firstResult {
			t.Errorf("iteration %d: checksum = %s, want %s (not deterministic)", i, result, firstResult)
		}
	}

	if len(firstResult) != 64 {
		t.Errorf("bundle checksum length = %d, want 64", len(firstResult))
	}
}

func TestComputeBundleChecksum_CoversPathsAndContents(t *testing.T) {
	sumA := strings.Repeat("a", 64)
	sumB := strings.Repeat("b", 64)
	base := ComputeBundleChecksum([]ChecksumRecord{{Checksum: sumA, FilePath: "one"}, {Checksum: sumB, FilePath: "two"}})

	tests := []struct {
		name    string
		records []ChecksumRecord
	}{
		{"renamed file", []ChecksumRecord{{Checksum: sumA, FilePath: "uno"}, {Checksum: sumB, FilePath: "two"}}},
		{"swapped contents", []ChecksumRecord{{Checksum: sumB, FilePath: "one"}, {Checksum: sumA, FilePath: "two"}}},
		{"removed file", []ChecksumRecord{{Checksum: sumA, FilePath: "one"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ComputeBundleChecksum(tt.records) == base {
				t.Errorf("checksum did not change")
			}
		})
	}

	// A newline in a name must not be able to imitate a second record
	forged := ComputeBundleChecksum([]ChecksumRecord{{Checksum: sumA, FilePath: "one\n" + sumB + "  ./two"}})
	if forged == base {
		t.Errorf("file name forged a manifest line")
	}
}

func TestComputeLegacyBundleChecksum_Known(t *testing.T) {
	// SHA256 of "abc\ndef": the version 1 value must never change
	got := ComputeLegacyBundleChecksum([]string{"def", "abc"})
	want := sha256.Sum256([]byte("abc\ndef"))
	if got != hex.EncodeToString(want[:]) {
		t.Errorf("legacy checksum = %s", got)
	}
}

func TestManifest_RoundTripAndSHA256Sum(t *testing.T) {
	names := []string{
		"plain.txt",
		"with space.txt",
		" leading and trailing ",
		"back\\slash",
		"new\nline",
		"carriage\rreturn",
		"my.bundle.txt",
		"photos.bundle/file",
		"sub dir/nested file",
	}

	dir := t.TempDir()
	for _, name := range names {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, ".bundle"), 0755); err != nil {
		t.Fatal(err)
	}

	cf := &ChecksumFile{}
	if err := cf.Compute(dir); err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	if len(cf.Records) != len(names) {
		t.Fatalf("got %d records, want %d: %v", len(cf.Records), len(names), cf.Records)
	}
	if err := cf.Save(dir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded := &ChecksumFile{}
	if err := loaded.Load(dir); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	corrupted, err := loaded.Verify(dir)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if len(corrupted) != 0 {
		t.Errorf("corrupted = %q, want none", corrupted)
	}

	// The bundle checksum is the SHA256 of the file on disk
	manifest, err := os.ReadFile(filepath.Join(dir, ".bundle", "SHA256SUM.txt"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(manifest)
	if got := ComputeBundleChecksum(loaded.Records); got != hex.EncodeToString(sum[:]) {
		t.Errorf("bundle checksum %s is not the SHA256 of the manifest", got)
	}

	// The manifest stays compatible with sha256sum(1)
	if _, err := exec.LookPath("sha256sum"); err != nil {
		t.Skip("sha256sum not installed")
	}
	cmd := exec.Command("sha256sum", "--check", "--strict", ".bundle/SHA256SUM.txt")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("sha256sum --check failed: %v\n%s", err, out)
	}
}

func TestLoad_RejectsBadManifests(t *testing.T) {
	sum := strings.Repeat("a", 64)
	tests := []struct {
		name string
		line string
	}{
		{"garbage", "garbage"},
		{"blank", ""},
		{"short checksum", "abc  ./file"},
		{"single space", sum + " ./file"},
		{"non-hex checksum", strings.Repeat("z", 64) + "  ./file"},
		{"uppercase checksum", strings.Repeat("A", 64) + "  ./file"},
		{"parent traversal", sum + "  ./../../etc/passwd"},
		{"absolute path", sum + "  /etc/passwd"},
		{"bad escape", "\\" + sum + "  ./a\\x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, ".bundle"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, ".bundle", "SHA256SUM.txt"), []byte(tt.line+"\n"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := (&ChecksumFile{}).Load(dir); err == nil {
				t.Errorf("Load() accepted %q", tt.line)
			}
		})
	}
}

func TestVerify_ReportsUntrackedAndReplacedFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other"), []byte("y"), 0644); err != nil {
		t.Fatal(err)
	}
	cf := &ChecksumFile{}
	if err := cf.Compute(dir); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "extra"), []byte("z"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "other")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "other"), 0755); err != nil {
		t.Fatal(err)
	}

	corrupted, err := cf.Verify(dir)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if want := []string{"extra", "other"}; !slices.Equal(corrupted, want) {
		t.Errorf("corrupted = %q, want %q", corrupted, want)
	}
}

func TestChecksumFile_ComputeAndVerify(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create .bundle directory
	bundleDir := filepath.Join(tmpDir, ".bundle")
	if err := os.Mkdir(bundleDir, 0755); err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}

	// Compute checksums
	cf := &ChecksumFile{}
	if err := cf.Compute(tmpDir); err != nil {
		t.Fatalf("Compute() error = %v", err)
	}

	if len(cf.Records) != 2 {
		t.Errorf("got %d records, want 2", len(cf.Records))
	}

	// Save
	if err := cf.Save(tmpDir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load
	cf2 := &ChecksumFile{}
	if err := cf2.Load(tmpDir); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cf2.Records) != len(cf.Records) {
		t.Errorf("loaded %d records, want %d", len(cf2.Records), len(cf.Records))
	}

	// Verify
	corrupted, err := cf2.Verify(tmpDir)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if len(corrupted) != 0 {
		t.Errorf("got %d corrupted files, want 0", len(corrupted))
	}

	// Modify file and verify again
	if err := os.WriteFile(file1, []byte("modified"), 0644); err != nil {
		t.Fatalf("failed to modify test file: %v", err)
	}
	corrupted, err = cf2.Verify(tmpDir)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if len(corrupted) != 1 {
		t.Errorf("got %d corrupted files, want 1", len(corrupted))
	}
}

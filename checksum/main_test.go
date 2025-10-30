package checksum

import (
	"math/rand"
	"os"
	"path/filepath"
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
	checksums := []string{
		"a1b2c3d4e5f67890123456789012345678901234567890123456789012345678",
		"b2c3d4e5f67890123456789012345678901234567890123456789012345678a1",
		"c3d4e5f67890123456789012345678901234567890123456789012345678a1b2",
	}

	var firstResult string
	for i := 0; i < 100; i++ {
		// Shuffle
		shuffled := make([]string, len(checksums))
		copy(shuffled, checksums)
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

// Package checksum provides SHA256 checksum computation and verification for
// bundle files. It supports deterministic bundle checksums and streaming I/O
// for efficient handling of large files.
//
// The package computes individual file checksums and combines them into a
// deterministic bundle checksum by sorting and hashing all file checksums.
// This ensures the same set of files always produces the same bundle checksum,
// regardless of scan order.
//
// Example usage:
//
//	// Compute checksums for a directory
//	files := &checksum.ChecksumFile{}
//	err := files.Compute("/path/to/files")
//
//	// Save to .bundle/SHA256SUM.txt
//	err = files.Save("/path/to/bundle")
//
//	// Verify integrity
//	corrupted, err := files.Verify("/path/to/bundle")
//	if len(corrupted) > 0 {
//	    fmt.Printf("Corrupted: %v\n", corrupted)
//	}
//
//	// Compute bundle checksum
//	checksums := []string{"abc123...", "def456..."}
//	bundleChecksum := checksum.ComputeBundleChecksum(checksums)
package checksum

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ChecksumRecord represents a single file checksum entry.
//
// Each record maps a file's relative path to its SHA256 checksum.
// Relative paths are normalized to use forward slashes and are relative
// to the bundle root directory.
//
// Example:
//
//	record := ChecksumRecord{
//	    Checksum: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
//	    FilePath: "documents/report.pdf",
//	}
type ChecksumRecord struct {
	Checksum string // SHA256 hash (64 hex characters)
	FilePath string // Relative path from bundle root
}

// ChecksumFile represents the entire SHA256SUM.txt file.
//
// It contains all checksum records for files in a bundle. Records are
// stored sorted by checksum for deterministic output.
//
// Example:
//
//	files := &ChecksumFile{
//	    Records: []ChecksumRecord{
//	        {Checksum: "abc123...", FilePath: "file1.txt"},
//	        {Checksum: "def456...", FilePath: "file2.txt"},
//	    },
//	}
type ChecksumFile struct {
	Records   []ChecksumRecord
	TotalSize int64 // Total size of all files in bytes
}

// ComputeBundleChecksum generates a deterministic bundle checksum from file checksums.
//
// Algorithm:
//  1. Sort checksums lexicographically
//  2. Concatenate with Unix newlines
//  3. Compute SHA256 of the concatenated string
//
// This ensures the same set of files always produces the same bundle checksum,
// regardless of the order in which files are scanned or processed.
//
// Example:
//
//	checksums := []string{
//	    "def456...",  // file2.txt
//	    "abc123...",  // file1.txt
//	    "ghi789...",  // file3.txt
//	}
//	bundleChecksum := checksum.ComputeBundleChecksum(checksums)
//	// Result: SHA256("abc123...\ndef456...\nghi789...")
//
// Parameters:
//   - checksums: slice of SHA256 checksums (64 hex characters each)
//
// Returns:
//   - string: SHA256 hash of sorted, concatenated checksums (64 hex characters)
func ComputeBundleChecksum(checksums []string) string {
	// Sort checksums for determinism
	sorted := make([]string, len(checksums))
	copy(sorted, checksums)
	sort.Strings(sorted)

	// Concatenate with Unix newlines
	combined := strings.Join(sorted, "\n")

	// Compute SHA256 of concatenated string
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// Load reads SHA256SUM.txt and parses checksum records.
//
// The file format is compatible with sha256sum(1):
//
//	<checksum>  ./<relative_path>
//
// Each line contains a 64-character hex checksum, two spaces, and a file path.
// Paths are relative to the bundle root and prefixed with "./".
//
// Example SHA256SUM.txt:
//
//	e3b0c44...  ./file1.txt
//	a1b2c3d...  ./dir/file2.pdf
//
// Example usage:
//
//	files := &checksum.ChecksumFile{}
//	err := files.Load("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Loaded %d checksums\n", len(files.Records))
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - error: if .bundle/SHA256SUM.txt cannot be read or parsed
func (cf *ChecksumFile) Load(bundlePath string) error {
	sumFile := filepath.Join(bundlePath, ".bundle", "SHA256SUM.txt")
	file, err := os.Open(sumFile)
	if err != nil {
		return err
	}
	defer file.Close()

	cf.Records = []ChecksumRecord{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			cf.Records = append(cf.Records, ChecksumRecord{
				Checksum: parts[0],
				FilePath: strings.TrimPrefix(parts[1], "./"),
			})
		}
	}
	return scanner.Err()
}

// Save writes checksums to SHA256SUM.txt in sorted order.
//
// Records are sorted by checksum for deterministic output. The file format
// is compatible with sha256sum(1) for verification.
//
// Example:
//
//	files := &checksum.ChecksumFile{}
//	files.Compute("/path/to/files")
//	err := files.Save("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Output format:
//
//	<checksum>  ./<relative_path>
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - error: if .bundle/SHA256SUM.txt cannot be created or written
func (cf *ChecksumFile) Save(bundlePath string) error {
	sumFile := filepath.Join(bundlePath, ".bundle", "SHA256SUM.txt")

	// Sort by checksum for determinism
	sort.Slice(cf.Records, func(i, j int) bool {
		return cf.Records[i].Checksum < cf.Records[j].Checksum
	})

	file, err := os.Create(sumFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, record := range cf.Records {
		fmt.Fprintf(writer, "%s  ./%s\n", record.Checksum, record.FilePath)
	}
	return writer.Flush()
}

// Compute scans a directory and computes checksums for all files.
//
// It walks the directory tree, excluding the .bundle/ subdirectory, and computes
// SHA256 checksums for all regular files using streaming I/O. Symlinks are
// not followed.
//
// Example:
//
//	files := &checksum.ChecksumFile{}
//	err := files.Compute("/path/to/files")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Computed %d checksums\n", len(files.Records))
//	for _, record := range files.Records {
//	    fmt.Printf("%s  %s\n", record.Checksum, record.FilePath)
//	}
//
// Parameters:
//   - bundlePath: absolute or relative path to the directory to scan
//
// Returns:
//   - error: if directory cannot be walked or checksums cannot be computed
func (cf *ChecksumFile) Compute(bundlePath string) error {
	cf.Records = []ChecksumRecord{}
	cf.TotalSize = 0

	err := filepath.Walk(bundlePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .bundle subdirectory
		if info.IsDir() {
			if info.Name() == ".bundle" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip if path contains .bundle (for nested cases)
		if strings.Contains(path, ".bundle") {
			return nil
		}

		// Compute checksum
		checksum, err := ComputeFileSHA256(path)
		if err != nil {
			return fmt.Errorf("failed to compute checksum for %s: %w", path, err)
		}

		// Get relative path
		relPath, err := filepath.Rel(bundlePath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		cf.Records = append(cf.Records, ChecksumRecord{
			Checksum: checksum,
			FilePath: relPath,
		})
		
		// Track total size
		cf.TotalSize += info.Size()

		return nil
	})

	return err
}

// Verify recomputes checksums and compares against stored values.
//
// It recomputes the SHA256 checksum for each file and compares it against
// the stored checksum. Files that are missing or have mismatched checksums
// are returned in the corrupted list.
//
// Example:
//
//	files := &checksum.ChecksumFile{}
//	files.Load("/path/to/bundle")
//	corrupted, err := files.Verify("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if len(corrupted) > 0 {
//	    fmt.Printf("Corrupted files:\n")
//	    for _, path := range corrupted {
//	        fmt.Printf("  %s\n", path)
//	    }
//	} else {
//	    fmt.Println("All files verified")
//	}
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - []string: list of relative paths to corrupted or missing files
//   - error: if checksums cannot be computed or files cannot be read
func (cf *ChecksumFile) Verify(bundlePath string) ([]string, error) {
	corrupted := []string{}

	for _, record := range cf.Records {
		filePath := filepath.Join(bundlePath, record.FilePath)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			corrupted = append(corrupted, record.FilePath)
			continue
		}

		// Recompute checksum
		checksum, err := ComputeFileSHA256(filePath)
		if err != nil {
			return nil, err
		}

		// Compare
		if checksum != record.Checksum {
			corrupted = append(corrupted, record.FilePath)
		}
	}

	return corrupted, nil
}

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

// ChecksumRecord represents a single file checksum entry
type ChecksumRecord struct {
	Checksum string // SHA256 hash (64 hex characters)
	FilePath string // Relative path from bundle root
}

// ChecksumFile represents the entire SHA256SUM.txt file
type ChecksumFile struct {
	Records []ChecksumRecord
}

// ComputeBundleChecksum generates a deterministic bundle checksum from file checksums
// Algorithm: sort checksums lexicographically, concatenate with newlines, hash result
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

// Load reads SHA256SUM.txt and parses checksum records
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

// Save writes checksums to SHA256SUM.txt in sorted order
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

// Compute scans a directory and computes checksums for all files
func (cf *ChecksumFile) Compute(bundlePath string) error {
	cf.Records = []ChecksumRecord{}

	err := filepath.Walk(bundlePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and .bundle subdirectory
		if info.IsDir() || strings.Contains(path, ".bundle") {
			if strings.Contains(path, ".bundle") {
				return filepath.SkipDir
			}
			return nil
		}

		// Compute checksum
		checksum, err := ComputeFileSHA256(path)
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(bundlePath, path)
		if err != nil {
			return err
		}

		cf.Records = append(cf.Records, ChecksumRecord{
			Checksum: checksum,
			FilePath: relPath,
		})

		return nil
	})

	return err
}

// Verify recomputes checksums and compares against stored values
// Returns list of corrupted file paths
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

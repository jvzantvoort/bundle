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
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// ComputeFileSHA256 computes the SHA256 checksum of a file using streaming I/O.
//
// It uses streaming I/O to avoid loading the entire file into memory, making it
// suitable for large files. The file is read in chunks and hashed incrementally.
//
// Example:
//
//	checksum, err := checksum.ComputeFileSHA256("/path/to/largefile.iso")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("SHA256: %s\n", checksum)
//
// Parameters:
//   - filePath: absolute or relative path to the file
//
// Returns:
//   - string: SHA256 checksum as 64 hex characters
//   - error: if file cannot be opened or read
func ComputeFileSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

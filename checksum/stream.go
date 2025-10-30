package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// ComputeFileSHA256 computes the SHA256 checksum of a file using streaming I/O
// to avoid loading the entire file into memory. Suitable for large files.
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

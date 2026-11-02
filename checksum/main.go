// Package checksum provides SHA256 checksum computation and verification for
// bundle files. It supports deterministic bundle checksums and streaming I/O
// for efficient handling of large files.
//
// The package computes individual file checksums and combines them into a
// deterministic bundle checksum by hashing the manifest, which lists every file
// path with its checksum, sorted by path. The same files under the same paths
// always produce the same bundle checksum, regardless of scan order.
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
//	bundleChecksum := checksum.ComputeBundleChecksum(files.Records)
package checksum

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
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
// stored sorted by file path for deterministic output.
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

// ValidSHA256 reports whether s is a lowercase hexadecimal SHA256 checksum.
func ValidSHA256(s string) bool {
	if len(s) != sha256.Size*2 || strings.ToLower(s) != s {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// ComputeBundleChecksum generates a deterministic bundle checksum from records.
//
// Algorithm:
//  1. Sort records by file path (byte-wise)
//  2. Render them as SHA256SUM.txt lines
//  3. Compute SHA256 of the rendered manifest
//
// The result covers both file names and contents, and equals the SHA256 of
// .bundle/SHA256SUM.txt, so it can be checked with sha256sum(1). Renaming a file
// changes the bundle checksum; scan order does not.
//
// Example:
//
//	records := []checksum.ChecksumRecord{
//	    {Checksum: "def456...", FilePath: "file2.txt"},
//	    {Checksum: "abc123...", FilePath: "file1.txt"},
//	}
//	bundleChecksum := checksum.ComputeBundleChecksum(records)
//	// Result: SHA256("abc123...  ./file1.txt\ndef456...  ./file2.txt\n")
//
// Parameters:
//   - records: checksum records for all files in the bundle
//
// Returns:
//   - string: SHA256 hash of the manifest (64 hex characters)
func ComputeBundleChecksum(records []ChecksumRecord) string {
	hash := sha256.Sum256(renderManifest(records))
	return hex.EncodeToString(hash[:])
}

// ComputeLegacyBundleChecksum generates the checksum used by version 1 bundles.
//
// It hashes the sorted, newline-joined file checksums and ignores file names.
// It exists only to verify and migrate version 1 bundles; new bundles use
// ComputeBundleChecksum.
//
// Parameters:
//   - checksums: slice of SHA256 checksums (64 hex characters each)
//
// Returns:
//   - string: SHA256 hash of sorted, concatenated checksums (64 hex characters)
func ComputeLegacyBundleChecksum(checksums []string) string {
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

// manifestEscaper escapes file names the way sha256sum(1) does.
var manifestEscaper = strings.NewReplacer("\\", "\\\\", "\n", "\\n", "\r", "\\r")

// renderManifest renders records as SHA256SUM.txt content, sorted by file path.
func renderManifest(records []ChecksumRecord) []byte {
	sorted := make([]ChecksumRecord, len(records))
	copy(sorted, records)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].FilePath < sorted[j].FilePath
	})

	var manifest strings.Builder
	for _, record := range sorted {
		if strings.ContainsAny(record.FilePath, "\\\n\r") {
			manifest.WriteString("\\")
		}
		manifest.WriteString(record.Checksum)
		manifest.WriteString("  ./")
		manifest.WriteString(manifestEscaper.Replace(record.FilePath))
		manifest.WriteString("\n")
	}
	return []byte(manifest.String())
}

// parseManifestLine parses a single SHA256SUM.txt line.
func parseManifestLine(line string) (ChecksumRecord, error) {
	escaped := strings.HasPrefix(line, "\\")
	line = strings.TrimPrefix(line, "\\")

	const hashLen = 64
	if len(line) < hashLen+2 || line[hashLen:hashLen+2] != "  " {
		return ChecksumRecord{}, fmt.Errorf("expected \"<checksum>  <path>\"")
	}
	sum := line[:hashLen]
	if _, err := hex.DecodeString(sum); err != nil || strings.ToLower(sum) != sum {
		return ChecksumRecord{}, fmt.Errorf("invalid checksum %q", sum)
	}

	path := strings.TrimPrefix(line[hashLen+2:], "./")
	if escaped {
		var err error
		if path, err = unescapePath(path); err != nil {
			return ChecksumRecord{}, err
		}
	}
	if path == "" || !filepath.IsLocal(filepath.FromSlash(path)) {
		return ChecksumRecord{}, fmt.Errorf("path %q is outside the bundle", path)
	}
	return ChecksumRecord{Checksum: sum, FilePath: path}, nil
}

// unescapePath reverses manifestEscaper.
func unescapePath(path string) (string, error) {
	var out strings.Builder
	for i := 0; i < len(path); i++ {
		if path[i] != '\\' {
			out.WriteByte(path[i])
			continue
		}
		i++
		if i == len(path) {
			return "", fmt.Errorf("invalid escape in path %q", path)
		}
		switch path[i] {
		case '\\':
			out.WriteByte('\\')
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		default:
			return "", fmt.Errorf("invalid escape in path %q", path)
		}
	}
	return out.String(), nil
}

// walkFiles calls fn for every non-directory entry below root, skipping .bundle
// metadata directories. Only a directory named exactly .bundle is skipped, so
// names such as photos.bundle or my.bundle.txt are regular bundle content.
func walkFiles(root string, fn func(path, relPath string, info fs.FileInfo) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if path != root && info.Name() == ".bundle" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		return fn(path, filepath.ToSlash(relPath), info)
	})
}

// Load reads SHA256SUM.txt and parses checksum records.
//
// The file format is compatible with sha256sum(1):
//
//	<checksum>  ./<relative_path>
//
// Each line contains a 64-character hex checksum, two spaces, and a file path.
// Paths are relative to the bundle root and prefixed with "./". As in sha256sum(1),
// a line starting with a backslash holds a path with \\, \n and \r escaped.
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
//   - error: if .bundle/SHA256SUM.txt cannot be read, has a malformed line, or
//     lists a path outside the bundle
func (cf *ChecksumFile) Load(bundlePath string) error {
	sumFile := filepath.Join(bundlePath, ".bundle", "SHA256SUM.txt")
	file, err := os.Open(sumFile)
	if err != nil {
		return err
	}
	defer file.Close()

	cf.Records = []ChecksumRecord{}
	scanner := bufio.NewScanner(file)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		record, err := parseManifestLine(scanner.Text())
		if err != nil {
			return fmt.Errorf("SHA256SUM.txt line %d: %w", lineNo, err)
		}
		cf.Records = append(cf.Records, record)
	}
	return scanner.Err()
}

// Save writes checksums to SHA256SUM.txt in sorted order.
//
// Records are sorted by file path for deterministic output. The file format
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

	// Sort by file path for determinism
	sort.Slice(cf.Records, func(i, j int) bool {
		return cf.Records[i].FilePath < cf.Records[j].FilePath
	})

	return os.WriteFile(sumFile, renderManifest(cf.Records), 0644)
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

	return walkFiles(bundlePath, func(path, relPath string, info fs.FileInfo) error {
		// Compute checksum
		checksum, err := ComputeFileSHA256(path)
		if err != nil {
			return fmt.Errorf("failed to compute checksum for %s: %w", path, err)
		}

		cf.Records = append(cf.Records, ChecksumRecord{
			Checksum: checksum,
			FilePath: relPath,
		})

		// Track total size
		cf.TotalSize += info.Size()

		return nil
	})
}

// Verify recomputes checksums and compares the bundle against its manifest.
//
// It recomputes the SHA256 checksum for each listed file and compares it against
// the stored checksum. Files that are missing or have mismatched checksums, and
// files that exist but are not listed, are returned in the corrupted list.
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
//   - []string: sorted relative paths of corrupted, missing or unlisted files
//   - error: if checksums cannot be computed or files cannot be read
func (cf *ChecksumFile) Verify(bundlePath string) ([]string, error) {
	corrupted, err := cf.VerifyRecords(bundlePath)
	if err != nil {
		return nil, err
	}

	untracked, err := cf.Untracked(bundlePath)
	if err != nil {
		return nil, err
	}

	corrupted = append(corrupted, untracked...)
	sort.Strings(corrupted)
	return corrupted, nil
}

// VerifyRecords checks only the files listed in the manifest.
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - []string: relative paths of listed files that are missing or modified
//   - error: if checksums cannot be computed or files cannot be read
func (cf *ChecksumFile) VerifyRecords(bundlePath string) ([]string, error) {
	corrupted := []string{}

	for _, record := range cf.Records {
		filePath := filepath.Join(bundlePath, filepath.FromSlash(record.FilePath))

		// Check if file exists
		info, err := os.Stat(filePath)
		if os.IsNotExist(err) || (err == nil && info.IsDir()) {
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

// Untracked lists files in the bundle that the manifest does not mention.
//
// Parameters:
//   - bundlePath: absolute or relative path to the bundle directory
//
// Returns:
//   - []string: relative paths of files not listed in the manifest
//   - error: if the directory cannot be walked
func (cf *ChecksumFile) Untracked(bundlePath string) ([]string, error) {
	tracked := make(map[string]struct{}, len(cf.Records))
	for _, record := range cf.Records {
		tracked[record.FilePath] = struct{}{}
	}

	untracked := []string{}
	err := walkFiles(bundlePath, func(_, relPath string, _ fs.FileInfo) error {
		if _, ok := tracked[relPath]; !ok {
			untracked = append(untracked, relPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return untracked, nil
}

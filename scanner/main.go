// Package scanner provides directory traversal utilities for bundle operations.
//
// It implements efficient directory scanning with automatic exclusion of the
// .bundle/ metadata directory. Supports both regular file scanning and symlink
// following.
//
// Example usage:
//
//	// Scan directory (exclude .bundle/)
//	files, err := scanner.ScanDirectory("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, file := range files {
//	    fmt.Println(file)
//	}
//
//	// Scan with symlink following
//	files, err = scanner.ScanWithSymlinks("/path/to/bundle")
package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// ScanDirectory walks a directory tree and returns all file paths, excluding .bundle/.
//
// It performs a recursive walk of the directory tree, collecting regular files
// only. The .bundle/ directory is skipped entirely. Symlinks are not followed.
//
// Example:
//
//	files, err := scanner.ScanDirectory("/path/to/photos")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Found %d files\n", len(files))
//	for _, file := range files {
//	    fmt.Println(file)
//	}
//
// Parameters:
//   - rootPath: absolute or relative path to the directory to scan
//
// Returns:
//   - []string: slice of absolute paths to regular files
//   - error: if directory cannot be walked or accessed
func ScanDirectory(rootPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .bundle directory entirely
		if info.IsDir() && info.Name() == ".bundle" {
			return filepath.SkipDir
		}

		// Skip directories, only collect files
		if info.IsDir() {
			return nil
		}

		// Skip if path contains .bundle (in case of nested)
		if strings.Contains(path, ".bundle") {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// ScanWithSymlinks is like ScanDirectory but follows symlinks.
//
// It walks the directory tree and follows symbolic links to their targets.
// Broken symlinks are skipped. The .bundle/ directory is still excluded.
// Relative symlinks are resolved relative to their containing directory.
//
// Example:
//
//	files, err := scanner.ScanWithSymlinks("/path/to/photos")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, file := range files {
//	    fmt.Println(file)  // May include files outside rootPath (via symlinks)
//	}
//
// Parameters:
//   - rootPath: absolute or relative path to the directory to scan
//
// Returns:
//   - []string: slice of absolute paths to files (including symlink targets)
//   - error: if directory cannot be walked or accessed
func ScanWithSymlinks(rootPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .bundle directory
		if info.IsDir() && info.Name() == ".bundle" {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		if strings.Contains(path, ".bundle") {
			return nil
		}

		// Follow symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				// Skip broken symlinks
				return nil
			}
			// Make target absolute if relative
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(path), target)
			}
			files = append(files, target)
		} else {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

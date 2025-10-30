package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// ScanDirectory walks a directory tree and returns all file paths, excluding .bundle/
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

// ScanWithSymlinks is like ScanDirectory but follows symlinks
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

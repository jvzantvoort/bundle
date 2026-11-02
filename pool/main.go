// Package pool provides pool management for centralized bundle storage.
//
// A pool is a centralized location where bundles can be imported and stored.
// Pools are configured in the application configuration file and can have
// multiple destinations.
//
// Example configuration (~/.config/bundle/config.yaml):
//
//	pools:
//	  default:
//	    root: /mnt/bundles
//	    title: Default Bundle Pool
//	  backup:
//	    root: /backup/bundles
//	    title: Backup Pool
//
// Example usage:
//
//	// Get pool by name
//	pool, err := pool.GetPool("default")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Import bundle to pool
//	err = pool.Import("/path/to/bundle", false)
//
//	// List all bundles in pool
//	bundles, err := pool.ListBundles()
package pool

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/lock"
	"github.com/jvzantvoort/bundle/metadata"
	"github.com/jvzantvoort/bundle/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// stagingPrefix names the hidden directories that hold imports in progress.
const stagingPrefix = ".import-"

// Pool represents a centralized bundle storage location.
//
// A pool is a directory where bundles are stored with their checksums
// as directory names for content-addressable storage.
//
// Example:
//
//	pool := &Pool{
//	    Root:  "/mnt/bundles",
//	    Title: "Production Pool",
//	}
type Pool struct {
	Root  string // Root directory for bundle storage
	Title string // Human-readable pool title
}

// GetPool retrieves a pool configuration by name.
//
// It reads from the application configuration (viper) and returns
// the pool configuration. Returns error if pool is not found or
// configuration is invalid.
//
// Example:
//
//	pool, err := pool.GetPool("default")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Pool root: %s\n", pool.Root)
//
// Parameters:
//   - name: pool name from configuration
//
// Returns:
//   - *Pool: pool configuration
//   - error: if pool not found or invalid
func GetPool(name string) (*Pool, error) {
	log.Debugf("GetPool called with name: %s", name)

	if !viper.IsSet("pools." + name) {
		log.Debugf("Pool '%s' not found in configuration", name)
		log.Debugf("Available pools: %v", viper.GetStringMap("pools"))
		return nil, fmt.Errorf("pool '%s' not found in configuration", name)
	}

	root := viper.GetString(fmt.Sprintf("pools.%s.root", name))
	log.Debugf("Pool '%s' root from config: %s", name, root)

	if root == "" {
		log.Debugf("Pool '%s' has empty root directory", name)
		return nil, fmt.Errorf("pool '%s' has no root directory configured", name)
	}

	title := viper.GetString(fmt.Sprintf("pools.%s.title", name))
	if title == "" {
		title = name // Use name as fallback
		log.Debugf("Pool '%s' has no title, using name as fallback", name)
	} else {
		log.Debugf("Pool '%s' title from config: %s", name, title)
	}

	pool := &Pool{
		Root:  root,
		Title: title,
	}

	log.Debugf("Pool '%s' configuration loaded successfully:", name)
	log.Debugf("  Root:  %s", pool.Root)
	log.Debugf("  Title: %s", pool.Title)

	return pool, nil
}

// ListPools returns all configured pools.
//
// It reads the configuration and returns a map of pool names to Pool structs.
//
// Example:
//
//	pools, err := pool.ListPools()
//	for name, p := range pools {
//	    fmt.Printf("%s: %s (%s)\n", name, p.Title, p.Root)
//	}
//
// Returns:
//   - map[string]*Pool: map of pool names to configurations
//   - error: if configuration is invalid
func ListPools() (map[string]*Pool, error) {
	pools := make(map[string]*Pool)

	poolsConfig := viper.GetStringMap("pools")
	log.Debugf("ListPools: found %d pool(s) in configuration", len(poolsConfig))

	if len(poolsConfig) == 0 {
		log.Debugf("No pools configured")
		return pools, nil
	}

	log.Debugf("Pool names from configuration: %v", poolsConfig)

	for name := range poolsConfig {
		log.Debugf("Loading pool configuration for: %s", name)
		pool, err := GetPool(name)
		if err != nil {
			log.Debugf("Failed to load pool '%s': %v", name, err)
			return nil, fmt.Errorf("invalid pool '%s': %w", name, err)
		}
		pools[name] = pool
	}

	log.Debugf("Successfully loaded %d pool(s)", len(pools))

	return pools, nil
}

// Import copies or moves a bundle to the pool.
//
// The bundle is stored in the pool with its checksum as the directory name,
// ensuring content-addressable storage. The source is verified first and the
// copy is verified before it appears in the pool, so a failed import leaves no
// partial bundle behind. If move is true, the source bundle is removed only
// after the verified copy is in place.
//
// Version 1 bundles are refused until they are upgraded with bundle.Migrate,
// because the pool stores bundles under current checksums.
//
// Example:
//
//	pool, _ := pool.GetPool("default")
//	err := pool.Import("/path/to/bundle", false)  // Copy
//	err = pool.Import("/path/to/bundle", true)    // Move
//
// Parameters:
//   - bundlePath: path to the bundle to import
//   - move: if true, remove source after import
//
// Returns:
//   - error: if import fails; utils.ErrNotABundle, utils.ErrCorruptedBundle,
//     utils.ErrOutdatedBundle, utils.ErrBundleExists and utils.ErrInvalidPath
//     identify user errors
func (p *Pool) Import(bundlePath string, move bool) error {
	log.Debugf("Import called:")
	log.Debugf("  Pool:   %s (%s)", p.Title, p.Root)
	log.Debugf("  Source: %s", bundlePath)
	log.Debugf("  Mode:   %s", map[bool]string{true: "move", false: "copy"}[move])

	// Load bundle metadata to get checksum
	log.Debugf("Loading bundle metadata from: %s", bundlePath)
	meta, err := metadata.Load(bundlePath)
	if err != nil {
		log.Debugf("Failed to load metadata: %v", err)
		if os.IsNotExist(err) {
			return utils.ErrNotABundle
		}
		return fmt.Errorf("failed to load bundle metadata: %w", err)
	}

	log.Debugf("Bundle metadata loaded:")
	log.Debugf("  Title:    %s", meta.Title)
	log.Debugf("  Checksum: %s", meta.BundleChecksum)
	log.Debugf("  Author:   %s", meta.Author)

	// The checksum becomes a directory name, so it must be validated before use
	if err := meta.Validate(); err != nil {
		return fmt.Errorf("%w: META.json: %v", utils.ErrCorruptedBundle, err)
	}
	if meta.Version < metadata.CurrentVersion {
		return fmt.Errorf("%w: run 'bundle migrate %s' first", utils.ErrOutdatedBundle, bundlePath)
	}

	// Destination is root/checksum
	destPath := filepath.Join(p.Root, meta.BundleChecksum)
	log.Debugf("Destination path: %s", destPath)

	// Check if bundle already exists in pool
	if _, err := os.Stat(destPath); err == nil {
		log.Debugf("Bundle already exists at destination: %s", destPath)
		return fmt.Errorf("%w in pool: %s", utils.ErrBundleExists, meta.BundleChecksum)
	}

	// Ensure pool root exists
	log.Debugf("Ensuring pool root directory exists: %s", p.Root)
	if err := os.MkdirAll(p.Root, 0755); err != nil {
		log.Debugf("Failed to create pool directory: %v", err)
		return fmt.Errorf("failed to create pool directory: %w", err)
	}

	// A pool inside the bundle would be copied into itself and removed by a move
	if inside, err := isInside(p.Root, bundlePath); err != nil {
		return err
	} else if inside {
		return fmt.Errorf("%w: pool %s is inside bundle %s", utils.ErrInvalidPath, p.Root, bundlePath)
	}

	// A move deletes the source, so it must not race with another writer
	if move {
		bundleLock, err := lock.AcquireLock(bundlePath)
		if err != nil {
			return err
		}
		defer func() {
			if err := bundleLock.Release(); err != nil {
				log.Errorf("failed to release lock: %v", err)
			}
		}()
	}

	if err := verifyBundle(bundlePath); err != nil {
		return err
	}

	// Copy to a hidden staging directory in the pool, on the same filesystem as
	// the destination, so the final rename is atomic
	stagingRoot, err := os.MkdirTemp(p.Root, stagingPrefix+"*")
	if err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(stagingRoot); err != nil {
			log.Errorf("failed to remove staging directory: %v", err)
		}
	}()
	stagingPath := filepath.Join(stagingRoot, meta.BundleChecksum)

	log.Debugf("Copying bundle from %s to %s", bundlePath, stagingPath)
	if err := copyDir(bundlePath, stagingPath); err != nil {
		log.Debugf("Failed to copy bundle: %v", err)
		return fmt.Errorf("failed to copy bundle: %w", err)
	}
	log.Debugf("Bundle copied successfully")

	// The source lock must not lock the pool copy
	if err := os.Remove(filepath.Join(stagingPath, ".bundle", ".lock")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean up copy: %w", err)
	}

	if err := verifyBundle(stagingPath); err != nil {
		return fmt.Errorf("copy failed verification: %w", err)
	}

	if err := os.Rename(stagingPath, destPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w in pool: %s", utils.ErrBundleExists, meta.BundleChecksum)
		}
		return fmt.Errorf("failed to move bundle into pool: %w", err)
	}

	// If move, remove source
	if move {
		log.Debugf("Move mode: removing source directory: %s", bundlePath)
		if err := os.RemoveAll(bundlePath); err != nil {
			log.Debugf("Failed to remove source: %v", err)
			return fmt.Errorf("failed to remove source bundle: %w", err)
		}
		log.Debugf("Source directory removed successfully")
	}

	log.Debugf("Import completed successfully")
	return nil
}

// MigrationResult describes the outcome of migrating one pool entry.
type MigrationResult struct {
	Name        string // Directory name before migration
	NewChecksum string // Bundle checksum after migration, empty on failure
	Err         error  // Why the entry could not be migrated
}

// Migrate upgrades all version 1 bundles in the pool.
//
// Each bundle is migrated in place and its directory renamed to the new checksum.
// A failure does not stop the remaining bundles. If the pool already holds the
// new checksum, the entry is left in place and reported; nothing is deleted.
//
// Returns:
//   - []MigrationResult: one result per bundle that was changed or failed
//   - error: if the pool cannot be scanned
func (p *Pool) Migrate() ([]MigrationResult, error) {
	entries, err := os.ReadDir(p.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool directory: %w", err)
	}

	results := []MigrationResult{}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		bundlePath := filepath.Join(p.Root, entry.Name())
		if _, err := os.Stat(filepath.Join(bundlePath, ".bundle")); err != nil {
			log.Debugf("Skipping non-bundle entry: %s", entry.Name())
			continue
		}

		newChecksum, changed, err := p.migrateEntry(entry.Name())
		if err != nil {
			results = append(results, MigrationResult{Name: entry.Name(), Err: err})
		} else if changed {
			results = append(results, MigrationResult{Name: entry.Name(), NewChecksum: newChecksum})
		}
	}
	return results, nil
}

// migrateEntry migrates one pool directory and renames it to its checksum.
func (p *Pool) migrateEntry(name string) (string, bool, error) {
	bundlePath := filepath.Join(p.Root, name)

	newChecksum, changed, err := bundle.Migrate(bundlePath)
	if err != nil {
		return "", false, err
	}
	if newChecksum == name {
		return newChecksum, changed, nil
	}

	// An entry from an interrupted migration can reach this point without having
	// been checked by bundle.Migrate
	if err := verifyBundle(bundlePath); err != nil {
		return "", false, err
	}

	destPath := filepath.Join(p.Root, newChecksum)
	if _, err := os.Stat(destPath); err == nil {
		return "", false, fmt.Errorf("%w in pool: %s", utils.ErrBundleExists, newChecksum)
	}
	if err := os.Rename(bundlePath, destPath); err != nil {
		return "", false, fmt.Errorf("failed to rename bundle: %w", err)
	}
	return newChecksum, true, nil
}

// verifyBundle returns utils.ErrCorruptedBundle if the bundle fails verification.
func verifyBundle(path string) error {
	verified, corrupted, err := bundle.Verify(path)
	if err != nil {
		return err
	}
	if !verified {
		return fmt.Errorf("%w: %s", utils.ErrCorruptedBundle, strings.Join(corrupted, ", "))
	}
	return nil
}

// isInside reports whether path is dir itself or located below it.
func isInside(path, dir string) (bool, error) {
	resolvedPath, err := resolvePath(path)
	if err != nil {
		return false, err
	}
	resolvedDir, err := resolvePath(dir)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(resolvedDir, resolvedPath)
	if err != nil {
		return false, err
	}
	return filepath.IsLocal(rel), nil
}

// resolvePath returns the absolute path with symlinks resolved.
func resolvePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absPath)
}

// ListBundles returns all bundles in the pool.
//
// It scans the pool directory and returns metadata for all bundles found.
// Each bundle is stored as a directory named by its checksum.
//
// Example:
//
//	pool, _ := pool.GetPool("default")
//	bundles, err := pool.ListBundles()
//	for _, meta := range bundles {
//	    fmt.Printf("%s: %s\n", meta.BundleChecksum[:8], meta.Title)
//	}
//
// Returns:
//   - []*metadata.Metadata: list of bundle metadata
//   - error: if pool cannot be scanned
func (p *Pool) ListBundles() ([]*metadata.Metadata, error) {
	var bundles []*metadata.Metadata

	log.Debugf("ListBundles called for pool: %s (%s)", p.Title, p.Root)

	// Check if pool directory exists
	if _, err := os.Stat(p.Root); os.IsNotExist(err) {
		log.Debugf("Pool directory does not exist: %s", p.Root)
		return bundles, nil // Empty pool
	}

	log.Debugf("Scanning pool directory: %s", p.Root)

	// Scan pool directory
	entries, err := os.ReadDir(p.Root)
	if err != nil {
		log.Debugf("Failed to read pool directory: %v", err)
		return nil, fmt.Errorf("failed to read pool directory: %w", err)
	}

	log.Debugf("Found %d entries in pool directory", len(entries))

	// Load metadata for each bundle
	validBundles := 0
	skippedEntries := 0

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			log.Debugf("Skipping entry: %s", entry.Name())
			skippedEntries++
			continue
		}

		bundlePath := filepath.Join(p.Root, entry.Name())
		log.Debugf("Loading bundle metadata from: %s", bundlePath)

		meta, err := metadata.Load(bundlePath)
		if err != nil {
			// Skip invalid bundles
			log.Debugf("Skipping invalid bundle %s: %v", entry.Name(), err)
			skippedEntries++
			continue
		}
		if err := meta.Validate(); err != nil {
			log.Debugf("Skipping invalid bundle %s: %v", entry.Name(), err)
			skippedEntries++
			continue
		}
		if meta.BundleChecksum != entry.Name() {
			log.Debugf("Skipping bundle %s: directory name does not match checksum %s", entry.Name(), meta.BundleChecksum)
			skippedEntries++
			continue
		}

		log.Debugf("Bundle loaded: %s (%s)", meta.Title, meta.BundleChecksum[:12])
		bundles = append(bundles, meta)
		validBundles++
	}

	log.Debugf("ListBundles completed:")
	log.Debugf("  Total entries:   %d", len(entries))
	log.Debugf("  Valid bundles:   %d", validBundles)
	log.Debugf("  Skipped entries: %d", skippedEntries)

	return bundles, nil
}

// GetBundlePath returns the full path to a bundle in the pool.
//
// Parameters:
//   - checksum: bundle checksum
//
// Returns:
//   - string: full path to bundle
func (p *Pool) GetBundlePath(checksum string) string {
	return filepath.Join(p.Root, checksum)
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	// Get source info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file and flushes it to disk, because a move deletes
// the source once the copy is in place.
func copyFile(src, dst string) (err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := dstFile.Close(); err == nil {
			err = closeErr
		}
	}()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return dstFile.Sync()
}

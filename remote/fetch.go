package remote

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jvzantvoort/bundle/checksum"
	"github.com/jvzantvoort/bundle/metadata"
	"github.com/jvzantvoort/bundle/state"
	"github.com/jvzantvoort/bundle/tag"
	"github.com/jvzantvoort/bundle/utils"
	log "github.com/sirupsen/logrus"
)

// Fetch downloads a bundle into a new local directory.
//
// The server is not trusted: the manifest must hash to the requested checksum
// and stay inside the bundle before anything is written, and each file must match
// its manifest entry. The caller still has to verify the finished bundle, which
// pool.Receive does.
//
// Parameters:
//   - sum: bundle checksum
//   - dir: directory to create
//
// Returns:
//   - error: utils.ErrCorruptedBundle if what the server sent does not match sum,
//     utils.ErrOutdatedBundle if the bundle must be migrated on the server first
func (c *Client) Fetch(ctx context.Context, sum, dir string) error {
	details, err := c.GetBundle(ctx, sum)
	if err != nil {
		return err
	}
	if details.Checksum != sum {
		return fmt.Errorf("%w: server returned bundle %q for %s", utils.ErrCorruptedBundle, details.Checksum, sum)
	}
	if details.Version != metadata.CurrentVersion {
		return fmt.Errorf("%w: run 'bundle migrate --remote %s %s' first", utils.ErrOutdatedBundle, c.Name, sum)
	}

	entries, err := c.ListFiles(ctx, sum)
	if err != nil {
		return err
	}
	records, err := checkManifest(entries)
	if err != nil {
		return err
	}
	if computed := checksum.ComputeBundleChecksum(records); computed != sum {
		return fmt.Errorf("%w: manifest from the server hashes to %s", utils.ErrCorruptedBundle, computed)
	}

	if err := os.MkdirAll(filepath.Join(dir, ".bundle"), 0755); err != nil {
		return err
	}

	var totalSize int64
	for _, entry := range entries {
		log.Debugf("Downloading %s", entry.Path)
		if err := c.downloadFile(ctx, sum, dir, entry); err != nil {
			return fmt.Errorf("failed to download %s: %w", entry.Path, err)
		}
		totalSize += entry.SizeBytes
	}

	files := &checksum.ChecksumFile{Records: records, TotalSize: totalSize}
	if err := files.Save(dir); err != nil {
		return fmt.Errorf("failed to save checksums: %w", err)
	}
	bundleState := &state.State{Verified: true, LastChecked: time.Now(), Replicas: []string{}, SizeBytes: totalSize}
	if err := bundleState.Save(dir); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}
	bundleTags := &tag.Tags{Tags: []string{}}
	bundleTags.Add(details.Tags...)
	if err := bundleTags.Save(dir); err != nil {
		return fmt.Errorf("failed to save tags: %w", err)
	}
	meta := &metadata.Metadata{
		Title:          details.Title,
		CreatedAt:      details.CreatedAt,
		BundleChecksum: sum,
		Author:         details.Author,
		Version:        metadata.CurrentVersion,
	}
	if err := meta.Save(dir); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}
	return nil
}

// checkManifest refuses manifest entries that could write outside the bundle,
// overwrite its metadata or collide with each other.
func checkManifest(entries []ManifestEntry) ([]checksum.ChecksumRecord, error) {
	records := make([]checksum.ChecksumRecord, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if !safeManifestPath(entry.Path) {
			return nil, fmt.Errorf("%w: unsafe path in manifest from the server: %q", utils.ErrCorruptedBundle, entry.Path)
		}
		if _, ok := seen[entry.Path]; ok {
			return nil, fmt.Errorf("%w: duplicate path in manifest from the server: %q", utils.ErrCorruptedBundle, entry.Path)
		}
		seen[entry.Path] = struct{}{}
		if !checksum.ValidSHA256(entry.Checksum) || entry.SizeBytes < 0 {
			return nil, fmt.Errorf("%w: invalid manifest entry from the server: %q", utils.ErrCorruptedBundle, entry.Path)
		}
		records = append(records, checksum.ChecksumRecord{Checksum: entry.Checksum, FilePath: entry.Path})
	}
	return records, nil
}

// safeManifestPath reports whether a slash-separated path stays inside the bundle
// and outside every .bundle metadata directory.
func safeManifestPath(path string) bool {
	if path == "" || !filepath.IsLocal(filepath.FromSlash(path)) {
		return false
	}
	return !slices.ContainsFunc(strings.Split(path, "/"), func(segment string) bool {
		return segment == "" || segment == "." || segment == ".." || segment == ".bundle"
	})
}

// downloadFile stores one file and checks it against its manifest entry. It reads
// at most one byte more than announced, so a server cannot fill the disk.
func (c *Client) downloadFile(ctx context.Context, sum, dir string, entry ManifestEntry) (err error) {
	target := filepath.Join(dir, filepath.FromSlash(entry.Path))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	// O_EXCL also catches paths that collide on a case-insensitive filesystem
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()

	resp, err := c.send(ctx, http.MethodGet, c.poolPath("bundles", sum, "files", entry.Path), nil, nil, -1)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(resp.Body, entry.SizeBytes+1))
	if err != nil {
		return err
	}
	if written != entry.SizeBytes {
		return fmt.Errorf("%w: expected %d bytes, got %d or more", utils.ErrCorruptedBundle, entry.SizeBytes, written)
	}
	if actual := hex.EncodeToString(hash.Sum(nil)); actual != entry.Checksum {
		return fmt.Errorf("%w: expected checksum %s, got %s", utils.ErrCorruptedBundle, entry.Checksum, actual)
	}
	return nil
}

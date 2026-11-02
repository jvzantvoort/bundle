package remote

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/metadata"
	"github.com/jvzantvoort/bundle/utils"
	log "github.com/sirupsen/logrus"
)

// Results of a push.
const (
	PushStatusPushed = "pushed" // The server now holds the bundle because of this push
	PushStatusExists = "exists" // The server already held the bundle; nothing changed
)

// PushResult describes a finished push.
type PushResult struct {
	Status    string
	Checksum  string
	FileCount int
	SizeBytes int64
}

// Push uploads a local bundle to the pool.
//
// The bundle is verified first. If the server already holds it nothing is
// uploaded, and its title and tags on the server are left alone.
//
// An interrupted push can be run again. The session is created with an
// Idempotency-Key derived from the request, so a server that honours the key
// continues it and only the files it still misses are sent; a server that ignores
// the key starts over in a new session. A session the server rejected is cancelled,
// so that it does not block the next attempt until it expires.
//
// Parameters:
//   - bundlePath: path to the local bundle
//
// Returns:
//   - *PushResult: what happened
//   - error: utils.ErrNotABundle, utils.ErrCorruptedBundle, utils.ErrOutdatedBundle
//     and utils.ErrInvalidPath identify problems with the local bundle
func (c *Client) Push(ctx context.Context, bundlePath string) (*PushResult, error) {
	request, sizes, err := loadPushRequest(bundlePath)
	if err != nil {
		return nil, err
	}

	result := &PushResult{Status: PushStatusPushed, Checksum: request.Checksum, FileCount: len(request.Files)}
	for _, entry := range request.Files {
		result.SizeBytes += entry.SizeBytes
	}

	if err := c.checkLimits(ctx, request, result.SizeBytes); err != nil {
		return nil, err
	}

	session, err := c.openUpload(ctx, request)
	if err != nil {
		// The server refuses an upload of a bundle it already holds
		if hasStatus(err, http.StatusConflict) {
			if _, getErr := c.GetBundle(ctx, request.Checksum); getErr == nil {
				result.Status = PushStatusExists
				return result, nil
			}
		}
		return nil, err
	}
	if session.Status == "complete" {
		result.Status = PushStatusExists
		return result, nil
	}

	pushErr := c.uploadFiles(ctx, bundlePath, session, request, sizes)
	if pushErr == nil {
		pushErr = c.completeUpload(ctx, session.ID, request.Checksum)
	}
	if pushErr == nil {
		return result, nil
	}

	// Another client may have published the same bundle in the meantime
	if hasStatus(pushErr, http.StatusConflict) {
		if _, err := c.GetBundle(ctx, request.Checksum); err == nil {
			result.Status = PushStatusExists
			return result, nil
		}
	}

	var apiErr *APIError
	if errors.As(pushErr, &apiErr) {
		if err := c.cancelUpload(ctx, session.ID); err != nil {
			log.Debugf("failed to cancel upload %s: %v", session.ID, err)
		}
		return nil, pushErr
	}
	return nil, fmt.Errorf("%w (upload %s stays open; run push again)", pushErr, session.ID)
}

// loadPushRequest verifies a local bundle and builds its upload request.
func loadPushRequest(bundlePath string) (*createUploadRequest, map[string]int64, error) {
	if _, err := metadata.Load(bundlePath); os.IsNotExist(err) {
		return nil, nil, utils.ErrNotABundle
	}

	verified, corrupted, err := bundle.Verify(bundlePath)
	if err != nil {
		return nil, nil, err
	}
	if !verified {
		return nil, nil, fmt.Errorf("%w: %s", utils.ErrCorruptedBundle, strings.Join(corrupted, ", "))
	}

	local, err := bundle.Load(bundlePath)
	if err != nil {
		return nil, nil, err
	}
	if local.Metadata.Version < metadata.CurrentVersion {
		return nil, nil, fmt.Errorf("%w: run 'bundle migrate %s' first", utils.ErrOutdatedBundle, bundlePath)
	}
	if len(local.Files.Records) == 0 {
		return nil, nil, fmt.Errorf("%w: the server does not accept a bundle without files: %s", utils.ErrInvalidPath, bundlePath)
	}

	entries := make([]ManifestEntry, 0, len(local.Files.Records))
	sizes := make(map[string]int64, len(local.Files.Records))
	for _, record := range local.Files.Records {
		info, err := os.Stat(filepath.Join(bundlePath, filepath.FromSlash(record.FilePath)))
		if err != nil {
			return nil, nil, err
		}
		entries = append(entries, ManifestEntry{Path: record.FilePath, Checksum: record.Checksum, SizeBytes: info.Size()})
		sizes[record.FilePath] = info.Size()
	}

	tags := local.Tags.List()
	if tags == nil {
		tags = []string{}
	}
	return &createUploadRequest{
		Checksum: local.Metadata.BundleChecksum,
		Metadata: BundleMetadata{
			Title:     local.Metadata.Title,
			CreatedAt: local.Metadata.CreatedAt,
			Author:    local.Metadata.Author,
			Version:   local.Metadata.Version,
		},
		Tags:  tags,
		Files: entries,
	}, sizes, nil
}

// checkLimits refuses an upload the server has announced it will not accept.
func (c *Client) checkLimits(ctx context.Context, request *createUploadRequest, totalSize int64) error {
	capabilities, err := c.Capabilities(ctx)
	if err != nil {
		return err
	}
	if !slices.Contains(capabilities.BundleVersions, request.Metadata.Version) {
		return fmt.Errorf("%w: server does not accept version %d bundles", utils.ErrRemoteRejected, request.Metadata.Version)
	}

	limits := capabilities.UploadLimits
	if limits.MaxManifestEntries > 0 && len(request.Files) > limits.MaxManifestEntries {
		return fmt.Errorf("%w: bundle has %d files, the server accepts %d", utils.ErrRemoteRejected, len(request.Files), limits.MaxManifestEntries)
	}
	if limits.MaxBundleBytes > 0 && totalSize > limits.MaxBundleBytes {
		return fmt.Errorf("%w: bundle has %d bytes, the server accepts %d", utils.ErrRemoteRejected, totalSize, limits.MaxBundleBytes)
	}
	if limits.MaxFileBytes > 0 {
		for _, entry := range request.Files {
			if entry.SizeBytes > limits.MaxFileBytes {
				return fmt.Errorf("%w: %s has %d bytes, the server accepts %d per file", utils.ErrRemoteRejected, entry.Path, entry.SizeBytes, limits.MaxFileBytes)
			}
		}
	}
	return nil
}

// openUpload creates an upload session, or continues the one from an earlier
// attempt, and returns its current state.
func (c *Client) openUpload(ctx context.Context, request *createUploadRequest) (*upload, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(data)

	session, err := c.createUpload(ctx, request, "push-"+hex.EncodeToString(hash[:]))
	if err != nil {
		return nil, err
	}
	if session.Status == "open" || session.Status == "complete" {
		return session, nil
	}

	// The key replayed a session that was cancelled or has expired
	log.Debugf("upload %s is %s, starting a new one", session.ID, session.Status)
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return nil, err
	}
	session, err = c.createUpload(ctx, request, "push-"+hex.EncodeToString(random))
	if err != nil {
		return nil, err
	}
	if session.Status != "open" {
		return nil, fmt.Errorf("new upload %s is %s", session.ID, session.Status)
	}
	return session, nil
}

// createUpload posts the request and reads back the session, because a replayed
// response describes the session as it was when first created.
func (c *Client) createUpload(ctx context.Context, request *createUploadRequest, key string) (*upload, error) {
	var created upload
	header := http.Header{"Idempotency-Key": {key}}
	if _, err := c.sendJSON(ctx, http.MethodPost, c.poolPath("uploads"), header, request, &created); err != nil {
		return nil, err
	}
	var session upload
	if _, err := c.sendJSON(ctx, http.MethodGet, c.poolPath("uploads", created.ID), nil, nil, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// uploadFiles sends the files the session still misses.
func (c *Client) uploadFiles(ctx context.Context, bundlePath string, session *upload, request *createUploadRequest, sizes map[string]int64) error {
	checksums := make(map[string]string, len(request.Files))
	pending := make([]string, 0, len(request.Files))
	for _, entry := range request.Files {
		checksums[entry.Path] = entry.Checksum
		pending = append(pending, entry.Path)
	}

	// missing_files is optional in the API
	if session.MissingFiles != nil {
		pending = session.MissingFiles
	} else if session.FileCount > 0 && session.UploadedFileCount >= session.FileCount {
		pending = nil
	}

	for _, path := range pending {
		sum, ok := checksums[path]
		if !ok {
			return fmt.Errorf("server asked for a file that is not in the manifest: %q", path)
		}
		log.Debugf("Uploading %s", path)
		err := c.uploadFile(ctx, bundlePath, session.ID, path, sum, sizes[path])
		// Without missing_files this can be a file sent by an earlier attempt. If
		// the session itself is no longer writable, completing it reports that.
		if hasStatus(err, http.StatusConflict) && session.MissingFiles == nil {
			log.Debugf("Server already has %s", path)
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to upload %s: %w", path, err)
		}
	}
	return nil
}

// uploadFile streams one file to the session.
func (c *Client) uploadFile(ctx context.Context, bundlePath, id, path, sum string, size int64) error {
	file, err := os.Open(filepath.Join(bundlePath, filepath.FromSlash(path)))
	if err != nil {
		return err
	}
	defer file.Close()

	header := http.Header{
		"Content-Type":   {"application/octet-stream"},
		"Content-Sha256": {sum},
	}
	// Go sends a non-nil body of unknown length as chunked, even an empty one
	var body io.Reader = file
	if size == 0 {
		body = http.NoBody
	}
	resp, err := c.send(ctx, http.MethodPut, c.poolPath("uploads", id, "files", path), header, body, size)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// completeUpload asks the server to verify and publish the bundle.
func (c *Client) completeUpload(ctx context.Context, id, sum string) error {
	var published Bundle
	if _, err := c.sendJSON(ctx, http.MethodPost, c.poolPath("uploads", id, "complete"), nil, nil, &published); err != nil {
		return err
	}
	if published.Checksum != sum {
		return fmt.Errorf("server published checksum %q, expected %s", published.Checksum, sum)
	}
	return nil
}

// cancelUpload removes a session.
func (c *Client) cancelUpload(ctx context.Context, id string) error {
	_, err := c.sendJSON(ctx, http.MethodDelete, c.poolPath("uploads", id), nil, nil, nil)
	return err
}

package remote

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jvzantvoort/bundle/utils"
)

// BundleSummary is the BundleSummary schema of the server API.
type BundleSummary struct {
	Checksum  string    `json:"checksum"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	Version   int       `json:"version"`
	FileCount int       `json:"file_count"`
	SizeBytes int64     `json:"size_bytes"`
	Verified  bool      `json:"verified"`
}

// BundleState is the BundleState schema of the server API.
type BundleState struct {
	Verified    bool       `json:"verified"`
	LastChecked *time.Time `json:"last_checked"`
	Replicas    []string   `json:"replicas"`
	SizeBytes   int64      `json:"size_bytes"`
}

// Bundle is the Bundle schema of the server API.
type Bundle struct {
	BundleSummary
	Metadata BundleMetadata `json:"metadata"`
	State    BundleState    `json:"state"`
	Tags     []string       `json:"tags"`
	Revision string         `json:"revision"`
}

// BundleMetadata is the BundleMetadata schema of the server API.
type BundleMetadata struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Author    string    `json:"author"`
	Version   int       `json:"version"`
}

// ManifestEntry is one file of a bundle manifest.
type ManifestEntry struct {
	Path      string `json:"path"`
	Checksum  string `json:"checksum"`
	SizeBytes int64  `json:"size_bytes"`
}

// page is the common shape of the paginated responses.
type page[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// Capabilities is the Capabilities schema of the server API.
type Capabilities struct {
	BundleVersions []int `json:"bundle_versions"`
	UploadLimits   struct {
		MaxManifestEntries int   `json:"max_manifest_entries"`
		MaxFileBytes       int64 `json:"max_file_bytes"`
		MaxBundleBytes     int64 `json:"max_bundle_bytes"`
	} `json:"upload_limits"`
}

// CorruptedFile is one problem found by a server-side verification.
type CorruptedFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// VerificationResult is the outcome of a server-side verification.
type VerificationResult struct {
	Verified       bool            `json:"verified"`
	CheckedAt      time.Time       `json:"checked_at"`
	FilesChecked   int             `json:"files_checked"`
	CorruptedFiles []CorruptedFile `json:"corrupted_files"`
}

// task is the Task schema of the server API.
type task struct {
	ID     string              `json:"id"`
	Status string              `json:"status"`
	Result *VerificationResult `json:"result"`
	Error  string              `json:"error"`
}

// MigrationResult is the outcome of a server-side migration.
type MigrationResult struct {
	Status   string `json:"status"`
	Checksum string `json:"checksum"`
}

// createUploadRequest is the CreateUploadRequest schema of the server API.
type createUploadRequest struct {
	Checksum string          `json:"checksum"`
	Metadata BundleMetadata  `json:"metadata"`
	Tags     []string        `json:"tags"`
	Files    []ManifestEntry `json:"files"`
}

// upload is the Upload schema of the server API.
type upload struct {
	ID                string   `json:"id"`
	Status            string   `json:"status"`
	FileCount         int      `json:"file_count"`
	UploadedFileCount int      `json:"uploaded_file_count"`
	MissingFiles      []string `json:"missing_files"`
}

// APIError is an error response from the server.
type APIError struct {
	Status  int
	Code    string
	Message string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	parts := []string{fmt.Sprintf("server returned %d", e.Status)}
	if e.Code != "" {
		parts = append(parts, e.Code)
	}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	return strings.Join(parts, ": ")
}

// Unwrap makes a rejected request a user error. Server failures stay system errors.
func (e *APIError) Unwrap() error {
	if e.Status >= 400 && e.Status < 500 {
		return utils.ErrRemoteRejected
	}
	return nil
}

// hasStatus reports whether err is an APIError with the given HTTP status.
func hasStatus(err error, status int) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == status
}

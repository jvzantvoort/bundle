// Package remotetest provides an in-memory Bundle Server for tests.
//
// It enforces the parts of openapi.yaml a client can get wrong: strict request
// bodies, the bearer token, Content-SHA256, If-Match, Idempotency-Key and single
// percent-encoded path segments for file paths.
package remotetest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jvzantvoort/bundle/checksum"
)

var pathPattern = regexp.MustCompile(`^[^/].*$`)

// Bundle is a bundle stored by the server. Tests can set Manifest to make the
// server announce something else than the files it holds.
type Bundle struct {
	Title     string
	Author    string
	CreatedAt time.Time
	Version   int
	Tags      []string
	Files     map[string][]byte
	Manifest  []Entry
	Revision  int
	Corrupted bool
}

// Entry is one manifest entry.
type Entry struct {
	Path      string `json:"path"`
	Checksum  string `json:"checksum"`
	SizeBytes int64  `json:"size_bytes"`
}

type upload struct {
	id       string
	pool     string
	sum      string
	status   string
	metadata metadataBody
	tags     []string
	manifest []Entry
	received map[string][]byte
}

type metadataBody struct {
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Author    string    `json:"author"`
	Version   int       `json:"version"`
}

type createBody struct {
	Checksum string       `json:"checksum"`
	Metadata metadataBody `json:"metadata"`
	Tags     []string     `json:"tags"`
	Files    []Entry      `json:"files"`
}

type idempotent struct {
	bodyHash string
	uploadID string
}

// Server is the fake server.
type Server struct {
	*httptest.Server

	Token string // Required bearer token

	// Limits returned by /capabilities; zero means large
	MaxEntries   int
	MaxFileBytes int64

	// DropUpload, if set, closes the connection instead of accepting a file
	DropUpload func(path string) bool

	// RewriteUpload, if set, replaces the body of an upload after it was received
	RewriteUpload func(path string, data []byte) []byte

	// MissingBundleIs500 answers a request for a missing bundle with 500 instead of 404
	MissingBundleIs500 bool

	// IgnoreIdempotency opens a new session for every upload request, whatever its
	// Idempotency-Key and whatever other sessions are active for the bundle
	IgnoreIdempotency bool

	mu      sync.Mutex
	pools   map[string]map[string]*Bundle
	uploads map[string]*upload
	keys    map[string]idempotent
	tasks   map[string]map[string]any
	nextID  int
	puts    []string
}

// New starts a server with an empty pool named "default".
func New(token string) *Server {
	s := &Server{
		Token:   token,
		pools:   map[string]map[string]*Bundle{"default": {}},
		uploads: map[string]*upload{},
		keys:    map[string]idempotent{},
		tasks:   map[string]map[string]any{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/capabilities", s.capabilities)
	mux.HandleFunc("GET /api/v1/tasks/{id}", s.getTask)
	mux.HandleFunc("GET /api/v1/pools/{pool}/bundles", s.listBundles)
	mux.HandleFunc("GET /api/v1/pools/{pool}/bundles/{sum}", s.getBundle)
	mux.HandleFunc("PATCH /api/v1/pools/{pool}/bundles/{sum}", s.patchBundle)
	mux.HandleFunc("GET /api/v1/pools/{pool}/bundles/{sum}/files", s.listFiles)
	mux.HandleFunc("GET /api/v1/pools/{pool}/bundles/{sum}/files/{file}", s.downloadFile)
	mux.HandleFunc("POST /api/v1/pools/{pool}/bundles/{sum}/verify", s.verify)
	mux.HandleFunc("GET /api/v1/pools/{pool}/bundles/{sum}/tags", s.listTags)
	mux.HandleFunc("POST /api/v1/pools/{pool}/bundles/{sum}/tags", s.modifyTags)
	mux.HandleFunc("DELETE /api/v1/pools/{pool}/bundles/{sum}/tags", s.modifyTags)
	mux.HandleFunc("POST /api/v1/pools/{pool}/bundles/{sum}/migrate", s.migrate)
	mux.HandleFunc("POST /api/v1/pools/{pool}/uploads", s.createUpload)
	mux.HandleFunc("GET /api/v1/pools/{pool}/uploads/{id}", s.getUpload)
	mux.HandleFunc("DELETE /api/v1/pools/{pool}/uploads/{id}", s.cancelUpload)
	mux.HandleFunc("PUT /api/v1/pools/{pool}/uploads/{id}/files/{file}", s.uploadFile)
	mux.HandleFunc("POST /api/v1/pools/{pool}/uploads/{id}/complete", s.completeUpload)

	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+s.Token {
			fail(w, http.StatusUnauthorized, "unauthorized", "a valid bearer token is required")
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		mux.ServeHTTP(w, r)
	}))
	return s
}

// APIURL returns the API root.
func (s *Server) APIURL() string {
	return s.Server.URL + "/api/v1"
}

// Add stores a bundle and returns its checksum, computed by the bundle's version.
func (s *Server) Add(pool string, bundle *Bundle) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if bundle.Version == 0 {
		bundle.Version = 2
	}
	if bundle.Author == "" {
		bundle.Author = "tester"
	}
	if bundle.CreatedAt.IsZero() {
		bundle.CreatedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	}
	if bundle.Tags == nil {
		bundle.Tags = []string{}
	}
	if bundle.Manifest == nil {
		bundle.Manifest = ManifestOf(bundle.Files)
	}
	if bundle.Revision == 0 {
		bundle.Revision = 1
	}
	sum := SumOf(bundle.Manifest)
	if bundle.Version == 1 {
		// Any stable value will do: this checksum is only used as a name
		hash := sha256.Sum256([]byte("v1" + sum))
		sum = hex.EncodeToString(hash[:])
	}
	if s.pools[pool] == nil {
		s.pools[pool] = map[string]*Bundle{}
	}
	s.pools[pool][sum] = bundle
	return sum
}

// Get returns a stored bundle, or nil.
func (s *Server) Get(pool, sum string) *Bundle {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pools[pool][sum]
}

// Puts returns the paths of all accepted file uploads, and clears the list.
func (s *Server) Puts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	puts := s.puts
	s.puts = nil
	return puts
}

// OpenUploads returns how many upload sessions are open.
func (s *Server) OpenUploads() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, session := range s.uploads {
		if session.status == "open" {
			count++
		}
	}
	return count
}

// ManifestOf builds the manifest for a set of files.
func ManifestOf(files map[string][]byte) []Entry {
	manifest := make([]Entry, 0, len(files))
	for path, data := range files {
		hash := sha256.Sum256(data)
		manifest = append(manifest, Entry{Path: path, Checksum: hex.EncodeToString(hash[:]), SizeBytes: int64(len(data))})
	}
	sort.Slice(manifest, func(i, j int) bool { return manifest[i].Path < manifest[j].Path })
	return manifest
}

// SumOf computes the version 2 bundle checksum of a manifest.
func SumOf(manifest []Entry) string {
	records := make([]checksum.ChecksumRecord, len(manifest))
	for i, entry := range manifest {
		records[i] = checksum.ChecksumRecord{Checksum: entry.Checksum, FilePath: entry.Path}
	}
	return checksum.ComputeBundleChecksum(records)
}

func fail(w http.ResponseWriter, status int, code, message string) {
	reply(w, status, map[string]string{"code": code, "message": message})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (s *Server) capabilities(w http.ResponseWriter, _ *http.Request) {
	entries, fileBytes := s.MaxEntries, s.MaxFileBytes
	if entries == 0 {
		entries = 1 << 20
	}
	if fileBytes == 0 {
		fileBytes = 1 << 40
	}
	reply(w, http.StatusOK, map[string]any{
		"api_version":         "1.0.0",
		"server_version":      "test",
		"bundle_versions":     []int{1, 2},
		"checksum_algorithms": []string{"sha256"},
		"upload_limits": map[string]any{
			"max_manifest_entries": entries,
			"max_file_bytes":       fileBytes,
			"max_bundle_bytes":     int64(1) << 50,
			"upload_ttl_seconds":   3600,
		},
	})
}

func summary(sum string, bundle *Bundle) map[string]any {
	var size int64
	for _, entry := range bundle.Manifest {
		size += entry.SizeBytes
	}
	return map[string]any{
		"checksum":   sum,
		"title":      bundle.Title,
		"author":     bundle.Author,
		"created_at": bundle.CreatedAt,
		"version":    bundle.Version,
		"file_count": len(bundle.Manifest),
		"size_bytes": size,
		"verified":   !bundle.Corrupted,
	}
}

func details(sum string, bundle *Bundle) map[string]any {
	out := summary(sum, bundle)
	out["metadata"] = map[string]any{
		"title":      bundle.Title,
		"created_at": bundle.CreatedAt,
		"author":     bundle.Author,
		"version":    bundle.Version,
	}
	out["state"] = map[string]any{"verified": !bundle.Corrupted, "replicas": []string{}, "size_bytes": out["size_bytes"]}
	out["tags"] = bundle.Tags
	out["revision"] = revision(bundle)
	return out
}

func revision(bundle *Bundle) string {
	return `"rev-` + strconv.Itoa(bundle.Revision) + `"`
}

// bundleOf finds the bundle of a request, or responds with 404.
func (s *Server) bundleOf(w http.ResponseWriter, r *http.Request) (string, *Bundle) {
	sum := r.PathValue("sum")
	bundle := s.pools[r.PathValue("pool")][sum]
	if bundle == nil {
		if s.MissingBundleIs500 {
			fail(w, http.StatusInternalServerError, "bundle_read_failed", "directory is not a bundle")
			return sum, nil
		}
		fail(w, http.StatusNotFound, "bundle_not_found", "No bundle "+sum)
	}
	return sum, bundle
}

func paginate[T any](w http.ResponseWriter, r *http.Request, items []T) {
	number, size := 1, 50
	if value := r.URL.Query().Get("page"); value != "" {
		number, _ = strconv.Atoi(value)
	}
	if value := r.URL.Query().Get("page_size"); value != "" {
		size, _ = strconv.Atoi(value)
	}
	if number < 1 || size < 1 || size > 100 {
		fail(w, http.StatusBadRequest, "invalid_page", "invalid page or page_size")
		return
	}
	start := min((number-1)*size, len(items))
	end := min(start+size, len(items))
	reply(w, http.StatusOK, map[string]any{"items": items[start:end], "page": number, "page_size": size, "total": len(items)})
}

func (s *Server) listBundles(w http.ResponseWriter, r *http.Request) {
	bundles, ok := s.pools[r.PathValue("pool")]
	if !ok {
		fail(w, http.StatusNotFound, "pool_not_found", "No pool "+r.PathValue("pool"))
		return
	}
	sums := make([]string, 0, len(bundles))
	for sum := range bundles {
		sums = append(sums, sum)
	}
	sort.Strings(sums)
	items := make([]map[string]any, len(sums))
	for i, sum := range sums {
		items[i] = summary(sum, bundles[sum])
	}
	paginate(w, r, items)
}

func (s *Server) getBundle(w http.ResponseWriter, r *http.Request) {
	if sum, bundle := s.bundleOf(w, r); bundle != nil {
		w.Header().Set("ETag", revision(bundle))
		reply(w, http.StatusOK, details(sum, bundle))
	}
}

// precondition checks If-Match, or responds with an error.
func precondition(w http.ResponseWriter, r *http.Request, bundle *Bundle) bool {
	match := r.Header.Get("If-Match")
	if match == "" {
		fail(w, http.StatusBadRequest, "missing_if_match", "If-Match is required")
		return false
	}
	if match != revision(bundle) {
		fail(w, http.StatusPreconditionFailed, "revision_mismatch", "The bundle changed")
		return false
	}
	return true
}

func (s *Server) patchBundle(w http.ResponseWriter, r *http.Request) {
	sum, bundle := s.bundleOf(w, r)
	if bundle == nil || !precondition(w, r, bundle) {
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if !decodeStrict(w, r, &body) {
		return
	}
	bundle.Title = body.Title
	bundle.Revision++
	reply(w, http.StatusOK, details(sum, bundle))
}

func (s *Server) listFiles(w http.ResponseWriter, r *http.Request) {
	if _, bundle := s.bundleOf(w, r); bundle != nil {
		paginate(w, r, bundle.Manifest)
	}
}

func (s *Server) downloadFile(w http.ResponseWriter, r *http.Request) {
	_, bundle := s.bundleOf(w, r)
	if bundle == nil {
		return
	}
	data, ok := bundle.Files[r.PathValue("file")]
	if !ok {
		fail(w, http.StatusNotFound, "file_not_found", "No file "+r.PathValue("file"))
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(data)
}

func (s *Server) verify(w http.ResponseWriter, r *http.Request) {
	_, bundle := s.bundleOf(w, r)
	if bundle == nil {
		return
	}
	s.nextID++
	id := "task-" + strconv.Itoa(s.nextID)
	corrupted := []map[string]string{}
	if bundle.Corrupted {
		corrupted = append(corrupted, map[string]string{"path": bundle.Manifest[0].Path, "reason": "checksum_mismatch"})
	}
	s.tasks[id] = map[string]any{
		"id": id, "type": "verify", "status": "done", "created_at": time.Now(),
		"result": map[string]any{
			"verified": !bundle.Corrupted, "checked_at": time.Now(),
			"files_checked": len(bundle.Manifest), "corrupted_files": corrupted,
		},
	}
	reply(w, http.StatusAccepted, map[string]any{"id": id, "type": "verify", "status": "queued", "created_at": time.Now()})
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	task, ok := s.tasks[r.PathValue("id")]
	if !ok {
		fail(w, http.StatusNotFound, "task_not_found", "No task")
		return
	}
	reply(w, http.StatusOK, task)
}

func (s *Server) listTags(w http.ResponseWriter, r *http.Request) {
	if _, bundle := s.bundleOf(w, r); bundle != nil {
		reply(w, http.StatusOK, map[string]any{"tags": bundle.Tags, "total": len(bundle.Tags)})
	}
}

func (s *Server) modifyTags(w http.ResponseWriter, r *http.Request) {
	_, bundle := s.bundleOf(w, r)
	if bundle == nil || !precondition(w, r, bundle) {
		return
	}
	var body struct {
		Tags []string `json:"tags"`
	}
	if !decodeStrict(w, r, &body) {
		return
	}
	for _, name := range body.Tags {
		index := slices.Index(bundle.Tags, name)
		if r.Method == http.MethodPost && index < 0 {
			bundle.Tags = append(bundle.Tags, name)
		} else if r.Method == http.MethodDelete && index >= 0 {
			bundle.Tags = slices.Delete(bundle.Tags, index, index+1)
		}
	}
	sort.Strings(bundle.Tags)
	bundle.Revision++
	w.Header().Set("ETag", revision(bundle))
	reply(w, http.StatusOK, map[string]any{"tags": bundle.Tags, "total": len(bundle.Tags)})
}

func (s *Server) migrate(w http.ResponseWriter, r *http.Request) {
	sum, bundle := s.bundleOf(w, r)
	if bundle == nil {
		return
	}
	if bundle.Version == 2 {
		reply(w, http.StatusOK, map[string]any{"status": "current", "checksum": sum, "version": 2})
		return
	}
	pool := s.pools[r.PathValue("pool")]
	newSum := SumOf(bundle.Manifest)
	delete(pool, sum)
	bundle.Version = 2
	pool[newSum] = bundle
	reply(w, http.StatusOK, map[string]any{"status": "migrated", "checksum": newSum, "version": 2})
}

// decodeStrict decodes a JSON body, refusing unknown fields as the API does.
func decodeStrict(w http.ResponseWriter, r *http.Request, out any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		fail(w, http.StatusBadRequest, "invalid_body", err.Error())
		return false
	}
	return true
}

func (s *Server) createUpload(w http.ResponseWriter, r *http.Request) {
	pool := r.PathValue("pool")
	if _, ok := s.pools[pool]; !ok {
		fail(w, http.StatusNotFound, "pool_not_found", "No pool "+pool)
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	bodyHash := sha256.Sum256(data)
	hash := hex.EncodeToString(bodyHash[:])

	key := r.Header.Get("Idempotency-Key")
	if previous, ok := s.keys[key]; key != "" && ok && !s.IgnoreIdempotency {
		if previous.bodyHash != hash {
			fail(w, http.StatusConflict, "idempotency_conflict", "key reused with a different body")
			return
		}
		reply(w, http.StatusCreated, s.uploadBody(s.uploads[previous.uploadID]))
		return
	}

	r.Body = io.NopCloser(bytes.NewReader(data))
	var body createBody
	if !decodeStrict(w, r, &body) {
		return
	}
	if len(body.Files) == 0 || body.Metadata.Version != 2 || body.Metadata.Author == "" || body.Tags == nil {
		fail(w, http.StatusBadRequest, "invalid_upload", "files, author, version 2 and tags are required")
		return
	}
	for _, entry := range body.Files {
		if !pathPattern.MatchString(entry.Path) || !checksum.ValidSHA256(entry.Checksum) {
			fail(w, http.StatusBadRequest, "invalid_entry", "invalid manifest entry "+entry.Path)
			return
		}
	}
	if SumOf(body.Files) != body.Checksum {
		fail(w, http.StatusBadRequest, "checksum_mismatch", "checksum does not match the manifest")
		return
	}
	if _, exists := s.pools[pool][body.Checksum]; exists {
		fail(w, http.StatusConflict, "bundle_exists", "bundle already exists")
		return
	}
	for _, session := range s.uploads {
		if session.pool == pool && session.sum == body.Checksum && session.status == "open" && !s.IgnoreIdempotency {
			fail(w, http.StatusConflict, "upload_active", "an upload for this bundle is active")
			return
		}
	}

	s.nextID++
	session := &upload{
		id:       fmt.Sprintf("00000000-0000-4000-8000-%012d", s.nextID),
		pool:     pool,
		sum:      body.Checksum,
		status:   "open",
		metadata: body.Metadata,
		tags:     body.Tags,
		manifest: body.Files,
		received: map[string][]byte{},
	}
	s.uploads[session.id] = session
	if key != "" {
		s.keys[key] = idempotent{bodyHash: hash, uploadID: session.id}
	}
	reply(w, http.StatusCreated, s.uploadBody(session))
}

func (s *Server) uploadBody(session *upload) map[string]any {
	missing := []string{}
	for _, entry := range session.manifest {
		if _, ok := session.received[entry.Path]; !ok {
			missing = append(missing, entry.Path)
		}
	}
	return map[string]any{
		"id":                  session.id,
		"pool":                session.pool,
		"checksum":            session.sum,
		"status":              session.status,
		"file_count":          len(session.manifest),
		"uploaded_file_count": len(session.received),
		"missing_files":       missing,
		"created_at":          time.Now(),
		"expires_at":          time.Now().Add(time.Hour),
	}
}

func (s *Server) sessionOf(w http.ResponseWriter, r *http.Request) *upload {
	session := s.uploads[r.PathValue("id")]
	if session == nil || session.pool != r.PathValue("pool") {
		fail(w, http.StatusNotFound, "upload_not_found", "No upload")
		return nil
	}
	return session
}

func (s *Server) getUpload(w http.ResponseWriter, r *http.Request) {
	if session := s.sessionOf(w, r); session != nil {
		reply(w, http.StatusOK, s.uploadBody(session))
	}
}

func (s *Server) cancelUpload(w http.ResponseWriter, r *http.Request) {
	session := s.sessionOf(w, r)
	if session == nil {
		return
	}
	if session.status == "complete" {
		fail(w, http.StatusConflict, "upload_complete", "already completed")
		return
	}
	session.status = "cancelled"
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request) {
	session := s.sessionOf(w, r)
	if session == nil {
		return
	}
	path := r.PathValue("file")
	if s.DropUpload != nil && s.DropUpload(path) {
		panic(http.ErrAbortHandler)
	}
	if session.status != "open" {
		fail(w, http.StatusConflict, "upload_closed", "upload is "+session.status)
		return
	}
	index := slices.IndexFunc(session.manifest, func(entry Entry) bool { return entry.Path == path })
	if index < 0 {
		fail(w, http.StatusNotFound, "file_not_found", "not in the manifest: "+path)
		return
	}
	if _, done := session.received[path]; done {
		fail(w, http.StatusConflict, "file_accepted", "already accepted: "+path)
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if s.RewriteUpload != nil {
		data = s.RewriteUpload(path, data)
	}
	hash := sha256.Sum256(data)
	actual := hex.EncodeToString(hash[:])
	if r.Header.Get("Content-SHA256") != session.manifest[index].Checksum {
		fail(w, http.StatusBadRequest, "invalid_digest", "Content-SHA256 does not match the manifest")
		return
	}
	if actual != session.manifest[index].Checksum {
		fail(w, http.StatusUnprocessableEntity, "checksum_mismatch", "body does not match the manifest")
		return
	}
	session.received[path] = data
	s.puts = append(s.puts, path)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) completeUpload(w http.ResponseWriter, r *http.Request) {
	session := s.sessionOf(w, r)
	if session == nil {
		return
	}
	if session.status != "open" {
		fail(w, http.StatusConflict, "upload_closed", "upload is "+session.status)
		return
	}
	if len(session.received) != len(session.manifest) {
		fail(w, http.StatusConflict, "files_missing", "missing "+strings.Join(s.uploadBody(session)["missing_files"].([]string), ", "))
		return
	}
	session.status = "complete"
	bundle := &Bundle{
		Title:     session.metadata.Title,
		Author:    session.metadata.Author,
		CreatedAt: session.metadata.CreatedAt,
		Version:   2,
		Tags:      session.tags,
		Files:     session.received,
		Manifest:  session.manifest,
		Revision:  1,
	}
	s.pools[session.pool][session.sum] = bundle
	w.Header().Set("Location", "/api/v1/pools/"+session.pool+"/bundles/"+session.sum)
	reply(w, http.StatusCreated, details(session.sum, bundle))
}

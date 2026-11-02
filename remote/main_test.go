package remote_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/checksum"
	"github.com/jvzantvoort/bundle/metadata"
	"github.com/jvzantvoort/bundle/pool"
	"github.com/jvzantvoort/bundle/remote"
	"github.com/jvzantvoort/bundle/remote/remotetest"
	"github.com/jvzantvoort/bundle/tag"
	"github.com/jvzantvoort/bundle/utils"
	"github.com/spf13/viper"
)

const testToken = "s3cret"

func newClient(t *testing.T, server *remotetest.Server) *remote.Client {
	t.Helper()
	t.Setenv("BUNDLE_TOKEN", "")
	viper.Set("remotes.test.url", server.APIURL())
	viper.Set("remotes.test.token", server.Token)
	client, err := remote.New("test", "")
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func newServer(t *testing.T) *remotetest.Server {
	t.Helper()
	server := remotetest.New(testToken)
	t.Cleanup(server.Close)
	return server
}

// newBundle creates a local bundle with awkward file names.
func newBundle(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "data")
	files := map[string]string{
		"plain.txt":            "plain",
		"dir/sub dir/a b.txt":  "spaces",
		"dir/100%.txt":         "percent",
		"dir/hash#tag+plus":    "reserved",
		"empty":                "",
		"photos.bundle/x.jpeg": "not metadata",
	}
	for path, data := range files {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := bundle.Create(dir, "Title"); err != nil {
		t.Fatal(err)
	}
	tags := &tag.Tags{Tags: []string{}}
	tags.Add("two", "one")
	if err := tags.Save(dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestNewRejectsBadConfiguration(t *testing.T) {
	for name, url := range map[string]string{"nourl": "", "ftp": "ftp://host/api", "relative": "host/api"} {
		viper.Set("remotes."+name+".url", url)
		if _, err := remote.New(name, ""); !errors.Is(err, utils.ErrRemoteConfig) {
			t.Errorf("%s: err = %v, want ErrRemoteConfig", name, err)
		}
	}
	if _, err := remote.New("missing", ""); !errors.Is(err, utils.ErrRemoteConfig) {
		t.Errorf("missing remote: err = %v", err)
	}
	viper.Set("remotes.good.url", "http://localhost/api/v1/")
	if _, err := remote.New("good", "../other"); !errors.Is(err, utils.ErrRemoteConfig) {
		t.Errorf("pool with a path: err = %v", err)
	}
	client, err := remote.New("good", "")
	if err != nil {
		t.Fatal(err)
	}
	if client.BaseURL != "http://localhost/api/v1" || client.Pool != "default" {
		t.Errorf("client = %+v", client)
	}
}

func TestPushPublishesBundle(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	dir := newBundle(t)

	result, err := client.Push(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != remote.PushStatusPushed || result.FileCount != 6 {
		t.Fatalf("result = %+v", result)
	}

	meta, err := metadata.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Checksum != meta.BundleChecksum {
		t.Errorf("checksum = %s, want %s", result.Checksum, meta.BundleChecksum)
	}
	stored := server.Get("default", meta.BundleChecksum)
	if stored == nil {
		t.Fatal("bundle not on the server")
	}
	if stored.Title != "Title" || stored.Author != meta.Author || !stored.CreatedAt.Equal(meta.CreatedAt) {
		t.Errorf("stored metadata = %+v", stored)
	}
	if !slices.Equal(stored.Tags, []string{"one", "two"}) {
		t.Errorf("stored tags = %v", stored.Tags)
	}
	if string(stored.Files["dir/sub dir/a b.txt"]) != "spaces" || string(stored.Files["dir/100%.txt"]) != "percent" {
		t.Errorf("stored files = %v", stored.Files)
	}
	if data, ok := stored.Files["empty"]; !ok || len(data) != 0 {
		t.Errorf("empty file = %q, %v", data, ok)
	}
}

func TestPushExistingBundleUploadsNothing(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	dir := newBundle(t)

	if _, err := client.Push(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	server.Puts()

	result, err := client.Push(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != remote.PushStatusExists {
		t.Errorf("status = %s", result.Status)
	}
	if puts := server.Puts(); len(puts) != 0 {
		t.Errorf("uploaded %v", puts)
	}
}

func TestPushResumesAfterConnectionLoss(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	dir := newBundle(t)

	server.DropUpload = func(path string) bool { return path == "plain.txt" }
	if _, err := client.Push(context.Background(), dir); err == nil || !strings.Contains(err.Error(), "run push again") {
		t.Fatalf("err = %v, want a hint to run push again", err)
	}
	if server.OpenUploads() != 1 {
		t.Fatalf("open uploads = %d, want the session kept for resuming", server.OpenUploads())
	}
	before := server.Puts()
	if len(before) == 0 {
		t.Fatal("expected files before plain.txt to be uploaded")
	}

	server.DropUpload = nil
	if _, err := client.Push(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	after := server.Puts()
	for _, path := range before {
		if slices.Contains(after, path) {
			t.Errorf("%s was uploaded twice", path)
		}
	}
	if len(before)+len(after) != 6 {
		t.Errorf("uploaded %v then %v, want 6 files in total", before, after)
	}
}

func TestPushLeavesOutIgnoredFiles(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	dir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"keep.txt": "keep", "SHA256SUM.txt": "ignored"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := bundle.Create(dir, "Title", "SHA256SUM.txt"); err != nil {
		t.Fatal(err)
	}

	result, err := client.Push(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	stored := server.Get("default", result.Checksum)
	if stored == nil {
		t.Fatal("bundle not on the server")
	}
	if _, ok := stored.Files["SHA256SUM.txt"]; ok || len(stored.Files) != 1 {
		t.Errorf("stored files = %v", stored.Files)
	}
}

func TestPushDoesNotDependOnMissingBundleStatus(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	dir := newBundle(t)
	server.MissingBundleIs500 = true

	result, err := client.Push(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != remote.PushStatusPushed {
		t.Errorf("status = %s", result.Status)
	}

	result, err = client.Push(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != remote.PushStatusExists {
		t.Errorf("second push status = %s", result.Status)
	}
}

func TestPushStartsOverWhenServerIgnoresIdempotencyKey(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	dir := newBundle(t)
	server.IgnoreIdempotency = true

	server.DropUpload = func(path string) bool { return path == "plain.txt" }
	if _, err := client.Push(context.Background(), dir); err == nil {
		t.Fatal("expected the first push to fail")
	}
	server.Puts()

	server.DropUpload = nil
	result, err := client.Push(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if server.Get("default", result.Checksum) == nil {
		t.Error("bundle not on the server")
	}
	if puts := server.Puts(); len(puts) != 6 {
		t.Errorf("uploaded %v, want all 6 files again", puts)
	}
	if server.OpenUploads() != 1 {
		t.Errorf("open uploads = %d, want the interrupted session left open", server.OpenUploads())
	}
}

func TestPushConflictWithoutBundleIsAnError(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	dir := newBundle(t)

	server.DropUpload = func(path string) bool { return path == "plain.txt" }
	if _, err := client.Push(context.Background(), dir); err == nil {
		t.Fatal("expected the first push to fail")
	}
	if server.OpenUploads() != 1 {
		t.Fatalf("open uploads = %d, want the session kept", server.OpenUploads())
	}
	server.DropUpload = nil

	// A different request gets a different key, so the open session blocks it
	tags, err := tag.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	tags.Add("other-request")
	if err := tags.Save(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Push(context.Background(), dir); !errors.Is(err, utils.ErrRemoteRejected) {
		t.Errorf("err = %v, want ErrRemoteRejected", err)
	}
}

func TestPushCancelsRejectedUploadAndCanRetry(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	dir := newBundle(t)

	// Change the file between verification and upload
	server.RewriteUpload = func(path string, data []byte) []byte {
		if path == "plain.txt" {
			return []byte("changed in transit")
		}
		return data
	}
	if _, err := client.Push(context.Background(), dir); !errors.Is(err, utils.ErrRemoteRejected) {
		t.Fatalf("err = %v, want ErrRemoteRejected", err)
	}
	if server.OpenUploads() != 0 {
		t.Fatalf("open uploads = %d, want the rejected session cancelled", server.OpenUploads())
	}

	// The same key now replays the cancelled session, which must not end the push
	server.RewriteUpload = nil
	result, err := client.Push(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if server.Get("default", result.Checksum) == nil {
		t.Error("bundle not on the server")
	}
}

func TestWrongTokenIsAUserError(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	server.Token = "rotated"

	_, err := client.Push(context.Background(), newBundle(t))
	if !errors.Is(err, utils.ErrRemoteRejected) || utils.ExitCodeFromError(err) != 1 {
		t.Errorf("err = %v, exit code = %d", err, utils.ExitCodeFromError(err))
	}
}

func TestPushRefusals(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	ctx := context.Background()

	if _, err := client.Push(ctx, t.TempDir()); !errors.Is(err, utils.ErrNotABundle) {
		t.Errorf("not a bundle: err = %v", err)
	}

	corrupted := newBundle(t)
	if err := os.WriteFile(filepath.Join(corrupted, "plain.txt"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Push(ctx, corrupted); !errors.Is(err, utils.ErrCorruptedBundle) {
		t.Errorf("corrupted: err = %v", err)
	}

	empty := t.TempDir()
	if _, err := bundle.Create(empty, "Empty"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Push(ctx, empty); !errors.Is(err, utils.ErrInvalidPath) {
		t.Errorf("empty: err = %v", err)
	}

	legacy := newBundle(t)
	meta, err := metadata.Load(legacy)
	if err != nil {
		t.Fatal(err)
	}
	files := &checksum.ChecksumFile{}
	if err := files.Load(legacy); err != nil {
		t.Fatal(err)
	}
	sums := make([]string, len(files.Records))
	for i, record := range files.Records {
		sums[i] = record.Checksum
	}
	meta.Version = metadata.VersionLegacy
	meta.BundleChecksum = checksum.ComputeLegacyBundleChecksum(sums)
	if err := meta.Save(legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Push(ctx, legacy); !errors.Is(err, utils.ErrOutdatedBundle) {
		t.Errorf("version 1: err = %v", err)
	}

	server.MaxEntries = 1
	if _, err := client.Push(ctx, newBundle(t)); !errors.Is(err, utils.ErrRemoteRejected) {
		t.Errorf("too many files: err = %v", err)
	}
	server.MaxEntries = 0
	server.MaxFileBytes = 1
	if _, err := client.Push(ctx, newBundle(t)); !errors.Is(err, utils.ErrRemoteRejected) {
		t.Errorf("file too large: err = %v", err)
	}
	if puts := server.Puts(); len(puts) != 0 {
		t.Errorf("uploaded %v, want nothing", puts)
	}
}

func TestReadOperations(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	ctx := context.Background()

	// More than one page of files and of bundles
	files := map[string][]byte{}
	for i := 0; i < 250; i++ {
		files["f"+strings.Repeat("x", i%7)+string(rune('a'+i%26))+string(rune('a'+i/26))] = []byte{byte(i)}
	}
	sum := server.Add("default", &remotetest.Bundle{Title: "Many", Files: files, Tags: []string{"t"}})
	for i := 0; i < 120; i++ {
		server.Add("default", &remotetest.Bundle{Title: "B", Files: map[string][]byte{"a": {byte(i), 1}}})
	}

	bundles, err := client.ListBundles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundles) != 121 {
		t.Errorf("bundles = %d, want 121", len(bundles))
	}
	entries, err := client.ListFiles(ctx, sum)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(files) {
		t.Errorf("entries = %d, want %d", len(entries), len(files))
	}
	details, err := client.GetBundle(ctx, sum)
	if err != nil {
		t.Fatal(err)
	}
	if details.Title != "Many" || details.FileCount != len(files) || details.Revision == "" {
		t.Errorf("details = %+v", details)
	}

	if _, err := client.GetBundle(ctx, "../../etc"); !errors.Is(err, utils.ErrInvalidPath) {
		t.Errorf("bad checksum: err = %v", err)
	}
	missing := strings.Repeat("0", 64)
	_, err = client.GetBundle(ctx, missing)
	var apiErr *remote.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound || apiErr.Code != "bundle_not_found" {
		t.Errorf("missing bundle: err = %v", err)
	}
	if utils.ExitCodeFromError(err) != 1 {
		t.Errorf("exit code = %d, want 1", utils.ExitCodeFromError(err))
	}
}

func TestWriteOperations(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	ctx := context.Background()
	sum := server.Add("default", &remotetest.Bundle{Title: "Old", Files: map[string][]byte{"a": []byte("x")}})

	updated, err := client.Rename(ctx, sum, "New")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "New" {
		t.Errorf("title = %s", updated.Title)
	}

	tags, err := client.AddTags(ctx, sum, []string{"b", "a"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(tags, []string{"a", "b"}) {
		t.Errorf("tags = %v", tags)
	}
	if tags, err = client.RemoveTags(ctx, sum, []string{"a", "absent"}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(tags, []string{"b"}) {
		t.Errorf("tags = %v", tags)
	}
	if tags, err = client.ListTags(ctx, sum); err != nil || !slices.Equal(tags, []string{"b"}) {
		t.Errorf("tags = %v, err = %v", tags, err)
	}
}

func TestVerifyAndMigrate(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	ctx := context.Background()

	good := server.Add("default", &remotetest.Bundle{Title: "G", Files: map[string][]byte{"a": []byte("x")}})
	result, err := client.Verify(ctx, good)
	if err != nil || !result.Verified || result.FilesChecked != 1 {
		t.Errorf("result = %+v, err = %v", result, err)
	}

	bad := server.Add("default", &remotetest.Bundle{Title: "B", Corrupted: true, Files: map[string][]byte{"b": []byte("y")}})
	result, err = client.Verify(ctx, bad)
	if err != nil {
		t.Fatal(err)
	}
	if result.Verified || len(result.CorruptedFiles) != 1 || result.CorruptedFiles[0].Reason != "checksum_mismatch" {
		t.Errorf("result = %+v", result)
	}

	old := server.Add("default", &remotetest.Bundle{Title: "O", Version: 1, Files: map[string][]byte{"c": []byte("z")}})
	migrated, err := client.Migrate(ctx, old)
	if err != nil {
		t.Fatal(err)
	}
	if migrated.Status != "migrated" || migrated.Checksum == old {
		t.Errorf("migrated = %+v", migrated)
	}
	if current, err := client.Migrate(ctx, migrated.Checksum); err != nil || current.Status != "current" {
		t.Errorf("current = %+v, err = %v", current, err)
	}
}

// receive downloads into a new pool and returns the pool.
func receive(t *testing.T, client *remote.Client, sum string) (*pool.Pool, error) {
	t.Helper()
	p := &pool.Pool{Root: t.TempDir(), Title: "T"}
	err := p.Receive(sum, func(staging string) error {
		return client.Fetch(context.Background(), sum, staging)
	})
	return p, err
}

// checkPoolEmpty fails if a refused download left anything in the pool.
func checkPoolEmpty(t *testing.T, p *pool.Pool) {
	t.Helper()
	entries, err := os.ReadDir(p.Root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("pool holds %v after a refused download", entries)
	}
}

func TestFetchRoundTrip(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	dir := newBundle(t)

	pushed, err := client.Push(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}

	p, err := receive(t, client, pushed.Checksum)
	if err != nil {
		t.Fatal(err)
	}
	stored := p.GetBundlePath(pushed.Checksum)
	verified, corrupted, err := bundle.Verify(stored)
	if err != nil || !verified {
		t.Fatalf("verified = %v, corrupted = %v, err = %v", verified, corrupted, err)
	}

	got, err := bundle.Load(stored)
	if err != nil {
		t.Fatal(err)
	}
	if got.Metadata.Title != "Title" || got.Metadata.BundleChecksum != pushed.Checksum || got.Metadata.Version != metadata.CurrentVersion {
		t.Errorf("metadata = %+v", got.Metadata)
	}
	if !slices.Equal(got.Tags.List(), []string{"one", "two"}) {
		t.Errorf("tags = %v", got.Tags.List())
	}
	if got.State.SizeBytes != pushed.SizeBytes {
		t.Errorf("size = %d, want %d", got.State.SizeBytes, pushed.SizeBytes)
	}
	data, err := os.ReadFile(filepath.Join(stored, "dir", "100%.txt"))
	if err != nil || string(data) != "percent" {
		t.Errorf("file = %q, err = %v", data, err)
	}
}

func TestFetchRefusesUntrustedManifest(t *testing.T) {
	escape := filepath.Join(t.TempDir(), "outside")
	content := []byte("evil")
	entry := remotetest.ManifestOf(map[string][]byte{"x": content})[0]

	for name, path := range map[string]string{
		"parent":        "../evil",
		"nested parent": "a/../../evil",
		"absolute":      escape,
		"metadata":      ".bundle/META.json",
		"nested meta":   "a/.bundle/x",
		"dot":           "./x",
		"empty segment": "a//x",
		"empty":         "",
	} {
		t.Run(name, func(t *testing.T) {
			server := newServer(t)
			client := newClient(t, server)
			entry := entry
			entry.Path = path
			sum := server.Add("default", &remotetest.Bundle{
				Title: "T", Files: map[string][]byte{path: content}, Manifest: []remotetest.Entry{entry},
			})
			p, err := receive(t, client, sum)
			if !errors.Is(err, utils.ErrCorruptedBundle) {
				t.Errorf("err = %v, want ErrCorruptedBundle", err)
			}
			checkPoolEmpty(t, p)
			if _, err := os.Stat(escape); err == nil {
				t.Error("file written outside the bundle")
			}
		})
	}

	t.Run("duplicate", func(t *testing.T) {
		server := newServer(t)
		client := newClient(t, server)
		sum := server.Add("default", &remotetest.Bundle{
			Title: "T", Files: map[string][]byte{"x": content}, Manifest: []remotetest.Entry{entry, entry},
		})
		p, err := receive(t, client, sum)
		if !errors.Is(err, utils.ErrCorruptedBundle) {
			t.Errorf("err = %v", err)
		}
		checkPoolEmpty(t, p)
	})
}

func TestFetchRefusesWrongContent(t *testing.T) {
	good := []byte("good")
	manifest := remotetest.ManifestOf(map[string][]byte{"x": good})

	for name, body := range map[string][]byte{
		"tampered":  []byte("evil"),
		"truncated": []byte("go"),
		"oversized": []byte(strings.Repeat("A", 1<<20)),
	} {
		t.Run(name, func(t *testing.T) {
			server := newServer(t)
			client := newClient(t, server)
			sum := server.Add("default", &remotetest.Bundle{Title: "T", Files: map[string][]byte{"x": body}, Manifest: manifest})
			p, err := receive(t, client, sum)
			if !errors.Is(err, utils.ErrCorruptedBundle) {
				t.Errorf("err = %v, want ErrCorruptedBundle", err)
			}
			checkPoolEmpty(t, p)
		})
	}

	t.Run("manifest does not match the checksum", func(t *testing.T) {
		server := newServer(t)
		client := newClient(t, server)
		sum := server.Add("default", &remotetest.Bundle{Title: "T", Files: map[string][]byte{"x": good}})
		server.Get("default", sum).Manifest = remotetest.ManifestOf(map[string][]byte{"y": good})
		p, err := receive(t, client, sum)
		if !errors.Is(err, utils.ErrCorruptedBundle) {
			t.Errorf("err = %v, want ErrCorruptedBundle", err)
		}
		checkPoolEmpty(t, p)
	})

	t.Run("version 1", func(t *testing.T) {
		server := newServer(t)
		client := newClient(t, server)
		sum := server.Add("default", &remotetest.Bundle{Title: "T", Version: 1, Files: map[string][]byte{"x": good}})
		p, err := receive(t, client, sum)
		if !errors.Is(err, utils.ErrOutdatedBundle) {
			t.Errorf("err = %v, want ErrOutdatedBundle", err)
		}
		checkPoolEmpty(t, p)
	})
}

// Fetch must catch wrong content itself, not rely on the later pool verification,
// and must stop reading a file that is larger than announced.
func TestFetchChecksEachFile(t *testing.T) {
	good := []byte("good")
	manifest := remotetest.ManifestOf(map[string][]byte{"x": good})

	for name, body := range map[string][]byte{
		"tampered":  []byte("evil"),
		"truncated": []byte("go"),
		"oversized": []byte(strings.Repeat("A", 1<<20)),
	} {
		t.Run(name, func(t *testing.T) {
			server := newServer(t)
			client := newClient(t, server)
			sum := server.Add("default", &remotetest.Bundle{Title: "T", Files: map[string][]byte{"x": body}, Manifest: manifest})

			dir := filepath.Join(t.TempDir(), "bundle")
			if err := client.Fetch(context.Background(), sum, dir); !errors.Is(err, utils.ErrCorruptedBundle) {
				t.Errorf("err = %v, want ErrCorruptedBundle", err)
			}
			info, err := os.Stat(filepath.Join(dir, "x"))
			if err != nil {
				t.Fatal(err)
			}
			if info.Size() > int64(len(good))+1 {
				t.Errorf("stored %d bytes of a file announced as %d", info.Size(), len(good))
			}
		})
	}
}

func TestReceiveRefusesExistingAndBadChecksum(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	sum := server.Add("default", &remotetest.Bundle{Title: "T", Files: map[string][]byte{"x": []byte("y")}})

	p, err := receive(t, client, sum)
	if err != nil {
		t.Fatal(err)
	}
	err = p.Receive(sum, func(string) error {
		t.Error("fill called for a bundle the pool holds")
		return nil
	})
	if !errors.Is(err, utils.ErrBundleExists) {
		t.Errorf("err = %v, want ErrBundleExists", err)
	}
	if err := p.Receive("../evil", func(string) error { return nil }); !errors.Is(err, utils.ErrInvalidPath) {
		t.Errorf("err = %v, want ErrInvalidPath", err)
	}
}

func TestFetchStopsWhenCancelled(t *testing.T) {
	server := newServer(t)
	client := newClient(t, server)
	sum := server.Add("default", &remotetest.Bundle{Title: "T", Files: map[string][]byte{"x": []byte("y")}})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := &pool.Pool{Root: t.TempDir()}
	err := p.Receive(sum, func(staging string) error { return client.Fetch(ctx, sum, staging) })
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
	checkPoolEmpty(t, p)
}

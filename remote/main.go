// Package remote is a client for the Bundle Server HTTP API described in
// openapi.yaml.
//
// A remote is configured by name and identifies a server and a default pool on it:
//
//	default_remote: origin
//	remotes:
//	  origin:
//	    url: http://bundles.example.org:3110/api/v1
//	    token: secret
//	    pool: default
//
// Everything a server returns is treated as untrusted: downloaded manifests and
// files are checked before they can reach a local pool.
//
// Example usage:
//
//	client, err := remote.New("origin", "")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	bundles, err := client.ListBundles(ctx)
package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jvzantvoort/bundle/checksum"
	"github.com/jvzantvoort/bundle/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	// tokenEnv overrides the token from the configuration file.
	tokenEnv = "BUNDLE_TOKEN"

	// maxPageSize is the largest page_size the API allows.
	maxPageSize = 100

	// maxJSONBytes bounds a JSON response, which holds at most one page.
	maxJSONBytes = 64 << 20

	// maxErrorBytes bounds the part of an error response that is read.
	maxErrorBytes = 64 << 10
)

var (
	poolPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

	// taskPollInterval is the first delay between task status requests.
	taskPollInterval = 200 * time.Millisecond

	// maxTaskPollInterval is the longest delay between task status requests.
	maxTaskPollInterval = 5 * time.Second
)

// Client talks to one pool on one Bundle Server.
type Client struct {
	Name    string // Remote name from the configuration
	BaseURL string // API root, including /api/v1
	Pool    string // Pool on the server

	token string
	http  *http.Client
}

// DefaultName returns the configured default remote, if any.
func DefaultName() string {
	return viper.GetString("default_remote")
}

// New creates a client for a remote from the configuration.
//
// Parameters:
//   - name: remote name from the configuration
//   - pool: pool on the server; empty selects the remote's pool, then "default"
//
// Returns:
//   - *Client: the client
//   - error: utils.ErrRemoteConfig if the remote is missing or invalid
func New(name, pool string) (*Client, error) {
	key := "remotes." + name
	if !viper.IsSet(key) {
		return nil, fmt.Errorf("%w: remote '%s' not found in configuration", utils.ErrRemoteConfig, name)
	}

	baseURL := strings.TrimRight(viper.GetString(key+".url"), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("%w: remote '%s' needs an http or https url", utils.ErrRemoteConfig, name)
	}

	if pool == "" {
		pool = viper.GetString(key + ".pool")
	}
	if pool == "" {
		pool = "default"
	}
	if !poolPattern.MatchString(pool) {
		return nil, fmt.Errorf("%w: invalid pool name %q", utils.ErrRemoteConfig, pool)
	}

	token := os.Getenv(tokenEnv)
	if token == "" {
		token = viper.GetString(key + ".token")
	}
	if token != "" && parsed.Scheme == "http" && !isLoopback(parsed.Hostname()) {
		// Not logged: logs go to stdout, which holds the JSON output
		fmt.Fprintf(os.Stderr, "warning: remote '%s' sends its token unencrypted over http\n", name)
	}

	// No overall timeout: file transfers can legitimately take hours, and
	// completing an upload verifies the whole bundle before the server responds
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 30 * time.Minute

	return &Client{
		Name:    name,
		BaseURL: baseURL,
		Pool:    pool,
		token:   token,
		http:    &http.Client{Transport: transport},
	}, nil
}

// isLoopback reports whether host is this machine.
func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// checkChecksum returns utils.ErrInvalidPath unless sum is a bundle checksum.
func checkChecksum(sum string) error {
	if !checksum.ValidSHA256(sum) {
		return fmt.Errorf("%w: not a bundle checksum: %q", utils.ErrInvalidPath, sum)
	}
	return nil
}

// poolPath builds a path below /pools/{pool}, escaping each segment.
func (c *Client) poolPath(segments ...string) string {
	path := "/pools/" + url.PathEscape(c.Pool)
	for _, segment := range segments {
		path += "/" + url.PathEscape(segment)
	}
	return path
}

// send performs a request and returns the response if its status is 2xx.
func (c *Client) send(ctx context.Context, method, path string, header http.Header, body io.Reader, size int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	if size >= 0 {
		req.ContentLength = size
	}
	for key, values := range header {
		req.Header[key] = values
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	log.Debugf("%s %s", method, req.URL)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}
	defer resp.Body.Close()

	apiErr := &APIError{Status: resp.StatusCode}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBytes))
	if json.Unmarshal(data, apiErr) != nil || (apiErr.Code == "" && apiErr.Message == "") {
		apiErr.Message = strings.TrimSpace(string(data))
	}
	return nil, apiErr
}

// sendJSON performs a request with an optional JSON body and decodes the response
// into out, if out is not nil.
func (c *Client) sendJSON(ctx context.Context, method, path string, header http.Header, in, out any) (http.Header, error) {
	if header == nil {
		header = http.Header{}
	}
	var body io.Reader
	size := int64(-1)
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		body, size = bytes.NewReader(data), int64(len(data))
		header.Set("Content-Type", "application/json")
	}

	resp, err := c.send(ctx, method, path, header, body, size)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if out != nil {
		if err := json.NewDecoder(io.LimitReader(resp.Body, maxJSONBytes)).Decode(out); err != nil {
			return nil, fmt.Errorf("invalid response to %s %s: %w", method, path, err)
		}
	}
	return resp.Header, nil
}

// listAll collects every item of a paginated resource.
func listAll[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	items := []T{}
	for number := 1; ; number++ {
		query := url.Values{"page": {strconv.Itoa(number)}, "page_size": {strconv.Itoa(maxPageSize)}}
		var current page[T]
		if _, err := c.sendJSON(ctx, http.MethodGet, path+"?"+query.Encode(), nil, nil, &current); err != nil {
			return nil, err
		}
		items = append(items, current.Items...)
		if len(current.Items) == 0 || len(items) >= current.Total {
			return items, nil
		}
	}
}

// Capabilities returns the formats and limits of the server.
func (c *Client) Capabilities(ctx context.Context) (*Capabilities, error) {
	var capabilities Capabilities
	if _, err := c.sendJSON(ctx, http.MethodGet, "/capabilities", nil, nil, &capabilities); err != nil {
		return nil, err
	}
	return &capabilities, nil
}

// ListBundles returns all bundles in the pool.
func (c *Client) ListBundles(ctx context.Context) ([]BundleSummary, error) {
	return listAll[BundleSummary](ctx, c, c.poolPath("bundles"))
}

// GetBundle returns the details of a bundle.
func (c *Client) GetBundle(ctx context.Context, sum string) (*Bundle, error) {
	if err := checkChecksum(sum); err != nil {
		return nil, err
	}
	var bundle Bundle
	header, err := c.sendJSON(ctx, http.MethodGet, c.poolPath("bundles", sum), nil, nil, &bundle)
	if err != nil {
		return nil, err
	}
	if etag := header.Get("ETag"); etag != "" {
		bundle.Revision = etag
	}
	return &bundle, nil
}

// ListFiles returns the manifest of a bundle.
func (c *Client) ListFiles(ctx context.Context, sum string) ([]ManifestEntry, error) {
	if err := checkChecksum(sum); err != nil {
		return nil, err
	}
	return listAll[ManifestEntry](ctx, c, c.poolPath("bundles", sum, "files"))
}

// Verify runs a server-side verification and waits for its result.
//
// A bundle that fails verification is a result with Verified false, not an error.
func (c *Client) Verify(ctx context.Context, sum string) (*VerificationResult, error) {
	if err := checkChecksum(sum); err != nil {
		return nil, err
	}
	var current task
	if _, err := c.sendJSON(ctx, http.MethodPost, c.poolPath("bundles", sum, "verify"), nil, nil, &current); err != nil {
		return nil, err
	}

	delay := taskPollInterval
	for {
		switch current.Status {
		case "done":
			if current.Result == nil {
				return nil, fmt.Errorf("verification task %s finished without a result", current.ID)
			}
			return current.Result, nil
		case "failed":
			return nil, fmt.Errorf("verification task %s failed: %s", current.ID, current.Error)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
		delay = min(delay*2, maxTaskPollInterval)

		path := "/tasks/" + url.PathEscape(current.ID)
		if _, err := c.sendJSON(ctx, http.MethodGet, path, nil, nil, &current); err != nil {
			return nil, err
		}
	}
}

// Rename changes the title of a bundle.
func (c *Client) Rename(ctx context.Context, sum, title string) (*Bundle, error) {
	current, err := c.GetBundle(ctx, sum)
	if err != nil {
		return nil, err
	}
	var updated Bundle
	body := map[string]string{"title": title}
	if _, err := c.sendJSON(ctx, http.MethodPatch, c.poolPath("bundles", sum), ifMatch(current), body, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// ListTags returns the tags of a bundle.
func (c *Client) ListTags(ctx context.Context, sum string) ([]string, error) {
	if err := checkChecksum(sum); err != nil {
		return nil, err
	}
	var tags struct {
		Tags []string `json:"tags"`
	}
	if _, err := c.sendJSON(ctx, http.MethodGet, c.poolPath("bundles", sum, "tags"), nil, nil, &tags); err != nil {
		return nil, err
	}
	return tags.Tags, nil
}

// AddTags adds tags to a bundle and returns the complete tag list.
func (c *Client) AddTags(ctx context.Context, sum string, tags []string) ([]string, error) {
	return c.modifyTags(ctx, http.MethodPost, sum, tags)
}

// RemoveTags removes tags from a bundle and returns the complete tag list.
func (c *Client) RemoveTags(ctx context.Context, sum string, tags []string) ([]string, error) {
	return c.modifyTags(ctx, http.MethodDelete, sum, tags)
}

func (c *Client) modifyTags(ctx context.Context, method, sum string, tags []string) ([]string, error) {
	current, err := c.GetBundle(ctx, sum)
	if err != nil {
		return nil, err
	}
	var updated struct {
		Tags []string `json:"tags"`
	}
	body := map[string][]string{"tags": tags}
	if _, err := c.sendJSON(ctx, method, c.poolPath("bundles", sum, "tags"), ifMatch(current), body, &updated); err != nil {
		return nil, err
	}
	return updated.Tags, nil
}

// ifMatch builds the precondition that rejects a write if the bundle changed
// after it was read.
func ifMatch(bundle *Bundle) http.Header {
	return http.Header{"If-Match": {bundle.Revision}}
}

// Migrate upgrades a version 1 bundle on the server.
func (c *Client) Migrate(ctx context.Context, sum string) (*MigrationResult, error) {
	if err := checkChecksum(sum); err != nil {
		return nil, err
	}
	var result MigrationResult
	if _, err := c.sendJSON(ctx, http.MethodPost, c.poolPath("bundles", sum, "migrate"), nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

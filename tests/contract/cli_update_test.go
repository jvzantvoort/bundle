package contract_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestCLI_UpdateMedia(t *testing.T) {
	bin, repoRoot := buildCLI(t)
	dir := filepath.Join(t.TempDir(), "data")
	var img bytes.Buffer
	if err := png.Encode(&img, image.NewGray(image.Rect(0, 0, 3, 2))); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "pic.png"), img.String())
	writeFile(t, filepath.Join(dir, "notes.txt"), "hello")
	if _, stderr, exit, _ := runCmd(bin, repoRoot, "create", dir, "--title", "Media"); exit != 0 {
		t.Fatalf("create exit = %d: %s", exit, stderr)
	}

	if _, _, exit, _ := runCmd(bin, repoRoot, "update", dir); exit != 1 {
		t.Errorf("update without an option: exit = %d, want 1", exit)
	}
	if _, _, exit, _ := runCmd(bin, repoRoot, "update", t.TempDir(), "--media"); exit != 1 {
		t.Errorf("update of a directory that is not a bundle: exit = %d, want 1", exit)
	}

	out, stderr, exit, _ := runCmd(bin, repoRoot, "update", dir, "--media", "--json")
	if exit != 0 {
		t.Fatalf("update --media exit = %d: %s", exit, stderr)
	}
	var result struct {
		Status string `json:"status"`
		Media  struct {
			Added, Skipped, Unsupported, Files int
		} `json:"media"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("update output is not JSON: %v\n%s", err, out)
	}
	if result.Status != "updated" || result.Media.Added != 1 || result.Media.Unsupported != 1 || result.Media.Files != 1 {
		t.Errorf("update result = %+v", result)
	}

	var doc struct {
		Files map[string]struct {
			Type       string `json:"type"`
			Resolution struct{ Width, Height int }
		} `json:"files"`
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".bundle", "MEDIA.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if entry := doc.Files["pic.png"]; entry.Type != "image" || entry.Resolution.Width != 3 || entry.Resolution.Height != 2 {
		t.Errorf("MEDIA.json entry = %+v", entry)
	}

	out, _, exit, _ = runCmd(bin, repoRoot, "update", dir, "--media", "--json")
	if exit != 0 || json.Unmarshal([]byte(out), &result) != nil || result.Media.Skipped != 1 {
		t.Errorf("second update: exit %d, %+v", exit, result)
	}

	// The bundle identity is unchanged
	if _, stderr, exit, _ := runCmd(bin, repoRoot, "verify", dir); exit != 0 {
		t.Errorf("verify after update: exit = %d: %s", exit, stderr)
	}
}

package contract_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// testBinPath returns where to build the CLI in dir. Windows only runs files with
// an executable extension.
func testBinPath(dir string) string {
	name := "bundle-test-bin"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}

// buildCLI compiles the CLI into a temp directory and returns its path and the repo root.
func buildCLI(t *testing.T) (string, string) {
	t.Helper()
	bin := testBinPath(t.TempDir())
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Join(cwd, "..", "..")
	if out, err := exec.Command("go", "build", "-o", bin, filepath.Join(repoRoot, "cmd", "bundle")).CombinedOutput(); err != nil {
		t.Fatalf("failed to build cli: %v\n%s", err, out)
	}
	return bin, repoRoot
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCLI_VerifyExitCodes(t *testing.T) {
	bin, repoRoot := buildCLI(t)
	dir := filepath.Join(t.TempDir(), "data")
	writeFile(t, filepath.Join(dir, "my file.txt"), "x")

	if _, stderr, exit, _ := runCmd(bin, repoRoot, "create", dir, "--title", "T"); exit != 0 {
		t.Fatalf("create exit = %d: %s", exit, stderr)
	}
	if out, _, exit, _ := runCmd(bin, repoRoot, "verify", dir); exit != 0 {
		t.Fatalf("verify of a new bundle with a space in a name: exit = %d: %s", exit, out)
	}

	writeFile(t, filepath.Join(dir, "extra"), "y")
	out, _, exit, _ := runCmd(bin, repoRoot, "verify", dir, "-j")
	if exit != 1 {
		t.Fatalf("verify with an extra file: exit = %d, want 1: %s", exit, out)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid json: %v: %s", err, out)
	}
	if resp["status"] != "invalid" {
		t.Errorf("status = %v, want invalid", resp["status"])
	}

	if _, _, exit, _ := runCmd(bin, repoRoot, "verify", t.TempDir()); exit != 1 {
		t.Errorf("verify of a non-bundle: exit = %d, want 1", exit)
	}
}

func TestCLI_LocalInfoAndVerifyOutput(t *testing.T) {
	bin, repoRoot := buildCLI(t)
	dir := filepath.Join(t.TempDir(), "data")
	writeFile(t, filepath.Join(dir, "a.txt"), "a")
	writeFile(t, filepath.Join(dir, "sub", "b.txt"), "b")

	if _, stderr, exit, _ := runCmd(bin, repoRoot, "create", dir, "--title", "Shown title"); exit != 0 {
		t.Fatalf("create exit = %d: %s", exit, stderr)
	}

	out, _, exit, _ := runCmd(bin, repoRoot, "info", dir)
	if exit != 0 {
		t.Fatalf("info exit = %d: %s", exit, out)
	}
	for _, want := range []string{"Title:    Shown title", "Files:    2"} {
		if !strings.Contains(out, want) {
			t.Errorf("info output misses %q: %s", want, out)
		}
	}
	if out, _, _, _ := runCmd(bin, repoRoot, "info", dir, "-j"); strings.Contains(out, "Title:    ") {
		t.Errorf("info -j also printed the summary: %s", out)
	}

	check := func(wantStatus string, wantExit, wantCorrupted int) {
		t.Helper()
		out, _, exit, _ := runCmd(bin, repoRoot, "verify", dir, "-j")
		if exit != wantExit {
			t.Fatalf("verify exit = %d, want %d: %s", exit, wantExit, out)
		}
		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(out), &resp); err != nil {
			t.Fatalf("invalid json: %v: %s", err, out)
		}
		if resp["status"] != wantStatus || resp["files_checked"] != float64(2) {
			t.Errorf("status = %v, files_checked = %v", resp["status"], resp["files_checked"])
		}
		if checked, err := time.Parse(time.RFC3339, fmt.Sprint(resp["last_verified"])); err != nil || time.Since(checked) > time.Hour {
			t.Errorf("last_verified = %v", resp["last_verified"])
		}
		if corrupted, ok := resp["corrupted_files"].([]interface{}); !ok || len(corrupted) != wantCorrupted {
			t.Errorf("corrupted_files = %v, want %d entries", resp["corrupted_files"], wantCorrupted)
		}
	}
	check("valid", 0, 0)
	writeFile(t, filepath.Join(dir, "a.txt"), "changed")
	check("invalid", 1, 1)
}

func TestCLI_CreateRefusals(t *testing.T) {
	bin, repoRoot := buildCLI(t)
	base := t.TempDir()

	missing := filepath.Join(base, "typo")
	if _, _, exit, _ := runCmd(bin, repoRoot, "create", missing); exit != 1 {
		t.Errorf("create on missing path: exit = %d, want 1", exit)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Errorf("create made the missing directory")
	}

	dir := filepath.Join(base, "data")
	writeFile(t, filepath.Join(dir, "f"), "x")
	if _, stderr, exit, _ := runCmd(bin, repoRoot, "create", dir, "--title", "T"); exit != 0 {
		t.Fatalf("create exit = %d: %s", exit, stderr)
	}
	if _, _, exit, _ := runCmd(bin, repoRoot, "create", dir); exit != 1 {
		t.Errorf("create on existing bundle: exit = %d, want 1", exit)
	}
}

func TestCLI_Version(t *testing.T) {
	bin, repoRoot := buildCLI(t)

	out, stderr, exit, _ := runCmd(bin, repoRoot, "--version")
	if exit != 0 {
		t.Fatalf("--version exit = %d: %s", exit, stderr)
	}
	if !strings.HasPrefix(out, "bundle version ") {
		t.Errorf("--version output = %q", out)
	}
}

func TestCLI_Ignore(t *testing.T) {
	bin, repoRoot := buildCLI(t)
	dir := filepath.Join(t.TempDir(), "data")
	writeFile(t, filepath.Join(dir, "keep.txt"), "keep")
	writeFile(t, filepath.Join(dir, "SHA256SUM.txt"), "other")
	writeFile(t, filepath.Join(dir, "sub", "SHA256SUM.txt"), "other")

	if _, stderr, exit, _ := runCmd(bin, repoRoot, "create", dir, "--ignore", "["); exit != 1 {
		t.Errorf("create with a bad pattern: exit = %d, want 1: %s", exit, stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, ".bundle")); !os.IsNotExist(err) {
		t.Errorf("create with a bad pattern left .bundle behind: %v", err)
	}

	out, stderr, exit, _ := runCmd(bin, repoRoot, "create", dir, "-i", "SHA256SUM.txt", "-j")
	if exit != 0 {
		t.Fatalf("create exit = %d: %s", exit, stderr)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid json: %v: %s", err, out)
	}
	if resp["files"] != float64(1) {
		t.Errorf("files = %v, want 1", resp["files"])
	}
	if out, _, exit, _ := runCmd(bin, repoRoot, "verify", dir); exit != 0 {
		t.Errorf("verify exit = %d: %s", exit, out)
	}
}

// Results go to stdout and logs to stderr, so --json output stays valid JSON even
// with -v, and plain output carries no log prefixes.
func TestCLI_LogsGoToStderr(t *testing.T) {
	bin, repoRoot := buildCLI(t)
	dir := filepath.Join(t.TempDir(), "data")
	writeFile(t, filepath.Join(dir, "a.txt"), "a")
	if _, stderr, exit, _ := runCmd(bin, repoRoot, "create", dir, "--title", "T"); exit != 0 {
		t.Fatalf("create exit = %d: %s", exit, stderr)
	}

	for _, args := range [][]string{
		{"verify", dir},
		{"info", dir},
		{"list", dir},
		{"tag", "list", dir},
		{"rename", dir, "New title"},
	} {
		out, stderr, exit, _ := runCmd(bin, repoRoot, append(args, "-j", "-v")...)
		if exit != 0 {
			t.Fatalf("%v: exit = %d: %s", args, exit, stderr)
		}
		if !json.Valid([]byte(out)) {
			t.Errorf("%v -j -v: stdout is not only JSON: %s", args, out)
		}
		if !strings.Contains(stderr, "level=debug") {
			t.Errorf("%v -v: no debug log on stderr: %s", args, stderr)
		}
	}

	out, stderr, exit, _ := runCmd(bin, repoRoot, "verify", dir)
	if exit != 0 || !strings.HasPrefix(out, "Bundle Integrity: VALID\n") || strings.Contains(out, "level=") {
		t.Errorf("verify: exit = %d, stdout = %q, want the result only", exit, out)
	}
	if strings.Contains(stderr, "Bundle Integrity") {
		t.Errorf("verify printed its result to stderr: %s", stderr)
	}
}

// verify prints the table of the CLI contract, and --json adds the reason for
// each file next to the list of paths.
func TestCLI_VerifyReportsWhatIsWrong(t *testing.T) {
	bin, repoRoot := buildCLI(t)
	dir := filepath.Join(t.TempDir(), "data")
	writeFile(t, filepath.Join(dir, "a.txt"), "a")
	writeFile(t, filepath.Join(dir, "b.txt"), "b")
	writeFile(t, filepath.Join(dir, "c.txt"), "c")
	if _, stderr, exit, _ := runCmd(bin, repoRoot, "create", dir, "--title", "T"); exit != 0 {
		t.Fatalf("create exit = %d: %s", exit, stderr)
	}

	out, _, exit, _ := runCmd(bin, repoRoot, "verify", dir)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if exit != 0 || len(lines) != 4 ||
		lines[0] != "Bundle Integrity: VALID" ||
		lines[1] != strings.Repeat("-", len(lines[0])) ||
		lines[2] != "Files Checked: 3" ||
		!strings.HasPrefix(lines[3], "Last Verified: ") {
		t.Errorf("valid bundle: exit = %d, stdout:\n%s", exit, out)
	}
	if _, err := time.ParseInLocation("2006-01-02 15:04:05", strings.TrimPrefix(lines[len(lines)-1], "Last Verified: "), time.Local); err != nil {
		t.Errorf("Last Verified is not a time: %v", err)
	}

	writeFile(t, filepath.Join(dir, "a.txt"), "changed")
	if err := os.Remove(filepath.Join(dir, "c.txt")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "extra.txt"), "x")

	out, _, exit, _ = runCmd(bin, repoRoot, "verify", dir)
	lines = strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if exit != 1 || len(lines) != 7 ||
		lines[0] != "Bundle Integrity: INVALID" ||
		lines[1] != strings.Repeat("-", len(lines[0])) ||
		lines[2] != "Files Checked: 3" ||
		lines[3] != "Corrupted Files:" ||
		!strings.HasPrefix(lines[4], "- a.txt (expected: ") || !strings.Contains(lines[4], "..., found: ") ||
		lines[5] != "- c.txt (file missing)" ||
		lines[6] != "- extra.txt (not in manifest)" {
		t.Errorf("invalid bundle: exit = %d, stdout:\n%s", exit, out)
	}

	out, _, exit, _ = runCmd(bin, repoRoot, "verify", dir, "-j")
	var resp struct {
		Status         string   `json:"status"`
		CorruptedFiles []string `json:"corrupted_files"`
		Problems       []struct {
			Path, Reason, Expected, Found string
		} `json:"problems"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid json: %v: %s", err, out)
	}
	if exit != 1 || resp.Status != "invalid" || strings.Join(resp.CorruptedFiles, ",") != "a.txt,c.txt,extra.txt" || len(resp.Problems) != 3 {
		t.Fatalf("json: exit = %d, %+v", exit, resp)
	}
	a, c, extra := resp.Problems[0], resp.Problems[1], resp.Problems[2]
	if a.Reason != "checksum_mismatch" || len(a.Expected) != 64 || len(a.Found) != 64 || a.Expected == a.Found {
		t.Errorf("a.txt = %+v", a)
	}
	if c.Reason != "missing" || len(c.Expected) != 64 || c.Found != "" {
		t.Errorf("c.txt = %+v", c)
	}
	if extra.Reason != "unlisted" || extra.Expected != "" || extra.Found != "" {
		t.Errorf("extra.txt = %+v", extra)
	}
}

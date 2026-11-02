package contract_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildCLI compiles the CLI into a temp directory and returns its path and the repo root.
func buildCLI(t *testing.T) (string, string) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "bundle-test-bin")
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
	if err := json.Unmarshal([]byte(extractJSON(out)), &resp); err != nil {
		t.Fatalf("invalid json: %v: %s", err, out)
	}
	if resp["status"] != "invalid" {
		t.Errorf("status = %v, want invalid", resp["status"])
	}

	if _, _, exit, _ := runCmd(bin, repoRoot, "verify", t.TempDir()); exit != 1 {
		t.Errorf("verify of a non-bundle: exit = %d, want 1", exit)
	}
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

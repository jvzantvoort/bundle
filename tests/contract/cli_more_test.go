package contract_test

import (
    "encoding/json"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
)

// This test covers create, info, verify, list, and rename in JSON and non-JSON modes.
func TestCLI_More(t *testing.T) {
    tmp := t.TempDir()
    bin := filepath.Join(tmp, "bundle-test-bin")
    cwd, _ := os.Getwd()
    repoRoot := filepath.Join(cwd, "..", "..")
    cmdPath := filepath.Join(repoRoot, "cmd", "bundle")

    build := exec.Command("go", "build", "-o", bin, cmdPath)
    build.Stdout = os.Stdout
    build.Stderr = os.Stderr
    if err := build.Run(); err != nil {
        t.Fatalf("failed to build cli: %v", err)
    }

    // Sanity: ensure help contains 'list' subcommand
    helpOut, helpErr, _, _ := runCmd(bin, repoRoot, "--help")
    if !strings.Contains(helpOut, "list") && !strings.Contains(helpErr, "list") {
        t.Fatalf("built binary help missing 'list' subcommand; helpOut=%s helpErr=%s", helpOut, helpErr)
    }

    // create sample data
    dataDir := filepath.Join(tmp, "data2")
    if err := os.MkdirAll(dataDir, 0755); err != nil {
        t.Fatalf("mkdir data: %v", err)
    }
    f1 := filepath.Join(dataDir, "x.txt")
    if err := os.WriteFile(f1, []byte("abc"), 0644); err != nil {
        t.Fatalf("write file: %v", err)
    }

    // Create bundle (non-JSON)
    out, stderr, exit, err := runCmd(bin, repoRoot, "create", dataDir, "--title", "More Test")
    if err != nil || exit != 0 {
        t.Fatalf("create failed: err=%v exit=%d out=%s errout=%s", err, exit, out, stderr)
    }

    // Info JSON
    out, stderr, exit, err = runCmd(bin, repoRoot, "info", dataDir, "-j")
    if err != nil || exit != 0 {
        t.Fatalf("info -j failed: err=%v exit=%d out=%s errout=%s", err, exit, out, stderr)
    }
    var infoResp map[string]interface{}
    if err := json.Unmarshal([]byte(extractJSON(out)), &infoResp); err != nil {
        t.Fatalf("invalid json from info: %v out=%s errout=%s", err, out, stderr)
    }
    if infoResp["path"] == nil {
        t.Fatalf("info json missing path: %v", infoResp)
    }

    // Verify JSON (should be valid)
    out, stderr, exit, err = runCmd(bin, repoRoot, "verify", dataDir, "-j")
    if err != nil || exit != 0 {
        t.Fatalf("verify -j failed: err=%v exit=%d out=%s errout=%s", err, exit, out, stderr)
    }
    var verResp map[string]interface{}
    if err := json.Unmarshal([]byte(extractJSON(out)), &verResp); err != nil {
        t.Fatalf("invalid json from verify: %v out=%s errout=%s", err, out, stderr)
    }
    if verResp["status"] == nil {
        t.Fatalf("verify json missing status: %v", verResp)
    }

    // List JSON
    out, stderr, exit, err = runCmd(bin, repoRoot, "list", dataDir, "-j")
    if err != nil || exit != 0 {
        t.Fatalf("list -j failed: err=%v exit=%d out=%s errout=%s", err, exit, out, stderr)
    }
    var listResp map[string]interface{}
    if err := json.Unmarshal([]byte(extractJSON(out)), &listResp); err != nil {
        t.Fatalf("invalid json from list: %v out=%s errout=%s", err, out, stderr)
    }
    if listResp["path"] == nil {
        t.Fatalf("list json missing path: %v", listResp)
    }

    // Rename (JSON)
    out, stderr, exit, err = runCmd(bin, repoRoot, "rename", dataDir, "NewTitle", "-j")
    // Note: rename uses positional args, ensure exit 0
    if err != nil || exit != 0 {
        t.Fatalf("rename failed: err=%v exit=%d out=%s errout=%s", err, exit, out, stderr)
    }
    var renResp map[string]interface{}
    if err := json.Unmarshal([]byte(extractJSON(out)), &renResp); err != nil {
        t.Fatalf("invalid json from rename: %v out=%s errout=%s", err, out, stderr)
    }
    if renResp["title"] != "NewTitle" {
        t.Fatalf("rename did not set title: %v", renResp)
    }
}

// extractJSON finds the first '{' and returns substring from there, or the original string if not found.
func extractJSON(s string) string {
    if i := strings.Index(s, "{"); i >= 0 {
        return s[i:]
    }
    return s
}

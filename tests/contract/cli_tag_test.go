package contract_test

import (
    "context"
    "encoding/json"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
    "time"
)

// helper runs a command and returns stdout, stderr, exit code and error
func runCmd(bin string, workdir string, args ...string) (string, string, int, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    cmd := exec.CommandContext(ctx, bin, args...)
    if workdir != "" {
        cmd.Dir = workdir
    }

    stdoutPipe, _ := cmd.StdoutPipe()
    stderrPipe, _ := cmd.StderrPipe()

    if err := cmd.Start(); err != nil {
        return "", "", -1, err
    }

    outBytes, _ := io.ReadAll(stdoutPipe)
    errBytes, _ := io.ReadAll(stderrPipe)

    err := cmd.Wait()
    exit := 0
    if cmd.ProcessState != nil {
        exit = cmd.ProcessState.ExitCode()
    }
    return string(outBytes), string(errBytes), exit, err
}

func TestTagCLI_AddListRemove_JSON(t *testing.T) {
    // Build binary
    tmp := t.TempDir()
    bin := filepath.Join(tmp, "bundle-test-bin")
    // Build CLI binary from repo root
    cwd, _ := os.Getwd()
    repoRoot := filepath.Join(cwd, "..", "..")
    cmdPath := filepath.Join(repoRoot, "cmd", "bundle")
    build := exec.Command("go", "build", "-o", bin, cmdPath)
    build.Stdout = os.Stdout
    build.Stderr = os.Stderr
    if err := build.Run(); err != nil {
        t.Fatalf("failed to build cli: %v", err)
    }

    // Create a sample bundle directory with files
    dataDir := filepath.Join(tmp, "data")
    if err := os.MkdirAll(dataDir, 0755); err != nil {
        t.Fatalf("mkdir data: %v", err)
    }
    f1 := filepath.Join(dataDir, "a.txt")
    if err := os.WriteFile(f1, []byte("hello"), 0644); err != nil {
        t.Fatalf("write file: %v", err)
    }

    // Run create
    _, _, exit, err := runCmd(bin, repoRoot, "create", dataDir, "--title", "CLI Test")
    if err != nil {
        t.Fatalf("create command failed: %v (exit %d)", err, exit)
    }
    if exit != 0 {
        t.Fatalf("create exit code = %d, want 0", exit)
    }

    // Add tags
    out, stderr, exit, err := runCmd(bin, repoRoot, "tag", "add", dataDir, "Travel", "Photos", "2025", "-j")
    if err != nil {
        t.Fatalf("tag add failed: %v (exit %d) out=%s errout=%s", err, exit, out, stderr)
    }
    if exit != 0 {
        t.Fatalf("tag add exit = %d want 0", exit)
    }

    var addResp map[string]interface{}
    if err := json.Unmarshal([]byte(out), &addResp); err != nil {
        t.Fatalf("invalid json from tag add: %v; out=%s errout=%s", err, out, stderr)
    }
    if addResp["status"] != "added" {
        t.Fatalf("unexpected add status: %v", addResp["status"])
    }
    tags, ok := addResp["tags"].([]interface{})
    if !ok || len(tags) < 1 {
        t.Fatalf("tags missing in add response: %v", addResp["tags"])
    }

    // List tags
    out, stderr, exit, err = runCmd(bin, repoRoot, "tag", "list", dataDir, "-j")
    if err != nil {
        t.Fatalf("tag list failed: %v (exit %d) out=%s errout=%s", err, exit, out, stderr)
    }
    if exit != 0 {
        t.Fatalf("tag list exit = %d want 0", exit)
    }
    var listResp map[string]interface{}
    if err := json.Unmarshal([]byte(out), &listResp); err != nil {
        t.Fatalf("invalid json from tag list: %v; out=%s errout=%s", err, out, stderr)
    }
    if listResp["path"] == nil {
        t.Fatalf("list response missing path: %v", listResp)
    }
    ltags, ok := listResp["tags"].([]interface{})
    if !ok {
        t.Fatalf("list tags missing or wrong type: %v", listResp["tags"])
    }
    if len(ltags) < 3 {
        t.Fatalf("expected >=3 tags after add, got %d", len(ltags))
    }

    // Remove a tag
    out, stderr, exit, err = runCmd(bin, repoRoot, "tag", "remove", dataDir, "photos", "-j")
    if err != nil {
        t.Fatalf("tag remove failed: %v (exit %d) out=%s errout=%s", err, exit, out, stderr)
    }
    if exit != 0 {
        t.Fatalf("tag remove exit = %d want 0", exit)
    }
    var remResp map[string]interface{}
    if err := json.Unmarshal([]byte(out), &remResp); err != nil {
        t.Fatalf("invalid json from tag remove: %v; out=%s errout=%s", err, out, stderr)
    }
    if remResp["status"] != "removed" {
        t.Fatalf("unexpected remove status: %v", remResp["status"])
    }

    // Error case: non-existent path should exit 1
    _, _, exit, _ = runCmd(bin, repoRoot, "tag", "add", "/nonexistent/path/hopefully", "x")
    if exit != 1 {
        t.Fatalf("expected exit 1 for non-existent path, got %d", exit)
    }
}

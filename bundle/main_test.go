package bundle

import (
    "io/ioutil"
    "os"
    "path/filepath"
    "testing"
)

// TestCreateLoadVerify performs an end-to-end create, load, verify and corruption detection
func TestCreateLoadVerify(t *testing.T) {
    dir := t.TempDir()

    // Create some test files
    files := []struct{
        name string
        data string
    }{
        {"a.txt", "hello"},
        {"sub/b.txt", "world"},
    }

    for _, f := range files {
        p := filepath.Join(dir, f.name)
        if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
            t.Fatalf("mkdir: %v", err)
        }
        if err := ioutil.WriteFile(p, []byte(f.data), 0644); err != nil {
            t.Fatalf("write: %v", err)
        }
    }

    // Create bundle
    b, err := Create(dir, "Test Bundle")
    if err != nil {
        t.Fatalf("Create failed: %v", err)
    }
    if b == nil {
        t.Fatalf("Create returned nil bundle")
    }

    // Load bundle
    lb, err := Load(dir)
    if err != nil {
        t.Fatalf("Load failed: %v", err)
    }
    if lb.Metadata.Title != "Test Bundle" {
        t.Fatalf("unexpected title: %s", lb.Metadata.Title)
    }

    // Verify should succeed
    ok, corrupted, err := Verify(dir)
    if err != nil {
        t.Fatalf("Verify error: %v", err)
    }
    if !ok || len(corrupted) != 0 {
        t.Fatalf("expected verify ok, got corrupted=%v", corrupted)
    }

    // Corrupt a file
    f0 := filepath.Join(dir, "a.txt")
    if err := ioutil.WriteFile(f0, []byte("corrupt"), 0644); err != nil {
        t.Fatalf("corrupt write failed: %v", err)
    }

    ok, corrupted, err = Verify(dir)
    if err != nil {
        t.Fatalf("Verify after corrupt error: %v", err)
    }
    if ok {
        t.Fatalf("expected verify to detect corruption")
    }
    if len(corrupted) == 0 {
        t.Fatalf("expected corrupted list, got none")
    }
}

// TestLoadNonBundle ensures Load returns error for non-bundle directory
func TestLoadNonBundle(t *testing.T) {
    dir := t.TempDir()
    // Ensure no .bundle exists
    _, err := Load(dir)
    if err == nil {
        t.Fatalf("expected error loading non-bundle dir")
    }
}

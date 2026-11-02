package contract_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jvzantvoort/bundle/remote/remotetest"
)

// remoteCLI runs the CLI with a configuration holding one remote and one local pool.
type remoteCLI struct {
	bin  string
	home string
	pool string
}

func newRemoteCLI(t *testing.T, server *remotetest.Server, token string) *remoteCLI {
	t.Helper()
	bin, _ := buildCLI(t)
	home := t.TempDir()
	poolRoot := filepath.Join(t.TempDir(), "pool")
	config := fmt.Sprintf(`default_remote: origin
remotes:
  origin:
    url: %s
    token: %s
pools:
  local:
    root: %s
`, server.APIURL(), token, filepath.ToSlash(poolRoot))
	writeFile(t, filepath.Join(home, ".config", "bundle", "config.yaml"), config)
	return &remoteCLI{bin: bin, home: home, pool: poolRoot}
}

func (c *remoteCLI) run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(c.bin, args...)
	cmd.Dir = c.home
	env := []string{}
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "BUNDLE_TOKEN=") {
			env = append(env, entry)
		}
	}
	cmd.Env = append(env, "HOME="+c.home, "USERPROFILE="+c.home)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if _, isExit := err.(*exec.ExitError); err != nil && !isExit {
		t.Fatal(err)
	}
	return stdout.String(), stderr.String(), cmd.ProcessState.ExitCode()
}

// runJSON runs a command that must succeed and decodes its JSON output, which
// must be the only thing on stdout.
func (c *remoteCLI) runJSON(t *testing.T, args ...string) map[string]interface{} {
	t.Helper()
	out, stderr, exit := c.run(t, append(args, "-j")...)
	if exit != 0 {
		t.Fatalf("%v: exit = %d\nstdout: %s\nstderr: %s", args, exit, out, stderr)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("%v: invalid json: %v: %s", args, err, out)
	}
	return resp
}

func TestCLI_PushAndRemoteCommands(t *testing.T) {
	server := remotetest.New("tok")
	defer server.Close()
	cli := newRemoteCLI(t, server, "tok")

	dir := filepath.Join(t.TempDir(), "data")
	writeFile(t, filepath.Join(dir, "sub", "my file.txt"), "hello")
	if _, stderr, exit := cli.run(t, "create", dir, "--title", "Mine"); exit != 0 {
		t.Fatalf("create exit = %d: %s", exit, stderr)
	}

	// push uses default_remote
	pushed := cli.runJSON(t, "push", dir)
	if pushed["status"] != "pushed" || pushed["remote"] != "origin" || pushed["pool"] != "default" {
		t.Fatalf("push = %v", pushed)
	}
	sum := pushed["checksum"].(string)

	if again := cli.runJSON(t, "push", dir); again["status"] != "exists" {
		t.Errorf("second push = %v", again)
	}

	listed := cli.runJSON(t, "list_bundles", "--remote", "origin")
	if listed["count"] != float64(1) {
		t.Errorf("list_bundles = %v", listed)
	}
	if out, stderr, exit := cli.run(t, "list_bundles", "--remote", "origin"); exit != 0 || !strings.Contains(out, sum) {
		t.Errorf("list_bundles table must show the full checksum: exit = %d\n%s\n%s", exit, out, stderr)
	}

	info := cli.runJSON(t, "info", "--remote", "origin", sum)
	if info["title"] != "Mine" || info["files"] != float64(1) {
		t.Errorf("info = %v", info)
	}
	if out, _, exit := cli.run(t, "info", "--remote", "origin", sum); exit != 0 || !strings.Contains(out, "Mine") {
		t.Errorf("info output: exit = %d: %s", exit, out)
	}

	files := cli.runJSON(t, "list", "--remote", "origin", sum)
	if files["total_files"] != float64(1) || files["total_size"] != float64(5) {
		t.Errorf("list = %v", files)
	}

	if verified := cli.runJSON(t, "verify", "--remote", "origin", sum); verified["status"] != "valid" {
		t.Errorf("verify = %v", verified)
	}

	if renamed := cli.runJSON(t, "rename", "--remote", "origin", sum, "Renamed"); renamed["title"] != "Renamed" {
		t.Errorf("rename = %v", renamed)
	}

	cli.runJSON(t, "tag", "add", "--remote", "origin", sum, "b", "a")
	cli.runJSON(t, "tag", "remove", "--remote", "origin", sum, "b")
	tags := cli.runJSON(t, "tag", "list", "--remote", "origin", sum)
	if fmt.Sprint(tags["tags"]) != "[a]" {
		t.Errorf("tags = %v", tags)
	}

	if migrated := cli.runJSON(t, "migrate", "--remote", "origin", sum); !strings.Contains(fmt.Sprint(migrated["bundles"]), "current") {
		t.Errorf("migrate = %v", migrated)
	}

	imported := cli.runJSON(t, "import", "--remote", "origin", sum, "--pool", "local")
	if imported["status"] != "imported" {
		t.Errorf("import = %v", imported)
	}
	stored := filepath.Join(cli.pool, sum)
	if out, stderr, exit := cli.run(t, "verify", stored); exit != 0 {
		t.Errorf("verify of the imported bundle: exit = %d\n%s\n%s", exit, out, stderr)
	}
	local := cli.runJSON(t, "info", stored)
	if local["title"] != "Renamed" || fmt.Sprint(local["tags"]) != "[a]" {
		t.Errorf("imported info = %v", local)
	}

	// Importing twice is the same user error as for a local import
	if _, _, exit := cli.run(t, "import", "--remote", "origin", sum, "--pool", "local"); exit != 1 {
		t.Errorf("second import: exit = %d, want 1", exit)
	}
}

func TestCLI_RemoteErrors(t *testing.T) {
	server := remotetest.New("tok")
	defer server.Close()
	cli := newRemoteCLI(t, server, "wrong")
	missing := strings.Repeat("0", 64)

	dir := filepath.Join(t.TempDir(), "data")
	writeFile(t, filepath.Join(dir, "a"), "x")
	if _, stderr, exit := cli.run(t, "create", dir, "--title", "T"); exit != 0 {
		t.Fatalf("create exit = %d: %s", exit, stderr)
	}

	for name, args := range map[string][]string{
		"wrong token":       {"push", dir},
		"unknown remote":    {"info", "--remote", "nope", missing},
		"not a checksum":    {"info", "--remote", "origin", "../../etc"},
		"local pool flag":   {"list_bundles", "--remote", "origin", "--pool", "x"},
		"move from remote":  {"import", "--remote", "origin", "--pool", "local", "--move", missing},
		"bad import target": {"import", "--remote", "origin", "--pool", "nope", missing},
		"local migrate":     {"migrate", "--remote", "origin", "--pool", "x"},
		"no path":           {"push"},
		"not a bundle":      {"push", t.TempDir()},
	} {
		t.Run(name, func(t *testing.T) {
			if out, stderr, exit := cli.run(t, args...); exit != 1 {
				t.Errorf("exit = %d, want 1\n%s\n%s", exit, out, stderr)
			}
		})
	}

	// The token must reach the server only in the header, never in output
	out, stderr, _ := cli.run(t, "push", dir)
	if !strings.Contains(out+stderr, "unauthorized") {
		t.Errorf("expected the server's error:\n%s\n%s", out, stderr)
	}
	if strings.Contains(out+stderr, "wrong") {
		t.Errorf("token in output:\n%s\n%s", out, stderr)
	}

	// create has no remote form
	if _, _, exit := cli.run(t, "create", dir, "--remote", "origin"); exit == 0 {
		t.Error("create accepted --remote")
	}
}

func TestCLI_PushWithoutRemote(t *testing.T) {
	server := remotetest.New("tok")
	defer server.Close()
	cli := newRemoteCLI(t, server, "tok")
	config := filepath.Join(cli.home, ".config", "bundle", "config.yaml")
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, config, strings.Replace(string(data), "default_remote: origin\n", "", 1))

	out, stderr, exit := cli.run(t, "push", t.TempDir())
	if exit != 1 || !strings.Contains(out+stderr, "default_remote") {
		t.Errorf("exit = %d\n%s\n%s", exit, out, stderr)
	}
}

func TestCLI_RemoteVerifyInvalidExitsOne(t *testing.T) {
	server := remotetest.New("tok")
	defer server.Close()
	cli := newRemoteCLI(t, server, "tok")
	sum := server.Add("default", &remotetest.Bundle{Title: "Bad", Corrupted: true, Files: map[string][]byte{"a": []byte("x")}})

	out, _, exit := cli.run(t, "verify", "--remote", "origin", sum)
	if exit != 1 || !strings.Contains(out, "checksum_mismatch") {
		t.Errorf("exit = %d: %s", exit, out)
	}
}

func TestCLI_RemotePoolAndMigrateAll(t *testing.T) {
	server := remotetest.New("tok")
	defer server.Close()
	cli := newRemoteCLI(t, server, "tok")
	old := server.Add("archive", &remotetest.Bundle{Title: "Old", Version: 1, Files: map[string][]byte{"a": []byte("x")}})
	current := server.Add("archive", &remotetest.Bundle{Title: "New", Files: map[string][]byte{"b": []byte("y")}})

	if listed := cli.runJSON(t, "list_bundles", "--remote", "origin", "--remote-pool", "archive"); listed["count"] != float64(2) {
		t.Errorf("list_bundles = %v", listed)
	}

	out, stderr, exit := cli.run(t, "import", "--remote", "origin", "--remote-pool", "archive", "--pool", "local", old)
	if exit != 1 || !strings.Contains(out+stderr, "migrate") {
		t.Errorf("import of version 1: exit = %d\n%s\n%s", exit, out, stderr)
	}

	migrated := cli.runJSON(t, "migrate", "--remote", "origin", "--remote-pool", "archive")
	bundles := migrated["bundles"].([]interface{})
	if len(bundles) != 1 || bundles[0].(map[string]interface{})["name"] != old {
		t.Errorf("migrate = %v", migrated)
	}
	if server.Get("archive", current) == nil {
		t.Error("up-to-date bundle was touched")
	}
}

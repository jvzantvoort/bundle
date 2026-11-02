/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"

	"github.com/jvzantvoort/bundle/metadata"
	"github.com/jvzantvoort/bundle/pool"
	"github.com/jvzantvoort/bundle/remote"
	"github.com/jvzantvoort/bundle/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var remoteName string
var remotePool string

// addRemoteFlags lets a command work on a server instead of the local filesystem.
func addRemoteFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&remoteName, "remote", "", "use this configured server; the path argument becomes a bundle checksum")
	addRemotePoolFlag(cmd)
}

func addRemotePoolFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&remotePool, "remote-pool", "", "pool on the server (default: the pool of the remote, then \"default\")")
}

// remoteClient returns the client for the --remote flag, or exits.
func remoteClient(name string) *remote.Client {
	client, err := remote.New(name, remotePool)
	if err != nil {
		log.Error(err)
		os.Exit(utils.ExitCodeFromError(err))
	}
	return client
}

// remoteContext returns a context cancelled by an interrupt, so that a download
// can clean up its staging directory.
func remoteContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

// exitRemote reports a failed remote operation and exits.
func exitRemote(err error) {
	log.Error(err)
	exitCode(err)
}

// exitCode exits with the code for err.
func exitCode(err error) {
	os.Exit(utils.ExitCodeFromError(err))
}

// outputRemoteJSON writes JSON with the remote and pool added, or exits.
func outputRemoteJSON(client *remote.Client, out map[string]interface{}) {
	out["remote"] = client.Name
	out["pool"] = client.Pool
	if err := utils.OutputJSON(out); err != nil {
		log.Errorf("failed to output json: %v", err)
		os.Exit(2)
	}
}

func remoteVerify(sum string) {
	client := remoteClient(remoteName)
	ctx, stop := remoteContext()
	defer stop()

	result, err := client.Verify(ctx, sum)
	if err != nil {
		exitRemote(err)
	}

	corrupted := make([]string, 0, len(result.CorruptedFiles))
	for _, file := range result.CorruptedFiles {
		corrupted = append(corrupted, file.Path)
	}
	if jsonOutput {
		outputRemoteJSON(client, map[string]interface{}{
			"status":          map[bool]string{true: "valid", false: "invalid"}[result.Verified],
			"checksum":        sum,
			"files_checked":   result.FilesChecked,
			"last_verified":   result.CheckedAt.UTC().Format("2006-01-02T15:04:05Z"),
			"corrupted_files": corrupted,
		})
	} else if result.Verified {
		log.Info("Bundle Integrity: VALID")
	} else {
		log.Info("Bundle Integrity: INVALID")
		for _, file := range result.CorruptedFiles {
			log.Infof("Corrupted: %s (%s)", file.Path, file.Reason)
		}
	}

	if !result.Verified {
		os.Exit(1)
	}
}

func remoteInfo(sum string) {
	client := remoteClient(remoteName)
	ctx, stop := remoteContext()
	defer stop()

	details, err := client.GetBundle(ctx, sum)
	if err != nil {
		exitRemote(err)
	}

	created := details.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
	if jsonOutput {
		outputRemoteJSON(client, map[string]interface{}{
			"title":      details.Title,
			"checksum":   details.Checksum,
			"files":      details.FileCount,
			"size_bytes": details.SizeBytes,
			"created_at": created,
			"author":     details.Author,
			"verified":   details.Verified,
			"version":    details.Version,
			"tags":       details.Tags,
			"replicas":   details.State.Replicas,
		})
		return
	}

	fmt.Printf("Title:    %s\n", details.Title)
	fmt.Printf("Checksum: %s\n", details.Checksum)
	fmt.Printf("Author:   %s\n", details.Author)
	fmt.Printf("Created:  %s\n", created)
	fmt.Printf("Files:    %d\n", details.FileCount)
	fmt.Printf("Size:     %s\n", formatBytes(details.SizeBytes))
	fmt.Printf("Verified: %t\n", details.Verified)
	fmt.Printf("Tags:     %s\n", strings.Join(details.Tags, ", "))
}

func remoteList(sum string) {
	client := remoteClient(remoteName)
	ctx, stop := remoteContext()
	defer stop()

	entries, err := client.ListFiles(ctx, sum)
	if err != nil {
		exitRemote(err)
	}

	var totalSize int64
	for _, entry := range entries {
		totalSize += entry.SizeBytes
	}

	if jsonOutput {
		outputRemoteJSON(client, map[string]interface{}{
			"checksum":    sum,
			"files":       entries,
			"total_files": len(entries),
			"total_size":  totalSize,
		})
		return
	}

	table := utils.OutputTable(os.Stdout)
	table.Header("Filename", "Checksum", "Size")
	for _, entry := range entries {
		_ = table.Append([]string{entry.Path, entry.Checksum, formatBytes(entry.SizeBytes)})
	}
	_ = table.Render()
}

func remoteListBundles() {
	client := remoteClient(remoteName)
	ctx, stop := remoteContext()
	defer stop()

	bundles, err := client.ListBundles(ctx)
	if err != nil {
		exitRemote(err)
	}

	if jsonOutput {
		type bundleInfo struct {
			Checksum  string `json:"checksum"`
			Title     string `json:"title"`
			Author    string `json:"author"`
			CreatedAt string `json:"created_at"`
		}
		bundleList := make([]bundleInfo, len(bundles))
		for i, summary := range bundles {
			bundleList[i] = bundleInfo{
				Checksum:  summary.Checksum,
				Title:     summary.Title,
				Author:    summary.Author,
				CreatedAt: summary.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			}
		}
		outputRemoteJSON(client, map[string]interface{}{"bundles": bundleList, "count": len(bundles)})
		return
	}

	if len(bundles) == 0 {
		log.Info("No bundles found in pool")
		return
	}

	sort.Slice(bundles, func(i, j int) bool {
		return bundles[i].Title < bundles[j].Title
	})

	// Remote commands take the checksum as their argument, so it is not truncated
	table := utils.OutputTable(os.Stdout)
	table.Header("Checksum", "Title", "Author", "Created")
	for _, summary := range bundles {
		_ = table.Append([]string{
			summary.Checksum,
			summary.Title,
			summary.Author,
			summary.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	fmt.Printf("Pool: %s (%s)\n\n", client.Pool, client.Name)
	_ = table.Render()
	fmt.Printf("\nTotal: %d bundles\n", len(bundles))
}

func remoteRename(sum, title string) {
	client := remoteClient(remoteName)
	ctx, stop := remoteContext()
	defer stop()

	updated, err := client.Rename(ctx, sum, title)
	if err != nil {
		exitRemote(err)
	}

	if jsonOutput {
		outputRemoteJSON(client, map[string]interface{}{
			"status":   "renamed",
			"checksum": sum,
			"title":    updated.Title,
		})
		return
	}
	log.Infof("Title updated: %s", updated.Title)
}

// outputRemoteTags prints the tags of a bundle on a server.
func outputRemoteTags(client *remote.Client, status, sum string, tags []string) {
	if jsonOutput {
		out := map[string]interface{}{"checksum": sum, "tags": tags}
		if status != "" {
			out["status"] = status
		}
		outputRemoteJSON(client, out)
		return
	}
	for _, name := range tags {
		fmt.Println(name)
	}
}

func remoteTagList(sum string) {
	client := remoteClient(remoteName)
	ctx, stop := remoteContext()
	defer stop()

	tags, err := client.ListTags(ctx, sum)
	if err != nil {
		exitRemote(err)
	}
	outputRemoteTags(client, "", sum, tags)
}

func remoteTagAdd(sum string, names []string) {
	client := remoteClient(remoteName)
	ctx, stop := remoteContext()
	defer stop()

	tags, err := client.AddTags(ctx, sum, names)
	if err != nil {
		exitRemote(err)
	}
	outputRemoteTags(client, "added", sum, tags)
}

func remoteTagRemove(sum string, names []string) {
	client := remoteClient(remoteName)
	ctx, stop := remoteContext()
	defer stop()

	tags, err := client.RemoveTags(ctx, sum, names)
	if err != nil {
		exitRemote(err)
	}
	outputRemoteTags(client, "removed", sum, tags)
}

// remoteMigrate upgrades one bundle, or with no checksum every version 1 bundle
// in the pool on the server.
func remoteMigrate(args []string) {
	client := remoteClient(remoteName)
	ctx, stop := remoteContext()
	defer stop()

	checksums := args
	if len(args) == 0 {
		bundles, err := client.ListBundles(ctx)
		if err != nil {
			exitRemote(err)
		}
		for _, summary := range bundles {
			if summary.Version < metadata.CurrentVersion {
				checksums = append(checksums, summary.Checksum)
			}
		}
	}

	type bundleResult struct {
		Name     string `json:"name"`
		Status   string `json:"status,omitempty"`
		Checksum string `json:"checksum,omitempty"`
		Error    string `json:"error,omitempty"`
	}

	exitCode := 0
	out := []bundleResult{}
	for _, sum := range checksums {
		result, err := client.Migrate(ctx, sum)
		if err != nil {
			exitCode = max(exitCode, utils.ExitCodeFromError(err))
			out = append(out, bundleResult{Name: sum, Error: err.Error()})
			if !jsonOutput {
				log.Errorf("%s: %v", sum, err)
			}
			continue
		}
		out = append(out, bundleResult{Name: sum, Status: result.Status, Checksum: result.Checksum})
		if !jsonOutput {
			log.Infof("%s -> %s (%s)", sum, result.Checksum, result.Status)
		}
	}

	if jsonOutput {
		outputRemoteJSON(client, map[string]interface{}{"bundles": out})
	} else {
		log.Infof("%d bundle(s) processed", len(out))
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

// remoteImport downloads a bundle from the server into a local pool.
func remoteImport(sum, poolName string) {
	client := remoteClient(remoteName)

	p, err := pool.GetPool(poolName)
	if err != nil {
		log.Errorf("Pool error: %v", err)
		os.Exit(1)
	}

	ctx, stop := remoteContext()
	defer stop()

	err = p.Receive(sum, func(stagingPath string) error {
		return client.Fetch(ctx, sum, stagingPath)
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Error("Import interrupted")
		} else {
			log.Errorf("Import failed: %v", err)
		}
		os.Exit(utils.ExitCodeFromError(err))
	}

	if jsonOutput {
		outputRemoteJSON(client, map[string]interface{}{
			"status":     "imported",
			"operation":  "downloaded",
			"checksum":   sum,
			"local_pool": poolName,
			"pool_root":  p.Root,
		})
		return
	}
	log.Infof("Bundle downloaded to pool '%s'", poolName)
	log.Infof("Pool: %s", p.Root)
}

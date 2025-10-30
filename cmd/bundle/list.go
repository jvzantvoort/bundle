/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
    "os"
    "path/filepath"
    "strings"
    "strconv"

    "github.com/jvzantvoort/bundle/messages"
    "github.com/jvzantvoort/bundle/bundle"
    "github.com/jvzantvoort/bundle/utils"
    "github.com/spf13/cobra"
    log "github.com/sirupsen/logrus"
)

// ListCmd represents the list command
var ListCmd = &cobra.Command{
    Use:   "list",
    Short: messages.GetShort("list"),
    Long:  messages.GetLong("list"),
    Run:   handleListCmd,
}

func init() {
    rootCmd.AddCommand(ListCmd)
}

func handleListCmd(cmd *cobra.Command, args []string) {
    if verbose {
        log.SetLevel(log.DebugLevel)
    }
    log.Debugf("%s: start", cmd.Use)
    defer log.Debugf("%s: end", cmd.Use)

    if len(args) != 1 {
        log.Error("No path provided")
        if err := cmd.Help(); err != nil {
            log.Error(err)
        }
        os.Exit(1)
    }

    path := args[0]
    b, err := bundle.Load(path)
    if err != nil {
        if os.IsNotExist(err) || strings.Contains(err.Error(), "not a bundle") {
            log.Errorf("Not a bundle: %v", err)
            os.Exit(1)
        }
        log.Errorf("System error: %v", err)
        os.Exit(2)
    }

    // Prepare file entries
    type fileEntry struct {
        Path     string `json:"path"`
        Checksum string `json:"checksum"`
        Size     int64  `json:"size_bytes"`
    }

    entries := []fileEntry{}
    var totalSize int64
    for _, r := range b.Files.Records {
        p := filepath.Join(b.Path, r.FilePath)
        var size int64
        if info, err := os.Stat(p); err == nil {
            size = info.Size()
            totalSize += size
        }
        entries = append(entries, fileEntry{
            Path:     r.FilePath,
            Checksum: r.Checksum,
            Size:     size,
        })
    }

    if jsonOutput {
        out := map[string]interface{}{
            "path":       b.Path,
            "files":      entries,
            "total_files": len(entries),
            "total_size": totalSize,
        }
        if err := utils.OutputJSON(out); err != nil {
            log.Errorf("failed to output json: %v", err)
            os.Exit(2)
        }
        return
    }

    // Human-readable table output
    table := utils.OutputTable(os.Stdout)
    table.Header("Filename", "Checksum", "Size")
    for _, e := range entries {
        _ = table.Append([]string{e.Path, e.Checksum, formatBytes(e.Size)})
    }
    _ = table.Render()
    log.Debugf("\nTotal: %d files, %s", len(entries), formatBytes(totalSize))
}

// formatBytes formats bytes into human-friendly string (KB/MB/GB)
func formatBytes(b int64) string {
    const unit = 1024
    if b < unit {
        return strconv.FormatInt(b, 10) + " B"
    }
    div, exp := int64(unit), 0
    for n := b / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    value := float64(b) / float64(div)
    units := []string{"KB", "MB", "GB", "TB"}
    // Format with 1 decimal place
    s := strconv.FormatFloat(value, 'f', 1, 64)
    return s + " " + units[exp]
}


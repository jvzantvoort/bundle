/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"fmt"
	"os"

	"github.com/jvzantvoort/bundle/messages"
	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/utils"
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"
)

// InfoCmd represents the info command
var InfoCmd = &cobra.Command{
	Use:   messages.GetUse("info"),
	Short: messages.GetShort("info"),
	Long:  messages.GetLong("info"),
	Run:   handleInfoCmd,
}

func init() {
	rootCmd.AddCommand(InfoCmd)
	InfoCmd.Flags().StringP("tag", "T", "", "mark every line with this tag")
	InfoCmd.Flags().StringP("title", "t", "", "log the contents of this file")
	addRemoteFlags(InfoCmd)
}

func handleInfoCmd(cmd *cobra.Command, args []string) {
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

	if remoteName != "" {
		remoteInfo(path)
		return
	}

	b, err := bundle.Load(path)
	if err != nil {
		log.Errorf("System error: %v", err)
		os.Exit(2)
	}

	if jsonOutput {
		out := map[string]interface{}{
			"path":       b.Path,
			"title":      "",
			"checksum":   "",
			"files":      0,
			"size_bytes": 0,
			"created_at": "",
			"author":     "",
			"verified":   nil,
			"tags":       []string{},
			"replicas":   []string{},
		}
		if b.Metadata != nil {
			out["title"] = b.Metadata.Title
			out["checksum"] = b.Metadata.BundleChecksum
			out["created_at"] = b.Metadata.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
			out["author"] = b.Metadata.Author
		}
		if b.State != nil {
			out["files"] = len(b.Files.Records)
			out["size_bytes"] = b.State.SizeBytes
			out["verified"] = b.State.Verified
		}
		if b.Tags != nil {
			out["tags"] = b.Tags.List()
		}
		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
		return
	}

	// Human-readable summary
	fmt.Printf("Path:     %s\n", b.Path)
	if b.Metadata != nil {
		fmt.Printf("Title:    %s\n", b.Metadata.Title)
		fmt.Printf("Checksum: %s\n", b.Metadata.BundleChecksum)
		fmt.Printf("Author:   %s\n", b.Metadata.Author)
		fmt.Printf("Created:  %s\n", b.Metadata.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"))
	}
	if b.State != nil {
		fmt.Printf("Files:    %d\n", len(b.Files.Records))
		fmt.Printf("Size:     %s\n", formatBytes(b.State.SizeBytes))
	}
}

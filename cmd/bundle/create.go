/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"os"

	"github.com/jvzantvoort/bundle/messages"
	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/utils"
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"
)

// CreateCmd represents the create command
var CreateCmd = &cobra.Command{
	Use:   messages.GetUse("create"),
	Short: messages.GetShort("create"),
	Long:  messages.GetLong("create"),
	Run:   handleCreateCmd,
}

func init() {
	rootCmd.AddCommand(CreateCmd)
	CreateCmd.Flags().StringP("tag", "T", "", "mark every line with this tag")
	CreateCmd.Flags().StringP("title", "t", "", "log the contents of this file")
}

func handleCreateCmd(cmd *cobra.Command, args []string) {
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
	title := GetString(*cmd, "title")

	b, err := bundle.Create(path, title)
	if err != nil {
		// Distinguish common user errors vs system errors where possible
		if os.IsNotExist(err) {
			log.Errorf("directory does not exist: %s", path)
			os.Exit(1)
		}
		// lock.AcquireLock returns an error string for lock contention; treat other errors as system errors
		log.Errorf("System error: %v", err)
		os.Exit(2)
	}

	// Print a human-readable summary similar to the CLI contract
	log.Info("Bundle Created")
	log.Info("--------------")
	log.Infof("Path:     %s", b.Path)
	if b.Metadata != nil {
		log.Infof("Checksum: %s", b.Metadata.BundleChecksum)
		log.Infof("Title:    %s", b.Metadata.Title)
		log.Infof("Created:  %s", b.Metadata.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	if b.Files != nil {
		log.Infof("Files:    %d", len(b.Files.Records))
	}
	if b.State != nil {
		log.Infof("Size:     %d bytes", b.State.SizeBytes)
	}

	if jsonOutput {
		out := map[string]interface{}{
			"status":     "created",
			"path":       b.Path,
			"checksum":   "",
			"files":      0,
			"size_bytes": 0,
			"title":      "",
			"created_at": "",
		}
		if b.Metadata != nil {
			out["checksum"] = b.Metadata.BundleChecksum
			out["title"] = b.Metadata.Title
			out["created_at"] = b.Metadata.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		if b.Files != nil {
			out["files"] = len(b.Files.Records)
		}
		if b.State != nil {
			out["size_bytes"] = b.State.SizeBytes
		}

		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
	}
}

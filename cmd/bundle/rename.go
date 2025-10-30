/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"os"
	"strings"

	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/messages"
	"github.com/jvzantvoort/bundle/metadata"
	"github.com/jvzantvoort/bundle/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// RenameCmd represents the rename command.
//
// It updates the title (human-readable name) of a bundle without changing
// the checksum or other metadata. The title is stored in .bundle/META.json.
//
// Usage:
//   bundle rename <path> <new_title>
//
// Example:
//   bundle rename ./my-bundle "Updated Title"
//   bundle rename /data/bundle "New Name" --json
var RenameCmd = &cobra.Command{
	Use:   messages.GetUse("rename"),
	Short: messages.GetShort("rename"),
	Long:  messages.GetLong("rename"),
	Run:   handleRenameCmd,
}

func init() {
	rootCmd.AddCommand(RenameCmd)
}

// handleRenameCmd processes the rename command.
//
// It validates arguments, loads the bundle metadata, updates the title,
// and saves the changes. Outputs either JSON or human-readable confirmation.
func handleRenameCmd(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugf("%s: start", cmd.Use)
	defer log.Debugf("%s: end", cmd.Use)

	// Validate arguments
	if len(args) != 2 {
		log.Error("Usage: bundle rename <path> <new_title>")
		if err := cmd.Help(); err != nil {
			log.Error(err)
		}
		os.Exit(1)
	}

	path := args[0]
	newTitle := args[1]

	log.Debugf("Updating title for bundle: %s", path)
	log.Debugf("New title: %s", newTitle)

	// Verify bundle exists
	b, err := bundle.Load(path)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "not a bundle") {
			log.Errorf("Not a bundle: %v", err)
			os.Exit(1)
		}
		log.Errorf("System error: %v", err)
		os.Exit(2)
	}

	if b.Metadata == nil {
		log.Errorf("bundle metadata missing")
		os.Exit(2)
	}

	oldTitle := b.Metadata.Title
	log.Debugf("Old title: %s", oldTitle)

	// Update title using metadata helper
	if err := metadata.UpdateTitle(path, newTitle); err != nil {
		log.Errorf("Failed to update title: %v", err)
		os.Exit(2)
	}

	log.Debugf("Title updated successfully")

	// Output results
	if jsonOutput {
		out := map[string]interface{}{
			"status":    "renamed",
			"path":      path,
			"old_title": oldTitle,
			"new_title": newTitle,
			"title":     newTitle, // For backward compatibility
		}
		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
		return
	}

	log.Infof("Title updated: %s → %s", oldTitle, newTitle)
}

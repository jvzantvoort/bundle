/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"os"
	"strings"

	"github.com/jvzantvoort/bundle/messages"
	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/utils"
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"
)

// RenameCmd represents the rename command
var RenameCmd = &cobra.Command{
	Use:   messages.GetUse("rename"),
	Short: messages.GetShort("rename"),
	Long:  messages.GetLong("rename"),
	Run:   handleRenameCmd,
}

func init() {
	rootCmd.AddCommand(RenameCmd)
	RenameCmd.Flags().StringP("tag", "T", "", "mark every line with this tag")
	RenameCmd.Flags().StringP("title", "t", "", "log the contents of this file")
}

func handleRenameCmd(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugf("%s: start", cmd.Use)
	defer log.Debugf("%s: end", cmd.Use)

	if len(args) != 2 {
		log.Error("Usage: bundle rename <path> <new_title>")
		if err := cmd.Help(); err != nil {
			log.Error(err)
		}
		os.Exit(1)
	}

	path := args[0]
	newTitle := args[1]

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

	b.Metadata.Title = newTitle
	if err := b.Metadata.Save(path); err != nil {
		log.Errorf("System error: %v", err)
		os.Exit(2)
	}

	if jsonOutput {
		out := map[string]interface{}{
			"status": "renamed",
			"path":   b.Path,
			"title":  b.Metadata.Title,
		}
		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
		return
	}

	log.Infof("Title updated: %s", b.Metadata.Title)
}

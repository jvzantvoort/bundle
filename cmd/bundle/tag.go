/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"fmt"
	"os"

	"github.com/jvzantvoort/bundle/messages"
	"github.com/jvzantvoort/bundle/tag"
	"github.com/jvzantvoort/bundle/utils"
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"
)

// TagCmd represents the tag command
var TagCmd = &cobra.Command{
	Use:   messages.GetUse("tag"),
	Short: messages.GetShort("tag"),
	Long:  messages.GetLong("tag"),
	Run:   handleTagCmd,
}

func init() {
	rootCmd.AddCommand(TagCmd)
	TagCmd.Flags().StringP("tag", "T", "", "mark every line with this tag")
	TagCmd.Flags().StringP("title", "t", "", "log the contents of this file")

	// Subcommands: add, remove, list
	TagCmd.AddCommand(tagAddCmd)
	TagCmd.AddCommand(tagRemoveCmd)
	TagCmd.AddCommand(tagListCmd)
}

func handleTagCmd(cmd *cobra.Command, args []string) {
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
}

// tag add
var tagAddCmd = &cobra.Command{
	Use:   messages.GetUse("tag_add"),
	Short: messages.GetShort("tag_add"),
	Long:  messages.GetLong("tag_add"),
	Run:   handleTagAddCmd,
}

func handleTagAddCmd(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugf("%s: start", cmd.Use)
	defer log.Debugf("%s: end", cmd.Use)

	if len(args) < 2 {
		log.Error("Usage: bundle tag add <path> <tag> [<tag>...]")
		if err := cmd.Help(); err != nil {
			log.Error(err)
		}
		os.Exit(1)
	}

	path := args[0]
	// Validate path exists and is a directory (user error if not)
	if fi, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			log.Errorf("Path does not exist: %s", path)
			os.Exit(1)
		}
		log.Errorf("System error: %v", err)
		os.Exit(2)
	} else if !fi.IsDir() {
		log.Errorf("Path is not a directory: %s", path)
		os.Exit(1)
	}
	tags := args[1:]

	t, err := tag.Load(path)
	if err != nil {
		log.Errorf("System error: %v", err)
		os.Exit(2)
	}

	t.Add(tags...)
	if err := t.Save(path); err != nil {
		log.Errorf("System error: %v", err)
		os.Exit(2)
	}

	jsonOut := jsonOutput
	if jsonOut {
		out := map[string]interface{}{
			"status": "added",
			"path":   path,
			"tags":   t.List(),
		}
		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
		return
	}

	log.Debug("Tags Added")
	// Print tags
	for _, v := range t.List() {
		fmt.Println(v)
	}
}

// tag remove
var tagRemoveCmd = &cobra.Command{
	Use:   messages.GetUse("tag_remove"),
	Short: messages.GetShort("tag_remove"),
	Long:  messages.GetLong("tag_remove"),
	Run:   handleTagRemoveCmd,
}

func handleTagRemoveCmd(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugf("%s: start", cmd.Use)
	defer log.Debugf("%s: end", cmd.Use)

	if len(args) < 2 {
		log.Error("Usage: bundle tag remove <path> <tag> [<tag>...]")
		if err := cmd.Help(); err != nil {
			log.Error(err)
		}
		os.Exit(1)
	}

	path := args[0]
	// Validate path exists and is a directory (user error if not)
	if fi, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			log.Errorf("Path does not exist: %s", path)
			os.Exit(1)
		}
		log.Errorf("System error: %v", err)
		os.Exit(2)
	} else if !fi.IsDir() {
		log.Errorf("Path is not a directory: %s", path)
		os.Exit(1)
	}
	tags := args[1:]

	t, err := tag.Load(path)
	if err != nil {
		log.Errorf("System error: %v", err)
		os.Exit(2)
	}

	t.Remove(tags...)
	if err := t.Save(path); err != nil {
		log.Errorf("System error: %v", err)
		os.Exit(2)
	}

	jsonOut := jsonOutput
	if jsonOut {
		out := map[string]interface{}{
			"status": "removed",
			"path":   path,
			"tags":   tags,
		}
		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
		return
	}

	log.Debug("Tags Removed")
}

// tag list
var tagListCmd = &cobra.Command{
	Use:   messages.GetUse("tag_list"),
	Short: messages.GetShort("tag_list"),
	Long:  messages.GetLong("tag_list"),
	Run:   handleTagListCmd,
}

func handleTagListCmd(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugf("%s: start", cmd.Use)
	defer log.Debugf("%s: end", cmd.Use)

	if len(args) != 1 {
		log.Error("Usage: bundle tag list <path>")
		if err := cmd.Help(); err != nil {
			log.Error(err)
		}
		os.Exit(1)
	}

	path := args[0]
	// Validate path exists and is a directory (user error if not)
	if fi, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			log.Errorf("Path does not exist: %s", path)
			os.Exit(1)
		}
		log.Errorf("System error: %v", err)
		os.Exit(2)
	} else if !fi.IsDir() {
		log.Errorf("Path is not a directory: %s", path)
		os.Exit(1)
	}
	t, err := tag.Load(path)
	if err != nil {
		log.Errorf("System error: %v", err)
		os.Exit(2)
	}

	jsonOut := jsonOutput
	if jsonOut {
		out := map[string]interface{}{
			"path": path,
			"tags": t.List(),
		}
		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
		return
	}

	if len(t.Tags) == 0 {
		log.Debug("No tags")
		return
	}

	for _, v := range t.List() {
		fmt.Println(v)
	}
}

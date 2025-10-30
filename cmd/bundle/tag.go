/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"fmt"
	"os"

	"github.com/jvzantvoort/bundle/messages"
	"github.com/jvzantvoort/bundle/tag"
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
	Use:   "add",
	Short: "Add tags to a bundle",
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

	log.Info("Tags Added")
	// Print tags
	for _, v := range t.List() {
		fmt.Println(v)
	}
}

// tag remove
var tagRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove tags from a bundle",
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

	log.Info("Tags Removed")
}

// tag list
var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tags for a bundle",
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
	t, err := tag.Load(path)
	if err != nil {
		log.Errorf("System error: %v", err)
		os.Exit(2)
	}

	if len(t.Tags) == 0 {
		log.Info("No tags")
		return
	}

	for _, v := range t.List() {
		fmt.Println(v)
	}
}

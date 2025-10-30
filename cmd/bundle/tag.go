/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"os"

	"github.com/jvzantvoort/bundle/messages"
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

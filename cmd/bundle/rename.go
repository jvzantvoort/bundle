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

	if len(args) != 1 {
		log.Error("No path provided")
		if err := cmd.Help(); err != nil {
			log.Error(err)
		}
		os.Exit(1)
	}
}

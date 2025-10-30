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
}

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
}

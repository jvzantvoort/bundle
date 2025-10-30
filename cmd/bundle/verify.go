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

// VerifyCmd represents the verify command
var VerifyCmd = &cobra.Command{
	Use:   messages.GetUse("verify"),
	Short: messages.GetShort("verify"),
	Long:  messages.GetLong("verify"),
	Run:   handleVerifyCmd,
}

func init() {
	rootCmd.AddCommand(VerifyCmd)
	VerifyCmd.Flags().StringP("tag", "T", "", "mark every line with this tag")
	VerifyCmd.Flags().StringP("title", "t", "", "log the contents of this file")
}

func handleVerifyCmd(cmd *cobra.Command, args []string) {
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

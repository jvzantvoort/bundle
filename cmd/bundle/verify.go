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

	path := args[0]

	verified, corrupted, err := bundle.Verify(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Errorf("directory does not exist: %s", path)
			os.Exit(1)
		}
		log.Errorf("System error: %v", err)
		os.Exit(2)
	}

	if verified {
		log.Info("Bundle Integrity: VALID")
	} else {
		log.Info("Bundle Integrity: INVALID")
	}

	if jsonOutput {
		out := map[string]interface{}{
			"status":        "",
			"files_checked": 0,
			"last_verified": "",
			"corrupted_files": corrupted,
		}
		if verified {
			out["status"] = "valid"
		} else {
			out["status"] = "invalid"
		}
		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
	}
}

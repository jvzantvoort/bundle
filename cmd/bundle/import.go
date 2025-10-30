/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"os"

	"github.com/jvzantvoort/bundle/messages"
	"github.com/jvzantvoort/bundle/pool"
	"github.com/jvzantvoort/bundle/utils"
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"
)

// ImportCmd represents the import command
var ImportCmd = &cobra.Command{
	Use:   messages.GetUse("import"),
	Short: messages.GetShort("import"),
	Long:  messages.GetLong("import"),
	Run:   handleImportCmd,
}

func init() {
	rootCmd.AddCommand(ImportCmd)
	ImportCmd.Flags().StringP("pool", "p", "default", "pool name to import to")
	ImportCmd.Flags().BoolP("move", "m", false, "move bundle instead of copy")
}

func handleImportCmd(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugf("%s: start", cmd.Use)
	defer log.Debugf("%s: end", cmd.Use)

	if len(args) != 1 {
		log.Error("Usage: bundle import <path> [--pool <name>] [--move]")
		if err := cmd.Help(); err != nil {
			log.Error(err)
		}
		os.Exit(1)
	}

	bundlePath := args[0]
	poolName, _ := cmd.Flags().GetString("pool")
	moveFlag, _ := cmd.Flags().GetBool("move")

	// Get pool configuration
	p, err := pool.GetPool(poolName)
	if err != nil {
		log.Errorf("Pool error: %v", err)
		os.Exit(1)
	}

	// Import bundle
	if err := p.Import(bundlePath, moveFlag); err != nil {
		log.Errorf("Import failed: %v", err)
		os.Exit(2)
	}

	if jsonOutput {
		operation := "copied"
		if moveFlag {
			operation = "moved"
		}

		out := map[string]interface{}{
			"status":    "imported",
			"operation": operation,
			"pool":      poolName,
			"pool_root": p.Root,
			"source":    bundlePath,
		}
		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
		return
	}

	action := "copied"
	if moveFlag {
		action = "moved"
	}
	log.Infof("Bundle %s to pool '%s'", action, poolName)
	log.Infof("Pool: %s", p.Root)
}

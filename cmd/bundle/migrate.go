/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"os"

	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/messages"
	"github.com/jvzantvoort/bundle/pool"
	"github.com/jvzantvoort/bundle/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// MigrateCmd represents the migrate command
var MigrateCmd = &cobra.Command{
	Use:   messages.GetUse("migrate"),
	Short: messages.GetShort("migrate"),
	Long:  messages.GetLong("migrate"),
	Run:   handleMigrateCmd,
}

func init() {
	rootCmd.AddCommand(MigrateCmd)
	MigrateCmd.Flags().StringP("pool", "p", "", "migrate all bundles in this pool")
}

func handleMigrateCmd(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugf("%s: start", cmd.Use)
	defer log.Debugf("%s: end", cmd.Use)

	poolName, _ := cmd.Flags().GetString("pool")

	if (poolName == "") == (len(args) == 0) || len(args) > 1 {
		log.Error("Usage: bundle migrate <path> | bundle migrate --pool <name>")
		if err := cmd.Help(); err != nil {
			log.Error(err)
		}
		os.Exit(1)
	}

	if poolName != "" {
		migratePool(poolName)
		return
	}
	migrateBundle(args[0])
}

// migrateBundle upgrades a single bundle in place.
func migrateBundle(path string) {
	newChecksum, changed, err := bundle.Migrate(path)
	if err != nil {
		log.Errorf("Migration failed: %v", err)
		os.Exit(utils.ExitCodeFromError(err))
	}

	if jsonOutput {
		out := map[string]interface{}{
			"status":   map[bool]string{true: "migrated", false: "current"}[changed],
			"path":     path,
			"checksum": newChecksum,
		}
		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
		return
	}

	if changed {
		log.Infof("Bundle migrated, new checksum: %s", newChecksum)
	} else {
		log.Info("Bundle is already up to date")
	}
}

// migratePool upgrades every bundle in a pool and reports each one.
func migratePool(poolName string) {
	p, err := pool.GetPool(poolName)
	if err != nil {
		log.Errorf("Pool error: %v", err)
		os.Exit(1)
	}

	results, err := p.Migrate()
	if err != nil {
		log.Errorf("Migration failed: %v", err)
		os.Exit(2)
	}

	type bundleResult struct {
		Name     string `json:"name"`
		Checksum string `json:"checksum,omitempty"`
		Error    string `json:"error,omitempty"`
	}

	exitCode := 0
	out := []bundleResult{}
	for _, result := range results {
		entry := bundleResult{Name: result.Name, Checksum: result.NewChecksum}
		if result.Err != nil {
			entry.Error = result.Err.Error()
			exitCode = 1
			if !jsonOutput {
				log.Errorf("%s: %v", result.Name, result.Err)
			}
		} else if !jsonOutput {
			log.Infof("%s -> %s", result.Name, result.NewChecksum)
		}
		out = append(out, entry)
	}

	if jsonOutput {
		if err := utils.OutputJSON(map[string]interface{}{"pool": poolName, "bundles": out}); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
	} else {
		log.Infof("%d bundle(s) processed", len(results))
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

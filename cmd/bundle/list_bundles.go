/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/jvzantvoort/bundle/messages"
	"github.com/jvzantvoort/bundle/pool"
	"github.com/jvzantvoort/bundle/utils"
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"
)

// ListBundlesCmd represents the list_bundles command
var ListBundlesCmd = &cobra.Command{
	Use:   messages.GetUse("list_bundles"),
	Short: messages.GetShort("list_bundles"),
	Long:  messages.GetLong("list_bundles"),
	Run:   handleListBundlesCmd,
}

func init() {
	rootCmd.AddCommand(ListBundlesCmd)
	ListBundlesCmd.Flags().StringP("pool", "p", "default", "pool name to list bundles from")
}

func handleListBundlesCmd(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugf("%s: start", cmd.Use)
	defer log.Debugf("%s: end", cmd.Use)

	poolName, _ := cmd.Flags().GetString("pool")

	// Get pool configuration
	p, err := pool.GetPool(poolName)
	if err != nil {
		log.Errorf("Pool error: %v", err)
		os.Exit(1)
	}

	// List bundles
	bundles, err := p.ListBundles()
	if err != nil {
		log.Errorf("Failed to list bundles: %v", err)
		os.Exit(2)
	}

	if jsonOutput {
		type bundleInfo struct {
			Checksum  string `json:"checksum"`
			Title     string `json:"title"`
			Author    string `json:"author"`
			CreatedAt string `json:"created_at"`
		}

		bundleList := make([]bundleInfo, len(bundles))
		for i, meta := range bundles {
			bundleList[i] = bundleInfo{
				Checksum:  meta.BundleChecksum,
				Title:     meta.Title,
				Author:    meta.Author,
				CreatedAt: meta.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			}
		}

		out := map[string]interface{}{
			"pool":    poolName,
			"root":    p.Root,
			"bundles": bundleList,
			"count":   len(bundles),
		}
		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
		return
	}

	// Human-readable table output
	if len(bundles) == 0 {
		log.Info("No bundles found in pool")
		return
	}

	// Sort bundles by title
	sort.Slice(bundles, func(i, j int) bool {
		return bundles[i].Title < bundles[j].Title
	})

	table := utils.OutputTable(os.Stdout)
	table.Header("Checksum", "Title", "Author", "Created")

	for _, meta := range bundles {
		table.Append([]string{
			meta.BundleChecksum[:12] + "...", // Truncate checksum
			meta.Title,
			meta.Author,
			meta.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	fmt.Printf("Pool: %s (%s)\n\n", p.Title, p.Root)
	table.Render()
	fmt.Printf("\nTotal: %d bundles\n", len(bundles))
}

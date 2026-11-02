/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"os"

	"github.com/jvzantvoort/bundle/messages"
	"github.com/jvzantvoort/bundle/remote"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// PushCmd represents the push command
var PushCmd = &cobra.Command{
	Use:   messages.GetUse("push"),
	Short: messages.GetShort("push"),
	Long:  messages.GetLong("push"),
	Run:   handlePushCmd,
}

func init() {
	rootCmd.AddCommand(PushCmd)
	PushCmd.Flags().StringVar(&remoteName, "remote", "", "configured server to push to (default: default_remote from the configuration)")
	addRemotePoolFlag(PushCmd)
}

func handlePushCmd(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugf("%s: start", cmd.Use)
	defer log.Debugf("%s: end", cmd.Use)

	if len(args) != 1 {
		log.Error("Usage: bundle push <path> [--remote <name>] [--remote-pool <name>]")
		if err := cmd.Help(); err != nil {
			log.Error(err)
		}
		os.Exit(1)
	}

	name := remoteName
	if name == "" {
		name = remote.DefaultName()
	}
	if name == "" {
		log.Error("No remote selected: use --remote or set default_remote in the configuration")
		os.Exit(1)
	}

	path := args[0]
	client := remoteClient(name)
	ctx, stop := remoteContext()
	defer stop()

	result, err := client.Push(ctx, path)
	if err != nil {
		log.Errorf("Push failed: %v", err)
		exitCode(err)
	}

	if jsonOutput {
		outputRemoteJSON(client, map[string]interface{}{
			"status":     result.Status,
			"path":       path,
			"checksum":   result.Checksum,
			"files":      result.FileCount,
			"size_bytes": result.SizeBytes,
		})
		return
	}

	if result.Status == remote.PushStatusExists {
		log.Infof("Bundle already on '%s', pool '%s': %s", client.Name, client.Pool, result.Checksum)
		return
	}
	log.Infof("Bundle pushed to '%s', pool '%s': %s", client.Name, client.Pool, result.Checksum)
	log.Infof("%d file(s), %s", result.FileCount, formatBytes(result.SizeBytes))
}

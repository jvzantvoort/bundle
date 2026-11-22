package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/media"
	"github.com/jvzantvoort/bundle/messages"
	"github.com/jvzantvoort/bundle/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// UpdateCmd represents the update command.
//
// It updates metadata that is derived from a bundle's files, in place,
// without changing the files, checksum, title or tags. --media stores the
// media metadata of the bundle's images and videos in .bundle/MEDIA.json.
//
// Usage:
//
//	bundle update <path> --media [--force]
var UpdateCmd = &cobra.Command{
	Use:   messages.GetUse("update"),
	Short: messages.GetShort("update"),
	Long:  messages.GetLong("update"),
	Run:   handleUpdateCmd,
}

func init() {
	rootCmd.AddCommand(UpdateCmd)
	UpdateCmd.Flags().Bool("media", false, "Look up EXIF data, resolution, capture time, GPS and device of images and videos")
	UpdateCmd.Flags().Bool("force", false, "Read every file again, also when its checksum is unchanged")
}

// handleUpdateCmd processes the update command.
func handleUpdateCmd(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugf("%s: start", cmd.Use)
	defer log.Debugf("%s: end", cmd.Use)

	updateMedia, _ := cmd.Flags().GetBool("media")
	force, _ := cmd.Flags().GetBool("force")
	if len(args) != 1 {
		log.Error("Usage: bundle update <path> --media [--force]")
		os.Exit(1)
	}
	if !updateMedia {
		log.Error("nothing to update: choose what to update, for example --media")
		os.Exit(1)
	}
	path := args[0]

	b, err := bundle.Load(path)
	if err != nil {
		if errors.Is(err, utils.ErrNotABundle) || os.IsNotExist(err) {
			log.Errorf("Not a bundle: %s", path)
			os.Exit(1)
		}
		log.Errorf("System error: %v", err)
		os.Exit(2)
	}

	summary, err := media.Scan(path, b.Files.Records, media.Options{
		Force:     force,
		Generator: "bundle " + strings.Fields(versionString())[0],
	})
	if err != nil {
		log.Errorf("Failed to update media metadata: %v", err)
		os.Exit(2)
	}

	if jsonOutput {
		out := map[string]any{"status": "updated", "path": path, "media": summary}
		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
		return
	}
	fmt.Printf("Media metadata updated: %s\n", media.Path(path))
	fmt.Printf("  %d added, %d updated, %d unchanged, %d unsupported, %d removed\n",
		summary.Added, summary.Updated, summary.Skipped, summary.Unsupported, summary.Removed)
	fmt.Printf("  %d media files, %d devices\n", summary.Files, summary.Devices)
}

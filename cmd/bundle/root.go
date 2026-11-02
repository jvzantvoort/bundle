package main

import (
	"os"
	"runtime/debug"

	"github.com/jvzantvoort/bundle/config"
	"github.com/jvzantvoort/bundle/messages"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var verbose bool
var jsonOutput bool

// version and revision are set at build time with -ldflags.
var version string
var revision string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   messages.GetUse("root"),
	Short: messages.GetShort("root"),
	Long:  messages.GetLong("root"),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// versionString returns the version set at build time. Binaries from go install
// have no build flags, so it falls back to the module version.
func versionString() string {
	current := version
	if current == "" {
		current = "dev"
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			current = info.Main.Version
		}
	}
	if revision != "" {
		current += " (" + revision + ")"
	}
	return current
}

func init() {
	// Initialize configuration system
	config.InitConfig()

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Version = versionString()

	// Setup logging
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:          true,
		DisableLevelTruncation: true,
		TimestampFormat:        "2006-01-02 15:04:05",
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output JSON")
}

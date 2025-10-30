package main

import (
	"fmt"
	"os"

	"github.com/jvzantvoort/bundle/config"
	"github.com/spf13/cobra"
)

var (
	verbose bool
	quiet   bool
)

var rootCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Bundle - content-addressable file bundle management",
	Long:  `Bundle Library provides tools for creating and managing immutable file bundles with SHA256-based integrity verification.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.SetLogLevel(verbose, quiet)
	},
}

func init() {
	config.InitConfig()
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output (DEBUG level)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Quiet output (errors only)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

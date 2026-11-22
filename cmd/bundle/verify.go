/*
Copyright © 2025 John van Zantvoort <john@vanzantvoort.org>
*/
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jvzantvoort/bundle/messages"
	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/checksum"
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
	addRemoteFlags(VerifyCmd)
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

	if remoteName != "" {
		remoteVerify(path)
		return
	}

	result, err := bundle.VerifyDetailed(path)
	if err != nil {
		code := utils.ExitCodeFromError(err)
		if code == 2 {
			log.Errorf("System error: %v", err)
		} else {
			log.Error(err)
		}
		os.Exit(code)
	}

	if jsonOutput {
		out := map[string]interface{}{
			"status":          map[bool]string{true: "valid", false: "invalid"}[result.Verified],
			"files_checked":   result.FilesChecked,
			"last_verified":   result.CheckedAt.UTC().Format("2006-01-02T15:04:05Z"),
			"corrupted_files": checksum.Paths(result.Problems),
			"problems":        result.Problems,
		}
		if err := utils.OutputJSON(out); err != nil {
			log.Errorf("failed to output json: %v", err)
			os.Exit(2)
		}
	} else {
		printVerification(result.Verified, result.FilesChecked, result.CheckedAt, result.Problems)
	}

	if !result.Verified {
		os.Exit(1)
	}
}

// printVerification prints a verification result in the table format of the
// CLI contract, for a local or a server-side verification.
func printVerification(verified bool, filesChecked int, checkedAt time.Time, problems []checksum.Problem) {
	header := "Bundle Integrity: VALID"
	if !verified {
		header = "Bundle Integrity: INVALID"
	}
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)))
	fmt.Printf("Files Checked: %d\n", filesChecked)
	if verified {
		fmt.Printf("Last Verified: %s\n", checkedAt.Local().Format("2006-01-02 15:04:05"))
		return
	}
	fmt.Println("Corrupted Files:")
	for _, problem := range problems {
		fmt.Printf("- %s (%s)\n", problem.Path, describeProblem(problem))
	}
}

// describeProblem explains a problem in a few words.
func describeProblem(problem checksum.Problem) string {
	switch problem.Reason {
	case checksum.ReasonChecksumMismatch:
		if problem.Expected != "" && problem.Found != "" {
			return fmt.Sprintf("expected: %s..., found: %s...", shortChecksum(problem.Expected), shortChecksum(problem.Found))
		}
		return "checksum mismatch"
	case checksum.ReasonMissing:
		return "file missing"
	case checksum.ReasonUnlisted:
		return "not in manifest"
	case checksum.ReasonManifest:
		return "manifest does not match bundle checksum"
	case "":
		return "corrupted"
	default:
		return problem.Reason
	}
}

// shortChecksum returns the first 8 characters of a checksum.
func shortChecksum(sum string) string {
	if len(sum) > 8 {
		return sum[:8]
	}
	return sum
}

// Package main implements the bundle CLI tool.
//
// It provides common utility functions shared across CLI commands for
// flag parsing and command handling.
package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// GetString retrieves a string flag value from a cobra command.
//
// It safely retrieves string flags with debug logging of the value.
// Returns empty string if flag is not set or doesn't exist.
//
// Example:
//
//	title := GetString(*cmd, "title")
//	if title != "" {
//	    fmt.Printf("Title: %s\n", title)
//	}
//
// Parameters:
//   - cmd: cobra command containing the flag
//   - name: flag name
//
// Returns:
//   - string: flag value or empty string
func GetString(cmd cobra.Command, name string) string {
	retv, _ := cmd.Flags().GetString(name)
	if len(retv) != 0 {
		log.Debugf("%s returned %s", name, retv)
	} else {
		log.Debugf("%s returned nothing", name)
	}
	return retv
}

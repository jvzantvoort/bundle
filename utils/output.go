// Package utils provides utility functions for CLI operations, error handling,
// and output formatting.
//
// It includes helpers for:
//   - JSON and table output formatting
//   - Error message formatting
//   - Exit code mapping
//   - File path operations
//   - Bundle directory detection
//
// Example usage:
//
//	// Output JSON
//	data := map[string]interface{}{"status": "ok", "files": 42}
//	utils.OutputJSON(data)
//
//	// Check if directory is a bundle
//	if utils.IsBundleDir("/path/to/dir") {
//	    fmt.Println("Is a bundle")
//	}
//
//	// Get exit code from error
//	code := utils.ExitCodeFromError(err)
//	os.Exit(code)
package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
)

// OutputJSON writes data as JSON to stdout.
//
// It serializes the data with 2-space indentation for readability.
//
// Example:
//
//	data := map[string]interface{}{
//	    "status": "created",
//	    "path": "/path/to/bundle",
//	    "files": 42,
//	}
//	if err := utils.OutputJSON(data); err != nil {
//	    log.Fatal(err)
//	}
//
// Output:
//
//	{
//	  "status": "created",
//	  "path": "/path/to/bundle",
//	  "files": 42
//	}
//
// Parameters:
//   - data: any JSON-serializable value
//
// Returns:
//   - error: if JSON encoding fails or write to stdout fails
func OutputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// OutputTable creates a table writer configured for bundle output.
//
// It returns a tablewriter.Table configured for formatting tabular output.
// The caller is responsible for setting headers, adding rows, and rendering.
//
// Example:
//
//	table := utils.OutputTable(os.Stdout)
//	table.SetHeader([]string{"File", "Checksum", "Size"})
//	table.Append([]string{"file1.txt", "abc123...", "1024"})
//	table.Append([]string{"file2.pdf", "def456...", "2048"})
//	table.Render()
//
// Parameters:
//   - writer: io.Writer to write table output (typically os.Stdout)
//
// Returns:
//   - *tablewriter.Table: configured table writer
func OutputTable(writer io.Writer) *tablewriter.Table {
	return tablewriter.NewWriter(writer)
}

// ErrorMessage writes an error message to stderr.
//
// It formats the message with "Error: " prefix and writes to stderr.
//
// Example:
//
//	utils.ErrorMessage("bundle not found: %s", path)
//	// Output to stderr: Error: bundle not found: /path/to/bundle
//
// Parameters:
//   - format: printf-style format string
//   - args: format arguments
func ErrorMessage(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// SystemErrorMessage writes a system error message to stderr.
//
// It formats the message with "System error: " prefix and writes to stderr.
// Use for errors that indicate system/infrastructure problems (I/O, permissions).
//
// Example:
//
//	utils.SystemErrorMessage("failed to write file: %v", err)
//	// Output to stderr: System error: failed to write file: permission denied
//
// Parameters:
//   - format: printf-style format string
//   - args: format arguments
func SystemErrorMessage(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "System error: "+format+"\n", args...)
}

package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
)

// OutputJSON writes data as JSON to stdout
func OutputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// OutputTable creates a table writer configured for bundle output
func OutputTable(writer io.Writer) *tablewriter.Table {
	return tablewriter.NewWriter(writer)
}

// ErrorMessage writes an error message to stderr
func ErrorMessage(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// SystemErrorMessage writes a system error message to stderr
func SystemErrorMessage(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "System error: "+format+"\n", args...)
}

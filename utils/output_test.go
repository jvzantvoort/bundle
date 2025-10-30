package utils

import (
	"bytes"
	"testing"
)

func TestOutputJSON(t *testing.T) {
	data := map[string]interface{}{
		"status": "success",
		"count":  42,
	}

	err := OutputJSON(data)
	if err != nil {
		t.Errorf("OutputJSON() error = %v", err)
	}
}

func TestOutputTable(t *testing.T) {
	var buf bytes.Buffer
	table := OutputTable(&buf)

	if table == nil {
		t.Error("OutputTable() returned nil")
	}
}

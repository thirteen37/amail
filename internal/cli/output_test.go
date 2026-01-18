package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

func TestIsJSONOutput(t *testing.T) {
	// Save original values
	origForceJSON := forceJSON
	origForceText := forceText
	defer func() {
		forceJSON = origForceJSON
		forceText = origForceText
	}()

	tests := []struct {
		name      string
		forceJSON bool
		forceText bool
		want      bool
	}{
		{
			name:      "force JSON overrides everything",
			forceJSON: true,
			forceText: true,
			want:      true,
		},
		{
			name:      "force text when no JSON flag",
			forceJSON: false,
			forceText: true,
			want:      false,
		},
		{
			name:      "force JSON flag",
			forceJSON: true,
			forceText: false,
			want:      true,
		},
		// Note: can't easily test TTY detection in unit tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forceJSON = tt.forceJSON
			forceText = tt.forceText
			if got := IsJSONOutput(); got != tt.want {
				t.Errorf("IsJSONOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrintJSON(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	testData := map[string]string{"key": "value"}
	err := PrintJSON(testData)
	if err != nil {
		t.Errorf("PrintJSON() error = %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result Response
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("Failed to parse JSON output: %v", err)
	}

	if !result.Success {
		t.Error("Expected success=true")
	}
	if result.Error != nil {
		t.Error("Expected error=nil")
	}
	if result.Data == nil {
		t.Error("Expected data to be present")
	}
}

func TestPrintJSONError(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	testErr := errors.New("test error message")
	err := PrintJSONError(testErr, "TEST_CODE")
	if err != nil {
		t.Errorf("PrintJSONError() error = %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result Response
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("Failed to parse JSON output: %v", err)
	}

	if result.Success {
		t.Error("Expected success=false")
	}
	if result.Error == nil {
		t.Error("Expected error to be present")
	}
	if result.Error.Message != "test error message" {
		t.Errorf("Expected error message 'test error message', got '%s'", result.Error.Message)
	}
	if result.Error.Code != "TEST_CODE" {
		t.Errorf("Expected error code 'TEST_CODE', got '%s'", result.Error.Code)
	}
	if result.Data != nil {
		t.Error("Expected data=nil for error response")
	}
}

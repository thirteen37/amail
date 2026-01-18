package cli

import (
	"encoding/json"
	"os"

	"golang.org/x/term"
)

var (
	forceJSON bool // --json flag
	forceText bool // --text flag
)

// Response wraps all JSON output in a consistent envelope
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo provides structured error information
type ErrorInfo struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// IsJSONOutput returns true if JSON output should be used
// Priority: --json flag > --text flag > TTY detection
func IsJSONOutput() bool {
	if forceJSON {
		return true
	}
	if forceText {
		return false
	}
	// Auto-detect: JSON when stdout is not a TTY (piped/redirected)
	return !term.IsTerminal(int(os.Stdout.Fd()))
}

// PrintJSON outputs data in the standard JSON envelope format
func PrintJSON(data interface{}) error {
	resp := Response{
		Success: true,
		Data:    data,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

// PrintJSONError outputs an error in the standard JSON envelope format
func PrintJSONError(err error, code string) error {
	resp := Response{
		Success: false,
		Error: &ErrorInfo{
			Message: err.Error(),
			Code:    code,
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

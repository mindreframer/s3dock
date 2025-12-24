package internal

import (
	"encoding/json"
	"os"
)

// OutputFormat represents the output format type
type OutputFormat int

const (
	// OutputFormatText is human-readable text output (default)
	OutputFormatText OutputFormat = iota
	// OutputFormatJSON is JSON output for programmatic consumption
	OutputFormatJSON
)

// OutputConfig holds the global output configuration
type OutputConfig struct {
	Format OutputFormat
}

// globalOutputConfig is the global output configuration
var globalOutputConfig = &OutputConfig{Format: OutputFormatText}

// SetOutputFormat sets the global output format
func SetOutputFormat(format OutputFormat) {
	globalOutputConfig.Format = format
}

// GetOutputFormat returns the current global output format
func GetOutputFormat() OutputFormat {
	return globalOutputConfig.Format
}

// IsJSONOutput returns true if JSON output is enabled
func IsJSONOutput() bool {
	return globalOutputConfig.Format == OutputFormatJSON
}

// OutputJSON writes a value as JSON to stdout
func OutputJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// OutputResult outputs the result in the appropriate format based on global config
// For JSON format, it wraps the result in a CommandResult with success=true
// For text format, it does nothing (caller handles text output)
func OutputResult(command string, data interface{}) error {
	if globalOutputConfig.Format == OutputFormatJSON {
		result := CommandResult{
			Success: true,
			Command: command,
			Data:    data,
		}
		return OutputJSON(result)
	}
	return nil
}

// OutputError outputs an error in the appropriate format based on global config
// For JSON format, it outputs a CommandResult with success=false
// For text format, it logs the error
func OutputError(command string, err error) {
	if globalOutputConfig.Format == OutputFormatJSON {
		result := CommandResult{
			Success: false,
			Command: command,
			Error:   err.Error(),
		}
		OutputJSON(result)
	} else {
		LogError("%v", err)
	}
}

package internal

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestLogLevels(t *testing.T) {
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	// Test each log level
	testCases := []struct {
		name        string
		level       LogLevel
		expected    []string
		notExpected []string
	}{
		{
			name:        "Error level only",
			level:       LogLevelError,
			expected:    []string{"[ERROR]"},
			notExpected: []string{"[INFO]", "[DEBUG]"},
		},
		{
			name:        "Info level",
			level:       LogLevelInfo,
			expected:    []string{"[ERROR]", "[INFO]"},
			notExpected: []string{"[DEBUG]"},
		},
		{
			name:        "Debug level",
			level:       LogLevelDebug,
			expected:    []string{"[ERROR]", "[INFO]", "[DEBUG]"},
			notExpected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset output buffer
			w.Truncate(0)

			// Set log level
			SetLogLevel(tc.level)

			// Log messages at each level
			LogError("test error message")
			LogInfo("test info message")
			LogDebug("test debug message")

			// Read output
			w.Close()
			var buf bytes.Buffer
			_, err := buf.ReadFrom(r)
			if err != nil {
				t.Fatalf("Failed to read output: %v", err)
			}
			output := buf.String()

			// Check expected messages
			for _, expected := range tc.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got: %s", expected, output)
				}
			}

			// Check not expected messages
			for _, notExpected := range tc.notExpected {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output to NOT contain '%s', got: %s", notExpected, output)
				}
			}

			// Recreate pipe for next test
			r, w, _ = os.Pipe()
			os.Stderr = w
		})
	}
}

func TestLogFormatting(t *testing.T) {
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	// Set to debug level to see all messages
	SetLogLevel(LogLevelDebug)

	// Test formatted messages
	LogInfo("Testing %s with %d parameters", "formatting", 2)
	LogDebug("Debug message with %s", "variable")

	// Read output
	w.Close()
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}
	output := buf.String()

	// Check that formatting worked
	if !strings.Contains(output, "Testing formatting with 2 parameters") {
		t.Errorf("Expected formatted info message, got: %s", output)
	}

	if !strings.Contains(output, "Debug message with variable") {
		t.Errorf("Expected formatted debug message, got: %s", output)
	}

	// Check timestamp format
	if !strings.Contains(output, "2025-") {
		t.Errorf("Expected timestamp in output, got: %s", output)
	}
}

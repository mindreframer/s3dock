package internal

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestSetOutputFormat(t *testing.T) {
	// Reset to default after test
	defer SetOutputFormat(OutputFormatText)

	// Test setting to JSON
	SetOutputFormat(OutputFormatJSON)
	if GetOutputFormat() != OutputFormatJSON {
		t.Errorf("Expected OutputFormatJSON, got %v", GetOutputFormat())
	}

	// Test setting back to Text
	SetOutputFormat(OutputFormatText)
	if GetOutputFormat() != OutputFormatText {
		t.Errorf("Expected OutputFormatText, got %v", GetOutputFormat())
	}
}

func TestIsJSONOutput(t *testing.T) {
	// Reset to default after test
	defer SetOutputFormat(OutputFormatText)

	// Default should be text
	if IsJSONOutput() {
		t.Error("Expected IsJSONOutput to be false by default")
	}

	// Set to JSON
	SetOutputFormat(OutputFormatJSON)
	if !IsJSONOutput() {
		t.Error("Expected IsJSONOutput to be true after setting JSON format")
	}
}

func TestOutputJSON(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"key": "value"}
	err := OutputJSON(data)
	if err != nil {
		t.Errorf("OutputJSON returned error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify it's valid JSON
	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("OutputJSON did not produce valid JSON: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("Expected key=value, got key=%s", result["key"])
	}
}

func TestCommandResult(t *testing.T) {
	result := CommandResult{
		Success: true,
		Command: "test",
		Data:    map[string]string{"foo": "bar"},
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Errorf("Failed to marshal CommandResult: %v", err)
	}

	var decoded CommandResult
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Errorf("Failed to unmarshal CommandResult: %v", err)
	}

	if decoded.Success != true {
		t.Error("Expected Success to be true")
	}
	if decoded.Command != "test" {
		t.Errorf("Expected Command to be 'test', got %s", decoded.Command)
	}
}

func TestOutputResult_JSONFormat(t *testing.T) {
	// Reset to default after test
	defer SetOutputFormat(OutputFormatText)

	SetOutputFormat(OutputFormatJSON)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := ListAppsResult{Apps: []string{"app1", "app2"}}
	err := OutputResult("list apps", data)
	if err != nil {
		t.Errorf("OutputResult returned error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify it's valid JSON with correct structure
	var result CommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("OutputResult did not produce valid JSON: %v", err)
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}
	if result.Command != "list apps" {
		t.Errorf("Expected Command to be 'list apps', got %s", result.Command)
	}
}

func TestOutputResult_TextFormat(t *testing.T) {
	// Reset to default after test
	defer SetOutputFormat(OutputFormatText)

	SetOutputFormat(OutputFormatText)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := ListAppsResult{Apps: []string{"app1", "app2"}}
	err := OutputResult("list apps", data)
	if err != nil {
		t.Errorf("OutputResult returned error: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// In text format, OutputResult should do nothing (return nil)
	if output != "" {
		t.Errorf("Expected no output in text format, got: %s", output)
	}
}

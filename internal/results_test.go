package internal

import (
	"encoding/json"
	"testing"
)

func TestBuildResult_JSON(t *testing.T) {
	result := BuildResult{
		ImageTag: "myapp:20250721-2118-f7a5a27",
		AppName:  "myapp",
		GitHash:  "f7a5a27",
		GitTime:  "20250721-2118",
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded BuildResult
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ImageTag != result.ImageTag {
		t.Errorf("ImageTag mismatch: expected %s, got %s", result.ImageTag, decoded.ImageTag)
	}
	if decoded.AppName != result.AppName {
		t.Errorf("AppName mismatch: expected %s, got %s", result.AppName, decoded.AppName)
	}
}

func TestPushResult_JSON(t *testing.T) {
	result := PushResult{
		ImageRef: "myapp:20250721-2118-f7a5a27",
		S3Key:    "images/myapp/202507/myapp-20250721-2118-f7a5a27.tar.gz",
		Checksum: "abc123",
		Size:     1024,
		Skipped:  false,
		Archived: false,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded PushResult
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Skipped != result.Skipped {
		t.Errorf("Skipped mismatch")
	}
}

func TestListAppsResult_JSON(t *testing.T) {
	result := ListAppsResult{
		Apps: []string{"app1", "app2", "app3"},
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ListAppsResult
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Apps) != 3 {
		t.Errorf("Expected 3 apps, got %d", len(decoded.Apps))
	}
}

func TestImageInfoToJSON(t *testing.T) {
	info := ImageInfo{
		AppName:   "myapp",
		Tag:       "20250721-2118-f7a5a27",
		S3Path:    "images/myapp/202507/myapp-20250721-2118-f7a5a27.tar.gz",
		YearMonth: "202507",
	}

	jsonInfo := info.ToJSON()

	if jsonInfo.AppName != info.AppName {
		t.Errorf("AppName mismatch")
	}
	if jsonInfo.Tag != info.Tag {
		t.Errorf("Tag mismatch")
	}
	if jsonInfo.S3Path != info.S3Path {
		t.Errorf("S3Path mismatch")
	}
	if jsonInfo.YearMonth != info.YearMonth {
		t.Errorf("YearMonth mismatch")
	}
}

func TestTagInfoToJSON(t *testing.T) {
	info := TagInfo{
		AppName:     "myapp",
		Version:     "v1.2.0",
		TargetImage: "myapp:20250721-2118-f7a5a27",
		S3Path:      "tags/myapp/v1.2.0.json",
	}

	jsonInfo := info.ToJSON()

	if jsonInfo.Version != info.Version {
		t.Errorf("Version mismatch")
	}
	if jsonInfo.TargetImage != info.TargetImage {
		t.Errorf("TargetImage mismatch")
	}
}

func TestEnvInfoToJSON(t *testing.T) {
	info := EnvInfo{
		AppName:     "myapp",
		Environment: "production",
		TargetType:  TargetTypeTag,
		TargetPath:  "tags/myapp/v1.2.0.json",
		SourceTag:   "v1.2.0",
		SourceImage: "myapp:20250721-2118-f7a5a27",
	}

	jsonInfo := info.ToJSON()

	if jsonInfo.Environment != info.Environment {
		t.Errorf("Environment mismatch")
	}
	if jsonInfo.TargetType != string(info.TargetType) {
		t.Errorf("TargetType mismatch")
	}
	if jsonInfo.SourceTag != info.SourceTag {
		t.Errorf("SourceTag mismatch")
	}
}

func TestCurrentResult_JSON(t *testing.T) {
	result := CurrentResult{
		AppName:     "myapp",
		Environment: "production",
		ImageRef:    "myapp:20250721-2118-f7a5a27",
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CurrentResult
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ImageRef != result.ImageRef {
		t.Errorf("ImageRef mismatch")
	}
}

func TestVersionResult_JSON(t *testing.T) {
	result := VersionResult{
		Version: "v1.0.0",
		Commit:  "abc123",
		Date:    "2025-01-01",
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded VersionResult
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Version != result.Version {
		t.Errorf("Version mismatch")
	}
}

func TestListTagForResult_JSON(t *testing.T) {
	result := ListTagForResult{
		AppName:     "myapp",
		Environment: "production",
		Tag:         "v1.2.0",
		Direct:      false,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ListTagForResult
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Tag != result.Tag {
		t.Errorf("Tag mismatch")
	}
	if decoded.Direct != result.Direct {
		t.Errorf("Direct mismatch")
	}
}

func TestCommandResultWithError_JSON(t *testing.T) {
	result := CommandResult{
		Success: false,
		Command: "current",
		Error:   "environment pointer not found: myapp/nonexistent",
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CommandResult
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Success != false {
		t.Error("Expected Success to be false")
	}
	if decoded.Error == "" {
		t.Error("Expected Error to be set")
	}
}

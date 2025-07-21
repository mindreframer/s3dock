package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateImagePointer(t *testing.T) {
	imageS3Path := "images/myapp/202507/myapp-20250721-1430-abc1234.tar.gz"
	gitHash := "abc1234"
	gitTime := "20250721-1430"
	sourceImage := "myapp:20250721-1430-abc1234"

	pointer, err := CreateImagePointer(imageS3Path, gitHash, gitTime, sourceImage)

	assert.NoError(t, err)
	assert.Equal(t, TargetTypeImage, pointer.TargetType)
	assert.Equal(t, imageS3Path, pointer.TargetPath)
	assert.Equal(t, gitHash, pointer.GitHash)
	assert.Equal(t, gitTime, pointer.GitTime)
	assert.Equal(t, sourceImage, pointer.SourceImage)
	assert.True(t, pointer.PromotedAt.Before(time.Now().Add(time.Second)))
}

func TestCreateTagPointer(t *testing.T) {
	tagS3Path := "tags/myapp/v1.2.0.json"
	gitHash := "abc1234"
	gitTime := "20250721-1430"
	sourceImage := "myapp:20250721-1430-abc1234"
	sourceTag := "v1.2.0"

	pointer, err := CreateTagPointer(tagS3Path, gitHash, gitTime, sourceImage, sourceTag)

	assert.NoError(t, err)
	assert.Equal(t, TargetTypeTag, pointer.TargetType)
	assert.Equal(t, tagS3Path, pointer.TargetPath)
	assert.Equal(t, gitHash, pointer.GitHash)
	assert.Equal(t, gitTime, pointer.GitTime)
	assert.Equal(t, sourceImage, pointer.SourceImage)
	assert.Equal(t, sourceTag, pointer.SourceTag)
	assert.True(t, pointer.PromotedAt.Before(time.Now().Add(time.Second)))
}

func TestPointerMetadataJSON(t *testing.T) {
	pointer := &PointerMetadata{
		TargetType:  TargetTypeImage,
		TargetPath:  "images/myapp/202507/myapp-20250721-1430-abc1234.tar.gz",
		PromotedAt:  time.Date(2025, 7, 21, 14, 30, 0, 0, time.UTC),
		PromotedBy:  "testuser",
		GitHash:     "abc1234",
		GitTime:     "20250721-1430",
		SourceImage: "myapp:20250721-1430-abc1234",
	}

	jsonData, err := pointer.ToJSON()
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "\"target_type\": \"image\"")
	assert.Contains(t, string(jsonData), "\"git_hash\": \"abc1234\"")

	parsed, err := PointerMetadataFromJSON(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, pointer.TargetType, parsed.TargetType)
	assert.Equal(t, pointer.TargetPath, parsed.TargetPath)
	assert.Equal(t, pointer.GitHash, parsed.GitHash)
	assert.Equal(t, pointer.GitTime, parsed.GitTime)
}

func TestGenerateTagKey(t *testing.T) {
	tests := []struct {
		appName  string
		version  string
		expected string
	}{
		{"myapp", "v1.2.0", "tags/myapp/v1.2.0.json"},
		{"api", "v2.1.5", "tags/api/v2.1.5.json"},
	}

	for _, test := range tests {
		result := GenerateTagKey(test.appName, test.version)
		assert.Equal(t, test.expected, result)
	}
}

func TestGeneratePointerKey(t *testing.T) {
	tests := []struct {
		appName     string
		environment string
		expected    string
	}{
		{"myapp", "production", "pointers/myapp/production.json"},
		{"api", "staging", "pointers/api/staging.json"},
	}

	for _, test := range tests {
		result := GeneratePointerKey(test.appName, test.environment)
		assert.Equal(t, test.expected, result)
	}
}

func TestParseImageReference(t *testing.T) {
	tests := []struct {
		imageRef        string
		expectedApp     string
		expectedGitTime string
		expectedGitHash string
		expectError     bool
	}{
		{
			"myapp:20250721-1430-abc1234",
			"myapp", "20250721-1430", "abc1234", false,
		},
		{
			"api:20250720-0900-def5678",
			"api", "20250720-0900", "def5678", false,
		},
		{
			"invalid-format",
			"", "", "", true,
		},
		{
			"app:invalid-tag-format",
			"", "", "", true,
		},
	}

	for _, test := range tests {
		t.Run(test.imageRef, func(t *testing.T) {
			appName, gitTime, gitHash, err := ParseImageReference(test.imageRef)

			if test.expectError {
				assert.Error(t, err, "Expected error for: %s", test.imageRef)
			} else {
				assert.NoError(t, err, "Unexpected error for: %s", test.imageRef)
				assert.Equal(t, test.expectedApp, appName)
				assert.Equal(t, test.expectedGitTime, gitTime)
				assert.Equal(t, test.expectedGitHash, gitHash)
			}
		})
	}
}
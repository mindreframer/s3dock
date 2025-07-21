package internal

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculateMetadata(t *testing.T) {
	data := strings.NewReader("test image data")
	gitHash := "abc1234"
	gitTime := "20250721-1430"
	imageTag := "myapp:latest"
	appName := "myapp"

	metadata, size, err := CalculateMetadata(data, gitHash, gitTime, imageTag, appName)

	assert.NoError(t, err)
	assert.Equal(t, int64(15), size)
	assert.Equal(t, "bf6d3bdce17efe14125f44654d4941cb", metadata.Checksum) // MD5 of "test image data"
	assert.Equal(t, gitHash, metadata.GitHash)
	assert.Equal(t, gitTime, metadata.GitTime)
	assert.Equal(t, imageTag, metadata.ImageTag)
	assert.Equal(t, appName, metadata.AppName)
	assert.Equal(t, size, metadata.Size)
	assert.True(t, metadata.CreatedAt.Before(time.Now().Add(time.Second)))
}

func TestImageMetadata_ToJSON(t *testing.T) {
	metadata := &ImageMetadata{
		Checksum:  "abc123",
		Size:      1024,
		CreatedAt: time.Date(2025, 7, 21, 14, 30, 0, 0, time.UTC),
		GitHash:   "def456",
		GitTime:   "20250721-1430",
		ImageTag:  "myapp:latest",
		AppName:   "myapp",
	}

	jsonData, err := metadata.ToJSON()

	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "\"checksum\": \"abc123\"")
	assert.Contains(t, string(jsonData), "\"size\": 1024")
	assert.Contains(t, string(jsonData), "\"git_hash\": \"def456\"")
}

func TestImageMetadataFromJSON(t *testing.T) {
	jsonData := `{
		"checksum": "abc123",
		"size": 1024,
		"created_at": "2025-07-21T14:30:00Z",
		"git_hash": "def456",
		"git_time": "20250721-1430",
		"image_tag": "myapp:latest",
		"app_name": "myapp"
	}`

	metadata, err := ImageMetadataFromJSON([]byte(jsonData))

	assert.NoError(t, err)
	assert.Equal(t, "abc123", metadata.Checksum)
	assert.Equal(t, int64(1024), metadata.Size)
	assert.Equal(t, "def456", metadata.GitHash)
	assert.Equal(t, "20250721-1430", metadata.GitTime)
	assert.Equal(t, "myapp:latest", metadata.ImageTag)
	assert.Equal(t, "myapp", metadata.AppName)
}

func TestGenerateMetadataKey(t *testing.T) {
	tests := []struct {
		imageKey     string
		expectedMeta string
	}{
		{
			"images/myapp/202507/myapp-20250721-1430-abc123.tar.gz",
			"images/myapp/202507/myapp-20250721-1430-abc123.json",
		},
		{
			"images/test/202501/test-20250101-0000-xyz789.tar.gz",
			"images/test/202501/test-20250101-0000-xyz789.json",
		},
	}

	for _, test := range tests {
		result := GenerateMetadataKey(test.imageKey)
		assert.Equal(t, test.expectedMeta, result)
	}
}

func TestGenerateArchiveKeys(t *testing.T) {
	imageKey := "images/myapp/202507/myapp-20250721-1430-abc123.tar.gz"
	timestamp := "20250722-1018"

	archiveImageKey, archiveMetaKey := GenerateArchiveKeys(imageKey, timestamp)

	expectedImageKey := "archive/myapp/202507/myapp-20250721-1430-abc123-archived-on-20250722-1018.tar.gz"
	expectedMetaKey := "archive/myapp/202507/myapp-20250721-1430-abc123-archived-on-20250722-1018.json"

	assert.Equal(t, expectedImageKey, archiveImageKey)
	assert.Equal(t, expectedMetaKey, archiveMetaKey)
}
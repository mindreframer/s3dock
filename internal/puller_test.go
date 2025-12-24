package internal

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestImagePuller_Pull_Success_DirectImage(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)

	// Create test data
	testContent := "mock image data"
	metadataJSON, imageData, _ := createTestMetadata(testContent)

	// Mock environment pointer
	envPointerJSON := `{
		"target_type": "image",
		"target_path": "images/myapp/202507/myapp-20250722-0039-abc1234.tar.gz",
		"promoted_at": "2025-07-22T13:34:24Z",
		"promoted_by": "testuser",
		"git_hash": "abc1234",
		"git_time": "20250722-0039",
		"source_image": "myapp:20250722-0039-abc1234"
	}`

	// Set up S3 mocks
	mockS3.On("Exists", mock.Anything, "test-bucket", "pointers/myapp/production.json").Return(true, nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "pointers/myapp/production.json").Return([]byte(envPointerJSON), nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "images/myapp/202507/myapp-20250722-0039-abc1234.json").Return([]byte(metadataJSON), nil)
	mockS3.On("DownloadStream", mock.Anything, "test-bucket", "images/myapp/202507/myapp-20250722-0039-abc1234.tar.gz").Return(io.NopCloser(bytes.NewReader(imageData)), nil)

	// Set up Docker mock
	mockDocker.On("ImageExists", mock.Anything, "myapp:20250722-0039-abc1234").Return(false, nil)
	mockDocker.On("ImportImage", mock.Anything, mock.AnythingOfType("*gzip.Reader")).Return(nil)

	puller := NewImagePuller(mockDocker, mockS3, "test-bucket")

	_, err := puller.Pull(context.Background(), "myapp", "production")

	assert.NoError(t, err)
	mockS3.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
}

func TestImagePuller_Pull_Success_TagReference(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)

	// Create test data
	testContent := "mock image data tag"
	metadataJSON, imageData, _ := createTestMetadata(testContent)

	// Mock environment pointer that references a tag
	envPointerJSON := `{
		"target_type": "tag",
		"target_path": "tags/myapp/v1.2.0.json",
		"promoted_at": "2025-07-22T13:34:24Z",
		"promoted_by": "testuser",
		"git_hash": "abc1234",
		"git_time": "20250722-0039",
		"source_tag": "v1.2.0"
	}`

	// Mock tag pointer that points to actual image
	tagPointerJSON := `{
		"target_type": "image",
		"target_path": "images/myapp/202507/myapp-20250722-0039-abc1234.tar.gz",
		"promoted_at": "2025-07-22T13:30:00Z",
		"promoted_by": "testuser",
		"git_hash": "abc1234",
		"git_time": "20250722-0039",
		"source_image": "myapp:20250722-0039-abc1234"
	}`

	// Set up S3 mocks
	mockS3.On("Exists", mock.Anything, "test-bucket", "pointers/myapp/staging.json").Return(true, nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "pointers/myapp/staging.json").Return([]byte(envPointerJSON), nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "tags/myapp/v1.2.0.json").Return([]byte(tagPointerJSON), nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "images/myapp/202507/myapp-20250722-0039-abc1234.json").Return([]byte(metadataJSON), nil)
	mockS3.On("DownloadStream", mock.Anything, "test-bucket", "images/myapp/202507/myapp-20250722-0039-abc1234.tar.gz").Return(io.NopCloser(bytes.NewReader(imageData)), nil)

	// Set up Docker mock
	mockDocker.On("ImageExists", mock.Anything, "myapp:20250722-0039-abc1234").Return(false, nil)
	mockDocker.On("ImportImage", mock.Anything, mock.AnythingOfType("*gzip.Reader")).Return(nil)

	puller := NewImagePuller(mockDocker, mockS3, "test-bucket")

	_, err := puller.Pull(context.Background(), "myapp", "staging")

	assert.NoError(t, err)
	mockS3.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
}

func TestImagePuller_Pull_EnvironmentNotFound(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)

	// Mock environment pointer doesn't exist
	mockS3.On("Exists", mock.Anything, "test-bucket", "pointers/myapp/nonexistent.json").Return(false, nil)

	puller := NewImagePuller(mockDocker, mockS3, "test-bucket")

	_, err := puller.Pull(context.Background(), "myapp", "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "environment pointer not found: myapp/nonexistent")
	mockS3.AssertExpectations(t)
}

func TestImagePuller_PullFromTag_Success(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)

	// Create test data
	testContent := "mock tag data"
	metadataJSON, imageData, _ := createTestMetadata(testContent)

	// Mock tag pointer
	tagPointerJSON := `{
		"target_type": "image",
		"target_path": "images/myapp/202507/myapp-20250722-0039-abc1234.tar.gz",
		"promoted_at": "2025-07-22T13:30:00Z",
		"promoted_by": "testuser",
		"git_hash": "abc1234",
		"git_time": "20250722-0039",
		"source_image": "myapp:20250722-0039-abc1234"
	}`

	// Set up S3 mocks
	mockS3.On("Exists", mock.Anything, "test-bucket", "tags/myapp/v1.2.0.json").Return(true, nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "tags/myapp/v1.2.0.json").Return([]byte(tagPointerJSON), nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "images/myapp/202507/myapp-20250722-0039-abc1234.json").Return([]byte(metadataJSON), nil)
	mockS3.On("DownloadStream", mock.Anything, "test-bucket", "images/myapp/202507/myapp-20250722-0039-abc1234.tar.gz").Return(io.NopCloser(bytes.NewReader(imageData)), nil)

	// Set up Docker mock
	mockDocker.On("ImageExists", mock.Anything, "myapp:20250722-0039-abc1234").Return(false, nil)
	mockDocker.On("ImportImage", mock.Anything, mock.AnythingOfType("*gzip.Reader")).Return(nil)

	puller := NewImagePuller(mockDocker, mockS3, "test-bucket")

	_, err := puller.PullFromTag(context.Background(), "myapp", "v1.2.0")

	assert.NoError(t, err)
	mockS3.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
}

func TestImagePuller_PullFromTag_TagNotFound(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)

	// Mock tag doesn't exist
	mockS3.On("Exists", mock.Anything, "test-bucket", "tags/myapp/v9.9.9.json").Return(false, nil)

	puller := NewImagePuller(mockDocker, mockS3, "test-bucket")

	_, err := puller.PullFromTag(context.Background(), "myapp", "v9.9.9")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag not found: myapp/v9.9.9")
	mockS3.AssertExpectations(t)
}

func TestImagePuller_Pull_ChecksumMismatch_RetrySuccess(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)

	// Create test data
	testContent := "correct data"
	metadataJSON, goodImageData, _ := createTestMetadata(testContent)

	// Create bad data with different checksum
	badImageData := createMockGzippedData("wrong data")

	// Mock environment pointer
	envPointerJSON := `{
		"target_type": "image",
		"target_path": "images/myapp/202507/myapp-20250722-0039-abc1234.tar.gz",
		"promoted_at": "2025-07-22T13:34:24Z",
		"promoted_by": "testuser",
		"git_hash": "abc1234",
		"git_time": "20250722-0039",
		"source_image": "myapp:20250722-0039-abc1234"
	}`

	// Set up S3 mocks
	mockS3.On("Exists", mock.Anything, "test-bucket", "pointers/myapp/production.json").Return(true, nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "pointers/myapp/production.json").Return([]byte(envPointerJSON), nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "images/myapp/202507/myapp-20250722-0039-abc1234.json").Return([]byte(metadataJSON), nil)

	// Remove Download mocks for tarball in retry test, only mock DownloadStream for each retry
	mockS3.On("DownloadStream", mock.Anything, "test-bucket", "images/myapp/202507/myapp-20250722-0039-abc1234.tar.gz").Return(io.NopCloser(bytes.NewReader(badImageData)), nil).Once()
	mockS3.On("DownloadStream", mock.Anything, "test-bucket", "images/myapp/202507/myapp-20250722-0039-abc1234.tar.gz").Return(io.NopCloser(bytes.NewReader(goodImageData)), nil).Once()

	// Set up Docker mock
	mockDocker.On("ImageExists", mock.Anything, "myapp:20250722-0039-abc1234").Return(false, nil)
	mockDocker.On("ImportImage", mock.Anything, mock.AnythingOfType("*gzip.Reader")).Return(nil)

	puller := NewImagePuller(mockDocker, mockS3, "test-bucket")

	_, err := puller.Pull(context.Background(), "myapp", "production")

	assert.NoError(t, err)
	mockS3.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
}

func TestImagePuller_Pull_DockerImportFailure(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)

	// Create test data
	testContent := "docker fail data"
	metadataJSON, imageData, _ := createTestMetadata(testContent)

	// Mock environment pointer
	envPointerJSON := `{
		"target_type": "image",
		"target_path": "images/myapp/202507/myapp-20250722-0039-abc1234.tar.gz",
		"promoted_at": "2025-07-22T13:34:24Z",
		"promoted_by": "testuser",
		"git_hash": "abc1234",
		"git_time": "20250722-0039",
		"source_image": "myapp:20250722-0039-abc1234"
	}`

	// Set up S3 mocks
	mockS3.On("Exists", mock.Anything, "test-bucket", "pointers/myapp/production.json").Return(true, nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "pointers/myapp/production.json").Return([]byte(envPointerJSON), nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "images/myapp/202507/myapp-20250722-0039-abc1234.json").Return([]byte(metadataJSON), nil)
	mockS3.On("DownloadStream", mock.Anything, "test-bucket", "images/myapp/202507/myapp-20250722-0039-abc1234.tar.gz").Return(io.NopCloser(bytes.NewReader(imageData)), nil)

	// Set up Docker mock to fail
	mockDocker.On("ImageExists", mock.Anything, "myapp:20250722-0039-abc1234").Return(false, nil)
	mockDocker.On("ImportImage", mock.Anything, mock.AnythingOfType("*gzip.Reader")).Return(assert.AnError)

	puller := NewImagePuller(mockDocker, mockS3, "test-bucket")

	_, err := puller.Pull(context.Background(), "myapp", "production")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to import image to Docker")
	mockS3.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
}

func TestImagePuller_Pull_Skip_ImageAlreadyExists(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)

	// Create test data
	testContent := "existing image data"
	metadataJSON, _, _ := createTestMetadata(testContent)

	// Mock environment pointer
	envPointerJSON := `{
		"target_type": "image",
		"target_path": "images/myapp/202507/myapp-20250722-0039-abc1234.tar.gz",
		"promoted_at": "2025-07-22T13:34:24Z",
		"promoted_by": "testuser",
		"git_hash": "abc1234",
		"git_time": "20250722-0039",
		"source_image": "myapp:20250722-0039-abc1234"
	}`

	// Set up S3 mocks - note that we only mock metadata download, not image download
	mockS3.On("Exists", mock.Anything, "test-bucket", "pointers/myapp/production.json").Return(true, nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "pointers/myapp/production.json").Return([]byte(envPointerJSON), nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "images/myapp/202507/myapp-20250722-0039-abc1234.json").Return([]byte(metadataJSON), nil)
	// No image download mock - should be skipped

	// Set up Docker mock - image already exists
	mockDocker.On("ImageExists", mock.Anything, "myapp:20250722-0039-abc1234").Return(true, nil)
	// No ImportImage mock - should be skipped

	puller := NewImagePuller(mockDocker, mockS3, "test-bucket")

	_, err := puller.Pull(context.Background(), "myapp", "production")

	assert.NoError(t, err)
	mockS3.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
}

// Helper functions
func createMockGzippedData(content string) []byte {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	gzipWriter.Write([]byte(content))
	gzipWriter.Close()
	return buf.Bytes()
}

func calculateExpectedChecksum(content string) string {
	data := createMockGzippedData(content)
	hasher := md5.New()
	hasher.Write(data)
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func createTestMetadata(content string) (string, []byte, string) {
	imageData := createMockGzippedData(content)
	checksum := calculateExpectedChecksum(content)

	metadataJSON := fmt.Sprintf(`{
		"checksum": "%s",
		"size": %d,
		"git_hash": "abc1234",
		"git_time": "20250722-0039",
		"image_tag": "myapp:20250722-0039-abc1234",
		"app_name": "myapp",
		"created_at": "2025-07-22T13:34:18Z"
	}`, checksum, len(imageData))

	return metadataJSON, imageData, checksum
}

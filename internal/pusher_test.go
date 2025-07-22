package internal

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) ExportImage(ctx context.Context, imageRef string) (io.ReadCloser, error) {
	args := m.Called(ctx, imageRef)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockDockerClient) ImportImage(ctx context.Context, tarStream io.Reader) error {
	args := m.Called(ctx, tarStream)
	return args.Error(0)
}

func (m *MockDockerClient) ImageExists(ctx context.Context, imageRef string) (bool, error) {
	args := m.Called(ctx, imageRef)
	return args.Bool(0), args.Error(1)
}

func (m *MockDockerClient) BuildImage(ctx context.Context, contextPath string, dockerfile string, tags []string) error {
	args := m.Called(ctx, contextPath, dockerfile, tags)
	return args.Error(0)
}

type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) Upload(ctx context.Context, bucket, key string, data io.Reader) error {
	args := m.Called(ctx, bucket, key, data)
	return args.Error(0)
}

func (m *MockS3Client) UploadWithProgress(ctx context.Context, bucket, key string, data io.Reader, size int64, description string) error {
	args := m.Called(ctx, bucket, key, data, size, description)
	return args.Error(0)
}

func (m *MockS3Client) Exists(ctx context.Context, bucket, key string) (bool, error) {
	args := m.Called(ctx, bucket, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockS3Client) Download(ctx context.Context, bucket, key string) ([]byte, error) {
	args := m.Called(ctx, bucket, key)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockS3Client) Copy(ctx context.Context, bucket, srcKey, dstKey string) error {
	args := m.Called(ctx, bucket, srcKey, dstKey)
	return args.Error(0)
}

func (m *MockS3Client) Delete(ctx context.Context, bucket, key string) error {
	args := m.Called(ctx, bucket, key)
	return args.Error(0)
}

func (m *MockS3Client) DownloadStream(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, bucket, key)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

type MockGitClient struct {
	mock.Mock
}

func (m *MockGitClient) GetCurrentHash(path string) (string, error) {
	args := m.Called(path)
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) GetCommitTimestamp(path string) (string, error) {
	args := m.Called(path)
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) IsRepositoryDirty(path string) (bool, error) {
	args := m.Called(path)
	return args.Bool(0), args.Error(1)
}

func TestExtractAppName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"myapp:latest", "myapp"},
		{"myapp", "myapp"},
		{"registry.com/myapp:v1.0", "myapp"},
		{"localhost:5000/myapp:latest", "myapp"},
		{"registry.io:443/namespace/myapp:v1.0", "myapp"},
	}

	for _, test := range tests {
		result := ExtractAppName(test.input)
		assert.Equal(t, test.expected, result, "Failed for input: %s", test.input)
	}
}

func TestImagePusher_Push_Success_NewImage(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)
	mockGit := new(MockGitClient)

	mockGit.On("GetCurrentHash", mock.Anything).Return("abc1234", nil)
	mockGit.On("GetCommitTimestamp", mock.Anything).Return("20250721-1430", nil)
	mockDocker.On("ExportImage", mock.Anything, "myapp:latest").Return(io.NopCloser(strings.NewReader("image data")), nil)

	// Metadata doesn't exist (new image)
	mockS3.On("Exists", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".json") && strings.HasPrefix(key, "images/")
	})).Return(false, nil)

	// Upload image and metadata
	mockS3.On("UploadWithProgress", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".tar.gz") && strings.HasPrefix(key, "images/")
	}), mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("string")).Return(nil)
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".json") && strings.HasPrefix(key, "images/")
	}), mock.Anything).Return(nil)

	// Mock audit log upload
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "audit/") && strings.Contains(key, "push")
	}), mock.Anything).Return(nil)

	pusher := NewImagePusher(mockDocker, mockS3, mockGit, "test-bucket")

	err := pusher.Push(context.Background(), "myapp:latest")

	assert.NoError(t, err)
	mockGit.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
	mockS3.AssertExpectations(t)
}

func TestImagePusher_Push_Success_ExistingSameChecksum(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)
	mockGit := new(MockGitClient)

	mockGit.On("GetCurrentHash", mock.Anything).Return("abc1234", nil)
	mockGit.On("GetCommitTimestamp", mock.Anything).Return("20250721-1430", nil)
	mockDocker.On("ExportImage", mock.Anything, "myapp:latest").Return(io.NopCloser(strings.NewReader("image data")), nil)

	// Metadata exists
	mockS3.On("Exists", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".json") && strings.HasPrefix(key, "images/")
	})).Return(true, nil)

	// Return existing metadata with same checksum (now gzipped)
	existingMetadata := &ImageMetadata{
		Checksum: "e3cb4936e6592acbef54276b4eb77d56", // MD5 of gzipped "image data"
		Size:     34,                                 // Size of compressed data
	}
	metadataJSON, _ := existingMetadata.ToJSON()
	mockS3.On("Download", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".json") && strings.HasPrefix(key, "images/")
	})).Return(metadataJSON, nil)

	// Mock audit log upload for skipped push
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "audit/") && strings.Contains(key, "push")
	}), mock.Anything).Return(nil)

	pusher := NewImagePusher(mockDocker, mockS3, mockGit, "test-bucket")

	err := pusher.Push(context.Background(), "myapp:latest")

	assert.NoError(t, err)
	mockGit.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
	mockS3.AssertExpectations(t)
}

func TestImagePusher_Push_GitError(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)
	mockGit := new(MockGitClient)

	mockGit.On("GetCurrentHash", mock.Anything).Return("", errors.New("git error"))

	pusher := NewImagePusher(mockDocker, mockS3, mockGit, "test-bucket")

	err := pusher.Push(context.Background(), "myapp:latest")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get git hash")
	mockGit.AssertExpectations(t)
}

func TestImagePusher_Push_Success_ChecksumMismatch(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)
	mockGit := new(MockGitClient)

	mockGit.On("GetCurrentHash", mock.Anything).Return("abc1234", nil)
	mockGit.On("GetCommitTimestamp", mock.Anything).Return("20250721-1430", nil)
	mockDocker.On("ExportImage", mock.Anything, "myapp:latest").Return(io.NopCloser(strings.NewReader("new image data")), nil)

	// Metadata exists
	mockS3.On("Exists", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".json") && strings.HasPrefix(key, "images/")
	})).Return(true, nil)

	// Return existing metadata with different checksum
	existingMetadata := &ImageMetadata{
		Checksum: "old-checksum-value",
		Size:     10,
	}
	metadataJSON, _ := existingMetadata.ToJSON()
	mockS3.On("Download", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".json") && strings.HasPrefix(key, "images/")
	})).Return(metadataJSON, nil)

	// Archive operations
	mockS3.On("Copy", mock.Anything, "test-bucket", mock.AnythingOfType("string"), mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "archive/")
	})).Return(nil)
	mockS3.On("Delete", mock.Anything, "test-bucket", mock.AnythingOfType("string")).Return(nil)

	// Upload new image and metadata
	mockS3.On("UploadWithProgress", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".tar.gz") && strings.HasPrefix(key, "images/")
	}), mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("string")).Return(nil)
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".json") && strings.HasPrefix(key, "images/")
	}), mock.Anything).Return(nil)

	// Mock audit log upload for push with archive
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "audit/") && strings.Contains(key, "push")
	}), mock.Anything).Return(nil)

	pusher := NewImagePusher(mockDocker, mockS3, mockGit, "test-bucket")

	err := pusher.Push(context.Background(), "myapp:latest")

	assert.NoError(t, err)
	mockGit.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
	mockS3.AssertExpectations(t)
}

func TestImagePusher_Push_DockerError(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)
	mockGit := new(MockGitClient)

	mockGit.On("GetCurrentHash", mock.Anything).Return("abc1234", nil)
	mockGit.On("GetCommitTimestamp", mock.Anything).Return("20250721-1430", nil)
	mockS3.On("Exists", mock.Anything, "test-bucket", mock.AnythingOfType("string")).Return(false, nil)
	mockDocker.On("ExportImage", mock.Anything, "myapp:latest").Return(io.NopCloser(strings.NewReader("")), errors.New("docker error"))

	pusher := NewImagePusher(mockDocker, mockS3, mockGit, "test-bucket")

	err := pusher.Push(context.Background(), "myapp:latest")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to export image")
	mockGit.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
	mockS3.AssertExpectations(t)
}

func TestImagePusher_Push_VerifyGzipCompression(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)
	mockGit := new(MockGitClient)

	originalData := "test image data that should be compressed"

	mockGit.On("GetCurrentHash", mock.Anything).Return("abc1234", nil)
	mockGit.On("GetCommitTimestamp", mock.Anything).Return("20250721-1430", nil)
	mockDocker.On("ExportImage", mock.Anything, "myapp:latest").Return(io.NopCloser(strings.NewReader(originalData)), nil)

	// Metadata doesn't exist (new image)
	mockS3.On("Exists", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".json") && strings.HasPrefix(key, "images/")
	})).Return(false, nil)

	var uploadedData *bytes.Buffer

	// Capture uploaded data to verify it's compressed
	mockS3.On("UploadWithProgress", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".tar.gz") && strings.HasPrefix(key, "images/")
	}), mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		reader := args.Get(3).(io.Reader)
		uploadedData = &bytes.Buffer{}
		io.Copy(uploadedData, reader)
	}).Return(nil)

	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".json") && strings.HasPrefix(key, "images/")
	}), mock.Anything).Return(nil)

	// Mock audit log upload
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "audit/") && strings.Contains(key, "push")
	}), mock.Anything).Return(nil)

	pusher := NewImagePusher(mockDocker, mockS3, mockGit, "test-bucket")

	err := pusher.Push(context.Background(), "myapp:latest")

	assert.NoError(t, err)
	assert.NotNil(t, uploadedData, "Should have captured uploaded data")

	// Verify the uploaded data is gzip compressed
	reader, err := gzip.NewReader(uploadedData)
	assert.NoError(t, err, "Uploaded data should be valid gzip")

	decompressed, err := io.ReadAll(reader)
	assert.NoError(t, err, "Should be able to decompress uploaded data")
	assert.Equal(t, originalData, string(decompressed), "Decompressed data should match original")

	reader.Close()

	mockGit.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
	mockS3.AssertExpectations(t)
}

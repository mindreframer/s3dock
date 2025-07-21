package internal

import (
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

type MockGitClient struct {
	mock.Mock
}

func (m *MockGitClient) GetCurrentHash() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) GetCommitTimestamp() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) IsRepositoryDirty() (bool, error) {
	args := m.Called()
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

func TestImagePusher_Push_Success(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)
	mockGit := new(MockGitClient)

	mockGit.On("GetCurrentHash").Return("abc1234", nil)
	mockDocker.On("ExportImage", mock.Anything, "myapp:latest").Return(io.NopCloser(strings.NewReader("image data")), nil)
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.AnythingOfType("string"), mock.Anything).Return(nil)

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

	mockGit.On("GetCurrentHash").Return("", errors.New("git error"))

	pusher := NewImagePusher(mockDocker, mockS3, mockGit, "test-bucket")

	err := pusher.Push(context.Background(), "myapp:latest")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get git hash")
	mockGit.AssertExpectations(t)
}

func TestImagePusher_Push_DockerError(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockS3 := new(MockS3Client)
	mockGit := new(MockGitClient)

	mockGit.On("GetCurrentHash").Return("abc1234", nil)
	mockDocker.On("ExportImage", mock.Anything, "myapp:latest").Return(io.NopCloser(strings.NewReader("")), errors.New("docker error"))

	pusher := NewImagePusher(mockDocker, mockS3, mockGit, "test-bucket")

	err := pusher.Push(context.Background(), "myapp:latest")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to export image")
	mockGit.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
}

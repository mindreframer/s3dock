package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageBuilder_Build_Success(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockGit := new(MockGitClient)

	mockGit.On("IsRepositoryDirty").Return(false, nil)
	mockGit.On("GetCurrentHash").Return("abc1234", nil)
	mockGit.On("GetCommitTimestamp").Return("20250721-1430", nil)
	mockDocker.On("BuildImage", context.Background(), ".", "Dockerfile", []string{"myapp:20250721-1430-abc1234"}).Return(nil)

	builder := NewImageBuilder(mockDocker, mockGit)

	tag, err := builder.Build(context.Background(), "myapp", ".", "Dockerfile")

	assert.NoError(t, err)
	assert.Equal(t, "myapp:20250721-1430-abc1234", tag)
	mockGit.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
}

func TestImageBuilder_Build_DirtyRepository(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockGit := new(MockGitClient)

	mockGit.On("IsRepositoryDirty").Return(true, nil)

	builder := NewImageBuilder(mockDocker, mockGit)

	tag, err := builder.Build(context.Background(), "myapp", ".", "Dockerfile")

	assert.Error(t, err)
	assert.Empty(t, tag)
	assert.Contains(t, err.Error(), "repository has uncommitted changes")
	mockGit.AssertExpectations(t)
}

func TestImageBuilder_Build_GitHashError(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockGit := new(MockGitClient)

	mockGit.On("IsRepositoryDirty").Return(false, nil)
	mockGit.On("GetCurrentHash").Return("", errors.New("git hash error"))

	builder := NewImageBuilder(mockDocker, mockGit)

	tag, err := builder.Build(context.Background(), "myapp", ".", "Dockerfile")

	assert.Error(t, err)
	assert.Empty(t, tag)
	assert.Contains(t, err.Error(), "failed to get git hash")
	mockGit.AssertExpectations(t)
}

func TestImageBuilder_Build_GitTimestampError(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockGit := new(MockGitClient)

	mockGit.On("IsRepositoryDirty").Return(false, nil)
	mockGit.On("GetCurrentHash").Return("abc1234", nil)
	mockGit.On("GetCommitTimestamp").Return("", errors.New("git timestamp error"))

	builder := NewImageBuilder(mockDocker, mockGit)

	tag, err := builder.Build(context.Background(), "myapp", ".", "Dockerfile")

	assert.Error(t, err)
	assert.Empty(t, tag)
	assert.Contains(t, err.Error(), "failed to get commit timestamp")
	mockGit.AssertExpectations(t)
}

func TestImageBuilder_Build_DockerBuildError(t *testing.T) {
	mockDocker := new(MockDockerClient)
	mockGit := new(MockGitClient)

	mockGit.On("IsRepositoryDirty").Return(false, nil)
	mockGit.On("GetCurrentHash").Return("abc1234", nil)
	mockGit.On("GetCommitTimestamp").Return("20250721-1430", nil)
	mockDocker.On("BuildImage", context.Background(), ".", "Dockerfile", []string{"myapp:20250721-1430-abc1234"}).Return(errors.New("docker build error"))

	builder := NewImageBuilder(mockDocker, mockGit)

	tag, err := builder.Build(context.Background(), "myapp", ".", "Dockerfile")

	assert.Error(t, err)
	assert.Empty(t, tag)
	assert.Contains(t, err.Error(), "failed to build image")
	mockGit.AssertExpectations(t)
	mockDocker.AssertExpectations(t)
}

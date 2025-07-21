package internal

import (
	"context"
	"fmt"
)

type ImageBuilder struct {
	docker DockerClient
	git    GitClient
}

func NewImageBuilder(docker DockerClient, git GitClient) *ImageBuilder {
	return &ImageBuilder{
		docker: docker,
		git:    git,
	}
}

func (b *ImageBuilder) Build(ctx context.Context, appName string, contextPath string, dockerfile string) (string, error) {
	LogInfo("Starting build for app: %s", appName)
	LogDebug("Build context: %s, Dockerfile: %s", contextPath, dockerfile)

	LogDebug("Checking if repository is clean")
	isDirty, err := b.git.IsRepositoryDirty()
	if err != nil {
		LogError("Failed to check repository status: %v", err)
		return "", fmt.Errorf("failed to check repository status: %w", err)
	}

	if isDirty {
		LogError("Repository has uncommitted changes - commit all changes before building")
		return "", fmt.Errorf("repository has uncommitted changes - commit all changes before building")
	}

	LogDebug("Repository is clean, proceeding with build")

	LogDebug("Getting git hash")
	gitHash, err := b.git.GetCurrentHash()
	if err != nil {
		LogError("Failed to get git hash: %v", err)
		return "", fmt.Errorf("failed to get git hash: %w", err)
	}

	LogDebug("Getting git commit timestamp")
	timestamp, err := b.git.GetCommitTimestamp()
	if err != nil {
		LogError("Failed to get commit timestamp: %v", err)
		return "", fmt.Errorf("failed to get commit timestamp: %w", err)
	}

	tag := fmt.Sprintf("%s:%s-%s", appName, timestamp, gitHash)
	LogDebug("Generated tag: %s", tag)

	LogInfo("Building image %s with tag %s", appName, tag)

	if err := b.docker.BuildImage(ctx, contextPath, dockerfile, []string{tag}); err != nil {
		LogError("Failed to build image %s: %v", tag, err)
		return "", fmt.Errorf("failed to build image: %w", err)
	}

	LogInfo("Successfully built %s", tag)
	return tag, nil
}

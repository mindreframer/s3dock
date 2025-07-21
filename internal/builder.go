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
	isDirty, err := b.git.IsRepositoryDirty()
	if err != nil {
		return "", fmt.Errorf("failed to check repository status: %w", err)
	}

	if isDirty {
		return "", fmt.Errorf("repository has uncommitted changes - commit all changes before building")
	}

	gitHash, err := b.git.GetCurrentHash()
	if err != nil {
		return "", fmt.Errorf("failed to get git hash: %w", err)
	}

	timestamp, err := b.git.GetCommitTimestamp()
	if err != nil {
		return "", fmt.Errorf("failed to get commit timestamp: %w", err)
	}

	tag := fmt.Sprintf("%s:%s-%s", appName, timestamp, gitHash)

	if err := b.docker.BuildImage(ctx, contextPath, dockerfile, []string{tag}); err != nil {
		return "", fmt.Errorf("failed to build image: %w", err)
	}

	fmt.Printf("Successfully built %s\n", tag)
	return tag, nil
}

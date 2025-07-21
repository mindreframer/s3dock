package internal

import (
	"context"
	"io"

	"github.com/docker/docker/client"
)

type DockerClientImpl struct {
	client *client.Client
}

func NewDockerClient() (*DockerClientImpl, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &DockerClientImpl{client: cli}, nil
}

func (d *DockerClientImpl) ExportImage(ctx context.Context, imageRef string) (io.ReadCloser, error) {
	return d.client.ImageSave(ctx, []string{imageRef})
}

func (d *DockerClientImpl) Close() error {
	return d.client.Close()
}

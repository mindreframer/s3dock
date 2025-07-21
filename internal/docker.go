package internal

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
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

func (d *DockerClientImpl) BuildImage(ctx context.Context, contextPath string, dockerfile string, tags []string) error {
	dockerfilePath := dockerfile
	if !filepath.IsAbs(dockerfile) {
		dockerfilePath = filepath.Join(contextPath, dockerfile)
	}

	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("dockerfile not found: %s", dockerfilePath)
	}

	tarReader, err := d.createBuildContext(contextPath)
	if err != nil {
		return err
	}
	defer tarReader.Close()

	response, err := d.client.ImageBuild(ctx, tarReader, types.ImageBuildOptions{
		Tags:       tags,
		Dockerfile: dockerfile,
	})
	if err != nil {
		return err
	}
	defer response.Body.Close()

	_, err = io.Copy(os.Stdout, response.Body)
	return err
}

func (d *DockerClientImpl) createBuildContext(contextPath string) (io.ReadCloser, error) {
	file, err := os.Open(contextPath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (d *DockerClientImpl) Close() error {
	return d.client.Close()
}

package internal

import (
	"archive/tar"
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
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		tw := tar.NewWriter(pw)
		defer tw.Close()

		err := filepath.Walk(contextPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(contextPath, path)
			if err != nil {
				return err
			}

			if relPath == "." {
				return nil
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = relPath

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tw, file)
			return err
		})

		if err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr, nil
}

func (d *DockerClientImpl) Close() error {
	return d.client.Close()
}

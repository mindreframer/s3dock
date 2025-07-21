package internal

import (
	"archive/tar"
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

// readDockerignore reads and parses .dockerignore patterns
func readDockerignore(contextPath string) ([]string, error) {
	dockerignorePath := filepath.Join(contextPath, ".dockerignore")

	file, err := os.Open(dockerignorePath)
	if os.IsNotExist(err) {
		return nil, nil // .dockerignore doesn't exist
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}

	return patterns, scanner.Err()
}

// shouldIgnore checks if a path should be ignored based on .dockerignore patterns
func shouldIgnore(path string, patterns []string) bool {
	// Convert path to use forward slashes for pattern matching
	normalizedPath := strings.ReplaceAll(path, string(os.PathSeparator), "/")

	for _, pattern := range patterns {
		// Handle directory patterns (ending with /)
		if strings.HasSuffix(pattern, "/") {
			dirPattern := strings.TrimSuffix(pattern, "/")
			// Check if the path starts with the directory pattern
			if strings.HasPrefix(normalizedPath, dirPattern+"/") || normalizedPath == dirPattern {
				return true
			}
			// Also check if any path component matches the directory pattern
			pathParts := strings.Split(normalizedPath, "/")
			for _, part := range pathParts {
				if part == dirPattern {
					return true
				}
			}
		}

		// Handle wildcard patterns (*)
		if strings.Contains(pattern, "*") {
			// Check if the filename matches the pattern
			filename := filepath.Base(normalizedPath)
			matched, _ := filepath.Match(pattern, filename)
			if matched {
				return true
			}
		}

		// Handle exact matches
		if normalizedPath == pattern {
			return true
		}

		// Handle prefix matches (for directory contents)
		if strings.HasPrefix(normalizedPath, pattern+"/") {
			return true
		}
	}
	return false
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
	// Read .dockerignore patterns
	patterns, err := readDockerignore(contextPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .dockerignore: %w", err)
	}

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

			// Check if this path should be ignored
			if shouldIgnore(relPath, patterns) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip files that are too large (over 100MB)
			if !info.IsDir() && info.Size() > 100*1024*1024 {
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

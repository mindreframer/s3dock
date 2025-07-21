package internal

import (
	"context"
	"io"
)

type DockerClient interface {
	ExportImage(ctx context.Context, imageRef string) (io.ReadCloser, error)
}

type S3Client interface {
	Upload(ctx context.Context, bucket, key string, data io.Reader) error
}

type GitClient interface {
	GetCurrentHash() (string, error)
}

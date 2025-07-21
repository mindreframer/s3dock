package internal

import (
	"context"
	"io"
)

type DockerClient interface {
	ExportImage(ctx context.Context, imageRef string) (io.ReadCloser, error)
	BuildImage(ctx context.Context, contextPath string, dockerfile string, tags []string) error
}

type S3Client interface {
	Upload(ctx context.Context, bucket, key string, data io.Reader) error
	Exists(ctx context.Context, bucket, key string) (bool, error)
	Download(ctx context.Context, bucket, key string) ([]byte, error)
	Copy(ctx context.Context, bucket, srcKey, dstKey string) error
	Delete(ctx context.Context, bucket, key string) error
}

type GitClient interface {
	GetCurrentHash() (string, error)
	GetCommitTimestamp() (string, error)
	IsRepositoryDirty() (bool, error)
}

package internal

import (
	"context"
	"io"
)

type DockerClient interface {
	ExportImage(ctx context.Context, imageRef string) (io.ReadCloser, error)
	ImportImage(ctx context.Context, tarStream io.Reader) error
	BuildImage(ctx context.Context, contextPath string, dockerfile string, tags []string, platform string) error
	ImageExists(ctx context.Context, imageRef string) (bool, error)
}

type S3Client interface {
	Upload(ctx context.Context, bucket, key string, data io.Reader) error
	UploadWithProgress(ctx context.Context, bucket, key string, data io.Reader, size int64, description string) error
	Exists(ctx context.Context, bucket, key string) (bool, error)
	Download(ctx context.Context, bucket, key string) ([]byte, error)
	DownloadStream(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	Copy(ctx context.Context, bucket, srcKey, dstKey string) error
	Delete(ctx context.Context, bucket, key string) error
	List(ctx context.Context, bucket, prefix string) ([]string, error)
}

type GitClient interface {
	GetCurrentHash(path string) (string, error)
	GetCommitTimestamp(path string) (string, error)
	IsRepositoryDirty(path string) (bool, error)
	FindRepositoryRoot(startPath string) (string, error)
}

package internal

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/schollz/progressbar/v3"
)

type S3ClientImpl struct {
	client      *s3.Client
	listClient  *s3.Client // Separate client for list operations (handles bucket-subdomain endpoints)
	uploader    *manager.Uploader
	keyPrefix   string // Prefix to add to keys for list operations
}

func NewS3Client(ctx context.Context) (*S3ClientImpl, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	var client *s3.Client
	var listClient *s3.Client
	var keyPrefix string

	endpointURL := os.Getenv("AWS_ENDPOINT_URL")
	if endpointURL != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpointURL)
			o.UsePathStyle = true
		})

		// Check if endpoint contains bucket name (e.g., https://bucket.s3.region.wasabisys.com)
		// If so, create a separate list client with the base endpoint
		baseEndpoint, bucket := extractBaseEndpoint(endpointURL)
		if baseEndpoint != "" && bucket != "" {
			LogDebug("Detected bucket-subdomain endpoint, creating separate list client")
			LogDebug("Base endpoint: %s, bucket prefix: %s", baseEndpoint, bucket)
			listClient = s3.NewFromConfig(cfg, func(o *s3.Options) {
				o.BaseEndpoint = aws.String(baseEndpoint)
				o.UsePathStyle = true
			})
			keyPrefix = bucket + "/"
		} else {
			listClient = client
		}
	} else {
		client = s3.NewFromConfig(cfg)
		listClient = client
	}

	uploader := manager.NewUploader(client)

	return &S3ClientImpl{
		client:     client,
		listClient: listClient,
		uploader:   uploader,
		keyPrefix:  keyPrefix,
	}, nil
}

// extractBaseEndpoint checks if an endpoint is a bucket-subdomain style endpoint
// (e.g., https://bucket.s3.region.wasabisys.com) and returns the base endpoint and bucket name
func extractBaseEndpoint(endpoint string) (baseEndpoint, bucket string) {
	// Remove protocol
	e := strings.TrimPrefix(endpoint, "https://")
	e = strings.TrimPrefix(e, "http://")

	// Check for patterns like: bucket.s3.region.provider.com
	// The bucket is the first part before .s3.
	if idx := strings.Index(e, ".s3."); idx > 0 {
		bucket = e[:idx]
		rest := e[idx+1:] // s3.region.provider.com
		baseEndpoint = strings.TrimPrefix(endpoint, "https://"+bucket+".")
		baseEndpoint = strings.TrimPrefix(baseEndpoint, "http://"+bucket+".")
		if strings.HasPrefix(endpoint, "https://") {
			baseEndpoint = "https://" + rest
		} else {
			baseEndpoint = "http://" + rest
		}
		return baseEndpoint, bucket
	}

	return "", ""
}

func (s *S3ClientImpl) Upload(ctx context.Context, bucket, key string, data io.Reader) error {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   data,
	})
	return err
}

func (s *S3ClientImpl) Exists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *S3ClientImpl) Download(ctx context.Context, bucket, key string) ([]byte, error) {
	downloader := manager.NewDownloader(s.client)
	buf := manager.NewWriteAtBuffer([]byte{})

	_, err := downloader.Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *S3ClientImpl) Copy(ctx context.Context, bucket, srcKey, dstKey string) error {
	copySource := bucket + "/" + srcKey
	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(copySource),
	})
	return err
}

func (s *S3ClientImpl) UploadWithProgress(ctx context.Context, bucket, key string, data io.Reader, size int64, description string) error {
	bar := progressbar.DefaultBytes(size, description)
	defer bar.Finish()

	reader := progressbar.NewReader(data, bar)

	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   &reader,
	})
	return err
}

func (s *S3ClientImpl) Delete(ctx context.Context, bucket, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

// DownloadStream streams an object from S3 as an io.ReadCloser.
func (s *S3ClientImpl) DownloadStream(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// List returns all keys with a given prefix
func (s *S3ClientImpl) List(ctx context.Context, bucket, prefix string) ([]string, error) {
	var keys []string

	// Use the list client and add key prefix if needed (for bucket-subdomain endpoints)
	actualPrefix := s.keyPrefix + prefix
	LogDebug("List: bucket=%s, prefix=%s, actualPrefix=%s", bucket, prefix, actualPrefix)

	paginator := s3.NewListObjectsV2Paginator(s.listClient, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(actualPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, obj := range page.Contents {
			// Strip the key prefix from results to match expected format
			key := *obj.Key
			if s.keyPrefix != "" && strings.HasPrefix(key, s.keyPrefix) {
				key = strings.TrimPrefix(key, s.keyPrefix)
			}
			keys = append(keys, key)
		}
	}

	return keys, nil
}

package internal

import (
	"context"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3ClientImpl struct {
	client   *s3.Client
	uploader *manager.Uploader
}

func NewS3Client(ctx context.Context) (*S3ClientImpl, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	var client *s3.Client
	if endpointURL := os.Getenv("AWS_ENDPOINT_URL"); endpointURL != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpointURL)
			o.UsePathStyle = true
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}
	
	uploader := manager.NewUploader(client)
	
	return &S3ClientImpl{client: client, uploader: uploader}, nil
}

func (s *S3ClientImpl) Upload(ctx context.Context, bucket, key string, data io.Reader) error {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   data,
	})
	return err
}
package internal

import (
	"context"
	"fmt"
	"time"
)

type ImagePusher struct {
	docker DockerClient
	s3     S3Client
	git    GitClient
	bucket string
}

func NewImagePusher(docker DockerClient, s3 S3Client, git GitClient, bucket string) *ImagePusher {
	return &ImagePusher{
		docker: docker,
		s3:     s3,
		git:    git,
		bucket: bucket,
	}
}

func (p *ImagePusher) Push(ctx context.Context, imageRef string) error {
	gitHash, err := p.git.GetCurrentHash()
	if err != nil {
		return fmt.Errorf("failed to get git hash: %w", err)
	}

	timestamp := time.Now().Format("20060102-1504")
	
	appName := ExtractAppName(imageRef)
	yearMonth := time.Now().Format("200601")
	
	filename := fmt.Sprintf("%s-%s-%s.tar.gz", appName, timestamp, gitHash)
	s3Key := fmt.Sprintf("images/%s/%s/%s", appName, yearMonth, filename)

	imageData, err := p.docker.ExportImage(ctx, imageRef)
	if err != nil {
		return fmt.Errorf("failed to export image: %w", err)
	}
	defer imageData.Close()

	if err := p.s3.Upload(ctx, p.bucket, s3Key, imageData); err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	fmt.Printf("Successfully pushed %s to s3://%s/%s\n", imageRef, p.bucket, s3Key)
	return nil
}

func ExtractAppName(imageRef string) string {
	lastSlash := -1
	
	for i, c := range imageRef {
		if c == '/' {
			lastSlash = i
		}
	}
	
	start := 0
	if lastSlash >= 0 {
		start = lastSlash + 1
	}
	
	end := len(imageRef)
	for i := start; i < len(imageRef); i++ {
		if imageRef[i] == ':' {
			end = i
			break
		}
	}
	
	return imageRef[start:end]
}
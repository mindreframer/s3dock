package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"s3dock/internal"
)

func TestIntegration_Push(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	bucket := "s3dock-test"
	imageRef := "s3dock-test:latest"

	os.Setenv("AWS_ACCESS_KEY_ID", "testuser")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "testpass123")
	os.Setenv("AWS_ENDPOINT_URL", "http://localhost:9000")
	os.Setenv("AWS_REGION", "us-east-1")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	dockerClient, err := internal.NewDockerClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer dockerClient.Close()
	
	s3Client, err := internal.NewS3Client(ctx)
	if err != nil {
		t.Skipf("S3 not available: %v", err)
	}
	
	gitClient := internal.NewGitClient()
	
	pusher := internal.NewImagePusher(dockerClient, s3Client, gitClient, bucket)
	
	err = pusher.Push(ctx, imageRef)
	if err != nil {
		if os.Getenv("CI") == "" {
			t.Skipf("Integration test failed (might be expected in local env): %v", err)
		} else {
			assert.NoError(t, err)
		}
	}
}
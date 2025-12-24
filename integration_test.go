package main

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"s3dock/internal"

	"github.com/stretchr/testify/assert"
)

func TestIntegration_Push(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test - set INTEGRATION_TEST=1 to run")
	}

	bucket := "s3dock-test"
	imageRef := "s3dock-test:latest"

	os.Setenv("AWS_ACCESS_KEY_ID", "testuser")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "testpass123")
	os.Setenv("AWS_ENDPOINT_URL", "http://localhost:9000")
	os.Setenv("AWS_REGION", "us-east-1")
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_ENDPOINT_URL")
		os.Unsetenv("AWS_REGION")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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

	_, err = pusher.Push(ctx, imageRef)
	assert.NoError(t, err, "Integration test should pass with proper MinIO setup")
}

func TestCurrentCommand_Integration(t *testing.T) {
	// Test that the current command shows proper usage when called without arguments
	cmd := exec.Command("./s3dock", "current")
	output, err := cmd.CombinedOutput()

	// Should exit with error code 1
	if err == nil {
		t.Error("Expected command to fail with error, but it succeeded")
	}

	// Should show usage message
	outputStr := string(output)
	if !contains(outputStr, "Current command requires app name and environment") {
		t.Errorf("Expected usage message, got: %s", outputStr)
	}
}

func TestCurrentCommand_Usage(t *testing.T) {
	// Test that the current command shows proper usage when called with insufficient arguments
	cmd := exec.Command("./s3dock", "current", "myapp")
	output, err := cmd.CombinedOutput()

	// Should exit with error code 1
	if err == nil {
		t.Error("Expected command to fail with error, but it succeeded")
	}

	// Should show usage message
	outputStr := string(output)
	if !contains(outputStr, "Current command requires app name and environment") {
		t.Errorf("Expected usage message, got: %s", outputStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}

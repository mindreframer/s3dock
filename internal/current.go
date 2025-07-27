package internal

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type CurrentService struct {
	s3     S3Client
	bucket string
}

func NewCurrentService(s3 S3Client, bucket string) *CurrentService {
	return &CurrentService{
		s3:     s3,
		bucket: bucket,
	}
}

// GetCurrentImage retrieves the current image reference for an app in a specific environment
func (c *CurrentService) GetCurrentImage(ctx context.Context, appName, environment string) (string, error) {
	LogInfo("Getting current image for %s in %s environment", appName, environment)

	// Get environment pointer
	envKey := GeneratePointerKey(appName, environment)
	LogDebug("Looking for environment pointer at: %s", envKey)

	exists, err := c.s3.Exists(ctx, c.bucket, envKey)
	if err != nil {
		LogError("Failed to check environment pointer existence: %v", err)
		return "", fmt.Errorf("failed to check environment pointer existence: %w", err)
	}

	if !exists {
		LogError("Environment pointer not found: %s/%s", appName, environment)
		return "", fmt.Errorf("environment pointer not found: %s/%s", appName, environment)
	}

	// Download environment pointer
	LogDebug("Downloading environment pointer")
	pointerData, err := c.s3.Download(ctx, c.bucket, envKey)
	if err != nil {
		LogError("Failed to download environment pointer: %v", err)
		return "", fmt.Errorf("failed to download environment pointer: %w", err)
	}

	pointer, err := PointerMetadataFromJSON(pointerData)
	if err != nil {
		LogError("Failed to parse environment pointer: %v", err)
		return "", fmt.Errorf("failed to parse environment pointer: %w", err)
	}

	LogDebug("Environment pointer type: %s, target: %s", pointer.TargetType, pointer.TargetPath)

	// Resolve to actual image path
	imageS3Path, err := ResolveImagePath(ctx, c.s3, c.bucket, pointer)
	if err != nil {
		LogError("Failed to resolve image path: %v", err)
		return "", fmt.Errorf("failed to resolve image path: %w", err)
	}

	// Extract image reference from S3 path
	imageRef, err := c.extractImageReferenceFromPath(imageS3Path)
	if err != nil {
		LogError("Failed to extract image reference from path: %v", err)
		return "", fmt.Errorf("failed to extract image reference from path: %w", err)
	}

	LogInfo("Current image for %s in %s: %s", appName, environment, imageRef)
	return imageRef, nil
}

// extractImageReferenceFromPath converts an S3 image path to an image reference
// Example: images/myapp/202507/myapp-20250721-1430-abc1234.tar.gz -> myapp:20250721-1430-abc1234
func (c *CurrentService) extractImageReferenceFromPath(s3Path string) (string, error) {
	// Validate that the path ends with .tar.gz
	if !strings.HasSuffix(s3Path, ".tar.gz") {
		return "", fmt.Errorf("invalid image path format: must end with .tar.gz")
	}

	// Remove .tar.gz extension
	baseName := strings.TrimSuffix(s3Path, ".tar.gz")

	// Get the filename part (last component of the path)
	filename := filepath.Base(baseName)

	// Split by dash to separate app name from timestamp-hash
	// Expected format: myapp-20250721-1430-abc1234
	parts := strings.SplitN(filename, "-", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid image filename format: %s", filename)
	}

	appName := parts[0]
	timestampHash := parts[1]

	// Validate timestamp-hash format (YYYYMMDD-HHMM-hash)
	// Should have exactly 2 dashes in the timestamp-hash part
	dashCount := strings.Count(timestampHash, "-")
	if dashCount != 2 {
		return "", fmt.Errorf("invalid timestamp-hash format: %s", timestampHash)
	}

	// Find the last dash to separate timestamp from hash
	lastDashIndex := strings.LastIndex(timestampHash, "-")
	if lastDashIndex == -1 {
		return "", fmt.Errorf("invalid timestamp-hash format: %s", timestampHash)
	}

	timestamp := timestampHash[:lastDashIndex]
	hash := timestampHash[lastDashIndex+1:]

	// Validate timestamp format (YYYYMMDD-HHMM)
	if len(timestamp) != 13 || timestamp[8] != '-' {
		return "", fmt.Errorf("invalid timestamp format: %s", timestamp)
	}

	// Validate hash (should be at least 5 characters)
	if len(hash) < 5 {
		return "", fmt.Errorf("invalid hash format: %s", hash)
	}

	imageRef := fmt.Sprintf("%s:%s-%s", appName, timestamp, hash)
	return imageRef, nil
}

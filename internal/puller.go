package internal

import (
	"compress/gzip"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/schollz/progressbar/v3"
)

type ImagePuller struct {
	docker DockerClient
	s3     S3Client
	bucket string
	audit  AuditLogger
}

func NewImagePuller(docker DockerClient, s3 S3Client, bucket string) *ImagePuller {
	auditLogger := NewS3AuditLogger(s3, bucket)
	return &ImagePuller{
		docker: docker,
		s3:     s3,
		bucket: bucket,
		audit:  auditLogger,
	}
}

// Pull image from environment (e.g., "myapp production")
func (p *ImagePuller) Pull(ctx context.Context, appName, environment string) error {
	LogInfo("Pulling %s from %s environment", appName, environment)

	// Get environment pointer
	envKey := GeneratePointerKey(appName, environment)
	LogDebug("Looking for environment pointer at: %s", envKey)

	exists, err := p.s3.Exists(ctx, p.bucket, envKey)
	if err != nil {
		LogError("Failed to check environment pointer existence: %v", err)
		return fmt.Errorf("failed to check environment pointer existence: %w", err)
	}

	if !exists {
		LogError("Environment pointer not found: %s/%s", appName, environment)
		return fmt.Errorf("environment pointer not found: %s/%s", appName, environment)
	}

	// Download environment pointer
	LogDebug("Downloading environment pointer")
	pointerData, err := p.s3.Download(ctx, p.bucket, envKey)
	if err != nil {
		LogError("Failed to download environment pointer: %v", err)
		return fmt.Errorf("failed to download environment pointer: %w", err)
	}

	pointer, err := PointerMetadataFromJSON(pointerData)
	if err != nil {
		LogError("Failed to parse environment pointer: %v", err)
		return fmt.Errorf("failed to parse environment pointer: %w", err)
	}

	LogDebug("Environment pointer type: %s, target: %s", pointer.TargetType, pointer.TargetPath)

	var imageS3Path string

	// Resolve target path based on pointer type
	switch pointer.TargetType {
	case TargetTypeImage:
		// Direct image reference
		imageS3Path = pointer.TargetPath
		LogDebug("Direct image reference: %s", imageS3Path)

	case TargetTypeTag:
		// Tag reference - need to resolve to image
		LogDebug("Tag reference, resolving: %s", pointer.TargetPath)
		tagData, err := p.s3.Download(ctx, p.bucket, pointer.TargetPath)
		if err != nil {
			LogError("Failed to download tag pointer: %v", err)
			return fmt.Errorf("failed to download tag pointer: %w", err)
		}

		tagPointer, err := PointerMetadataFromJSON(tagData)
		if err != nil {
			LogError("Failed to parse tag pointer: %v", err)
			return fmt.Errorf("failed to parse tag pointer: %w", err)
		}

		imageS3Path = tagPointer.TargetPath
		LogDebug("Resolved tag to image: %s", imageS3Path)

	default:
		LogError("Unknown pointer type: %s", pointer.TargetType)
		return fmt.Errorf("unknown pointer type: %s", pointer.TargetType)
	}

	// Download and import image
	return p.downloadAndImportImage(ctx, appName, environment, imageS3Path)
}

// PullFromTag pulls image directly from tag (e.g., "myapp v1.2.0")
func (p *ImagePuller) PullFromTag(ctx context.Context, appName, version string) error {
	LogInfo("Pulling %s tag %s", appName, version)

	// Get tag pointer
	tagKey := GenerateTagKey(appName, version)
	LogDebug("Looking for tag pointer at: %s", tagKey)

	exists, err := p.s3.Exists(ctx, p.bucket, tagKey)
	if err != nil {
		LogError("Failed to check tag existence: %v", err)
		return fmt.Errorf("failed to check tag existence: %w", err)
	}

	if !exists {
		LogError("Tag not found: %s/%s", appName, version)
		return fmt.Errorf("tag not found: %s/%s", appName, version)
	}

	// Download tag pointer
	LogDebug("Downloading tag pointer")
	tagData, err := p.s3.Download(ctx, p.bucket, tagKey)
	if err != nil {
		LogError("Failed to download tag pointer: %v", err)
		return fmt.Errorf("failed to download tag pointer: %w", err)
	}

	tagPointer, err := PointerMetadataFromJSON(tagData)
	if err != nil {
		LogError("Failed to parse tag pointer: %v", err)
		return fmt.Errorf("failed to parse tag pointer: %w", err)
	}

	imageS3Path := tagPointer.TargetPath
	LogDebug("Tag points to image: %s", imageS3Path)

	// Download and import image
	return p.downloadAndImportImage(ctx, appName, version, imageS3Path)
}

// downloadAndImportImage handles the core download, verify, and import logic
func (p *ImagePuller) downloadAndImportImage(ctx context.Context, appName, source, imageS3Path string) error {
	// Get metadata path
	metadataKey := GenerateMetadataKey(imageS3Path)
	LogDebug("Getting metadata from: %s", metadataKey)

	// Download metadata
	metadataData, err := p.s3.Download(ctx, p.bucket, metadataKey)
	if err != nil {
		LogError("Failed to download image metadata: %v", err)
		return fmt.Errorf("failed to download image metadata: %w", err)
	}

	metadata, err := ImageMetadataFromJSON(metadataData)
	if err != nil {
		LogError("Failed to parse image metadata: %v", err)
		return fmt.Errorf("failed to parse image metadata: %w", err)
	}

	LogDebug("Image metadata - size: %d bytes, checksum: %s", metadata.Size, metadata.Checksum)

	// Create temporary file for download
	tempFile, err := os.CreateTemp("", "s3dock-pull-*.tar.gz")
	if err != nil {
		LogError("Failed to create temp file: %v", err)
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Always cleanup temp file
	defer tempFile.Close()

	// Download with retries and checksum verification
	const maxRetries = 3
	var downloadErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		LogInfo("Downloading image (attempt %d/%d)", attempt, maxRetries)

		// Reset file position
		tempFile.Seek(0, 0)
		tempFile.Truncate(0)

		downloadErr = p.downloadImageWithProgress(ctx, imageS3Path, tempFile, metadata.Size)
		if downloadErr != nil {
			LogError("Download attempt %d failed: %v", attempt, downloadErr)
			continue
		}

		// Verify checksum
		tempFile.Seek(0, 0)
		actualChecksum, err := calculateFileChecksum(tempFile)
		if err != nil {
			LogError("Failed to calculate checksum (attempt %d): %v", attempt, err)
			downloadErr = err
			continue
		}

		if actualChecksum == metadata.Checksum {
			LogInfo("Checksum verified: %s", actualChecksum)
			break
		}

		LogError("Checksum mismatch (attempt %d): expected %s, got %s", attempt, metadata.Checksum, actualChecksum)
		downloadErr = fmt.Errorf("checksum mismatch: expected %s, got %s", metadata.Checksum, actualChecksum)
	}

	if downloadErr != nil {
		return fmt.Errorf("download failed after %d attempts: %w", maxRetries, downloadErr)
	}

	// Import to Docker
	LogInfo("Importing image to Docker")
	tempFile.Seek(0, 0)

	spinner := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription("Importing to Docker..."),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetWidth(50),
	)
	spinner.RenderBlank()

	err = p.importImageFromGzip(ctx, tempFile)
	spinner.Finish()

	if err != nil {
		LogError("Failed to import image to Docker: %v", err)
		return fmt.Errorf("failed to import image to Docker: %w", err)
	}

	LogInfo("Successfully pulled and imported %s from %s", appName, source)
	return nil
}

// downloadImageWithProgress downloads image from S3 with progress bar
func (p *ImagePuller) downloadImageWithProgress(ctx context.Context, imageS3Path string, dest io.WriteSeeker, expectedSize int64) error {
	// Note: We need to add a DownloadWithProgress method to S3Client interface
	// For now, use regular download - this will be enhanced
	LogDebug("Downloading image from S3: %s", imageS3Path)

	// Create progress bar
	bar := progressbar.DefaultBytes(expectedSize, "Downloading image")
	defer bar.Finish()

	// This is a placeholder - we'll need to enhance S3Client to support streaming downloads
	// For now, let's implement basic functionality
	data, err := p.s3.Download(ctx, p.bucket, imageS3Path)
	if err != nil {
		return err
	}

	// Write with progress tracking
	reader := strings.NewReader(string(data))
	progressReader := progressbar.NewReader(reader, bar)

	_, err = io.Copy(dest, &progressReader)
	return err
}

// importImageFromGzip decompresses and imports gzipped tar to Docker
func (p *ImagePuller) importImageFromGzip(ctx context.Context, gzipFile io.Reader) error {
	// Create gzip reader
	gzipReader, err := gzip.NewReader(gzipFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Import to Docker - this will preserve original tags
	err = p.docker.ImportImage(ctx, gzipReader)
	if err != nil {
		return fmt.Errorf("failed to import image: %w", err)
	}

	return nil
}

// calculateFileChecksum calculates MD5 checksum of file
func calculateFileChecksum(file io.ReadSeeker) (string, error) {
	hasher := md5.New()
	_, err := io.Copy(hasher, file)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

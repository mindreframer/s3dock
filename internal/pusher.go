package internal

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

type ImagePusher struct {
	docker DockerClient
	s3     S3Client
	git    GitClient
	bucket string
	audit  AuditLogger
}

func NewImagePusher(docker DockerClient, s3 S3Client, git GitClient, bucket string) *ImagePusher {
	auditLogger := NewS3AuditLogger(s3, bucket)
	return &ImagePusher{
		docker: docker,
		s3:     s3,
		git:    git,
		bucket: bucket,
		audit:  auditLogger,
	}
}

func (p *ImagePusher) Push(ctx context.Context, imageRef string) (*PushResult, error) {
	LogInfo("Pushing image %s to S3", imageRef)

	gitHash, err := p.git.GetCurrentHash(".")
	if err != nil {
		LogError("Failed to get git hash: %v", err)
		return nil, fmt.Errorf("failed to get git hash: %w", err)
	}

	gitTime, err := p.git.GetCommitTimestamp(".")
	if err != nil {
		LogError("Failed to get git timestamp: %v", err)
		return nil, fmt.Errorf("failed to get git timestamp: %w", err)
	}

	appName := ExtractAppName(imageRef)
	yearMonth := time.Now().Format("200601")

	filename := fmt.Sprintf("%s-%s-%s.tar.gz", appName, gitTime, gitHash)
	s3Key := fmt.Sprintf("images/%s/%s/%s", appName, yearMonth, filename)
	metadataKey := GenerateMetadataKey(s3Key)

	LogDebug("App name: %s, Git hash: %s, Git time: %s", appName, gitHash, gitTime)
	LogDebug("S3 key: %s, Metadata key: %s", s3Key, metadataKey)

	// Check if metadata exists and compare checksums
	LogDebug("Checking if metadata exists at %s", metadataKey)
	exists, err := p.s3.Exists(ctx, p.bucket, metadataKey)
	if err != nil {
		LogError("Failed to check metadata existence: %v", err)
		return nil, fmt.Errorf("failed to check metadata existence: %w", err)
	}

	LogDebug("Exporting Docker image %s", imageRef)
	var spinner *progressbar.ProgressBar
	if !IsJSONOutput() {
		spinner = progressbar.NewOptions(-1,
			progressbar.OptionSetDescription("Exporting Docker image..."),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionSetWidth(50),
		)
		spinner.RenderBlank()
	}

	imageData, err := p.docker.ExportImage(ctx, imageRef)
	if spinner != nil {
		spinner.Finish()
	}
	if err != nil {
		LogError("Failed to export image: %v", err)
		return nil, fmt.Errorf("failed to export image: %w", err)
	}
	defer imageData.Close()

	// Parse git time for tar normalization
	fixedTime, err := ParseGitTime(gitTime)
	if err != nil {
		LogError("Failed to parse git time: %v", err)
		return nil, fmt.Errorf("failed to parse git time: %w", err)
	}

	// Normalize tar timestamps and add gzip compression
	LogDebug("Normalizing tar timestamps to %s and compressing with gzip", fixedTime.Format("2006-01-02 15:04:05"))
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		
		// Create a pipe for tar normalization
		normPr, normPw := io.Pipe()
		
		// Goroutine 1: Normalize tar
		go func() {
			defer normPw.Close()
			if err := NormalizeTar(imageData, normPw, fixedTime); err != nil {
				LogError("Failed to normalize tar: %v", err)
				normPw.CloseWithError(err)
				return
			}
		}()
		
		// Goroutine 2: Compress normalized tar
		gzipWriter := gzip.NewWriter(pw)
		gzipWriter.ModTime = time.Time{} // Set to zero time for deterministic output
		defer gzipWriter.Close()

		if _, err := io.Copy(gzipWriter, normPr); err != nil {
			LogError("Failed to compress image data: %v", err)
			pw.CloseWithError(err)
			return
		}
	}()

	// Calculate metadata while buffering compressed data
	LogDebug("Calculating metadata for compressed image")
	var buf bytes.Buffer
	teeReader := io.TeeReader(pr, &buf)

	metadata, _, err := CalculateMetadata(teeReader, gitHash, gitTime, imageRef, appName)
	if err != nil {
		LogError("Failed to calculate metadata: %v", err)
		return nil, fmt.Errorf("failed to calculate metadata: %w", err)
	}

	LogDebug("Image checksum: %s, size: %d bytes", metadata.Checksum, metadata.Size)

	// If metadata exists, compare checksums
	if exists {
		LogDebug("Metadata exists, downloading for checksum comparison")
		existingMetadataBytes, err := p.s3.Download(ctx, p.bucket, metadataKey)
		if err != nil {
			LogError("Failed to download existing metadata: %v", err)
			return nil, fmt.Errorf("failed to download existing metadata: %w", err)
		}

		existingMetadata, err := ImageMetadataFromJSON(existingMetadataBytes)
		if err != nil {
			LogError("Failed to parse existing metadata: %v", err)
			return nil, fmt.Errorf("failed to parse existing metadata: %w", err)
		}

		LogDebug("Comparing checksums - existing: %s, new: %s", existingMetadata.Checksum, metadata.Checksum)
		if existingMetadata.Checksum == metadata.Checksum {
			LogInfo("Image %s already exists with same checksum, skipping upload", imageRef)

			// Log audit event for skipped upload
			auditEvent, err := CreatePushEvent(appName, gitHash, gitTime, imageRef, s3Key, metadata.Checksum, metadata.Size, true, false)
			if err == nil {
				p.audit.LogEvent(ctx, auditEvent)
			}

			return &PushResult{
				ImageRef: imageRef,
				S3Key:    s3Key,
				Checksum: metadata.Checksum,
				Size:     metadata.Size,
				Skipped:  true,
				Archived: false,
			}, nil
		}

		// Checksums don't match - archive existing files
		LogError("Checksum mismatch for %s (existing: %s, new: %s). Archiving existing files.",
			imageRef, existingMetadata.Checksum, metadata.Checksum)

		if err := p.archiveExistingFiles(ctx, s3Key, metadataKey); err != nil {
			LogError("Failed to archive existing files: %v", err)
			return nil, fmt.Errorf("failed to archive existing files: %w", err)
		}
	}

	// Upload new image
	LogDebug("Uploading image to S3: %s", s3Key)
	if err := p.s3.UploadWithProgress(ctx, p.bucket, s3Key, &buf, metadata.Size, "Uploading image"); err != nil {
		LogError("Failed to upload image to S3: %v", err)
		return nil, fmt.Errorf("failed to upload image to S3: %w", err)
	}

	// Upload metadata
	LogDebug("Uploading metadata to S3: %s", metadataKey)
	metadataJSON, err := metadata.ToJSON()
	if err != nil {
		LogError("Failed to serialize metadata: %v", err)
		return nil, fmt.Errorf("failed to serialize metadata: %w", err)
	}

	if err := p.s3.Upload(ctx, p.bucket, metadataKey, strings.NewReader(string(metadataJSON))); err != nil {
		LogError("Failed to upload metadata to S3: %v", err)
		return nil, fmt.Errorf("failed to upload metadata to S3: %w", err)
	}

	LogInfo("Successfully pushed %s to s3://%s/%s (checksum: %s)", imageRef, p.bucket, s3Key, metadata.Checksum)

	// Log audit event for successful upload
	wasArchived := exists // If metadata existed, we archived it
	auditEvent, err := CreatePushEvent(appName, gitHash, gitTime, imageRef, s3Key, metadata.Checksum, metadata.Size, false, wasArchived)
	if err == nil {
		p.audit.LogEvent(ctx, auditEvent)
	}

	return &PushResult{
		ImageRef: imageRef,
		S3Key:    s3Key,
		Checksum: metadata.Checksum,
		Size:     metadata.Size,
		Skipped:  false,
		Archived: wasArchived,
	}, nil
}

func (p *ImagePusher) archiveExistingFiles(ctx context.Context, imageS3Key, metadataKey string) error {
	timestamp := time.Now().Format("20060102-1504")
	archiveImageKey, archiveMetaKey := GenerateArchiveKeys(imageS3Key, timestamp)

	LogDebug("Archiving existing files - image: %s, metadata: %s", imageS3Key, metadataKey)
	LogDebug("Archive paths - image: %s, metadata: %s", archiveImageKey, archiveMetaKey)

	// Copy image to archive
	if err := p.s3.Copy(ctx, p.bucket, imageS3Key, archiveImageKey); err != nil {
		LogError("Failed to copy image to archive: %v", err)
		return fmt.Errorf("failed to copy image to archive: %w", err)
	}

	// Copy metadata to archive
	if err := p.s3.Copy(ctx, p.bucket, metadataKey, archiveMetaKey); err != nil {
		LogError("Failed to copy metadata to archive: %v", err)
		return fmt.Errorf("failed to copy metadata to archive: %w", err)
	}

	// Delete original files (they will be replaced)
	if err := p.s3.Delete(ctx, p.bucket, imageS3Key); err != nil {
		LogError("Failed to delete original image: %v", err)
		return fmt.Errorf("failed to delete original image: %w", err)
	}

	if err := p.s3.Delete(ctx, p.bucket, metadataKey); err != nil {
		LogError("Failed to delete original metadata: %v", err)
		return fmt.Errorf("failed to delete original metadata: %w", err)
	}

	LogInfo("Archived existing files to %s and %s", archiveImageKey, archiveMetaKey)
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

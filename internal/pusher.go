package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"
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

func (p *ImagePusher) Push(ctx context.Context, imageRef string) error {
	gitHash, err := p.git.GetCurrentHash()
	if err != nil {
		return fmt.Errorf("failed to get git hash: %w", err)
	}

	gitTime, err := p.git.GetCommitTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get git timestamp: %w", err)
	}

	appName := ExtractAppName(imageRef)
	yearMonth := time.Now().Format("200601")

	filename := fmt.Sprintf("%s-%s-%s.tar.gz", appName, gitTime, gitHash)
	s3Key := fmt.Sprintf("images/%s/%s/%s", appName, yearMonth, filename)
	metadataKey := GenerateMetadataKey(s3Key)

	// Check if metadata exists and compare checksums
	exists, err := p.s3.Exists(ctx, p.bucket, metadataKey)
	if err != nil {
		return fmt.Errorf("failed to check metadata existence: %w", err)
	}

	imageData, err := p.docker.ExportImage(ctx, imageRef)
	if err != nil {
		return fmt.Errorf("failed to export image: %w", err)
	}
	defer imageData.Close()

	// Calculate metadata while buffering data
	var buf bytes.Buffer
	teeReader := io.TeeReader(imageData, &buf)

	metadata, _, err := CalculateMetadata(teeReader, gitHash, gitTime, imageRef, appName)
	if err != nil {
		return fmt.Errorf("failed to calculate metadata: %w", err)
	}

	// If metadata exists, compare checksums
	if exists {
		existingMetadataBytes, err := p.s3.Download(ctx, p.bucket, metadataKey)
		if err != nil {
			return fmt.Errorf("failed to download existing metadata: %w", err)
		}

		existingMetadata, err := ImageMetadataFromJSON(existingMetadataBytes)
		if err != nil {
			return fmt.Errorf("failed to parse existing metadata: %w", err)
		}

		if existingMetadata.Checksum == metadata.Checksum {
			fmt.Printf("Image %s already exists with same checksum, skipping upload\n", imageRef)
			
			// Log audit event for skipped upload
			auditEvent, err := CreatePushEvent(appName, gitHash, gitTime, imageRef, s3Key, metadata.Checksum, metadata.Size, true, false)
			if err == nil {
				p.audit.LogEvent(ctx, auditEvent)
			}
			
			return nil
		}

		// Checksums don't match - archive existing files
		log.Printf("WARNING: Checksum mismatch for %s (existing: %s, new: %s). Archiving existing files.",
			imageRef, existingMetadata.Checksum, metadata.Checksum)

		if err := p.archiveExistingFiles(ctx, s3Key, metadataKey); err != nil {
			return fmt.Errorf("failed to archive existing files: %w", err)
		}
	}

	// Upload new image
	if err := p.s3.Upload(ctx, p.bucket, s3Key, &buf); err != nil {
		return fmt.Errorf("failed to upload image to S3: %w", err)
	}

	// Upload metadata
	metadataJSON, err := metadata.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	if err := p.s3.Upload(ctx, p.bucket, metadataKey, strings.NewReader(string(metadataJSON))); err != nil {
		return fmt.Errorf("failed to upload metadata to S3: %w", err)
	}

	fmt.Printf("Successfully pushed %s to s3://%s/%s (checksum: %s)\n", imageRef, p.bucket, s3Key, metadata.Checksum)
	
	// Log audit event for successful upload
	wasArchived := exists // If metadata existed, we archived it
	auditEvent, err := CreatePushEvent(appName, gitHash, gitTime, imageRef, s3Key, metadata.Checksum, metadata.Size, false, wasArchived)
	if err == nil {
		p.audit.LogEvent(ctx, auditEvent)
	}
	
	return nil
}

func (p *ImagePusher) archiveExistingFiles(ctx context.Context, imageS3Key, metadataKey string) error {
	timestamp := time.Now().Format("20060102-1504")
	archiveImageKey, archiveMetaKey := GenerateArchiveKeys(imageS3Key, timestamp)

	// Copy image to archive
	if err := p.s3.Copy(ctx, p.bucket, imageS3Key, archiveImageKey); err != nil {
		return fmt.Errorf("failed to copy image to archive: %w", err)
	}

	// Copy metadata to archive
	if err := p.s3.Copy(ctx, p.bucket, metadataKey, archiveMetaKey); err != nil {
		return fmt.Errorf("failed to copy metadata to archive: %w", err)
	}

	// Delete original files (they will be replaced)
	if err := p.s3.Delete(ctx, p.bucket, imageS3Key); err != nil {
		return fmt.Errorf("failed to delete original image: %w", err)
	}

	if err := p.s3.Delete(ctx, p.bucket, metadataKey); err != nil {
		return fmt.Errorf("failed to delete original metadata: %w", err)
	}

	fmt.Printf("Archived existing files to %s and %s\n", archiveImageKey, archiveMetaKey)
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

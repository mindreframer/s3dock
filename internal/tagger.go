package internal

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ImageTagger struct {
	s3     S3Client
	bucket string
	audit  AuditLogger
}

func NewImageTagger(s3Client S3Client, bucket string) *ImageTagger {
	auditLogger := NewS3AuditLogger(s3Client, bucket)
	return &ImageTagger{
		s3:     s3Client,
		bucket: bucket,
		audit:  auditLogger,
	}
}

func (t *ImageTagger) Tag(ctx context.Context, imageRef, version string) error {
	LogInfo("Creating tag %s for image %s", version, imageRef)

	// Parse image reference to extract components
	appName, gitTime, gitHash, err := ParseImageReference(imageRef)
	if err != nil {
		LogError("Failed to parse image reference: %v", err)
		return fmt.Errorf("failed to parse image reference: %w", err)
	}

	LogDebug("Parsed image reference - app: %s, git time: %s, git hash: %s", appName, gitTime, gitHash)

	// Construct expected image S3 path
	yearMonth := time.Now().Format("200601") // Use current year/month for lookup
	imageFilename := fmt.Sprintf("%s-%s-%s.tar.gz", appName, gitTime, gitHash)
	imageS3Path := fmt.Sprintf("images/%s/%s/%s", appName, yearMonth, imageFilename)

	LogDebug("Looking for image at S3 path: %s", imageS3Path)

	// Verify the image exists in S3
	exists, err := t.s3.Exists(ctx, t.bucket, imageS3Path)
	if err != nil {
		LogError("Failed to check if image exists: %v", err)
		return fmt.Errorf("failed to check if image exists: %w", err)
	}
	if !exists {
		LogError("Image not found in S3: %s", imageS3Path)
		return fmt.Errorf("image not found in S3: %s", imageS3Path)
	}

	// Create tag pointer
	tagKey := GenerateTagKey(appName, version)
	LogDebug("Creating tag pointer at S3 key: %s", tagKey)

	pointer, err := CreateImagePointer(imageS3Path, gitHash, gitTime, imageRef)
	if err != nil {
		LogError("Failed to create tag pointer: %v", err)
		return fmt.Errorf("failed to create tag pointer: %w", err)
	}

	// Upload tag to S3
	LogDebug("Uploading tag pointer to S3")
	pointerJSON, err := pointer.ToJSON()
	if err != nil {
		LogError("Failed to serialize tag pointer: %v", err)
		return fmt.Errorf("failed to serialize tag pointer: %w", err)
	}

	if err := t.s3.Upload(ctx, t.bucket, tagKey, strings.NewReader(string(pointerJSON))); err != nil {
		LogError("Failed to upload tag to S3: %v", err)
		return fmt.Errorf("failed to upload tag to S3: %w", err)
	}

	LogInfo("Successfully tagged %s as %s", imageRef, version)

	// Log audit event for tag creation
	auditEvent, err := CreateTagEvent(appName, gitHash, gitTime, imageRef, version, tagKey)
	if err == nil {
		t.audit.LogEvent(ctx, auditEvent)
	}

	return nil
}

type ImagePromoter struct {
	s3     S3Client
	bucket string
	audit  AuditLogger
}

func NewImagePromoter(s3Client S3Client, bucket string) *ImagePromoter {
	auditLogger := NewS3AuditLogger(s3Client, bucket)
	return &ImagePromoter{
		s3:     s3Client,
		bucket: bucket,
		audit:  auditLogger,
	}
}

func (p *ImagePromoter) Promote(ctx context.Context, source, environment string) error {
	LogInfo("Promoting %s to %s environment", source, environment)

	appName := ""
	var pointer *PointerMetadata
	var err error
	var gitTime, gitHash string

	// Determine if source is an image reference or a version tag
	if strings.Contains(source, ":") {
		// It's an image reference like myapp:20250721-2118-f7a5a27
		LogDebug("Source appears to be an image reference")
		appName, gitTime, gitHash, err = ParseImageReference(source)
		if err != nil {
			LogError("Failed to parse image reference: %v", err)
			return fmt.Errorf("failed to parse image reference: %w", err)
		}

		LogDebug("Parsed image reference - app: %s, git time: %s, git hash: %s", appName, gitTime, gitHash)

		// Construct expected image S3 path
		yearMonth := time.Now().Format("200601") // Use current year/month for lookup
		imageFilename := fmt.Sprintf("%s-%s-%s.tar.gz", appName, gitTime, gitHash)
		imageS3Path := fmt.Sprintf("images/%s/%s/%s", appName, yearMonth, imageFilename)

		LogDebug("Looking for image at S3 path: %s", imageS3Path)

		// Verify the image exists in S3
		exists, err := p.s3.Exists(ctx, p.bucket, imageS3Path)
		if err != nil {
			LogError("Failed to check if image exists: %v", err)
			return fmt.Errorf("failed to check if image exists: %w", err)
		}
		if !exists {
			LogError("Image not found in S3: %s", imageS3Path)
			return fmt.Errorf("image not found in S3: %s", imageS3Path)
		}

		// Create pointer directly to image
		LogDebug("Creating image pointer for promotion")
		pointer, err = CreateImagePointer(imageS3Path, gitHash, gitTime, source)
		if err != nil {
			LogError("Failed to create image pointer: %v", err)
			return fmt.Errorf("failed to create image pointer: %w", err)
		}

	} else {
		// It's a version tag like v1.2.0, need to determine app name from environment context
		// For now, extract from environment context or require app name
		// This is a simplification - in practice you might want to require app name
		LogError("Promoting from version tags requires specifying app name - use 'appname:version' format or direct image reference")
		return fmt.Errorf("promoting from version tags requires specifying app name - use 'appname:version' format or direct image reference")
	}

	// Check for existing pointer to track previous state and detect duplicates
	envKey := GeneratePointerKey(appName, environment)
	LogDebug("Environment pointer key: %s", envKey)

	var previousTarget string

	existingExists, err := p.s3.Exists(ctx, p.bucket, envKey)
	if err == nil && existingExists {
		LogDebug("Existing environment pointer found, checking previous target")
		existingData, err := p.s3.Download(ctx, p.bucket, envKey)
		if err == nil {
			existingPointer, err := PointerMetadataFromJSON(existingData)
			if err == nil {
				previousTarget = existingPointer.TargetPath
				LogDebug("Previous target: %s", previousTarget)

				// Check if we're promoting to the same target
				newTargetPath := pointer.TargetPath
				if existingPointer.TargetPath == newTargetPath {
					LogInfo("Environment %s is already pointing to %s, skipping promotion", environment, newTargetPath)
					return nil
				}
				LogDebug("Target changed from %s to %s, proceeding with promotion", existingPointer.TargetPath, newTargetPath)
			}
		}
	}

	// Upload pointer to environment
	LogDebug("Uploading environment pointer to S3")
	pointerJSON, err := pointer.ToJSON()
	if err != nil {
		LogError("Failed to serialize environment pointer: %v", err)
		return fmt.Errorf("failed to serialize environment pointer: %w", err)
	}

	if err := p.s3.Upload(ctx, p.bucket, envKey, strings.NewReader(string(pointerJSON))); err != nil {
		LogError("Failed to upload environment pointer to S3: %v", err)
		return fmt.Errorf("failed to upload environment pointer to S3: %w", err)
	}

	LogInfo("Successfully promoted %s to %s environment", source, environment)

	// Log audit event for promotion
	auditEvent, err := CreatePromotionEvent(appName, pointer.GitHash, pointer.GitTime, environment, source, "image", envKey, previousTarget)
	if err != nil {
		LogError("Failed to create promotion audit event: %v", err)
		return fmt.Errorf("failed to create promotion audit event: %w", err)
	}

	if err := p.audit.LogEvent(ctx, auditEvent); err != nil {
		LogError("Failed to log promotion audit event: %v", err)
		return fmt.Errorf("failed to log promotion audit event: %w", err)
	}

	return nil
}

func (p *ImagePromoter) PromoteFromTag(ctx context.Context, appName, version, environment string) error {
	LogInfo("Promoting %s %s to %s environment", appName, version, environment)

	// Download the tag to get image information
	tagKey := GenerateTagKey(appName, version)
	LogDebug("Looking for tag at S3 key: %s", tagKey)

	tagExists, err := p.s3.Exists(ctx, p.bucket, tagKey)
	if err != nil {
		LogError("Failed to check if tag exists: %v", err)
		return fmt.Errorf("failed to check if tag exists: %w", err)
	}
	if !tagExists {
		LogError("Tag not found: %s/%s", appName, version)
		return fmt.Errorf("tag not found: %s/%s", appName, version)
	}

	LogDebug("Downloading tag data from S3")
	tagData, err := p.s3.Download(ctx, p.bucket, tagKey)
	if err != nil {
		LogError("Failed to download tag: %v", err)
		return fmt.Errorf("failed to download tag: %w", err)
	}

	tagPointer, err := PointerMetadataFromJSON(tagData)
	if err != nil {
		LogError("Failed to parse tag: %v", err)
		return fmt.Errorf("failed to parse tag: %w", err)
	}

	LogDebug("Tag points to image: %s", tagPointer.SourceImage)

	// Create environment pointer that points to the tag
	LogDebug("Creating environment pointer that references tag")
	envPointer, err := CreateTagPointer(tagKey, tagPointer.GitHash, tagPointer.GitTime, tagPointer.SourceImage, version)
	if err != nil {
		LogError("Failed to create environment pointer: %v", err)
		return fmt.Errorf("failed to create environment pointer: %w", err)
	}

	// Check for existing pointer to track previous state and detect duplicates
	envKey := GeneratePointerKey(appName, environment)
	LogDebug("Environment pointer key: %s", envKey)

	var previousTarget string

	existingExists, err := p.s3.Exists(ctx, p.bucket, envKey)
	if err == nil && existingExists {
		LogDebug("Existing environment pointer found, checking previous target")
		existingData, err := p.s3.Download(ctx, p.bucket, envKey)
		if err == nil {
			existingPointer, err := PointerMetadataFromJSON(existingData)
			if err == nil {
				previousTarget = existingPointer.TargetPath
				LogDebug("Previous target: %s", previousTarget)

				// Check if we're promoting to the same target
				newTargetPath := envPointer.TargetPath
				if existingPointer.TargetPath == newTargetPath {
					LogInfo("Environment %s is already pointing to %s, skipping tag promotion", environment, newTargetPath)
					return nil
				}
				LogDebug("Target changed from %s to %s, proceeding with tag promotion", existingPointer.TargetPath, newTargetPath)
			}
		}
	}

	// Upload environment pointer
	LogDebug("Uploading environment pointer to S3")
	pointerJSON, err := envPointer.ToJSON()
	if err != nil {
		LogError("Failed to serialize environment pointer: %v", err)
		return fmt.Errorf("failed to serialize environment pointer: %w", err)
	}

	if err := p.s3.Upload(ctx, p.bucket, envKey, strings.NewReader(string(pointerJSON))); err != nil {
		LogError("Failed to upload environment pointer to S3: %v", err)
		return fmt.Errorf("failed to upload environment pointer to S3: %w", err)
	}

	LogInfo("Successfully promoted %s %s to %s environment", appName, version, environment)

	// Log audit event for tag-based promotion
	sourceRef := fmt.Sprintf("%s:%s", appName, version)
	auditEvent, err := CreatePromotionEvent(appName, tagPointer.GitHash, tagPointer.GitTime, environment, sourceRef, "tag", envKey, previousTarget)
	if err != nil {
		LogError("Failed to create tag promotion audit event: %v", err)
		return fmt.Errorf("failed to create tag promotion audit event: %w", err)
	}

	if err := p.audit.LogEvent(ctx, auditEvent); err != nil {
		LogError("Failed to log tag promotion audit event: %v", err)
		return fmt.Errorf("failed to log tag promotion audit event: %w", err)
	}

	return nil
}

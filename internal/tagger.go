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
	// Parse image reference to extract components
	appName, gitTime, gitHash, err := ParseImageReference(imageRef)
	if err != nil {
		return fmt.Errorf("failed to parse image reference: %w", err)
	}

	// Construct expected image S3 path
	yearMonth := time.Now().Format("200601") // Use current year/month for lookup
	imageFilename := fmt.Sprintf("%s-%s-%s.tar.gz", appName, gitTime, gitHash)
	imageS3Path := fmt.Sprintf("images/%s/%s/%s", appName, yearMonth, imageFilename)

	// Verify the image exists in S3
	exists, err := t.s3.Exists(ctx, t.bucket, imageS3Path)
	if err != nil {
		return fmt.Errorf("failed to check if image exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("image not found in S3: %s", imageS3Path)
	}

	// Create tag pointer
	tagKey := GenerateTagKey(appName, version)
	pointer, err := CreateImagePointer(imageS3Path, gitHash, gitTime, imageRef)
	if err != nil {
		return fmt.Errorf("failed to create tag pointer: %w", err)
	}

	// Upload tag to S3
	pointerJSON, err := pointer.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize tag pointer: %w", err)
	}

	if err := t.s3.Upload(ctx, t.bucket, tagKey, strings.NewReader(string(pointerJSON))); err != nil {
		return fmt.Errorf("failed to upload tag to S3: %w", err)
	}

	fmt.Printf("Successfully tagged %s as %s\n", imageRef, version)

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
	appName := ""
	var pointer *PointerMetadata
	var err error

	// Determine if source is an image reference or a version tag
	if strings.Contains(source, ":") {
		// It's an image reference like myapp:20250721-2118-f7a5a27
		appName, gitTime, gitHash, err := ParseImageReference(source)
		if err != nil {
			return fmt.Errorf("failed to parse image reference: %w", err)
		}

		// Construct expected image S3 path
		yearMonth := time.Now().Format("200601") // Use current year/month for lookup
		imageFilename := fmt.Sprintf("%s-%s-%s.tar.gz", appName, gitTime, gitHash)
		imageS3Path := fmt.Sprintf("images/%s/%s/%s", appName, yearMonth, imageFilename)

		// Verify the image exists in S3
		exists, err := p.s3.Exists(ctx, p.bucket, imageS3Path)
		if err != nil {
			return fmt.Errorf("failed to check if image exists: %w", err)
		}
		if !exists {
			return fmt.Errorf("image not found in S3: %s", imageS3Path)
		}

		// Create pointer directly to image
		pointer, err = CreateImagePointer(imageS3Path, gitHash, gitTime, source)
		if err != nil {
			return fmt.Errorf("failed to create image pointer: %w", err)
		}

	} else {
		// It's a version tag like v1.2.0, need to determine app name from environment context
		// For now, extract from environment context or require app name
		// This is a simplification - in practice you might want to require app name
		return fmt.Errorf("promoting from version tags requires specifying app name - use 'appname:version' format or direct image reference")
	}

	// Check for existing pointer to track previous state
	envKey := GeneratePointerKey(appName, environment)
	var previousTarget string

	existingExists, err := p.s3.Exists(ctx, p.bucket, envKey)
	if err == nil && existingExists {
		existingData, err := p.s3.Download(ctx, p.bucket, envKey)
		if err == nil {
			existingPointer, err := PointerMetadataFromJSON(existingData)
			if err == nil {
				previousTarget = existingPointer.TargetPath
			}
		}
	}

	// Upload pointer to environment
	pointerJSON, err := pointer.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize environment pointer: %w", err)
	}

	if err := p.s3.Upload(ctx, p.bucket, envKey, strings.NewReader(string(pointerJSON))); err != nil {
		return fmt.Errorf("failed to upload environment pointer to S3: %w", err)
	}

	fmt.Printf("Successfully promoted %s to %s environment\n", source, environment)

	// Log audit event for promotion
	auditEvent, err := CreatePromotionEvent(appName, pointer.GitHash, pointer.GitTime, environment, source, "image", envKey, previousTarget)
	if err == nil {
		p.audit.LogEvent(ctx, auditEvent)
	}

	return nil
}

func (p *ImagePromoter) PromoteFromTag(ctx context.Context, appName, version, environment string) error {
	// Download the tag to get image information
	tagKey := GenerateTagKey(appName, version)

	tagExists, err := p.s3.Exists(ctx, p.bucket, tagKey)
	if err != nil {
		return fmt.Errorf("failed to check if tag exists: %w", err)
	}
	if !tagExists {
		return fmt.Errorf("tag not found: %s/%s", appName, version)
	}

	tagData, err := p.s3.Download(ctx, p.bucket, tagKey)
	if err != nil {
		return fmt.Errorf("failed to download tag: %w", err)
	}

	tagPointer, err := PointerMetadataFromJSON(tagData)
	if err != nil {
		return fmt.Errorf("failed to parse tag: %w", err)
	}

	// Create environment pointer that points to the tag
	envPointer, err := CreateTagPointer(tagKey, tagPointer.GitHash, tagPointer.GitTime, tagPointer.SourceImage, version)
	if err != nil {
		return fmt.Errorf("failed to create environment pointer: %w", err)
	}

	// Check for existing pointer to track previous state
	envKey := GeneratePointerKey(appName, environment)
	var previousTarget string

	existingExists, err := p.s3.Exists(ctx, p.bucket, envKey)
	if err == nil && existingExists {
		existingData, err := p.s3.Download(ctx, p.bucket, envKey)
		if err == nil {
			existingPointer, err := PointerMetadataFromJSON(existingData)
			if err == nil {
				previousTarget = existingPointer.TargetPath
			}
		}
	}

	// Upload environment pointer
	pointerJSON, err := envPointer.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize environment pointer: %w", err)
	}

	if err := p.s3.Upload(ctx, p.bucket, envKey, strings.NewReader(string(pointerJSON))); err != nil {
		return fmt.Errorf("failed to upload environment pointer to S3: %w", err)
	}

	fmt.Printf("Successfully promoted %s %s to %s environment\n", appName, version, environment)

	// Log audit event for tag-based promotion
	sourceRef := fmt.Sprintf("%s:%s", appName, version)
	auditEvent, err := CreatePromotionEvent(appName, tagPointer.GitHash, tagPointer.GitTime, environment, sourceRef, "tag", envKey, previousTarget)
	if err == nil {
		p.audit.LogEvent(ctx, auditEvent)
	}

	return nil
}

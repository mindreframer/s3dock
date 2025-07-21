package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os/user"
	"strings"
	"time"
)

type TargetType string

const (
	TargetTypeImage TargetType = "image"
	TargetTypeTag   TargetType = "tag"
)

type PointerMetadata struct {
	TargetType  TargetType `json:"target_type"`
	TargetPath  string     `json:"target_path"`
	PromotedAt  time.Time  `json:"promoted_at"`
	PromotedBy  string     `json:"promoted_by"`
	GitHash     string     `json:"git_hash"`
	GitTime     string     `json:"git_time"`
	SourceImage string     `json:"source_image,omitempty"` // Original image reference if tagged
	SourceTag   string     `json:"source_tag,omitempty"`   // Source tag if promoted from tag
}

func (p *PointerMetadata) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

func PointerMetadataFromJSON(data []byte) (*PointerMetadata, error) {
	var pointer PointerMetadata
	if err := json.Unmarshal(data, &pointer); err != nil {
		return nil, err
	}
	return &pointer, nil
}

func CreateImagePointer(imageS3Path, gitHash, gitTime, sourceImage string) (*PointerMetadata, error) {
	promotedBy, err := getCurrentUser()
	if err != nil {
		promotedBy = "unknown"
	}

	return &PointerMetadata{
		TargetType:  TargetTypeImage,
		TargetPath:  imageS3Path,
		PromotedAt:  time.Now(),
		PromotedBy:  promotedBy,
		GitHash:     gitHash,
		GitTime:     gitTime,
		SourceImage: sourceImage,
	}, nil
}

func CreateTagPointer(tagS3Path, gitHash, gitTime, sourceImage, sourceTag string) (*PointerMetadata, error) {
	promotedBy, err := getCurrentUser()
	if err != nil {
		promotedBy = "unknown"
	}

	return &PointerMetadata{
		TargetType:  TargetTypeTag,
		TargetPath:  tagS3Path,
		PromotedAt:  time.Now(),
		PromotedBy:  promotedBy,
		GitHash:     gitHash,
		GitTime:     gitTime,
		SourceImage: sourceImage,
		SourceTag:   sourceTag,
	}, nil
}

func GenerateTagKey(appName, version string) string {
	return fmt.Sprintf("tags/%s/%s.json", appName, version)
}

func GeneratePointerKey(appName, environment string) string {
	return fmt.Sprintf("pointers/%s/%s.json", appName, environment)
}

func ResolveImagePath(ctx context.Context, s3Client S3Client, bucket string, pointer *PointerMetadata) (string, error) {
	switch pointer.TargetType {
	case TargetTypeImage:
		return pointer.TargetPath, nil
	case TargetTypeTag:
		// Download the tag to get the actual image path
		tagData, err := s3Client.Download(ctx, bucket, pointer.TargetPath)
		if err != nil {
			return "", fmt.Errorf("failed to download tag %s: %w", pointer.TargetPath, err)
		}

		tagPointer, err := PointerMetadataFromJSON(tagData)
		if err != nil {
			return "", fmt.Errorf("failed to parse tag %s: %w", pointer.TargetPath, err)
		}

		// Recursively resolve in case tag points to another tag (though unlikely)
		return ResolveImagePath(ctx, s3Client, bucket, tagPointer)
	default:
		return "", fmt.Errorf("unknown target type: %s", pointer.TargetType)
	}
}

func ParseImageReference(imageRef string) (appName, gitTime, gitHash string, err error) {
	// Parse myapp:20250721-2118-f7a5a27 format
	parts := strings.Split(imageRef, ":")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid image reference format: %s", imageRef)
	}

	appName = parts[0]
	tagPart := parts[1]

	// Extract git time and hash from tag
	// Expected format: 20250721-1430-f7a5a27 (exactly 2 dashes, gittime-githash)
	dashCount := strings.Count(tagPart, "-")
	if dashCount != 2 {
		return "", "", "", fmt.Errorf("invalid tag format in image reference: %s", imageRef)
	}

	dashIndex := strings.LastIndex(tagPart, "-")
	if dashIndex == -1 {
		return "", "", "", fmt.Errorf("invalid tag format in image reference: %s", imageRef)
	}

	gitHash = tagPart[dashIndex+1:]
	gitTime = tagPart[:dashIndex]

	// Additional validation: gitTime should look like timestamp (YYYYMMDD-HHMM)
	// and gitHash should be hex-like
	if len(gitHash) < 5 || len(gitTime) != 13 || gitTime[8] != '-' {
		return "", "", "", fmt.Errorf("invalid tag format in image reference: %s", imageRef)
	}

	return appName, gitTime, gitHash, nil
}

func getCurrentUser() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return user.Username, nil
}

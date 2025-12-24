package internal

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ListService provides listing functionality for images, tags, and environments
type ListService struct {
	s3     S3Client
	bucket string
}

// ImageInfo contains information about a pushed image
type ImageInfo struct {
	AppName   string
	Tag       string // e.g., 20250721-2118-f7a5a27
	S3Path    string
	YearMonth string
}

// TagInfo contains information about a semantic version tag
type TagInfo struct {
	AppName     string
	Version     string // e.g., v1.2.0
	TargetImage string // e.g., myapp:20250721-2118-f7a5a27
	S3Path      string
}

// EnvInfo contains information about an environment pointer
type EnvInfo struct {
	AppName     string
	Environment string
	TargetType  TargetType // "image" or "tag"
	TargetPath  string
	SourceTag   string // If promoted from a tag
	SourceImage string // Resolved image reference
}

func NewListService(s3 S3Client, bucket string) *ListService {
	return &ListService{
		s3:     s3,
		bucket: bucket,
	}
}

// ListImages returns all images for an app, optionally filtered by year-month
func (l *ListService) ListImages(ctx context.Context, appName string, yearMonth string) ([]ImageInfo, error) {
	LogInfo("Listing images for %s", appName)

	prefix := fmt.Sprintf("images/%s/", appName)
	if yearMonth != "" {
		prefix = fmt.Sprintf("images/%s/%s/", appName, yearMonth)
	}

	LogDebug("Listing S3 objects with prefix: %s", prefix)
	keys, err := l.s3.List(ctx, l.bucket, prefix)
	if err != nil {
		LogError("Failed to list images: %v", err)
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	var images []ImageInfo
	for _, key := range keys {
		// Only include .tar.gz files (skip .json metadata files)
		if !strings.HasSuffix(key, ".tar.gz") {
			continue
		}

		info, err := l.parseImagePath(key)
		if err != nil {
			LogDebug("Skipping invalid image path %s: %v", key, err)
			continue
		}
		images = append(images, info)
	}

	// Sort by tag (which includes timestamp) in descending order (newest first)
	sort.Slice(images, func(i, j int) bool {
		return images[i].Tag > images[j].Tag
	})

	LogInfo("Found %d images for %s", len(images), appName)
	return images, nil
}

// ListTags returns all semantic version tags for an app
func (l *ListService) ListTags(ctx context.Context, appName string) ([]TagInfo, error) {
	LogInfo("Listing tags for %s", appName)

	prefix := fmt.Sprintf("tags/%s/", appName)

	LogDebug("Listing S3 objects with prefix: %s", prefix)
	keys, err := l.s3.List(ctx, l.bucket, prefix)
	if err != nil {
		LogError("Failed to list tags: %v", err)
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	var tags []TagInfo
	for _, key := range keys {
		if !strings.HasSuffix(key, ".json") {
			continue
		}

		// Extract version from path: tags/myapp/v1.2.0.json -> v1.2.0
		base := filepath.Base(key)
		version := strings.TrimSuffix(base, ".json")

		// Download tag to get target image
		LogDebug("Downloading tag %s", key)
		tagData, err := l.s3.Download(ctx, l.bucket, key)
		if err != nil {
			LogDebug("Failed to download tag %s: %v", key, err)
			continue
		}

		pointer, err := PointerMetadataFromJSON(tagData)
		if err != nil {
			LogDebug("Failed to parse tag %s: %v", key, err)
			continue
		}

		tags = append(tags, TagInfo{
			AppName:     appName,
			Version:     version,
			TargetImage: pointer.SourceImage,
			S3Path:      key,
		})
	}

	// Sort by version (semantic versioning aware would be better, but string sort works for v-prefixed versions)
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Version > tags[j].Version
	})

	LogInfo("Found %d tags for %s", len(tags), appName)
	return tags, nil
}

// ListEnvironments returns all environment pointers for an app
func (l *ListService) ListEnvironments(ctx context.Context, appName string) ([]EnvInfo, error) {
	LogInfo("Listing environments for %s", appName)

	prefix := fmt.Sprintf("pointers/%s/", appName)

	LogDebug("Listing S3 objects with prefix: %s", prefix)
	keys, err := l.s3.List(ctx, l.bucket, prefix)
	if err != nil {
		LogError("Failed to list environments: %v", err)
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	var envs []EnvInfo
	for _, key := range keys {
		if !strings.HasSuffix(key, ".json") {
			continue
		}

		// Extract environment from path: pointers/myapp/production.json -> production
		base := filepath.Base(key)
		environment := strings.TrimSuffix(base, ".json")

		// Download pointer to get target info
		LogDebug("Downloading environment pointer %s", key)
		pointerData, err := l.s3.Download(ctx, l.bucket, key)
		if err != nil {
			LogDebug("Failed to download pointer %s: %v", key, err)
			continue
		}

		pointer, err := PointerMetadataFromJSON(pointerData)
		if err != nil {
			LogDebug("Failed to parse pointer %s: %v", key, err)
			continue
		}

		envs = append(envs, EnvInfo{
			AppName:     appName,
			Environment: environment,
			TargetType:  pointer.TargetType,
			TargetPath:  pointer.TargetPath,
			SourceTag:   pointer.SourceTag,
			SourceImage: pointer.SourceImage,
		})
	}

	// Sort alphabetically by environment name
	sort.Slice(envs, func(i, j int) bool {
		return envs[i].Environment < envs[j].Environment
	})

	LogInfo("Found %d environments for %s", len(envs), appName)
	return envs, nil
}

// ListApps returns all apps that have images, tags, or environments
func (l *ListService) ListApps(ctx context.Context) ([]string, error) {
	LogInfo("Listing all apps")

	appSet := make(map[string]bool)

	// Check images
	imageKeys, err := l.s3.List(ctx, l.bucket, "images/")
	if err == nil {
		for _, key := range imageKeys {
			parts := strings.Split(key, "/")
			if len(parts) >= 2 {
				appSet[parts[1]] = true
			}
		}
	}

	// Check tags
	tagKeys, err := l.s3.List(ctx, l.bucket, "tags/")
	if err == nil {
		for _, key := range tagKeys {
			parts := strings.Split(key, "/")
			if len(parts) >= 2 {
				appSet[parts[1]] = true
			}
		}
	}

	// Check pointers
	pointerKeys, err := l.s3.List(ctx, l.bucket, "pointers/")
	if err == nil {
		for _, key := range pointerKeys {
			parts := strings.Split(key, "/")
			if len(parts) >= 2 {
				appSet[parts[1]] = true
			}
		}
	}

	var apps []string
	for app := range appSet {
		apps = append(apps, app)
	}
	sort.Strings(apps)

	LogInfo("Found %d apps", len(apps))
	return apps, nil
}

// parseImagePath extracts image info from an S3 path
// Example: images/myapp/202507/myapp-20250721-2118-f7a5a27.tar.gz
func (l *ListService) parseImagePath(s3Path string) (ImageInfo, error) {
	parts := strings.Split(s3Path, "/")
	if len(parts) < 4 {
		return ImageInfo{}, fmt.Errorf("invalid image path format")
	}

	appName := parts[1]
	yearMonth := parts[2]
	filename := parts[3]

	// Remove .tar.gz extension
	base := strings.TrimSuffix(filename, ".tar.gz")

	// Extract tag from filename: myapp-20250721-2118-f7a5a27 -> 20250721-2118-f7a5a27
	prefix := appName + "-"
	if !strings.HasPrefix(base, prefix) {
		return ImageInfo{}, fmt.Errorf("filename doesn't match app name")
	}
	tag := strings.TrimPrefix(base, prefix)

	return ImageInfo{
		AppName:   appName,
		Tag:       tag,
		S3Path:    s3Path,
		YearMonth: yearMonth,
	}, nil
}

// GetTagForEnvironment returns the semantic version tag for an environment (if promoted via tag)
func (l *ListService) GetTagForEnvironment(ctx context.Context, appName, environment string) (string, error) {
	LogInfo("Getting tag for %s in %s environment", appName, environment)

	envKey := GeneratePointerKey(appName, environment)

	exists, err := l.s3.Exists(ctx, l.bucket, envKey)
	if err != nil {
		return "", fmt.Errorf("failed to check environment pointer: %w", err)
	}
	if !exists {
		return "", fmt.Errorf("environment pointer not found: %s/%s", appName, environment)
	}

	pointerData, err := l.s3.Download(ctx, l.bucket, envKey)
	if err != nil {
		return "", fmt.Errorf("failed to download environment pointer: %w", err)
	}

	pointer, err := PointerMetadataFromJSON(pointerData)
	if err != nil {
		return "", fmt.Errorf("failed to parse environment pointer: %w", err)
	}

	// If promoted via tag, return the source tag
	if pointer.TargetType == TargetTypeTag && pointer.SourceTag != "" {
		return pointer.SourceTag, nil
	}

	// If promoted directly from image, there's no tag
	return "", nil
}

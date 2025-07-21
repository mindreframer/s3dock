package internal

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type ImageMetadata struct {
	Checksum   string    `json:"checksum"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
	GitHash    string    `json:"git_hash"`
	GitTime    string    `json:"git_time"`
	ImageTag   string    `json:"image_tag"`
	AppName    string    `json:"app_name"`
}

func (m *ImageMetadata) ToJSON() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

func ImageMetadataFromJSON(data []byte) (*ImageMetadata, error) {
	var metadata ImageMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

func CalculateMetadata(data io.Reader, gitHash, gitTime, imageTag, appName string) (*ImageMetadata, int64, error) {
	hasher := md5.New()
	size, err := io.Copy(hasher, data)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	metadata := &ImageMetadata{
		Checksum:  checksum,
		Size:      size,
		CreatedAt: time.Now(),
		GitHash:   gitHash,
		GitTime:   gitTime,
		ImageTag:  imageTag,
		AppName:   appName,
	}

	return metadata, size, nil
}

func GenerateMetadataKey(imageS3Key string) string {
	// Convert images/myapp/202507/myapp-20250721-2118-f7a5a27.tar.gz
	// to images/myapp/202507/myapp-20250721-2118-f7a5a27.json
	if len(imageS3Key) >= 11 && imageS3Key[:7] == "images/" {
		withoutExtension := imageS3Key[:len(imageS3Key)-7] // remove .tar.gz
		return withoutExtension + ".json"                  // keep in images/ folder, just change extension
	}
	return imageS3Key + ".json"
}

func GenerateArchiveKeys(imageS3Key string, timestamp string) (string, string) {
	// Generate archive keys with timestamp
	// images/myapp/202507/myapp-20250721-2118-f7a5a27.tar.gz
	// -> archive/myapp/202507/myapp-20250721-2118-f7a5a27-archived-on-20250722-1018.tar.gz

	archiveImageKey := ""
	archiveMetaKey := ""

	if len(imageS3Key) >= 11 && imageS3Key[:7] == "images/" {
		// Remove .tar.gz and add archive prefix with timestamp
		withoutExtension := imageS3Key[:len(imageS3Key)-7]
		archiveImageKey = "archive/" + withoutExtension[7:] + "-archived-on-" + timestamp + ".tar.gz"
		archiveMetaKey = "archive/" + withoutExtension[7:] + "-archived-on-" + timestamp + ".json"
	}

	return archiveImageKey, archiveMetaKey
}
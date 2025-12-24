package internal

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestImageTagger_Tag_Success(t *testing.T) {
	mockS3 := new(MockS3Client)

	// Mock image exists check
	mockS3.On("Exists", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".tar.gz") && strings.HasPrefix(key, "images/")
	})).Return(true, nil)

	// Mock tag upload
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "tags/") && strings.HasSuffix(key, ".json")
	}), mock.Anything).Return(nil)

	// Mock audit log upload
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "audit/") && strings.Contains(key, "tag")
	}), mock.Anything).Return(nil)

	tagger := NewImageTagger(mockS3, "test-bucket")

	_, err := tagger.Tag(context.Background(), "myapp:20250721-1430-abc1234", "v1.2.0")

	assert.NoError(t, err)
	mockS3.AssertExpectations(t)
}

func TestImageTagger_Tag_ImageNotFound(t *testing.T) {
	mockS3 := new(MockS3Client)

	// Mock image doesn't exist
	mockS3.On("Exists", mock.Anything, "test-bucket", mock.AnythingOfType("string")).Return(false, nil)

	tagger := NewImageTagger(mockS3, "test-bucket")

	_, err := tagger.Tag(context.Background(), "myapp:20250721-1430-abc1234", "v1.2.0")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image not found in S3")
	mockS3.AssertExpectations(t)
}

func TestImageTagger_Tag_InvalidImageReference(t *testing.T) {
	mockS3 := new(MockS3Client)
	tagger := NewImageTagger(mockS3, "test-bucket")

	_, err := tagger.Tag(context.Background(), "invalid-format", "v1.2.0")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse image reference")
}

func TestImagePromoter_Promote_DirectImage_Success(t *testing.T) {
	mockS3 := new(MockS3Client)

	// Mock image exists check
	mockS3.On("Exists", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasSuffix(key, ".tar.gz") && strings.HasPrefix(key, "images/")
	})).Return(true, nil)

	// Mock checking for existing pointer (for audit trail)
	mockS3.On("Exists", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "pointers/")
	})).Return(false, nil)

	// Mock environment pointer upload
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "pointers/") && strings.HasSuffix(key, ".json")
	}), mock.Anything).Return(nil)

	// Mock audit log upload
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "audit/") && strings.Contains(key, "promotion")
	}), mock.Anything).Return(nil)

	promoter := NewImagePromoter(mockS3, "test-bucket")

	_, err := promoter.Promote(context.Background(), "myapp:20250721-1430-abc1234", "production")

	assert.NoError(t, err)
	mockS3.AssertExpectations(t)
}

func TestImagePromoter_Promote_DirectImage_ImageNotFound(t *testing.T) {
	mockS3 := new(MockS3Client)

	// Mock image doesn't exist
	mockS3.On("Exists", mock.Anything, "test-bucket", mock.AnythingOfType("string")).Return(false, nil)

	promoter := NewImagePromoter(mockS3, "test-bucket")

	_, err := promoter.Promote(context.Background(), "myapp:20250721-1430-abc1234", "production")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image not found in S3")
	mockS3.AssertExpectations(t)
}

func TestImagePromoter_PromoteFromTag_Success(t *testing.T) {
	mockS3 := new(MockS3Client)

	// Mock tag exists check
	mockS3.On("Exists", mock.Anything, "test-bucket", "tags/myapp/v1.2.0.json").Return(true, nil)

	// Mock tag download
	tagPointer := &PointerMetadata{
		TargetType:  TargetTypeImage,
		TargetPath:  "images/myapp/202507/myapp-20250721-1430-abc1234.tar.gz",
		GitHash:     "abc1234",
		GitTime:     "20250721-1430",
		SourceImage: "myapp:20250721-1430-abc1234",
	}
	tagJSON, _ := tagPointer.ToJSON()
	mockS3.On("Download", mock.Anything, "test-bucket", "tags/myapp/v1.2.0.json").Return(tagJSON, nil)

	// Mock checking for existing pointer (for audit trail)
	mockS3.On("Exists", mock.Anything, "test-bucket", "pointers/myapp/staging.json").Return(false, nil)

	// Mock environment pointer upload
	mockS3.On("Upload", mock.Anything, "test-bucket", "pointers/myapp/staging.json", mock.Anything).Return(nil)

	// Mock audit log upload
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "audit/") && strings.Contains(key, "promotion")
	}), mock.Anything).Return(nil)

	promoter := NewImagePromoter(mockS3, "test-bucket")

	_, err := promoter.PromoteFromTag(context.Background(), "myapp", "v1.2.0", "staging")

	assert.NoError(t, err)
	mockS3.AssertExpectations(t)
}

func TestImagePromoter_PromoteFromTag_TagNotFound(t *testing.T) {
	mockS3 := new(MockS3Client)

	// Mock tag doesn't exist
	mockS3.On("Exists", mock.Anything, "test-bucket", "tags/myapp/v1.2.0.json").Return(false, nil)

	promoter := NewImagePromoter(mockS3, "test-bucket")

	_, err := promoter.PromoteFromTag(context.Background(), "myapp", "v1.2.0", "staging")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag not found")
	mockS3.AssertExpectations(t)
}

func TestImagePromoter_PromoteFromTag_DownloadError(t *testing.T) {
	mockS3 := new(MockS3Client)

	// Mock tag exists but download fails
	mockS3.On("Exists", mock.Anything, "test-bucket", "tags/myapp/v1.2.0.json").Return(true, nil)
	mockS3.On("Download", mock.Anything, "test-bucket", "tags/myapp/v1.2.0.json").Return([]byte{}, errors.New("download error"))

	promoter := NewImagePromoter(mockS3, "test-bucket")

	_, err := promoter.PromoteFromTag(context.Background(), "myapp", "v1.2.0", "staging")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download tag")
	mockS3.AssertExpectations(t)
}

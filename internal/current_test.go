package internal

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewCurrentService(t *testing.T) {
	mockS3 := &MockS3Client{}
	bucket := "test-bucket"

	service := NewCurrentService(mockS3, bucket)

	assert.NotNil(t, service)
	assert.Equal(t, mockS3, service.s3)
	assert.Equal(t, bucket, service.bucket)
}

func TestGetCurrentImage_Success_ImagePointer(t *testing.T) {
	mockS3 := &MockS3Client{}
	bucket := "test-bucket"
	service := NewCurrentService(mockS3, bucket)

	appName := "myapp"
	environment := "production"
	envKey := GeneratePointerKey(appName, environment)

	// Create a pointer that points directly to an image
	pointer := &PointerMetadata{
		TargetType: TargetTypeImage,
		TargetPath: "images/myapp/202507/myapp-20250721-1430-abc1234.tar.gz",
		PromotedAt: time.Now(),
		PromotedBy: "testuser",
		GitHash:    "abc1234",
		GitTime:    "20250721-1430",
	}

	pointerData, _ := json.Marshal(pointer)

	// Mock S3 calls
	mockS3.On("Exists", mock.Anything, bucket, envKey).Return(true, nil)
	mockS3.On("Download", mock.Anything, bucket, envKey).Return(pointerData, nil)

	ctx := context.Background()
	imageRef, err := service.GetCurrentImage(ctx, appName, environment)

	assert.NoError(t, err)
	assert.Equal(t, "myapp:20250721-1430-abc1234", imageRef)
	mockS3.AssertExpectations(t)
}

func TestGetCurrentImage_Success_TagPointer(t *testing.T) {
	mockS3 := &MockS3Client{}
	bucket := "test-bucket"
	service := NewCurrentService(mockS3, bucket)

	appName := "myapp"
	environment := "production"
	envKey := GeneratePointerKey(appName, environment)
	tagKey := "tags/myapp/v1.2.0.json"

	// Create environment pointer that points to a tag
	envPointer := &PointerMetadata{
		TargetType: TargetTypeTag,
		TargetPath: tagKey,
		PromotedAt: time.Now(),
		PromotedBy: "testuser",
		GitHash:    "abc1234",
		GitTime:    "20250721-1430",
	}

	// Create tag pointer that points to an image
	tagPointer := &PointerMetadata{
		TargetType: TargetTypeImage,
		TargetPath: "images/myapp/202507/myapp-20250721-1430-abc1234.tar.gz",
		PromotedAt: time.Now(),
		PromotedBy: "testuser",
		GitHash:    "abc1234",
		GitTime:    "20250721-1430",
	}

	envPointerData, _ := json.Marshal(envPointer)
	tagPointerData, _ := json.Marshal(tagPointer)

	// Mock S3 calls
	mockS3.On("Exists", mock.Anything, bucket, envKey).Return(true, nil)
	mockS3.On("Download", mock.Anything, bucket, envKey).Return(envPointerData, nil)
	mockS3.On("Download", mock.Anything, bucket, tagKey).Return(tagPointerData, nil)

	ctx := context.Background()
	imageRef, err := service.GetCurrentImage(ctx, appName, environment)

	assert.NoError(t, err)
	assert.Equal(t, "myapp:20250721-1430-abc1234", imageRef)
	mockS3.AssertExpectations(t)
}

func TestGetCurrentImage_EnvironmentNotFound(t *testing.T) {
	mockS3 := &MockS3Client{}
	bucket := "test-bucket"
	service := NewCurrentService(mockS3, bucket)

	appName := "myapp"
	environment := "production"
	envKey := GeneratePointerKey(appName, environment)

	// Mock S3 call to return false for exists
	mockS3.On("Exists", mock.Anything, bucket, envKey).Return(false, nil)

	ctx := context.Background()
	imageRef, err := service.GetCurrentImage(ctx, appName, environment)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "environment pointer not found")
	assert.Empty(t, imageRef)
	mockS3.AssertExpectations(t)
}

func TestGetCurrentImage_S3ExistsError(t *testing.T) {
	mockS3 := &MockS3Client{}
	bucket := "test-bucket"
	service := NewCurrentService(mockS3, bucket)

	appName := "myapp"
	environment := "production"
	envKey := GeneratePointerKey(appName, environment)

	// Mock S3 call to return error
	mockS3.On("Exists", mock.Anything, bucket, envKey).Return(false, errors.New("S3 error"))

	ctx := context.Background()
	imageRef, err := service.GetCurrentImage(ctx, appName, environment)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check environment pointer existence")
	assert.Empty(t, imageRef)
	mockS3.AssertExpectations(t)
}

func TestGetCurrentImage_DownloadError(t *testing.T) {
	mockS3 := &MockS3Client{}
	bucket := "test-bucket"
	service := NewCurrentService(mockS3, bucket)

	appName := "myapp"
	environment := "production"
	envKey := GeneratePointerKey(appName, environment)

	// Mock S3 calls
	mockS3.On("Exists", mock.Anything, bucket, envKey).Return(true, nil)
	mockS3.On("Download", mock.Anything, bucket, envKey).Return([]byte{}, errors.New("download error"))

	ctx := context.Background()
	imageRef, err := service.GetCurrentImage(ctx, appName, environment)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download environment pointer")
	assert.Empty(t, imageRef)
	mockS3.AssertExpectations(t)
}

func TestGetCurrentImage_InvalidPointerData(t *testing.T) {
	mockS3 := &MockS3Client{}
	bucket := "test-bucket"
	service := NewCurrentService(mockS3, bucket)

	appName := "myapp"
	environment := "production"
	envKey := GeneratePointerKey(appName, environment)

	// Mock S3 calls with invalid JSON
	mockS3.On("Exists", mock.Anything, bucket, envKey).Return(true, nil)
	mockS3.On("Download", mock.Anything, bucket, envKey).Return([]byte("invalid json"), nil)

	ctx := context.Background()
	imageRef, err := service.GetCurrentImage(ctx, appName, environment)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse environment pointer")
	assert.Empty(t, imageRef)
	mockS3.AssertExpectations(t)
}

func TestExtractImageReferenceFromPath_Success(t *testing.T) {
	mockS3 := &MockS3Client{}
	service := NewCurrentService(mockS3, "test-bucket")

	tests := []struct {
		s3Path   string
		expected string
		name     string
	}{
		{
			s3Path:   "images/myapp/202507/myapp-20250721-1430-abc1234.tar.gz",
			expected: "myapp:20250721-1430-abc1234",
			name:     "standard format",
		},
		{
			s3Path:   "images/api/202506/api-20250615-0930-def5678.tar.gz",
			expected: "api:20250615-0930-def5678",
			name:     "different app and date",
		},
		{
			s3Path:   "images/frontend/202508/frontend-20250801-2359-1234567.tar.gz",
			expected: "frontend:20250801-2359-1234567",
			name:     "different timestamp format",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := service.extractImageReferenceFromPath(test.s3Path)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestExtractImageReferenceFromPath_InvalidFormat(t *testing.T) {
	mockS3 := &MockS3Client{}
	service := NewCurrentService(mockS3, "test-bucket")

	tests := []struct {
		s3Path   string
		expected string
		name     string
	}{
		{
			s3Path:   "images/myapp/202507/myapp-20250721-abc1234.tar.gz",
			expected: "invalid timestamp-hash format",
			name:     "missing timestamp dash",
		},
		{
			s3Path:   "images/myapp/202507/myapp-20250721-1430.tar.gz",
			expected: "invalid timestamp-hash format",
			name:     "missing hash",
		},
		{
			s3Path:   "images/myapp/202507/myapp-20250721-1430-abc.tar.gz",
			expected: "invalid hash format",
			name:     "hash too short",
		},
		{
			s3Path:   "images/myapp/202507/myapp-202507211430-abc1234.tar.gz",
			expected: "invalid timestamp-hash format",
			name:     "timestamp missing dash",
		},
		{
			s3Path:   "images/myapp/202507/myapp.tar.gz",
			expected: "invalid image filename format",
			name:     "missing timestamp and hash",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := service.extractImageReferenceFromPath(test.s3Path)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.expected)
			assert.Empty(t, result)
		})
	}
}

func TestExtractImageReferenceFromPath_EdgeCases(t *testing.T) {
	mockS3 := &MockS3Client{}
	service := NewCurrentService(mockS3, "test-bucket")

	tests := []struct {
		s3Path   string
		expected string
		name     string
	}{
		{
			s3Path:   "images/myapp/202507/myapp-20250721-1430-abc1234",
			expected: "invalid image path format: must end with .tar.gz",
			name:     "missing .tar.gz extension",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := service.extractImageReferenceFromPath(test.s3Path)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.expected)
			assert.Empty(t, result)
		})
	}
}

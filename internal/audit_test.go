package internal

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreatePushEvent(t *testing.T) {
	appName := "myapp"
	gitHash := "abc1234"
	gitTime := "20250721-1430"
	imageRef := "myapp:20250721-1430-abc1234"
	s3Path := "images/myapp/202507/myapp-20250721-1430-abc1234.tar.gz"
	checksum := "d4f3c2b1a5e6f7g8h9"
	size := int64(1024)

	event, err := CreatePushEvent(appName, gitHash, gitTime, imageRef, s3Path, checksum, size, false, false)

	assert.NoError(t, err)
	assert.Equal(t, EventTypePush, event.EventType)
	assert.Equal(t, appName, event.AppName)
	assert.Equal(t, gitHash, event.GitHash)
	assert.Equal(t, gitTime, event.GitTime)
	assert.True(t, event.Timestamp.Before(time.Now().Add(time.Second)))

	details, ok := event.Details.(PushEventDetails)
	assert.True(t, ok)
	assert.Equal(t, imageRef, details.ImageReference)
	assert.Equal(t, s3Path, details.S3Path)
	assert.Equal(t, checksum, details.Checksum)
	assert.Equal(t, size, details.Size)
	assert.False(t, details.WasSkipped)
	assert.False(t, details.WasArchived)
}

func TestCreateTagEvent(t *testing.T) {
	appName := "myapp"
	gitHash := "abc1234"
	gitTime := "20250721-1430"
	imageRef := "myapp:20250721-1430-abc1234"
	version := "v1.2.0"
	tagPath := "tags/myapp/v1.2.0.json"

	event, err := CreateTagEvent(appName, gitHash, gitTime, imageRef, version, tagPath)

	assert.NoError(t, err)
	assert.Equal(t, EventTypeTag, event.EventType)
	assert.Equal(t, appName, event.AppName)
	assert.Equal(t, gitHash, event.GitHash)
	assert.Equal(t, gitTime, event.GitTime)

	details, ok := event.Details.(TagEventDetails)
	assert.True(t, ok)
	assert.Equal(t, imageRef, details.ImageReference)
	assert.Equal(t, version, details.Version)
	assert.Equal(t, tagPath, details.TagPath)
}

func TestCreatePromotionEvent(t *testing.T) {
	appName := "myapp"
	gitHash := "abc1234"
	gitTime := "20250721-1430"
	environment := "production"
	source := "myapp:20250721-1430-abc1234"
	sourceType := "image"
	pointerPath := "pointers/myapp/production.json"
	previousTarget := "images/myapp/202507/myapp-20250720-1045-def5678.tar.gz"

	event, err := CreatePromotionEvent(appName, gitHash, gitTime, environment, source, sourceType, pointerPath, previousTarget)

	assert.NoError(t, err)
	assert.Equal(t, EventTypePromotion, event.EventType)
	assert.Equal(t, appName, event.AppName)
	assert.Equal(t, gitHash, event.GitHash)
	assert.Equal(t, gitTime, event.GitTime)

	details, ok := event.Details.(PromotionEventDetails)
	assert.True(t, ok)
	assert.Equal(t, environment, details.Environment)
	assert.Equal(t, source, details.Source)
	assert.Equal(t, sourceType, details.SourceType)
	assert.Equal(t, pointerPath, details.PointerPath)
	assert.Equal(t, previousTarget, details.PreviousTarget)
}

func TestGenerateAuditKey(t *testing.T) {
	appName := "myapp"
	timestamp := time.Date(2025, 7, 21, 14, 30, 0, 0, time.UTC)
	eventType := EventTypePush
	gitHash := "abc1234"

	key := GenerateAuditKey(appName, timestamp, eventType, gitHash)

	expected := "audit/myapp/202507/20250721-1430-push-abc1234.json"
	assert.Equal(t, expected, key)
}

func TestAuditEventJSON(t *testing.T) {
	event := &AuditEvent{
		EventType: EventTypePush,
		Timestamp: time.Date(2025, 7, 21, 14, 30, 0, 0, time.UTC),
		User:      "testuser",
		AppName:   "myapp",
		GitHash:   "abc1234",
		GitTime:   "20250721-1430",
		Details: PushEventDetails{
			ImageReference: "myapp:20250721-1430-abc1234",
			S3Path:         "images/myapp/202507/myapp-20250721-1430-abc1234.tar.gz",
			Checksum:       "d4f3c2b1a5e6f7g8h9",
			Size:           1024,
		},
	}

	jsonData, err := event.ToJSON()
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "\"event_type\": \"push\"")
	assert.Contains(t, string(jsonData), "\"app_name\": \"myapp\"")
	assert.Contains(t, string(jsonData), "\"git_hash\": \"abc1234\"")

	parsed, err := AuditEventFromJSON(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, event.EventType, parsed.EventType)
	assert.Equal(t, event.AppName, parsed.AppName)
	assert.Equal(t, event.GitHash, parsed.GitHash)
}

func TestS3AuditLogger_LogEvent(t *testing.T) {
	mockS3 := new(MockS3Client)

	// Mock audit log upload
	mockS3.On("Upload", mock.Anything, "test-bucket", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, "audit/") && strings.Contains(key, "push")
	}), mock.Anything).Return(nil)

	logger := NewS3AuditLogger(mockS3, "test-bucket")

	event := &AuditEvent{
		EventType: EventTypePush,
		Timestamp: time.Now(),
		User:      "testuser",
		AppName:   "myapp",
		GitHash:   "abc1234",
		GitTime:   "20250721-1430",
		Details: PushEventDetails{
			ImageReference: "myapp:20250721-1430-abc1234",
			S3Path:         "images/myapp/202507/myapp-20250721-1430-abc1234.tar.gz",
		},
	}

	err := logger.LogEvent(context.Background(), event)

	assert.NoError(t, err)
	mockS3.AssertExpectations(t)
}

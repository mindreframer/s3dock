package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type EventType string

const (
	EventTypePush      EventType = "push"
	EventTypeTag       EventType = "tag"
	EventTypePromotion EventType = "promotion"
)

type AuditEvent struct {
	EventType EventType   `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	User      string      `json:"user"`
	AppName   string      `json:"app_name"`
	GitHash   string      `json:"git_hash"`
	GitTime   string      `json:"git_time"`
	Details   interface{} `json:"details"`
}

type PushEventDetails struct {
	ImageReference string `json:"image_reference"`
	S3Path         string `json:"s3_path"`
	Checksum       string `json:"checksum"`
	Size           int64  `json:"size"`
	WasSkipped     bool   `json:"was_skipped,omitempty"`
	WasArchived    bool   `json:"was_archived,omitempty"`
}

type TagEventDetails struct {
	ImageReference string `json:"image_reference"`
	Version        string `json:"version"`
	TagPath        string `json:"tag_path"`
}

type PromotionEventDetails struct {
	Environment    string `json:"environment"`
	Source         string `json:"source"`
	SourceType     string `json:"source_type"` // "image" or "tag"
	PointerPath    string `json:"pointer_path"`
	PreviousTarget string `json:"previous_target,omitempty"`
}

func (a *AuditEvent) ToJSON() ([]byte, error) {
	return json.MarshalIndent(a, "", "  ")
}

func AuditEventFromJSON(data []byte) (*AuditEvent, error) {
	var event AuditEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

func GenerateAuditKey(appName string, timestamp time.Time, eventType EventType, gitHash string) string {
	yearMonth := timestamp.Format("200601")
	timeStr := timestamp.Format("20060102-1504")
	return fmt.Sprintf("audit/%s/%s/%s-%s-%s.json", appName, yearMonth, timeStr, eventType, gitHash)
}

func CreatePushEvent(appName, gitHash, gitTime, imageRef, s3Path, checksum string, size int64, wasSkipped, wasArchived bool) (*AuditEvent, error) {
	user, err := getCurrentUser()
	if err != nil {
		user = "unknown"
	}

	details := PushEventDetails{
		ImageReference: imageRef,
		S3Path:         s3Path,
		Checksum:       checksum,
		Size:           size,
		WasSkipped:     wasSkipped,
		WasArchived:    wasArchived,
	}

	return &AuditEvent{
		EventType: EventTypePush,
		Timestamp: time.Now(),
		User:      user,
		AppName:   appName,
		GitHash:   gitHash,
		GitTime:   gitTime,
		Details:   details,
	}, nil
}

func CreateTagEvent(appName, gitHash, gitTime, imageRef, version, tagPath string) (*AuditEvent, error) {
	user, err := getCurrentUser()
	if err != nil {
		user = "unknown"
	}

	details := TagEventDetails{
		ImageReference: imageRef,
		Version:        version,
		TagPath:        tagPath,
	}

	return &AuditEvent{
		EventType: EventTypeTag,
		Timestamp: time.Now(),
		User:      user,
		AppName:   appName,
		GitHash:   gitHash,
		GitTime:   gitTime,
		Details:   details,
	}, nil
}

func CreatePromotionEvent(appName, gitHash, gitTime, environment, source, sourceType, pointerPath, previousTarget string) (*AuditEvent, error) {
	user, err := getCurrentUser()
	if err != nil {
		user = "unknown"
	}

	details := PromotionEventDetails{
		Environment:    environment,
		Source:         source,
		SourceType:     sourceType,
		PointerPath:    pointerPath,
		PreviousTarget: previousTarget,
	}

	return &AuditEvent{
		EventType: EventTypePromotion,
		Timestamp: time.Now(),
		User:      user,
		AppName:   appName,
		GitHash:   gitHash,
		GitTime:   gitTime,
		Details:   details,
	}, nil
}

type AuditLogger interface {
	LogEvent(ctx context.Context, event *AuditEvent) error
}

type S3AuditLogger struct {
	s3     S3Client
	bucket string
}

func NewS3AuditLogger(s3Client S3Client, bucket string) *S3AuditLogger {
	return &S3AuditLogger{
		s3:     s3Client,
		bucket: bucket,
	}
}

func (a *S3AuditLogger) LogEvent(ctx context.Context, event *AuditEvent) error {
	auditKey := GenerateAuditKey(event.AppName, event.Timestamp, event.EventType, event.GitHash)

	eventJSON, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize audit event: %w", err)
	}

	if err := a.s3.Upload(ctx, a.bucket, auditKey, strings.NewReader(string(eventJSON))); err != nil {
		return fmt.Errorf("failed to upload audit event to S3: %w", err)
	}

	return nil
}

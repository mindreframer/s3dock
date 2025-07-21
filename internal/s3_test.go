package internal

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestS3Client_NewS3Client(t *testing.T) {
	client, err := NewS3Client(context.Background())

	if err != nil {
		t.Skip("AWS credentials not available - skipping test")
		return
	}

	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
}

func TestS3Client_Upload(t *testing.T) {
	client, err := NewS3Client(context.Background())
	if err != nil {
		t.Skip("AWS credentials not available - skipping test")
		return
	}

	err = client.Upload(context.Background(), "nonexistent-bucket", "test-key", strings.NewReader("test data"))
	assert.Error(t, err)
}

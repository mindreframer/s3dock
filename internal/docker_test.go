package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDockerClient_NewDockerClient(t *testing.T) {
	client, err := NewDockerClient()
	
	if err != nil {
		t.Skip("Docker daemon not available - skipping test")
		return
	}
	
	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	
	defer client.Close()
}

func TestDockerClient_ExportImage(t *testing.T) {
	client, err := NewDockerClient()
	if err != nil {
		t.Skip("Docker daemon not available - skipping test")
		return
	}
	defer client.Close()

	_, err = client.ExportImage(context.Background(), "nonexistent:image")
	assert.Error(t, err)
}
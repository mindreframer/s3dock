package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitClient_GetCurrentHash(t *testing.T) {
	client := NewGitClient()

	hash, err := client.GetCurrentHash()

	if err != nil {
		t.Skip("Git repository not found - skipping test")
		return
	}

	assert.Len(t, hash, 7)
	assert.Regexp(t, "^[a-f0-9]{7}$", hash)
}

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_DefaultConfig(t *testing.T) {
	config, err := LoadConfig("")

	assert.NoError(t, err)
	assert.Equal(t, "default", config.DefaultProfile)
	assert.Contains(t, config.Profiles, "default")
	assert.Equal(t, "s3dock-containers", config.Profiles["default"].Bucket)
}

func TestLoadConfig_ExplicitPath(t *testing.T) {
	testConfigContent := `{
		"default_profile": "test",
		"profiles": {
			"test": {
				"bucket": "test-bucket",
				"region": "us-west-2"
			}
		}
	}`

	tmpFile := filepath.Join(t.TempDir(), "test-config.json5")
	err := os.WriteFile(tmpFile, []byte(testConfigContent), 0644)
	assert.NoError(t, err)

	config, err := LoadConfig(tmpFile)

	assert.NoError(t, err)
	assert.Equal(t, "test", config.DefaultProfile)
	assert.Equal(t, "test-bucket", config.Profiles["test"].Bucket)
	assert.Equal(t, "us-west-2", config.Profiles["test"].Region)
}

func TestLoadConfig_WithComments(t *testing.T) {
	testConfigContent := `{
		// This is the default profile
		"default_profile": "dev",
		"profiles": {
			"dev": {
				"bucket": "dev-bucket", // Development bucket
				"region": "us-east-1",
				/* 
				 * Local MinIO endpoint
				 */
				"endpoint": "http://localhost:9000"
			}
		}
	}`

	tmpFile := filepath.Join(t.TempDir(), "commented-config.json5")
	err := os.WriteFile(tmpFile, []byte(testConfigContent), 0644)
	assert.NoError(t, err)

	config, err := LoadConfig(tmpFile)

	assert.NoError(t, err)
	assert.Equal(t, "dev", config.DefaultProfile)
	assert.Equal(t, "dev-bucket", config.Profiles["dev"].Bucket)
	assert.Equal(t, "http://localhost:9000", config.Profiles["dev"].Endpoint)
}

func TestResolveConfig_DefaultProfile(t *testing.T) {
	testConfigContent := `{
		"default_profile": "staging",
		"profiles": {
			"staging": {
				"bucket": "staging-bucket",
				"region": "us-west-2"
			}
		}
	}`

	tmpFile := filepath.Join(t.TempDir(), "resolve-config.json5")
	err := os.WriteFile(tmpFile, []byte(testConfigContent), 0644)
	assert.NoError(t, err)

	resolved, err := ResolveConfig(tmpFile, "", "")

	assert.NoError(t, err)
	assert.Equal(t, "staging-bucket", resolved.Bucket)
	assert.Equal(t, "us-west-2", resolved.Region)
}

func TestResolveConfig_ProfileOverride(t *testing.T) {
	testConfigContent := `{
		"default_profile": "default",
		"profiles": {
			"default": {
				"bucket": "default-bucket",
				"region": "us-east-1"
			},
			"prod": {
				"bucket": "prod-bucket", 
				"region": "us-west-2"
			}
		}
	}`

	tmpFile := filepath.Join(t.TempDir(), "profile-override.json5")
	err := os.WriteFile(tmpFile, []byte(testConfigContent), 0644)
	assert.NoError(t, err)

	resolved, err := ResolveConfig(tmpFile, "prod", "")

	assert.NoError(t, err)
	assert.Equal(t, "prod-bucket", resolved.Bucket)
	assert.Equal(t, "us-west-2", resolved.Region)
}

func TestResolveConfig_BucketOverride(t *testing.T) {
	testConfigContent := `{
		"profiles": {
			"default": {
				"bucket": "config-bucket",
				"region": "us-east-1"
			}
		}
	}`

	tmpFile := filepath.Join(t.TempDir(), "bucket-override.json5")
	err := os.WriteFile(tmpFile, []byte(testConfigContent), 0644)
	assert.NoError(t, err)

	resolved, err := ResolveConfig(tmpFile, "default", "override-bucket")

	assert.NoError(t, err)
	assert.Equal(t, "override-bucket", resolved.Bucket)
}

func TestResolveConfig_EnvOverrides(t *testing.T) {
	testConfigContent := `{
		"profiles": {
			"default": {
				"bucket": "config-bucket",
				"region": "us-east-1"
			}
		}
	}`

	tmpFile := filepath.Join(t.TempDir(), "env-override.json5")
	err := os.WriteFile(tmpFile, []byte(testConfigContent), 0644)
	assert.NoError(t, err)

	os.Setenv("S3DOCK_BUCKET", "env-bucket")
	os.Setenv("AWS_REGION", "eu-west-1")
	defer func() {
		os.Unsetenv("S3DOCK_BUCKET")
		os.Unsetenv("AWS_REGION")
	}()

	resolved, err := ResolveConfig(tmpFile, "default", "")

	assert.NoError(t, err)
	assert.Equal(t, "env-bucket", resolved.Bucket)
	assert.Equal(t, "eu-west-1", resolved.Region)
}

func TestConfig_GetProfileNames(t *testing.T) {
	config := &Config{
		Profiles: map[string]Profile{
			"dev":     {Bucket: "dev-bucket"},
			"staging": {Bucket: "staging-bucket"},
			"prod":    {Bucket: "prod-bucket"},
		},
	}

	names := config.GetProfileNames()

	assert.Len(t, names, 3)
	assert.Contains(t, names, "dev")
	assert.Contains(t, names, "staging")
	assert.Contains(t, names, "prod")
}

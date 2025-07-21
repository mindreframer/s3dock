package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adhocore/jsonc"
)

type Config struct {
	DefaultProfile string             `json:"default_profile"`
	Profiles       map[string]Profile `json:"profiles"`
	Docker         DockerConfig       `json:"docker"`
	Naming         NamingConfig       `json:"naming"`
	Defaults       DefaultsConfig     `json:"defaults"`
}

type Profile struct {
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

type DockerConfig struct {
	Timeout     string `json:"timeout"`
	Compression string `json:"compression"`
}

type NamingConfig struct {
	IncludeGitBranch bool   `json:"include_git_branch"`
	TimestampFormat  string `json:"timestamp_format"`
	PathTemplate     string `json:"path_template"`
}

type DefaultsConfig struct {
	RetryCount int    `json:"retry_count"`
	LogLevel   string `json:"log_level"`
}

type ResolvedConfig struct {
	Bucket    string
	Region    string
	Endpoint  string
	AccessKey string
	SecretKey string

	DockerTimeout     string
	DockerCompression string

	IncludeGitBranch bool
	TimestampFormat  string
	PathTemplate     string

	RetryCount int
	LogLevel   string
}

func LoadConfig(configPath string) (*Config, error) {
	var actualPath string
	var err error

	if configPath != "" {
		actualPath = configPath
	} else {
		actualPath, err = findConfigFile()
		if err != nil {
			return getDefaultConfig(), nil
		}
	}

	data, err := os.ReadFile(actualPath)
	if err != nil {
		if configPath != "" {
			return nil, fmt.Errorf("failed to read config file %s: %w", actualPath, err)
		}
		return getDefaultConfig(), nil
	}

	j := jsonc.New()
	var config Config
	if err := j.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", actualPath, err)
	}

	return &config, nil
}

func findConfigFile() (string, error) {
	homeDir, _ := os.UserHomeDir()

	candidates := []string{
		"./s3dock.json5",
		filepath.Join(homeDir, ".s3dock", "config.json5"),
		"/etc/s3dock/config.json5",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no config file found")
}

func getDefaultConfig() *Config {
	return &Config{
		DefaultProfile: "default",
		Profiles: map[string]Profile{
			"default": {
				Bucket: "s3dock-containers",
				Region: "us-east-1",
			},
		},
		Docker: DockerConfig{
			Timeout:     "30s",
			Compression: "gzip",
		},
		Naming: NamingConfig{
			IncludeGitBranch: false,
			TimestampFormat:  "20060102-1504",
			PathTemplate:     "images/{app}/{year_month}/{filename}",
		},
		Defaults: DefaultsConfig{
			RetryCount: 3,
			LogLevel:   "info",
		},
	}
}

func ResolveConfig(configPath, profileName, bucketOverride string) (*ResolvedConfig, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	profile := profileName
	if profile == "" {
		profile = config.DefaultProfile
	}

	if profile == "" {
		profile = "default"
	}

	profileConfig, exists := config.Profiles[profile]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found in config", profile)
	}

	resolved := &ResolvedConfig{
		Bucket:            resolveBucket(bucketOverride, profileConfig.Bucket),
		Region:            resolveRegion(profileConfig.Region),
		Endpoint:          resolveEndpoint(profileConfig.Endpoint),
		AccessKey:         resolveAccessKey(profileConfig.AccessKey),
		SecretKey:         resolveSecretKey(profileConfig.SecretKey),
		DockerTimeout:     config.Docker.Timeout,
		DockerCompression: config.Docker.Compression,
		IncludeGitBranch:  config.Naming.IncludeGitBranch,
		TimestampFormat:   config.Naming.TimestampFormat,
		PathTemplate:      config.Naming.PathTemplate,
		RetryCount:        config.Defaults.RetryCount,
		LogLevel:          config.Defaults.LogLevel,
	}

	return resolved, nil
}

func resolveBucket(override, configValue string) string {
	if override != "" {
		return override
	}
	if env := os.Getenv("S3DOCK_BUCKET"); env != "" {
		return env
	}
	if configValue != "" {
		return configValue
	}
	return "s3dock-containers"
}

func resolveRegion(configValue string) string {
	if env := os.Getenv("AWS_REGION"); env != "" {
		return env
	}
	if configValue != "" {
		return configValue
	}
	return "us-east-1"
}

func resolveEndpoint(configValue string) string {
	if env := os.Getenv("AWS_ENDPOINT_URL"); env != "" {
		return env
	}
	return configValue
}

func resolveAccessKey(configValue string) string {
	if env := os.Getenv("AWS_ACCESS_KEY_ID"); env != "" {
		return env
	}
	return configValue
}

func resolveSecretKey(configValue string) string {
	if env := os.Getenv("AWS_SECRET_ACCESS_KEY"); env != "" {
		return env
	}
	return configValue
}

func (c *Config) GetProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}

func (c *Config) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Default Profile: %s\n", c.DefaultProfile))
	sb.WriteString("Profiles:\n")
	for name, profile := range c.Profiles {
		sb.WriteString(fmt.Sprintf("  %s:\n", name))
		sb.WriteString(fmt.Sprintf("    Bucket: %s\n", profile.Bucket))
		sb.WriteString(fmt.Sprintf("    Region: %s\n", profile.Region))
		if profile.Endpoint != "" {
			sb.WriteString(fmt.Sprintf("    Endpoint: %s\n", profile.Endpoint))
		}
	}
	return sb.String()
}

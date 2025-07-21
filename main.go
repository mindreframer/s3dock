package main

import (
	"context"
	"fmt"
	"os"

	"s3dock/internal"
)

type GlobalFlags struct {
	Config  string
	Profile string
	Bucket  string
	Help    bool
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	globalFlags, remaining := parseGlobalFlags(os.Args[1:])

	if globalFlags.Help || len(remaining) == 0 {
		printUsage()
		return
	}

	command := remaining[0]
	commandArgs := remaining[1:]

	switch command {
	case "push":
		handlePushCommand(globalFlags, commandArgs)
	case "config":
		handleConfigCommand(globalFlags, commandArgs)
	case "tag":
		fmt.Println("Tag functionality not yet implemented")
	case "pull":
		fmt.Println("Pull functionality not yet implemented")
	case "list":
		fmt.Println("List functionality not yet implemented")
	case "cleanup":
		fmt.Println("Cleanup functionality not yet implemented")
	case "deploy":
		fmt.Println("Deploy functionality not yet implemented")
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: s3dock [global-flags] <command> [command-flags]")
	fmt.Println("")
	fmt.Println("Global Flags:")
	fmt.Println("  --config <path>   Explicit config file path")
	fmt.Println("  --profile <name>  Profile to use from config")
	fmt.Println("  --bucket <name>   Override bucket name")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  push <image:tag>  Push Docker image to S3")
	fmt.Println("  config            Config file management")
	fmt.Println("  tag               Tag functionality (not implemented)")
	fmt.Println("  pull              Pull functionality (not implemented)")
	fmt.Println("  list              List functionality (not implemented)")
	fmt.Println("  cleanup           Cleanup functionality (not implemented)")
	fmt.Println("  deploy            Deploy functionality (not implemented)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  s3dock push myapp:latest")
	fmt.Println("  s3dock --profile dev push myapp:latest")
	fmt.Println("  s3dock --config ./test.json5 push myapp:latest")
	fmt.Println("  s3dock config show")
	fmt.Println("  s3dock config list")
}

func parseGlobalFlags(args []string) (*GlobalFlags, []string) {
	flags := &GlobalFlags{}

	var remaining []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--config":
			if i+1 < len(args) {
				flags.Config = args[i+1]
				i++
			}
		case "--profile", "-p":
			if i+1 < len(args) {
				flags.Profile = args[i+1]
				i++
			}
		case "--bucket", "-b":
			if i+1 < len(args) {
				flags.Bucket = args[i+1]
				i++
			}
		case "--help", "-h":
			flags.Help = true
		default:
			remaining = append(remaining, arg)
		}
	}

	return flags, remaining
}

func handlePushCommand(globalFlags *GlobalFlags, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: s3dock [global-flags] push <image:tag>")
		fmt.Println("")
		fmt.Println("Push a Docker image to S3 storage.")
		fmt.Println("")
		fmt.Println("Global Flags:")
		fmt.Println("  --config <path>   Explicit config file path")
		fmt.Println("  --profile <name>  Profile to use from config")
		fmt.Println("  --bucket <name>   Override bucket name")
		return
	}

	imageRef := args[0]

	resolved, err := internal.ResolveConfig(globalFlags.Config, globalFlags.Profile, globalFlags.Bucket)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := pushImageWithConfig(imageRef, resolved); err != nil {
		fmt.Printf("Error pushing image: %v\n", err)
		os.Exit(1)
	}
}

func handleConfigCommand(globalFlags *GlobalFlags, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: s3dock config <subcommand>")
		fmt.Println("")
		fmt.Println("Config Subcommands:")
		fmt.Println("  show [--profile <name>]  Show current config or specific profile")
		fmt.Println("  list                     List all profiles")
		fmt.Println("  init                     Create default config file")
		return
	}

	subcommand := args[0]

	switch subcommand {
	case "show":
		handleConfigShow(globalFlags, args[1:])
	case "list":
		handleConfigList(globalFlags, args[1:])
	case "init":
		handleConfigInit(globalFlags, args[1:])
	default:
		fmt.Printf("Unknown config subcommand: %s\n", subcommand)
	}
}

func handleConfigShow(globalFlags *GlobalFlags, args []string) {
	localFlags, _ := parseGlobalFlags(args)

	configPath := globalFlags.Config
	if localFlags.Config != "" {
		configPath = localFlags.Config
	}

	profileName := globalFlags.Profile
	if localFlags.Profile != "" {
		profileName = localFlags.Profile
	}

	config, err := internal.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if profileName != "" {
		profile, exists := config.Profiles[profileName]
		if !exists {
			fmt.Printf("Profile '%s' not found\n", profileName)
			os.Exit(1)
		}
		fmt.Printf("Profile: %s\n", profileName)
		fmt.Printf("  Bucket: %s\n", profile.Bucket)
		fmt.Printf("  Region: %s\n", profile.Region)
		if profile.Endpoint != "" {
			fmt.Printf("  Endpoint: %s\n", profile.Endpoint)
		}
		if profile.AccessKey != "" {
			fmt.Printf("  Access Key: %s\n", profile.AccessKey)
		}
		return
	}

	fmt.Print(config.String())
}

func handleConfigList(globalFlags *GlobalFlags, args []string) {
	config, err := internal.LoadConfig(globalFlags.Config)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Available profiles:\n")
	for _, name := range config.GetProfileNames() {
		marker := " "
		if name == config.DefaultProfile {
			marker = "*"
		}
		fmt.Printf("%s %s\n", marker, name)
	}
}

func handleConfigInit(globalFlags *GlobalFlags, args []string) {
	configPath := "s3dock.json5"
	if len(args) > 0 {
		configPath = args[0]
	}

	defaultContent := `{
  // s3dock configuration file
  "default_profile": "default",
  
  "profiles": {
    "default": {
      "bucket": "s3dock-containers",
      "region": "us-east-1"
      // Add endpoint, access_key, secret_key as needed
    }
  },
  
  "docker": {
    "timeout": "30s",
    "compression": "gzip"
  },
  
  "naming": {
    "include_git_branch": false,
    "timestamp_format": "20060102-1504", 
    "path_template": "images/{app}/{year_month}/{filename}"
  },
  
  "defaults": {
    "retry_count": 3,
    "log_level": "info"
  }
}`

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config file %s already exists\n", configPath)
		os.Exit(1)
	}

	if err := os.WriteFile(configPath, []byte(defaultContent), 0644); err != nil {
		fmt.Printf("Error creating config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created config file: %s\n", configPath)
}

func pushImageWithConfig(imageRef string, config *internal.ResolvedConfig) error {
	ctx := context.Background()

	os.Setenv("AWS_REGION", config.Region)
	if config.Endpoint != "" {
		os.Setenv("AWS_ENDPOINT_URL", config.Endpoint)
	}
	if config.AccessKey != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", config.AccessKey)
	}
	if config.SecretKey != "" {
		os.Setenv("AWS_SECRET_ACCESS_KEY", config.SecretKey)
	}

	dockerClient, err := internal.NewDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	s3Client, err := internal.NewS3Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	gitClient := internal.NewGitClient()

	pusher := internal.NewImagePusher(dockerClient, s3Client, gitClient, config.Bucket)

	return pusher.Push(ctx, imageRef)
}

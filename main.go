package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"s3dock/internal"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type GlobalFlags struct {
	Config   string
	Profile  string
	Bucket   string
	LogLevel int
	Help     bool
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	globalFlags, remaining := parseGlobalFlags(os.Args[1:])

	// Set log level from global flags
	if globalFlags.LogLevel > 0 {
		internal.SetLogLevel(internal.LogLevel(globalFlags.LogLevel))
	}

	if globalFlags.Help || len(remaining) == 0 {
		printUsage()
		return
	}

	command := remaining[0]
	commandArgs := remaining[1:]

	switch command {
	case "build":
		handleBuildCommand(globalFlags, commandArgs)
	case "push":
		handlePushCommand(globalFlags, commandArgs)
	case "config":
		handleConfigCommand(globalFlags, commandArgs)
	case "tag":
		handleTagCommand(globalFlags, commandArgs)
	case "promote":
		handlePromoteCommand(globalFlags, commandArgs)
	case "pull":
		handlePullCommand(globalFlags, commandArgs)
	case "current":
		handleCurrentCommand(globalFlags, commandArgs)
	case "version", "--version", "-v":
		handleVersionCommand(commandArgs)
	case "list":
		internal.LogInfo("List functionality not yet implemented")
	case "cleanup":
		internal.LogInfo("Cleanup functionality not yet implemented")
	case "deploy":
		internal.LogInfo("Deploy functionality not yet implemented")
	case "help", "--help", "-h":
		printUsage()
	default:
		internal.LogError("Unknown command: %s", command)
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
	fmt.Println("  --log-level <n>   Log level (1=error, 2=info, 3=debug)")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  build <app-name>    Build Docker image with git-based tag")
	fmt.Println("  push <image:tag>    Push Docker image to S3")
	fmt.Println("  tag <image> <ver>   Create semantic version tag")
	fmt.Println("  promote <src> <env> Promote image/tag to environment")
	fmt.Println("  pull <app> <env>    Pull image from environment")
	fmt.Println("  current <app> <env> Show current image for environment")
	fmt.Println("  config              Config file management")
	fmt.Println("  version             Show version information")
	fmt.Println("  list              List functionality (not implemented)")
	fmt.Println("  cleanup           Cleanup functionality (not implemented)")
	fmt.Println("  deploy            Deploy functionality (not implemented)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  s3dock build myapp")
	fmt.Println("  s3dock build myapp --path /path/to/repo")
	fmt.Println("  s3dock build myapp --dockerfile Dockerfile.prod")
	fmt.Println("  s3dock push myapp:20250721-2118-f7a5a27")
	fmt.Println("  s3dock tag myapp:20250721-2118-f7a5a27 v1.2.0")
	fmt.Println("  s3dock promote myapp:20250721-2118-f7a5a27 production")
	fmt.Println("  s3dock promote myapp v1.2.0 staging")
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
		case "--log-level", "-l":
			if i+1 < len(args) {
				level := 0
				fmt.Sscanf(args[i+1], "%d", &level)
				if level >= 1 && level <= 3 {
					flags.LogLevel = level
				} else {
					fmt.Fprintf(os.Stderr, "Invalid log level: %s (must be 1, 2, or 3)\n", args[i+1])
					os.Exit(1)
				}
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
		internal.LogError("Error loading config: %v", err)
		os.Exit(1)
	}

	if err := pushImageWithConfig(imageRef, resolved); err != nil {
		internal.LogError("Error pushing image: %v", err)
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
		internal.LogError("Error loading config: %v", err)
		os.Exit(1)
	}

	if profileName != "" {
		profile, exists := config.Profiles[profileName]
		if !exists {
			internal.LogError("Profile '%s' not found", profileName)
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
		internal.LogError("Error loading config: %v", err)
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
		internal.LogError("Config file %s already exists", configPath)
		os.Exit(1)
	}

	if err := os.WriteFile(configPath, []byte(defaultContent), 0644); err != nil {
		internal.LogError("Error creating config file: %v", err)
		os.Exit(1)
	}

	internal.LogInfo("Created config file: %s", configPath)
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

func handleBuildCommand(globalFlags *GlobalFlags, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: s3dock [global-flags] build <app-name> [build-flags]")
		fmt.Println("")
		fmt.Println("Build a Docker image with git-based tag.")
		fmt.Println("")
		fmt.Println("Build Flags:")
		fmt.Println("  --path <directory>   Git repository path (default: .)")
		fmt.Println("  --dockerfile <path>  Dockerfile to use (default: Dockerfile)")
		fmt.Println("  --context <path>     Build context path (default: .)")
		fmt.Println("  --platform <platform> Target platform (e.g., linux/amd64, linux/arm64)")
		fmt.Println("")
		fmt.Println("Note: If --path is specified but --context is not, both will use the same path.")
		fmt.Println("")
		fmt.Println("The image will be tagged as: <app-name>:<timestamp>-<git-hash>")
		fmt.Println("Example: myapp:20250721-2118-f7a5a27")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  s3dock build myapp")
		fmt.Println("  s3dock build myapp --path /path/to/repo")
		fmt.Println("  s3dock build myapp --path ./subdirectory")
		fmt.Println("  s3dock build myapp --path . --dockerfile Dockerfile.prod")
		fmt.Println("  s3dock build myapp --path /git/repo --context /build/context")
		fmt.Println("  s3dock build myapp --platform linux/amd64")
		fmt.Println("  s3dock build myapp --platform linux/arm64")
		return
	}

	appName := args[0]
	buildArgs := args[1:]

	dockerfile := "Dockerfile"
	contextPath := "."
	gitPath := "."
	platform := ""

	for i := 0; i < len(buildArgs); i++ {
		arg := buildArgs[i]
		switch arg {
		case "--path":
			if i+1 < len(buildArgs) {
				gitPath = buildArgs[i+1]
				i++
			}
		case "--dockerfile":
			if i+1 < len(buildArgs) {
				dockerfile = buildArgs[i+1]
				i++
			}
		case "--context":
			if i+1 < len(buildArgs) {
				contextPath = buildArgs[i+1]
				i++
			}
		case "--platform":
			if i+1 < len(buildArgs) {
				platform = buildArgs[i+1]
				i++
			}
		}
	}

	// If --path is specified but --context is not, use the same path for both
	if gitPath != "." && contextPath == "." {
		contextPath = gitPath
	}

	// Always try to find the git repository root
	gitClient := internal.NewGitClient()
	
	// First try to find repository from the gitPath
	if repoRoot, err := gitClient.FindRepositoryRoot(gitPath); err == nil {
		internal.LogDebug("Found git repository root from gitPath: %s", repoRoot)
		gitPath = repoRoot
	} else {
		// If that fails, try from the context path
		if repoRoot, err := gitClient.FindRepositoryRoot(contextPath); err == nil {
			internal.LogDebug("Found git repository root from contextPath: %s", repoRoot)
			gitPath = repoRoot
		} else {
			// Finally, try from current working directory
			if repoRoot, err := gitClient.FindRepositoryRoot("."); err == nil {
				internal.LogDebug("Found git repository root from current directory: %s", repoRoot)
				gitPath = repoRoot
			} else {
				internal.LogError("Could not find git repository: %v", err)
				os.Exit(1)
			}
		}
	}

	if err := buildImageWithConfig(appName, contextPath, dockerfile, gitPath, platform); err != nil {
		internal.LogError("Error building image: %v", err)
		os.Exit(1)
	}
}

func buildImageWithConfig(appName, contextPath, dockerfile, gitPath, platform string) error {
	ctx := context.Background()

	dockerClient, err := internal.NewDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	gitClient := internal.NewGitClient()

	builder := internal.NewImageBuilder(dockerClient, gitClient)

	_, err = builder.Build(ctx, appName, contextPath, dockerfile, gitPath, platform)
	return err
}

func handleTagCommand(globalFlags *GlobalFlags, args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: s3dock [global-flags] tag <image:tag> <version>")
		fmt.Println("")
		fmt.Println("Create a semantic version tag for an image.")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  s3dock tag myapp:20250721-2118-f7a5a27 v1.2.0")
		fmt.Println("  s3dock tag myapp:20250720-1045-def5678 v1.1.5")
		return
	}

	imageRef := args[0]
	version := args[1]

	resolved, err := internal.ResolveConfig(globalFlags.Config, globalFlags.Profile, globalFlags.Bucket)
	if err != nil {
		internal.LogError("Error loading config: %v", err)
		os.Exit(1)
	}

	if err := tagImageWithConfig(imageRef, version, resolved); err != nil {
		internal.LogError("Error tagging image: %v", err)
		os.Exit(1)
	}
}

func handlePromoteCommand(globalFlags *GlobalFlags, args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: s3dock [global-flags] promote <source> <environment>")
		fmt.Println("   or: s3dock [global-flags] promote <app> <version> <environment>")
		fmt.Println("")
		fmt.Println("Promote an image or tag to an environment.")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  s3dock promote myapp:20250721-2118-f7a5a27 production")
		fmt.Println("  s3dock promote myapp v1.2.0 staging")
		return
	}

	var source, environment, appName, version string
	if len(args) == 2 {
		// Direct image promotion: s3dock promote myapp:20250721-2118-f7a5a27 production
		source = args[0]
		environment = args[1]
	} else if len(args) == 3 {
		// Tag-based promotion: s3dock promote myapp v1.2.0 staging
		appName = args[0]
		version = args[1]
		environment = args[2]
	} else {
		internal.LogError("Invalid number of arguments")
		os.Exit(1)
	}

	resolved, err := internal.ResolveConfig(globalFlags.Config, globalFlags.Profile, globalFlags.Bucket)
	if err != nil {
		internal.LogError("Error loading config: %v", err)
		os.Exit(1)
	}

	if len(args) == 2 {
		if err := promoteImageWithConfig(source, environment, resolved); err != nil {
			internal.LogError("Error promoting image: %v", err)
			os.Exit(1)
		}
	} else {
		if err := promoteTagWithConfig(appName, version, environment, resolved); err != nil {
			internal.LogError("Error promoting tag: %v", err)
			os.Exit(1)
		}
	}
}

func tagImageWithConfig(imageRef, version string, config *internal.ResolvedConfig) error {
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

	s3Client, err := internal.NewS3Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	tagger := internal.NewImageTagger(s3Client, config.Bucket)

	return tagger.Tag(ctx, imageRef, version)
}

func promoteImageWithConfig(source, environment string, config *internal.ResolvedConfig) error {
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

	s3Client, err := internal.NewS3Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	promoter := internal.NewImagePromoter(s3Client, config.Bucket)

	return promoter.Promote(ctx, source, environment)
}

func promoteTagWithConfig(appName, version, environment string, config *internal.ResolvedConfig) error {
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

	s3Client, err := internal.NewS3Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	promoter := internal.NewImagePromoter(s3Client, config.Bucket)

	return promoter.PromoteFromTag(ctx, appName, version, environment)
}

func handlePullCommand(globalFlags *GlobalFlags, args []string) {
	if len(args) < 2 {
		internal.LogError("Pull command requires app name and environment/tag")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s pull <app> <environment>    # Pull from environment (e.g., production, staging)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s pull <app> <tag>           # Pull from tag (e.g., v1.2.0)\n", os.Args[0])
		os.Exit(1)
	}

	appName := args[0]
	target := args[1]

	// Determine if target is a version tag (starts with 'v') or environment
	if strings.HasPrefix(target, "v") && len(strings.Split(target, ".")) >= 2 {
		// It's a version tag like v1.2.0
		err := pullTagWithConfig(appName, target, globalFlags)
		if err != nil {
			internal.LogError("Failed to pull tag: %v", err)
			os.Exit(1)
		}
	} else {
		// It's an environment like production, staging
		err := pullImageWithConfig(appName, target, globalFlags)
		if err != nil {
			internal.LogError("Failed to pull image: %v", err)
			os.Exit(1)
		}
	}
}

func pullImageWithConfig(appName, environment string, globalFlags *GlobalFlags) error {
	config, err := internal.ResolveConfig(globalFlags.Config, globalFlags.Profile, globalFlags.Bucket)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Set environment variables for AWS configuration
	os.Setenv("AWS_REGION", config.Region)
	if config.Endpoint != "" {
		os.Setenv("AWS_ENDPOINT_URL", config.Endpoint)
	}
	if config.AccessKey != "" && config.SecretKey != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", config.AccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", config.SecretKey)
	}

	s3Client, err := internal.NewS3Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	dockerClient, err := internal.NewDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	puller := internal.NewImagePuller(dockerClient, s3Client, config.Bucket)

	return puller.Pull(ctx, appName, environment)
}

func pullTagWithConfig(appName, version string, globalFlags *GlobalFlags) error {
	config, err := internal.ResolveConfig(globalFlags.Config, globalFlags.Profile, globalFlags.Bucket)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Set environment variables for AWS configuration
	os.Setenv("AWS_REGION", config.Region)
	if config.Endpoint != "" {
		os.Setenv("AWS_ENDPOINT_URL", config.Endpoint)
	}
	if config.AccessKey != "" && config.SecretKey != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", config.AccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", config.SecretKey)
	}

	s3Client, err := internal.NewS3Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	dockerClient, err := internal.NewDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	puller := internal.NewImagePuller(dockerClient, s3Client, config.Bucket)

	return puller.PullFromTag(ctx, appName, version)
}

func handleCurrentCommand(globalFlags *GlobalFlags, args []string) {
	if len(args) < 2 {
		internal.LogError("Current command requires app name and environment")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s current <app> <environment>    # Show current image for environment (e.g., production, staging)\n", os.Args[0])
		os.Exit(1)
	}

	appName := args[0]
	environment := args[1]

	err := getCurrentImageWithConfig(appName, environment, globalFlags)
	if err != nil {
		internal.LogError("Failed to get current image: %v", err)
		os.Exit(1)
	}
}

func getCurrentImageWithConfig(appName, environment string, globalFlags *GlobalFlags) error {
	config, err := internal.ResolveConfig(globalFlags.Config, globalFlags.Profile, globalFlags.Bucket)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Set environment variables for AWS configuration
	os.Setenv("AWS_REGION", config.Region)
	if config.Endpoint != "" {
		os.Setenv("AWS_ENDPOINT_URL", config.Endpoint)
	}
	if config.AccessKey != "" && config.SecretKey != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", config.AccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", config.SecretKey)
	}

	s3Client, err := internal.NewS3Client(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	currentService := internal.NewCurrentService(s3Client, config.Bucket)

	imageRef, err := currentService.GetCurrentImage(ctx, appName, environment)
	if err != nil {
		return err
	}

	// Output the current image reference
	fmt.Println(imageRef)
	return nil
}

func handleVersionCommand(args []string) {
	showFull := false

	// Check for --full or --detailed flag
	for _, arg := range args {
		if arg == "--full" || arg == "--detailed" {
			showFull = true
			break
		}
	}

	if showFull {
		fmt.Printf("s3dock version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("built: %s\n", date)
	} else {
		fmt.Println(version)
	}
}

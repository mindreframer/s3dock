package main

import (
	"context"
	"fmt"
	"os"

	"s3dock/internal"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: s3dock <command>")
		fmt.Println("Commands: push, tag, pull, list, cleanup, deploy")
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "push":
		if len(os.Args) < 3 {
			fmt.Println("Usage: s3dock push <image:tag>")
			os.Exit(1)
		}
		
		bucket := os.Getenv("S3DOCK_BUCKET")
		if bucket == "" {
			fmt.Println("Error: S3DOCK_BUCKET environment variable not set")
			os.Exit(1)
		}
		
		imageRef := os.Args[2]
		if err := pushImage(imageRef, bucket); err != nil {
			fmt.Printf("Error pushing image: %v\n", err)
			os.Exit(1)
		}
		
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
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func pushImage(imageRef, bucket string) error {
	ctx := context.Background()
	
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
	
	pusher := internal.NewImagePusher(dockerClient, s3Client, gitClient, bucket)
	
	return pusher.Push(ctx, imageRef)
}
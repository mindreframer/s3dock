//go:build integration

package internal

import (
	"archive/tar"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
)

var (
	keepTmp  = flag.Bool("keep-tmp", false, "Keep temp directories after test runs")
	logLevel = flag.Int("log-level", 1, "Log verbosity level (1-3)")
)

// Test helper for logging at different verbosity levels
func logf(t *testing.T, level int, format string, args ...interface{}) {
	t.Helper()
	if *logLevel >= level {
		t.Logf("[level-%d] %s", level, fmt.Sprintf(format, args...))
	}
}

// setupTempGitRepo creates a temporary git repository with a minimal Dockerfile
func setupTempGitRepo(t *testing.T) (repoDir string, cleanup func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "s3dock-git-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	logf(t, 2, "Created temp dir: %s", dir)

	// Configure git for tests
	gitEnv := append(os.Environ(),
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com")

	// Init git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Env = gitEnv
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	logf(t, 3, "Git init completed")

	// Add minimal Dockerfile
	dockerfile := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(dockerfile, []byte("FROM busybox\nLABEL test=integration\n"), 0644); err != nil {
		t.Fatalf("failed to write Dockerfile: %v", err)
	}
	logf(t, 3, "Created Dockerfile")

	// git add Dockerfile
	cmd = exec.Command("git", "add", "Dockerfile")
	cmd.Dir = dir
	cmd.Env = gitEnv
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}

	// git commit
	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = dir
	cmd.Env = gitEnv
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}
	logf(t, 3, "Initial commit completed")

	cleanupFn := func() {
		if *keepTmp {
			fmt.Printf("[keep-tmp] Temp dir: %s\n", dir)
			return
		}
		logf(t, 3, "Cleaning up temp dir: %s", dir)
		os.RemoveAll(dir)
	}

	return dir, cleanupFn
}

// makeDirty modifies the repository to make it dirty
func makeDirty(t *testing.T, repoDir string) {
	t.Helper()
	// Add a new file
	testFile := filepath.Join(repoDir, "dirty-file.txt")
	if err := os.WriteFile(testFile, []byte("dirty content\n"), 0644); err != nil {
		t.Fatalf("failed to create dirty file: %v", err)
	}
	logf(t, 3, "Made repo dirty by adding file: %s", testFile)
}

// modifyExistingFile modifies an existing tracked file to make repo dirty
func modifyExistingFile(t *testing.T, repoDir string) {
	t.Helper()
	dockerfile := filepath.Join(repoDir, "Dockerfile")
	content := "FROM busybox\nLABEL test=integration\nLABEL modified=true\n"
	if err := os.WriteFile(dockerfile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to modify Dockerfile: %v", err)
	}
	logf(t, 3, "Modified existing Dockerfile")
}

// addCommit adds another commit to the repository
func addCommit(t *testing.T, repoDir string, message string) {
	t.Helper()
	gitEnv := append(os.Environ(),
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com")

	// Add a new file
	filename := strings.ReplaceAll(message, " ", "-") + ".txt"
	testFile := filepath.Join(repoDir, filename)
	if err := os.WriteFile(testFile, []byte(message+"\n"), 0644); err != nil {
		t.Fatalf("failed to create file for commit: %v", err)
	}

	// git add
	cmd := exec.Command("git", "add", filename)
	cmd.Dir = repoDir
	cmd.Env = gitEnv
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}

	// git commit
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoDir
	cmd.Env = gitEnv
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}
	logf(t, 3, "Added commit: %s", message)
}

// dockerAvailable checks if Docker is available
func dockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	return cmd.Run() == nil
}

// createTarBuildContext creates a proper tar archive for Docker build context
func createTarBuildContext(contextPath string) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		tw := tar.NewWriter(pw)
		defer tw.Close()

		err := filepath.Walk(contextPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(contextPath, path)
			if err != nil {
				return err
			}

			if relPath == "." {
				return nil
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = relPath

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tw, file)
			return err
		})

		if err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr, nil
}

// GitClientWithDir is a git client that works with a specific directory for testing
type GitClientWithDir struct {
	dir string
}

func NewGitClientWithDir(dir string) *GitClientWithDir {
	return &GitClientWithDir{dir: dir}
}

func (g *GitClientWithDir) GetCurrentHash() (string, error) {
	repo, err := git.PlainOpen(g.dir)
	if err != nil {
		return "", err
	}

	ref, err := repo.Head()
	if err != nil {
		return "", err
	}

	return ref.Hash().String()[:7], nil
}

func (g *GitClientWithDir) GetCommitTimestamp() (string, error) {
	repo, err := git.PlainOpen(g.dir)
	if err != nil {
		return "", err
	}

	ref, err := repo.Head()
	if err != nil {
		return "", err
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", err
	}

	return commit.Committer.When.Format("20060102-1504"), nil
}

func (g *GitClientWithDir) IsRepositoryDirty() (bool, error) {
	repo, err := git.PlainOpen(g.dir)
	if err != nil {
		return false, err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return false, err
	}

	status, err := worktree.Status()
	if err != nil {
		return false, err
	}

	return !status.IsClean(), nil
}

// TestIntegration_GitClient tests the GitClient interface directly
func TestIntegration_GitClient_DirectOperations(t *testing.T) {
	flag.Parse()
	t.Parallel()

	repoDir, cleanup := setupTempGitRepo(t)
	defer cleanup()

	gitClient := NewGitClientWithDir(repoDir)

	// Test GetCurrentHash
	hash, err := gitClient.GetCurrentHash()
	if err != nil {
		t.Fatalf("GetCurrentHash failed: %v", err)
	}
	if len(hash) != 7 {
		t.Errorf("expected hash length 7, got %d: %s", len(hash), hash)
	}
	logf(t, 2, "Git hash: %s", hash)

	// Test GetCommitTimestamp
	timestamp, err := gitClient.GetCommitTimestamp()
	if err != nil {
		t.Fatalf("GetCommitTimestamp failed: %v", err)
	}
	// Check format YYYYMMDD-HHMM
	if matched, _ := regexp.MatchString(`^\d{8}-\d{4}$`, timestamp); !matched {
		t.Errorf("invalid timestamp format: %s", timestamp)
	}
	logf(t, 2, "Git timestamp: %s", timestamp)

	// Test IsRepositoryDirty - should be clean
	isDirty, err := gitClient.IsRepositoryDirty()
	if err != nil {
		t.Fatalf("IsRepositoryDirty failed: %v", err)
	}
	if isDirty {
		t.Errorf("expected clean repository, but got dirty")
	}
	logf(t, 2, "Repository is clean: %t", !isDirty)
}

// TestIntegration_Build_CleanRepo_Succeeds tests successful build with clean repository
func TestIntegration_Build_CleanRepo_Succeeds(t *testing.T) {
	flag.Parse()
	t.Parallel()
	if !dockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	repoDir, cleanup := setupTempGitRepo(t)
	defer cleanup()

	// Use real clients
	gitClient := NewGitClientWithDir(repoDir)
	dockerClient, err := NewDockerClient()
	if err != nil {
		t.Fatalf("failed to create Docker client: %v", err)
	}
	defer dockerClient.Close()

	builder := NewImageBuilder(dockerClient, gitClient)

	ctx := context.Background()
	appName := "myapp"
	tag, err := builder.Build(ctx, appName, repoDir, "Dockerfile")
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	logf(t, 1, "Built image tag: %s", tag)

	// Check tag format: myapp:YYYYMMDD-HHMM-abcdefg
	expectedPattern := `^myapp:\d{8}-\d{4}-[a-f0-9]{7}$`
	if matched, _ := regexp.MatchString(expectedPattern, tag); !matched {
		t.Errorf("unexpected tag format: %s (expected pattern: %s)", tag, expectedPattern)
	}
}

// TestIntegration_Build_DirtyRepo_Fails tests that build fails with dirty repository
func TestIntegration_Build_DirtyRepo_Fails(t *testing.T) {
	flag.Parse()
	t.Parallel()
	if !dockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	repoDir, cleanup := setupTempGitRepo(t)
	defer cleanup()

	// Make repository dirty
	makeDirty(t, repoDir)

	// Use real clients
	gitClient := NewGitClientWithDir(repoDir)
	dockerClient, err := NewDockerClient()
	if err != nil {
		t.Fatalf("failed to create Docker client: %v", err)
	}
	defer dockerClient.Close()

	builder := NewImageBuilder(dockerClient, gitClient)

	ctx := context.Background()
	appName := "myapp"
	_, err = builder.Build(ctx, appName, repoDir, "Dockerfile")
	if err == nil {
		t.Fatalf("expected build to fail with dirty repository, but it succeeded")
	}

	expectedError := "repository has uncommitted changes"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("expected error to contain '%s', got: %v", expectedError, err)
	}
	logf(t, 1, "Build correctly failed with dirty repo: %v", err)
}

// TestIntegration_Build_ModifiedDockerfile_DetectsDirty tests dirty detection when Dockerfile is modified
func TestIntegration_Build_ModifiedDockerfile_DetectsDirty(t *testing.T) {
	flag.Parse()
	t.Parallel()
	if !dockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	repoDir, cleanup := setupTempGitRepo(t)
	defer cleanup()

	// First verify it works when clean
	gitClient := NewGitClientWithDir(repoDir)
	dockerClient, err := NewDockerClient()
	if err != nil {
		t.Fatalf("failed to create Docker client: %v", err)
	}
	defer dockerClient.Close()

	builder := NewImageBuilder(dockerClient, gitClient)
	ctx := context.Background()
	appName := "myapp"

	// First build should succeed
	tag1, err := builder.Build(ctx, appName, repoDir, "Dockerfile")
	if err != nil {
		t.Fatalf("first build failed: %v", err)
	}
	logf(t, 2, "First build succeeded: %s", tag1)

	// Modify Dockerfile
	modifyExistingFile(t, repoDir)

	// Second build should fail due to dirty state
	_, err = builder.Build(ctx, appName, repoDir, "Dockerfile")
	if err == nil {
		t.Fatalf("expected build to fail with modified Dockerfile, but it succeeded")
	}

	expectedError := "repository has uncommitted changes"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("expected error to contain '%s', got: %v", expectedError, err)
	}
	logf(t, 1, "Build correctly failed with modified Dockerfile: %v", err)
}

// TestIntegration_Build_MultipleCommits_TagFormat tests tag format with multiple commits
func TestIntegration_Build_MultipleCommits_TagFormat(t *testing.T) {
	flag.Parse()
	t.Parallel()
	if !dockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	repoDir, cleanup := setupTempGitRepo(t)
	defer cleanup()

	// Add more commits
	addCommit(t, repoDir, "second commit")
	addCommit(t, repoDir, "third commit")

	// Use real clients
	gitClient := NewGitClientWithDir(repoDir)
	dockerClient, err := NewDockerClient()
	if err != nil {
		t.Fatalf("failed to create Docker client: %v", err)
	}
	defer dockerClient.Close()

	builder := NewImageBuilder(dockerClient, gitClient)

	ctx := context.Background()
	appName := "testapp"
	tag, err := builder.Build(ctx, appName, repoDir, "Dockerfile")
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	logf(t, 1, "Built image tag with multiple commits: %s", tag)

	// Check tag format
	expectedPattern := `^testapp:\d{8}-\d{4}-[a-f0-9]{7}$`
	if matched, _ := regexp.MatchString(expectedPattern, tag); !matched {
		t.Errorf("unexpected tag format: %s (expected pattern: %s)", tag, expectedPattern)
	}

	// Verify tag reflects latest commit
	parts := strings.Split(tag, ":")
	if len(parts) != 2 {
		t.Errorf("invalid tag format: %s", tag)
		return
	}

	// Extract timestamp and hash from tag
	tagParts := strings.Split(parts[1], "-")
	if len(tagParts) != 3 {
		t.Errorf("invalid tag timestamp-hash format: %s", parts[1])
		return
	}

	gitHash, err := gitClient.GetCurrentHash()
	if err != nil {
		t.Fatalf("failed to get current hash: %v", err)
	}

	if tagParts[2] != gitHash {
		t.Errorf("tag hash %s doesn't match git hash %s", tagParts[2], gitHash)
	}
}

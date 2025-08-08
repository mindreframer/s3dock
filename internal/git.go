package internal

import (
	"github.com/go-git/go-git/v5"
)

type GitClientImpl struct{}

func NewGitClient() *GitClientImpl {
	return &GitClientImpl{}
}

func (g *GitClientImpl) GetCurrentHash(path string) (string, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return "", err
	}

	ref, err := repo.Head()
	if err != nil {
		return "", err
	}

	return ref.Hash().String()[:7], nil
}

func (g *GitClientImpl) GetCommitTimestamp(path string) (string, error) {
	repo, err := git.PlainOpen(path)
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

func (g *GitClientImpl) FindRepositoryRoot(startPath string) (string, error) {
	repo, err := git.PlainOpenWithOptions(startPath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return "", err
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	return workTree.Filesystem.Root(), nil
}

func (g *GitClientImpl) IsRepositoryDirty(path string) (bool, error) {
	repo, err := git.PlainOpen(path)
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

	// Check if there are any actual modifications (not just untracked files)
	hasModifications := false
	for _, fileStatus := range status {
		// Only consider files that have actual modifications (not just untracked)
		if fileStatus.Worktree != git.Untracked && fileStatus.Worktree != git.Unmodified {
			hasModifications = true
			break
		}
	}

	return hasModifications, nil
}

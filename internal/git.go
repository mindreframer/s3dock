package internal

import (
	"github.com/go-git/go-git/v5"
)

type GitClientImpl struct{}

func NewGitClient() *GitClientImpl {
	return &GitClientImpl{}
}

func (g *GitClientImpl) GetCurrentHash() (string, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return "", err
	}

	ref, err := repo.Head()
	if err != nil {
		return "", err
	}

	return ref.Hash().String()[:7], nil
}

func (g *GitClientImpl) GetCommitTimestamp() (string, error) {
	repo, err := git.PlainOpen(".")
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

func (g *GitClientImpl) IsRepositoryDirty() (bool, error) {
	repo, err := git.PlainOpen(".")
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

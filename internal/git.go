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

package repository

import (
	"fmt"
	"os"

	"github.com/amaya382/baretree/internal/config"
)

// FindRoot finds the baretree repository root from the current directory
func FindRoot(startPath string) (string, error) {
	return config.FindRepoRoot(startPath)
}

// GetBareRepoPath returns the path to the bare repository
func GetBareRepoPath(repoRoot string) (string, error) {
	bareDir, err := config.GetBareDir(repoRoot)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(bareDir); os.IsNotExist(err) {
		return "", fmt.Errorf("bare repository not found at %s", bareDir)
	}

	return bareDir, nil
}

// IsBaretreeRepo checks if the given path is a baretree repository
// by verifying git-config has baretree settings and bare repository is valid
func IsBaretreeRepo(path string) bool {
	return config.IsBaretreeRepoGit(path)
}

// GetBareDirName returns the bare directory name (.git) for the repo
func GetBareDirName(repoRoot string) string {
	return config.BareDir
}

// InitializeConfig initializes baretree configuration in the bare repository
func InitializeConfig(repoRoot, defaultBranch string) error {
	return config.InitializeBaretreeConfig(repoRoot, defaultBranch)
}

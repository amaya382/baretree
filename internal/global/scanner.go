package global

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/amaya382/baretree/internal/repository"
)

// RepoInfo holds information about a discovered repository
type RepoInfo struct {
	// Path is the absolute path to the repository
	Path string
	// RelativePath is the path relative to the root (e.g., "github.com/user/repo")
	RelativePath string
	// Name is the repository name (last component of the path)
	Name string
}

// ScanRepositories scans the given root directories for baretree repositories
func ScanRepositories(roots []string) ([]RepoInfo, error) {
	var repos []RepoInfo
	seen := make(map[string]bool)

	for _, root := range roots {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}

		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip directories we can't access
			}

			if !d.IsDir() {
				return nil
			}

			// Skip hidden directories (except the root itself)
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}

			// Check if this is a baretree repository
			if repository.IsBaretreeRepo(path) {
				if !seen[path] {
					seen[path] = true
					relPath, _ := filepath.Rel(root, path)
					repos = append(repos, RepoInfo{
						Path:         path,
						RelativePath: relPath,
						Name:         filepath.Base(path),
					})
				}
				return filepath.SkipDir // Don't descend into repositories
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return repos, nil
}

// FilterRepositories filters repositories by a query string
func FilterRepositories(repos []RepoInfo, query string) []RepoInfo {
	if query == "" {
		return repos
	}

	query = strings.ToLower(query)
	var filtered []RepoInfo

	for _, repo := range repos {
		// Match against relative path or name
		if strings.Contains(strings.ToLower(repo.RelativePath), query) ||
			strings.Contains(strings.ToLower(repo.Name), query) {
			filtered = append(filtered, repo)
		}
	}

	return filtered
}

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

// FilterRepositories filters repositories by a query string.
// Results are ordered with prefix matches first, then substring matches.
func FilterRepositories(repos []RepoInfo, query string) []RepoInfo {
	if query == "" {
		return repos
	}

	query = strings.ToLower(query)
	var prefixMatches []RepoInfo
	var substringMatches []RepoInfo

	for _, repo := range repos {
		relPathLower := strings.ToLower(repo.RelativePath)
		nameLower := strings.ToLower(repo.Name)

		// Check for prefix match (name starts with query)
		if strings.HasPrefix(nameLower, query) {
			prefixMatches = append(prefixMatches, repo)
		} else if strings.Contains(relPathLower, query) || strings.Contains(nameLower, query) {
			// Substring match (not a prefix match)
			substringMatches = append(substringMatches, repo)
		}
	}

	// Return prefix matches first, then substring matches
	return append(prefixMatches, substringMatches...)
}

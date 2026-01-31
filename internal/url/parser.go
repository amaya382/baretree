package url

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// RepoPath represents a parsed repository path
type RepoPath struct {
	Host string // e.g., "github.com"
	User string // e.g., "amaya382"
	Repo string // e.g., "baretree"
}

// String returns the full path representation (host/user/repo)
func (r *RepoPath) String() string {
	return path.Join(r.Host, r.User, r.Repo)
}

// sshURLRegex matches SSH URLs like git@github.com:user/repo.git
var sshURLRegex = regexp.MustCompile(`^(?:[\w-]+@)?([\w.-]+):(.+?)(?:\.git)?$`)

// Parse parses a repository URL or path and returns its components
// Supports:
//   - SSH URL: git@github.com:user/repo.git
//   - HTTPS URL: https://github.com/user/repo.git
//   - Short path: github.com/user/repo
//   - User/repo (requires defaultHost): user/repo
//   - Repo only (requires defaultHost and defaultUser): repo
func Parse(input string, defaultHost string, defaultUser string) (*RepoPath, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty repository path")
	}

	// Try HTTPS/HTTP URL format first: https://github.com/user/repo.git
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		u, err := url.Parse(input)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
		pathStr := strings.TrimPrefix(u.Path, "/")
		pathStr = strings.TrimSuffix(pathStr, ".git")
		parts := strings.Split(pathStr, "/")
		if len(parts) >= 2 {
			return &RepoPath{
				Host: u.Host,
				User: parts[0],
				Repo: strings.Join(parts[1:], "/"),
			}, nil
		}
		return nil, fmt.Errorf("invalid HTTPS URL format: %s", input)
	}

	// Try SSH URL format: git@github.com:user/repo.git
	if matches := sshURLRegex.FindStringSubmatch(input); matches != nil {
		host := matches[1]
		repoPath := strings.TrimSuffix(matches[2], ".git")
		parts := strings.Split(repoPath, "/")
		if len(parts) >= 2 {
			return &RepoPath{
				Host: host,
				User: parts[0],
				Repo: strings.Join(parts[1:], "/"),
			}, nil
		}
		return nil, fmt.Errorf("invalid SSH URL format: %s", input)
	}

	// Try short path format: github.com/user/repo or host/user/repo
	parts := strings.Split(input, "/")

	switch len(parts) {
	case 1:
		// Repo only: "repo"
		if defaultHost == "" || defaultUser == "" {
			return nil, fmt.Errorf("cannot determine host and user for: %s", input)
		}
		return &RepoPath{
			Host: defaultHost,
			User: defaultUser,
			Repo: parts[0],
		}, nil

	case 2:
		// User/repo: "user/repo"
		if defaultHost == "" {
			return nil, fmt.Errorf("cannot determine host for: %s", input)
		}
		return &RepoPath{
			Host: defaultHost,
			User: parts[0],
			Repo: parts[1],
		}, nil

	default:
		// Full path: "host/user/repo" or deeper
		return &RepoPath{
			Host: parts[0],
			User: parts[1],
			Repo: strings.Join(parts[2:], "/"),
		}, nil
	}
}

// ParseRemoteURL parses a git remote URL and returns the repository path
func ParseRemoteURL(remoteURL string) (*RepoPath, error) {
	return Parse(remoteURL, "", "")
}

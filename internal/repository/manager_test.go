package repository

import (
	"testing"
)

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "SSH URL with .git",
			url:      "git@github.com:user/myrepo.git",
			expected: "myrepo",
		},
		{
			name:     "SSH URL without .git",
			url:      "git@github.com:user/myrepo",
			expected: "myrepo",
		},
		{
			name:     "HTTPS URL with .git",
			url:      "https://github.com/user/myrepo.git",
			expected: "myrepo",
		},
		{
			name:     "HTTPS URL without .git",
			url:      "https://github.com/user/myrepo",
			expected: "myrepo",
		},
		{
			name:     "nested path",
			url:      "https://github.com/org/team/project.git",
			expected: "project",
		},
		{
			name:     "simple name",
			url:      "myrepo.git",
			expected: "myrepo",
		},
		{
			name:     "simple name without .git",
			url:      "myrepo",
			expected: "myrepo",
		},
		{
			name:     "GitLab SSH URL",
			url:      "git@gitlab.com:group/subgroup/project.git",
			expected: "project",
		},
		{
			name:     "local path",
			url:      "/home/user/repos/myproject.git",
			expected: "myproject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractRepoName(tt.url)
			if result != tt.expected {
				t.Errorf("ExtractRepoName(%q) = %q, expected %q", tt.url, result, tt.expected)
			}
		})
	}
}

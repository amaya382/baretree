package url

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		defaultHost string
		defaultUser string
		wantHost    string
		wantUser    string
		wantRepo    string
		wantErr     bool
	}{
		{
			name:     "SSH URL with .git",
			input:    "git@github.com:amaya382/baretree.git",
			wantHost: "github.com",
			wantUser: "amaya382",
			wantRepo: "baretree",
		},
		{
			name:     "SSH URL without .git",
			input:    "git@github.com:amaya382/baretree",
			wantHost: "github.com",
			wantUser: "amaya382",
			wantRepo: "baretree",
		},
		{
			name:     "HTTPS URL with .git",
			input:    "https://github.com/amaya382/baretree.git",
			wantHost: "github.com",
			wantUser: "amaya382",
			wantRepo: "baretree",
		},
		{
			name:     "HTTPS URL without .git",
			input:    "https://github.com/amaya382/baretree",
			wantHost: "github.com",
			wantUser: "amaya382",
			wantRepo: "baretree",
		},
		{
			name:     "Full path",
			input:    "github.com/amaya382/baretree",
			wantHost: "github.com",
			wantUser: "amaya382",
			wantRepo: "baretree",
		},
		{
			name:        "User/repo with default host",
			input:       "amaya382/baretree",
			defaultHost: "github.com",
			wantHost:    "github.com",
			wantUser:    "amaya382",
			wantRepo:    "baretree",
		},
		{
			name:        "Repo only with defaults",
			input:       "baretree",
			defaultHost: "github.com",
			defaultUser: "amaya382",
			wantHost:    "github.com",
			wantUser:    "amaya382",
			wantRepo:    "baretree",
		},
		{
			name:    "User/repo without default host",
			input:   "amaya382/baretree",
			wantErr: true,
		},
		{
			name:        "Repo only without default user",
			input:       "baretree",
			defaultHost: "github.com",
			wantErr:     true,
		},
		{
			name:    "Empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:     "GitLab SSH URL",
			input:    "git@gitlab.com:user/project.git",
			wantHost: "gitlab.com",
			wantUser: "user",
			wantRepo: "project",
		},
		{
			name:     "Nested repo path",
			input:    "github.com/org/group/repo",
			wantHost: "github.com",
			wantUser: "org",
			wantRepo: "group/repo",
		},
		{
			name:     "SSH URL with nested path",
			input:    "git@gitlab.com:org/group/repo.git",
			wantHost: "gitlab.com",
			wantUser: "org",
			wantRepo: "group/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input, tt.defaultHost, tt.defaultUser)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Host != tt.wantHost {
				t.Errorf("Parse() Host = %v, want %v", got.Host, tt.wantHost)
			}
			if got.User != tt.wantUser {
				t.Errorf("Parse() User = %v, want %v", got.User, tt.wantUser)
			}
			if got.Repo != tt.wantRepo {
				t.Errorf("Parse() Repo = %v, want %v", got.Repo, tt.wantRepo)
			}
		})
	}
}

func TestRepoPath_String(t *testing.T) {
	rp := &RepoPath{
		Host: "github.com",
		User: "amaya382",
		Repo: "baretree",
	}
	want := "github.com/amaya382/baretree"
	if got := rp.String(); got != want {
		t.Errorf("RepoPath.String() = %v, want %v", got, want)
	}
}

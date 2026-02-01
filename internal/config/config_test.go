package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Repository.DefaultBranch != "main" {
		t.Errorf("expected default_branch 'main', got %q", cfg.Repository.DefaultBranch)
	}

	if len(cfg.PostCreate) != 0 {
		t.Errorf("expected empty postcreate, got %d items", len(cfg.PostCreate))
	}
}

// createTestBareRepo creates a bare git repository for testing
func createTestBareRepo(t *testing.T, tempDir, bareDir string) string {
	t.Helper()
	barePath := filepath.Join(tempDir, bareDir)
	if err := os.MkdirAll(barePath, 0755); err != nil {
		t.Fatalf("failed to create bare dir: %v", err)
	}
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = barePath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}
	return barePath
}

func TestLoadConfig(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "baretree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create bare repository
	createTestBareRepo(t, tempDir, ".git")

	// Initialize config
	if err := InitializeBaretreeConfig(tempDir, "main"); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	// Load config
	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify values
	if cfg.Repository.DefaultBranch != "main" {
		t.Errorf("expected default_branch 'main', got %q", cfg.Repository.DefaultBranch)
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "baretree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	_, err = LoadConfig(tempDir)
	if err == nil {
		t.Error("expected error when config not found, got nil")
	}
}

func TestSaveConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "baretree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create bare repository first
	createTestBareRepo(t, tempDir, ".git")

	cfg := &Config{
		Repository: Repository{
			DefaultBranch: "main",
		},
		PostCreate: []PostCreateAction{
			{Source: ".env", Type: "symlink"},
		},
	}

	if err := SaveConfig(tempDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load and verify
	loaded, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if len(loaded.PostCreate) != len(cfg.PostCreate) {
		t.Errorf("expected %d postcreate items, got %d", len(cfg.PostCreate), len(loaded.PostCreate))
	}
}

func TestFindRepoRoot(t *testing.T) {
	// Create nested directory structure
	tempDir, err := os.MkdirTemp("", "baretree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create bare repository
	createTestBareRepo(t, tempDir, ".git")

	// Initialize config
	if err := InitializeBaretreeConfig(tempDir, "main"); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	// Create nested directories
	nestedDir := filepath.Join(tempDir, "main", "src", "components")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	// Find root from nested directory
	root, err := FindRepoRoot(nestedDir)
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	if root != tempDir {
		t.Errorf("expected root %q, got %q", tempDir, root)
	}
}

func TestFindRepoRootNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "baretree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	_, err = FindRepoRoot(tempDir)
	if err == nil {
		t.Error("expected error when repo root not found, got nil")
	}
}

func TestExportImportTOML(t *testing.T) {
	original := &Config{
		Repository: Repository{
			DefaultBranch: "main",
		},
		PostCreate: []PostCreateAction{
			{Source: ".env", Type: "symlink", Managed: false},
			{Source: ".gitignore", Type: "copy", Managed: true},
		},
	}

	// Export to TOML
	tomlContent, err := ExportConfigToTOML(original)
	if err != nil {
		t.Fatalf("failed to export: %v", err)
	}

	// Import from TOML
	imported, err := ImportConfigFromTOML(tomlContent)
	if err != nil {
		t.Fatalf("failed to import: %v", err)
	}

	// Verify
	if len(imported.PostCreate) != len(original.PostCreate) {
		t.Errorf("expected %d postcreate items, got %d", len(original.PostCreate), len(imported.PostCreate))
	}
}

func TestExportImportPostCreateTOML(t *testing.T) {
	original := []PostCreateAction{
		{Source: ".env", Type: "symlink", Managed: false},
		{Source: ".gitignore", Type: "copy", Managed: true},
		{Source: "direnv allow", Type: "command"},
	}

	// Export to TOML
	tomlContent, err := ExportPostCreateToTOML(original)
	if err != nil {
		t.Fatalf("failed to export: %v", err)
	}

	// Import from TOML
	imported, err := ImportPostCreateFromTOML(tomlContent)
	if err != nil {
		t.Fatalf("failed to import: %v", err)
	}

	// Verify
	if len(imported) != len(original) {
		t.Errorf("expected %d postcreate items, got %d", len(original), len(imported))
	}

	for i, a := range imported {
		if a.Source != original[i].Source {
			t.Errorf("item %d: expected source %q, got %q", i, original[i].Source, a.Source)
		}
		if a.Type != original[i].Type {
			t.Errorf("item %d: expected type %q, got %q", i, original[i].Type, a.Type)
		}
		if a.Managed != original[i].Managed {
			t.Errorf("item %d: expected managed %v, got %v", i, original[i].Managed, a.Managed)
		}
	}
}

func TestParsePostCreateEntry(t *testing.T) {
	tests := []struct {
		input    string
		expected PostCreateAction
		wantErr  bool
	}{
		{".env:symlink", PostCreateAction{Source: ".env", Type: "symlink", Managed: false}, false},
		{".gitignore:copy:managed", PostCreateAction{Source: ".gitignore", Type: "copy", Managed: true}, false},
		{"direnv allow:command", PostCreateAction{Source: "direnv allow", Type: "command"}, false},
		{"npm install:command", PostCreateAction{Source: "npm install", Type: "command"}, false},
		{"invalid", PostCreateAction{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parsePostCreateEntry(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePostCreateEntry(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("parsePostCreateEntry(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatPostCreateEntry(t *testing.T) {
	tests := []struct {
		input    PostCreateAction
		expected string
	}{
		{PostCreateAction{Source: ".env", Type: "symlink", Managed: false}, ".env:symlink"},
		{PostCreateAction{Source: ".gitignore", Type: "copy", Managed: true}, ".gitignore:copy:managed"},
		{PostCreateAction{Source: "direnv allow", Type: "command"}, "direnv allow:command"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatPostCreateEntry(tt.input)
			if result != tt.expected {
				t.Errorf("formatPostCreateEntry(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

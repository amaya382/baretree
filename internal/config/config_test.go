package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Repository.BareDir != ".bare" {
		t.Errorf("expected bare_dir '.bare', got %q", cfg.Repository.BareDir)
	}

	if len(cfg.Shared) != 0 {
		t.Errorf("expected empty shared, got %d items", len(cfg.Shared))
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
	createTestBareRepo(t, tempDir, ".bare")

	// Initialize config
	if err := InitializeBaretreeConfig(tempDir, ".bare", "main"); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	// Load config
	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify values
	if cfg.Repository.BareDir != ".bare" {
		t.Errorf("expected bare_dir '.bare', got %q", cfg.Repository.BareDir)
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
	createTestBareRepo(t, tempDir, ".bare")

	cfg := &Config{
		Repository: Repository{
			BareDir:       ".bare",
			DefaultBranch: "main",
		},
		Shared: []Shared{
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

	if len(loaded.Shared) != len(cfg.Shared) {
		t.Errorf("expected %d shared items, got %d", len(cfg.Shared), len(loaded.Shared))
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
	createTestBareRepo(t, tempDir, ".bare")

	// Initialize config
	if err := InitializeBaretreeConfig(tempDir, ".bare", "main"); err != nil {
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
			BareDir:       ".bare",
			DefaultBranch: "main",
		},
		Shared: []Shared{
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
	if len(imported.Shared) != len(original.Shared) {
		t.Errorf("expected %d shared items, got %d", len(original.Shared), len(imported.Shared))
	}
}

func TestExportImportSharedTOML(t *testing.T) {
	original := []Shared{
		{Source: ".env", Type: "symlink", Managed: false},
		{Source: ".gitignore", Type: "copy", Managed: true},
	}

	// Export to TOML
	tomlContent, err := ExportSharedToTOML(original)
	if err != nil {
		t.Fatalf("failed to export: %v", err)
	}

	// Import from TOML
	imported, err := ImportSharedFromTOML(tomlContent)
	if err != nil {
		t.Fatalf("failed to import: %v", err)
	}

	// Verify
	if len(imported) != len(original) {
		t.Errorf("expected %d shared items, got %d", len(original), len(imported))
	}

	for i, s := range imported {
		if s.Source != original[i].Source {
			t.Errorf("item %d: expected source %q, got %q", i, original[i].Source, s.Source)
		}
		if s.Type != original[i].Type {
			t.Errorf("item %d: expected type %q, got %q", i, original[i].Type, s.Type)
		}
		if s.Managed != original[i].Managed {
			t.Errorf("item %d: expected managed %v, got %v", i, original[i].Managed, s.Managed)
		}
	}
}

func TestParseSharedEntry(t *testing.T) {
	tests := []struct {
		input    string
		expected Shared
		wantErr  bool
	}{
		{".env:symlink", Shared{Source: ".env", Type: "symlink", Managed: false}, false},
		{".gitignore:copy:managed", Shared{Source: ".gitignore", Type: "copy", Managed: true}, false},
		{"invalid", Shared{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseSharedEntry(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSharedEntry(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("parseSharedEntry(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatSharedEntry(t *testing.T) {
	tests := []struct {
		input    Shared
		expected string
	}{
		{Shared{Source: ".env", Type: "symlink", Managed: false}, ".env:symlink"},
		{Shared{Source: ".gitignore", Type: "copy", Managed: true}, ".gitignore:copy:managed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatSharedEntry(tt.input)
			if result != tt.expected {
				t.Errorf("formatSharedEntry(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

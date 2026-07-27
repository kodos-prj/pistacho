package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kodos-prj/pistacho/pkg/config"
	"github.com/kodos-prj/pistacho/pkg/registry"
)

func TestListCommand_Execute(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "chisel-list-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create test configuration
	cfg := &config.Config{
		RegistryPath: registryPath,
	}

	// Create and populate registry with test packages
	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Add test packages
	pkg1 := &registry.Package{
		Name:        "bash",
		Version:     "5.2.002-1",
		Files:       []string{"/usr/bin/bash", "/usr/share/man/man1/bash.1"},
		Executables: []string{"bash"},
		InstallDate: time.Now().Format(time.RFC3339),
	}

	pkg2 := &registry.Package{
		Name:         "vim",
		Version:      "9.0.000-1",
		Files:        []string{"/usr/bin/vim", "/usr/share/vim"},
		Executables:  []string{"vim", "vi"},
		Dependencies: []string{"ncurses", "glibc"},
		InstallDate:  time.Now().Format(time.RFC3339),
	}

	if err := reg.AddPackage(pkg1); err != nil {
		t.Fatalf("failed to add package: %v", err)
	}
	if err := reg.AddPackage(pkg2); err != nil {
		t.Fatalf("failed to add package: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	// Test list command
	cmd := NewListCommand(cfg)

	t.Run("ListCompact", func(t *testing.T) {
		err := cmd.Execute(false)
		if err != nil {
			t.Errorf("Execute() failed: %v", err)
		}
	})

	t.Run("ListVerbose", func(t *testing.T) {
		err := cmd.Execute(true)
		if err != nil {
			t.Errorf("Execute() with verbose failed: %v", err)
		}
	})
}

func TestListCommand_EmptyRegistry(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "chisel-list-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "registry.json")

	// Create test configuration
	cfg := &config.Config{
		RegistryPath: registryPath,
	}

	// Create empty registry
	_, err = registry.NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Test list command with empty registry
	cmd := NewListCommand(cfg)
	err = cmd.Execute(false)
	if err != nil {
		t.Errorf("Execute() with empty registry failed: %v", err)
	}
}

func TestListCommand_NonExistentRegistry(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "chisel-list-noexist-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "nonexistent-registry.json")

	// Create test configuration
	cfg := &config.Config{
		RegistryPath: registryPath,
	}

	// Test list command with non-existent registry
	cmd := NewListCommand(cfg)
	err = cmd.Execute(false)

	// Should succeed with empty list (registry is created automatically)
	if err != nil {
		t.Errorf("Execute() with non-existent registry failed: %v", err)
	}
}

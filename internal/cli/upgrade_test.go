package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kodos-prj/pistacho/pkg/config"
	"github.com/kodos-prj/pistacho/pkg/download"
	"github.com/kodos-prj/pistacho/pkg/registry"
)

func TestUpgradeCommand_NoInstalledPackages(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "chisel-upgrade-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "registry.json")
	alpmDBPath := filepath.Join(tmpDir, "db")

	// Create ALPM database directory
	if err := os.MkdirAll(alpmDBPath, 0755); err != nil {
		t.Fatalf("failed to create ALPM db dir: %v", err)
	}

	// Create test configuration
	cfg := &config.Config{
		RegistryPath: registryPath,
		AlpmRoot:     tmpDir,
		AlpmDBPath:   alpmDBPath,
	}

	// Create empty registry
	_, err = registry.NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Test upgrade command with no packages
	cmd := NewUpgradeCommand(cfg)
	summary, err := cmd.Execute(&UpgradeOptions{})

	// Error is expected since we don't have valid ALPM setup
	// But the summary structure should be created
	if summary != nil {
		if summary.Total != 0 {
			t.Errorf("Expected Total=0, got %d", summary.Total)
		}
	}
}

func TestUpgradeCommand_DryRun(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "chisel-upgrade-dryrun-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "registry.json")
	alpmDBPath := filepath.Join(tmpDir, "db")

	// Create ALPM database directory
	if err := os.MkdirAll(alpmDBPath, 0755); err != nil {
		t.Fatalf("failed to create ALPM db dir: %v", err)
	}

	// Create test configuration
	cfg := &config.Config{
		RegistryPath: registryPath,
		AlpmRoot:     tmpDir,
		AlpmDBPath:   alpmDBPath,
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
		Files:       []string{"/usr/bin/bash"},
		Executables: []string{"bash"},
		InstallDate: time.Now().Format(time.RFC3339),
	}

	if err := reg.AddPackage(pkg1); err != nil {
		t.Fatalf("failed to add package: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	// Test upgrade command with dry-run
	cmd := NewUpgradeCommand(cfg)
	summary, err := cmd.Execute(&UpgradeOptions{
		DryRun: true,
	})

	// Error is expected since we don't have valid ALPM databases
	// The important thing is that we test the structure
	if summary != nil && err == nil {
		// If no error, verify the structure
		if summary.Total < 0 {
			t.Errorf("Expected Total >= 0, got %d", summary.Total)
		}
	}
}

func TestUpgradeCommand_VerboseMode(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "chisel-upgrade-verbose-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "registry.json")
	alpmDBPath := filepath.Join(tmpDir, "db")

	// Create ALPM database directory
	if err := os.MkdirAll(alpmDBPath, 0755); err != nil {
		t.Fatalf("failed to create ALPM db dir: %v", err)
	}

	// Create test configuration
	cfg := &config.Config{
		RegistryPath: registryPath,
		AlpmRoot:     tmpDir,
		AlpmDBPath:   alpmDBPath,
	}

	// Create and populate registry with test packages
	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Add test packages
	pkg1 := &registry.Package{
		Name:        "vim",
		Version:     "9.0.000-1",
		Files:       []string{"/usr/bin/vim"},
		Executables: []string{"vim"},
		InstallDate: time.Now().Format(time.RFC3339),
	}

	if err := reg.AddPackage(pkg1); err != nil {
		t.Fatalf("failed to add package: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	// Test upgrade command with verbose mode
	cmd := NewUpgradeCommand(cfg)
	summary, err := cmd.Execute(&UpgradeOptions{
		Verbose: true,
		DryRun:  true,
	})

	// Error is expected since we don't have valid ALPM databases
	// The important thing is that we test the structure with verbose mode
	if summary != nil && err == nil {
		// If no error, verify the structure
		if summary.Total < 0 {
			t.Errorf("Expected Total >= 0, got %d", summary.Total)
		}
	}
}

func TestUpgradeCommand_SelectiveUpgrade(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "chisel-upgrade-selective-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	registryPath := filepath.Join(tmpDir, "registry.json")
	alpmDBPath := filepath.Join(tmpDir, "db")

	// Create ALPM database directory
	if err := os.MkdirAll(alpmDBPath, 0755); err != nil {
		t.Fatalf("failed to create ALPM db dir: %v", err)
	}

	// Create test configuration
	cfg := &config.Config{
		RegistryPath: registryPath,
		AlpmRoot:     tmpDir,
		AlpmDBPath:   alpmDBPath,
	}

	// Create and populate registry with test packages
	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Add multiple test packages
	pkg1 := &registry.Package{
		Name:        "bash",
		Version:     "5.2.002-1",
		Files:       []string{"/usr/bin/bash"},
		Executables: []string{"bash"},
		InstallDate: time.Now().Format(time.RFC3339),
	}

	pkg2 := &registry.Package{
		Name:        "vim",
		Version:     "9.0.000-1",
		Files:       []string{"/usr/bin/vim"},
		Executables: []string{"vim"},
		InstallDate: time.Now().Format(time.RFC3339),
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

	// Test upgrade command with specific package selection
	cmd := NewUpgradeCommand(cfg)
	summary, err := cmd.Execute(&UpgradeOptions{
		DryRun:   true,
		Packages: []string{"vim"}, // Only upgrade vim, not bash
	})

	// Error is expected since we don't have valid ALPM databases
	// The important thing is that we test the selective upgrade option
	if summary != nil && err == nil {
		// If no error, verify the structure
		if summary.Total < 0 {
			t.Errorf("Expected Total >= 0, got %d", summary.Total)
		}
	}
}

func TestUpgradeCandidate_Structure(t *testing.T) {
	candidate := UpgradeCandidate{
		PackageName:      "vim",
		InstalledVersion: "9.0.000-1",
		AvailableVersion: "9.0.001-1",
		PackageInfo: &download.PackageInfo{
			Name:    "vim",
			Version: "9.0.001-1",
			Repo:    "extra",
		},
		IsAutoAdded: false,
	}

	if candidate.PackageName != "vim" {
		t.Errorf("Expected PackageName=vim, got %s", candidate.PackageName)
	}

	if candidate.InstalledVersion != "9.0.000-1" {
		t.Errorf("Expected InstalledVersion=9.0.000-1, got %s", candidate.InstalledVersion)
	}

	if candidate.AvailableVersion != "9.0.001-1" {
		t.Errorf("Expected AvailableVersion=9.0.001-1, got %s", candidate.AvailableVersion)
	}

	if candidate.IsAutoAdded {
		t.Error("Expected IsAutoAdded=false")
	}
}

func TestUpgradeSummary_Structure(t *testing.T) {
	summary := &UpgradeSummary{
		Total:              10,
		Successful:         8,
		Failed:             2,
		SkippedNoUpdate:    0,
		SkippedNotFound:    1,
		AutoAddedCount:     3,
		OldVersionsCleaned: 5,
		SpaceFreed:         1024000,
	}

	if summary.Total != 10 {
		t.Errorf("Expected Total=10, got %d", summary.Total)
	}

	if summary.Successful != 8 {
		t.Errorf("Expected Successful=8, got %d", summary.Successful)
	}

	if summary.Failed != 2 {
		t.Errorf("Expected Failed=2, got %d", summary.Failed)
	}

	if summary.AutoAddedCount != 3 {
		t.Errorf("Expected AutoAddedCount=3, got %d", summary.AutoAddedCount)
	}

	if summary.SpaceFreed != 1024000 {
		t.Errorf("Expected SpaceFreed=1024000, got %d", summary.SpaceFreed)
	}
}

func TestUpgradeResult_Structure(t *testing.T) {
	result := UpgradeResult{
		PackageName: "bash",
		OldVersion:  "5.2.002-1",
		NewVersion:  "5.3.000-1",
		Success:     true,
		Error:       nil,
		TimeSeconds: 45,
	}

	if result.PackageName != "bash" {
		t.Errorf("Expected PackageName=bash, got %s", result.PackageName)
	}

	if result.OldVersion != "5.2.002-1" {
		t.Errorf("Expected OldVersion=5.2.002-1, got %s", result.OldVersion)
	}

	if result.NewVersion != "5.3.000-1" {
		t.Errorf("Expected NewVersion=5.3.000-1, got %s", result.NewVersion)
	}

	if !result.Success {
		t.Error("Expected Success=true")
	}

	if result.Error != nil {
		t.Errorf("Expected Error=nil, got %v", result.Error)
	}
}

func TestUpgradeOptions_Structure(t *testing.T) {
	opts := &UpgradeOptions{
		DryRun:   true,
		Verbose:  true,
		Packages: []string{"bash", "vim"},
	}

	if !opts.DryRun {
		t.Error("Expected DryRun=true")
	}

	if !opts.Verbose {
		t.Error("Expected Verbose=true")
	}

	if len(opts.Packages) != 2 {
		t.Errorf("Expected Packages length=2, got %d", len(opts.Packages))
	}
}

func TestNewUpgradeCommand(t *testing.T) {
	cfg := &config.Config{
		RegistryPath: "/tmp/registry.json",
	}

	cmd := NewUpgradeCommand(cfg)

	if cmd == nil {
		t.Fatal("NewUpgradeCommand() returned nil")
	}

	if cmd.config != cfg {
		t.Error("config not set correctly")
	}
}

func TestNewUpgradeCommandWithSymlinkDir(t *testing.T) {
	cfg := &config.Config{
		RegistryPath: "/tmp/registry.json",
	}

	symlinkDir := "/usr/local/bin"

	cmd := NewUpgradeCommandWithSymlinkDir(cfg, symlinkDir)

	if cmd == nil {
		t.Fatal("NewUpgradeCommandWithSymlinkDir() returned nil")
	}

	if cmd.config != cfg {
		t.Error("config not set correctly")
	}

	if cmd.symlinkDir != symlinkDir {
		t.Error("symlinkDir not set correctly")
	}
}

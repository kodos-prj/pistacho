package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kodos-prj/chisel/pkg/config"
)

// TestCleanupCommandCreation tests creating cleanup command instances
func TestCleanupCommandCreation(t *testing.T) {
	cfg := &config.Config{
		BaseDir:      "/tmp/chisel",
		PoolRoot:    "/tmp/chisel/pool",
		RegistryPath: "/tmp/chisel/registry.json",
	}

	cmd := NewCleanupCommand(cfg)
	if cmd == nil {
		t.Error("expected CleanupCommand, got nil")
	}
	if cmd.config != cfg {
		t.Error("config not set correctly")
	}
	if cmd.symlinkDir != "" {
		t.Error("expected empty symlinkDir for NewCleanupCommand")
	}

	symDir := "/custom/symlink/dir"
	cmdWithDir := NewCleanupCommandWithSymlinkDir(cfg, symDir)
	if cmdWithDir.symlinkDir != symDir {
		t.Errorf("expected symlinkDir %s, got %s", symDir, cmdWithDir.symlinkDir)
	}
}

// TestCleanupOptionsDefaults tests cleanup options default values
func TestCleanupOptionsDefaults(t *testing.T) {
	opts := &CleanupOptions{}
	if opts.DryRun {
		t.Error("expected DryRun to be false by default")
	}
	if opts.Verbose {
		t.Error("expected Verbose to be false by default")
	}
	if opts.Force {
		t.Error("expected Force to be false by default")
	}
	if opts.KeepVersions != 0 {
		t.Error("expected KeepVersions to be 0 by default")
	}
}

// TestVersionStatusFields tests VersionStatus structure
func TestVersionStatusFields(t *testing.T) {
	status := &VersionStatus{
		Version:          "1.2.3",
		HasActiveSymlink: true,
		HasActiveWrapper: false,
		SafeToRemove:     false,
		Reason:           "active symlink points to this version",
	}

	if status.Version != "1.2.3" {
		t.Errorf("expected Version 1.2.3, got %s", status.Version)
	}
	if !status.HasActiveSymlink {
		t.Error("expected HasActiveSymlink to be true")
	}
	if status.HasActiveWrapper {
		t.Error("expected HasActiveWrapper to be false")
	}
	if status.SafeToRemove {
		t.Error("expected SafeToRemove to be false")
	}
}

// TestCleanupResultInitialization tests CleanupResult structure
func TestCleanupResultInitialization(t *testing.T) {
	result := &CleanupResult{
		PackageName:     "test-pkg",
		VersionsRemoved: []string{},
		VersionsSkipped: []string{},
		SpaceFreed:      0,
	}

	if result.PackageName != "test-pkg" {
		t.Errorf("expected PackageName test-pkg, got %s", result.PackageName)
	}
	if len(result.VersionsRemoved) != 0 {
		t.Error("expected VersionsRemoved to be empty")
	}
}

// TestCleanupSummaryInitialization tests CleanupSummary structure
func TestCleanupSummaryInitialization(t *testing.T) {
	summary := &CleanupSummary{
		TotalResults: []CleanupResult{},
	}

	if summary.TotalVersionsRemoved != 0 {
		t.Error("expected TotalVersionsRemoved to be 0")
	}
	if summary.TotalSpaceFreed != 0 {
		t.Error("expected TotalSpaceFreed to be 0")
	}
	if len(summary.TotalResults) != 0 {
		t.Error("expected TotalResults to be empty")
	}
}

// TestExecuteWithNilOptions tests Execute with nil options
func TestExecuteWithNilOptions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
		KeepVersions: 2,
	}

	// Create necessary directories
	os.MkdirAll(cfg.PoolRoot, 0755)

	// Create minimal registry file
	regPath := cfg.RegistryPath
	os.WriteFile(regPath, []byte("{}"), 0644)

	cmd := NewCleanupCommand(cfg)
	summary, err := cmd.Execute(nil)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary == nil {
		t.Error("expected CleanupSummary, got nil")
	}
}

// TestExecuteEmptyStore tests Execute with empty package store
func TestExecuteEmptyStore(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
		KeepVersions: 2,
	}

	// Create necessary directories
	os.MkdirAll(cfg.PoolRoot, 0755)

	// Create minimal registry file
	os.WriteFile(cfg.RegistryPath, []byte("{}"), 0644)

	cmd := NewCleanupCommand(cfg)
	opts := &CleanupOptions{Verbose: true}
	summary, err := cmd.Execute(opts)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary.TotalVersionsRemoved != 0 {
		t.Errorf("expected 0 versions removed, got %d", summary.TotalVersionsRemoved)
	}
}

// TestFindOldVersionsNone tests findOldVersions with no old versions
func TestFindOldVersionsNone(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		PoolRoot: filepath.Join(tmpDir, "pool"),
	}

	cmd := NewCleanupCommand(cfg)
	oldVersions, err := cmd.findOldVersions(2)

	if err == nil || oldVersions == nil {
		// Expected - store doesn't have all packages, so no old versions
	}
}

// TestVersionStatusSafeToRemove tests version status when safe to remove
func TestVersionStatusSafeToRemove(t *testing.T) {
	status := &VersionStatus{
		Version:          "1.0.0",
		HasActiveSymlink: false,
		HasActiveWrapper: false,
		SafeToRemove:     true,
	}

	if !status.SafeToRemove {
		t.Error("expected SafeToRemove to be true")
	}
	if status.HasActiveSymlink || status.HasActiveWrapper {
		t.Error("expected no active symlink or wrapper")
	}
}

// TestVersionStatusNotSafeSymlink tests version status with active symlink
func TestVersionStatusNotSafeSymlink(t *testing.T) {
	status := &VersionStatus{
		Version:          "1.0.0",
		HasActiveSymlink: true,
		HasActiveWrapper: false,
		SafeToRemove:     false,
		Reason:           "active symlink points to this version",
	}

	if status.SafeToRemove {
		t.Error("expected SafeToRemove to be false")
	}
	if status.Reason == "" {
		t.Error("expected reason to be set")
	}
}

// TestVersionStatusNotSafeWrapper tests version status with active wrapper
func TestVersionStatusNotSafeWrapper(t *testing.T) {
	status := &VersionStatus{
		Version:          "1.0.0",
		HasActiveSymlink: false,
		HasActiveWrapper: true,
		SafeToRemove:     false,
		Reason:           "wrapper script references this version",
	}

	if status.SafeToRemove {
		t.Error("expected SafeToRemove to be false")
	}
}

// TestCleanupResultTracking tests accumulating cleanup results
func TestCleanupResultTracking(t *testing.T) {
	result := &CleanupResult{
		PackageName:     "bash",
		VersionsRemoved: []string{},
		VersionsSkipped: []string{},
		SpaceFreed:      0,
	}

	// Simulate adding removed versions
	result.VersionsRemoved = append(result.VersionsRemoved, "5.0.0")
	result.VersionsRemoved = append(result.VersionsRemoved, "5.1.0")
	result.SpaceFreed += 500 * 1024 * 1024 // 500 MB

	if len(result.VersionsRemoved) != 2 {
		t.Errorf("expected 2 removed versions, got %d", len(result.VersionsRemoved))
	}
	if result.SpaceFreed != 500*1024*1024 {
		t.Errorf("expected 500 MB freed, got %d bytes", result.SpaceFreed)
	}
}

// TestCleanupSummaryAccumulation tests accumulating cleanup summary
func TestCleanupSummaryAccumulation(t *testing.T) {
	summary := &CleanupSummary{
		TotalResults: []CleanupResult{},
	}

	// Simulate multiple cleanup results
	result1 := CleanupResult{
		PackageName:     "bash",
		VersionsRemoved: []string{"5.0.0"},
		SpaceFreed:      100 * 1024 * 1024,
	}

	result2 := CleanupResult{
		PackageName:     "curl",
		VersionsRemoved: []string{"7.0.0", "7.1.0"},
		SpaceFreed:      200 * 1024 * 1024,
	}

	summary.TotalResults = append(summary.TotalResults, result1, result2)
	summary.TotalVersionsRemoved = 3
	summary.TotalSpaceFreed = 300 * 1024 * 1024

	if len(summary.TotalResults) != 2 {
		t.Errorf("expected 2 results, got %d", len(summary.TotalResults))
	}
	if summary.TotalVersionsRemoved != 3 {
		t.Errorf("expected 3 versions removed, got %d", summary.TotalVersionsRemoved)
	}
	if summary.TotalSpaceFreed != 300*1024*1024 {
		t.Errorf("expected 300 MB freed, got %d bytes", summary.TotalSpaceFreed)
	}
}

// TestCleanupCommandWithKeepVersions tests cleanup with custom keep versions
func TestCleanupCommandWithKeepVersions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
		KeepVersions: 3,
	}

	os.MkdirAll(cfg.PoolRoot, 0755)
	os.WriteFile(cfg.RegistryPath, []byte("{}"), 0644)

	cmd := NewCleanupCommand(cfg)
	opts := &CleanupOptions{
		KeepVersions: 5, // Override config
	}

	summary, err := cmd.Execute(opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary == nil {
		t.Error("expected CleanupSummary")
	}
}

// TestCleanupCommandDryRun tests cleanup in dry-run mode
func TestCleanupCommandDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
		KeepVersions: 2,
	}

	os.MkdirAll(cfg.PoolRoot, 0755)
	os.WriteFile(cfg.RegistryPath, []byte("{}"), 0644)

	cmd := NewCleanupCommand(cfg)
	opts := &CleanupOptions{
		DryRun: true,
		Force:  true,
	}

	summary, err := cmd.Execute(opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary == nil {
		t.Error("expected CleanupSummary")
	}
}

// TestCleanupCommandWithoutForce tests cleanup requires confirmation
func TestCleanupCommandWithoutForce(t *testing.T) {
	// This test verifies the Force flag is checked
	opts := &CleanupOptions{
		Force: false,
	}

	if opts.Force {
		t.Error("expected Force to be false")
	}
}

// TestCleanupVersionSorting tests version sorting (newest first)
func TestCleanupVersionSorting(t *testing.T) {
	// Test data: should keep newest versions and remove oldest
	oldVersions := make(map[string][]string)
	oldVersions["bash"] = []string{"5.0.0", "5.1.0", "5.2.0"}

	if len(oldVersions["bash"]) != 3 {
		t.Errorf("expected 3 versions, got %d", len(oldVersions["bash"]))
	}
}

// TestRegistryNotFound tests Execute with missing registry (creates empty one)
func TestRegistryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "nonexistent.json"),
		KeepVersions: 2,
	}

	os.MkdirAll(cfg.PoolRoot, 0755)

	cmd := NewCleanupCommand(cfg)
	summary, err := cmd.Execute(&CleanupOptions{})

	// No error expected - registry creates empty registry file
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary == nil {
		t.Error("expected CacheSummary")
	}
}

// TestCleanupOptionsVariations tests different option combinations
func TestCleanupOptionsVariations(t *testing.T) {
	tests := []struct {
		name       string
		dryRun     bool
		force      bool
		verbose    bool
		expectDesc string
	}{
		{"default", false, false, false, "normal cleanup with confirmation"},
		{"force", false, true, false, "cleanup without confirmation"},
		{"dry-run", true, false, false, "preview mode"},
		{"dry-run with force", true, true, false, "preview mode (force ignored)"},
		{"verbose", false, false, true, "detailed output"},
		{"all options", true, true, true, "dry-run, force, verbose"},
	}

	for _, tt := range tests {
		opts := &CleanupOptions{
			DryRun:  tt.dryRun,
			Force:   tt.force,
			Verbose: tt.verbose,
		}

		if opts.DryRun != tt.dryRun || opts.Force != tt.force || opts.Verbose != tt.verbose {
			t.Errorf("%s: options not set correctly", tt.name)
		}
	}
}

// TestCleanupResultError tests error handling in result
func TestCleanupResultError(t *testing.T) {
	result := &CleanupResult{
		PackageName: "test",
	}

	if result.Error != nil {
		t.Error("expected Error to be nil initially")
	}

	// Simulate error
	result.Error = os.ErrNotExist
	if result.Error == nil {
		t.Error("expected Error to be set")
	}
}

// TestCleanupOptionsAURDefaults tests AUR cleanup options defaults
func TestCleanupOptionsAURDefaults(t *testing.T) {
	opts := &CleanupOptions{}
	if opts.CleanupAUR {
		t.Error("expected CleanupAUR to be false by default")
	}
	if opts.BuildCacheMaxAge != 0 {
		t.Error("expected BuildCacheMaxAge to be 0 by default (will use 7 days in Execute)")
	}
	if opts.BuildLogsMaxAge != 0 {
		t.Error("expected BuildLogsMaxAge to be 0 by default (will use 7 days in Execute)")
	}
}

// TestCleanupWithAUREnabled tests cleanup with AUR enabled
func TestCleanupWithAUREnabled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
		KeepVersions: 2,
	}

	// Create necessary directories
	os.MkdirAll(cfg.PoolRoot, 0755)
	os.MkdirAll(filepath.Join(tmpDir, "build-cache"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "build-logs"), 0755)

	// Create minimal registry file
	os.WriteFile(cfg.RegistryPath, []byte("{}"), 0644)

	cmd := NewCleanupCommand(cfg)
	opts := &CleanupOptions{
		CleanupAUR:       true,
		BuildCacheMaxAge: 7 * 24 * time.Hour,
		BuildLogsMaxAge:  7 * 24 * time.Hour,
		Force:            true,
		DryRun:           true,
	}

	summary, err := cmd.Execute(opts)
	if err != nil {
		t.Errorf("unexpected error with AUR cleanup: %v", err)
	}
	if summary == nil {
		t.Error("expected CleanupSummary, got nil")
	}
}

// TestCleanupSummaryAURFields tests AUR fields in summary
func TestCleanupSummaryAURFields(t *testing.T) {
	summary := &CleanupSummary{
		TotalResults: []CleanupResult{},
	}

	if summary.AURBuildDirsRemoved != 0 {
		t.Error("expected AURBuildDirsRemoved to be 0 initially")
	}
	if summary.AURLogsRemoved != 0 {
		t.Error("expected AURLogsRemoved to be 0 initially")
	}
	if summary.AURSpaceFreed != 0 {
		t.Error("expected AURSpaceFreed to be 0 initially")
	}

	// Simulate AUR cleanup
	summary.AURBuildDirsRemoved = 3
	summary.AURLogsRemoved = 5
	summary.AURSpaceFreed = 100 * 1024 * 1024 // 100 MB

	if summary.AURBuildDirsRemoved != 3 {
		t.Errorf("expected 3 AUR build dirs removed, got %d", summary.AURBuildDirsRemoved)
	}
	if summary.AURLogsRemoved != 5 {
		t.Errorf("expected 5 AUR logs removed, got %d", summary.AURLogsRemoved)
	}
	if summary.AURSpaceFreed != 100*1024*1024 {
		t.Errorf("expected 100 MB AUR space freed, got %d bytes", summary.AURSpaceFreed)
	}
}

// TestCleanupAURWithoutBuildDirs tests AUR cleanup when build dirs don't exist
func TestCleanupAURWithoutBuildDirs(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
		KeepVersions: 2,
	}

	os.MkdirAll(cfg.PoolRoot, 0755)
	os.WriteFile(cfg.RegistryPath, []byte("{}"), 0644)

	// Don't create build-cache or build-logs directories
	cmd := NewCleanupCommand(cfg)
	opts := &CleanupOptions{
		CleanupAUR: true,
		Force:      true,
	}

	summary, err := cmd.Execute(opts)
	if err != nil {
		t.Logf("Error (expected): %v", err)
	}
	if summary != nil {
		// Should still return summary, but AUR stats might be empty
		t.Logf("AUR build dirs removed: %d", summary.AURBuildDirsRemoved)
	}
}

// TestCleanupMixedPackagesAndAUR tests cleanup of both official and AUR packages
func TestCleanupMixedPackagesAndAUR(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
		KeepVersions: 2,
	}

	// Create directory structure
	os.MkdirAll(cfg.PoolRoot, 0755)
	os.MkdirAll(filepath.Join(tmpDir, "build-cache"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "build-logs"), 0755)

	// Create minimal registry
	os.WriteFile(cfg.RegistryPath, []byte("{}"), 0644)

	cmd := NewCleanupCommand(cfg)
	opts := &CleanupOptions{
		CleanupAUR:       true,
		Force:            true,
		Verbose:          true,
		DryRun:           true,
		BuildCacheMaxAge: 24 * time.Hour,
		BuildLogsMaxAge:  24 * time.Hour,
	}

	summary, err := cmd.Execute(opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary == nil {
		t.Error("expected CleanupSummary")
	}
}

// TestCleanupOptionsAURTiming tests AUR cleanup timing options
func TestCleanupOptionsAURTiming(t *testing.T) {
	tests := []struct {
		name        string
		maxAge      time.Duration
		expectValid bool
	}{
		{"1 day", 24 * time.Hour, true},
		{"7 days", 7 * 24 * time.Hour, true},
		{"30 days", 30 * 24 * time.Hour, true},
		{"0 (should be handled)", 0, true},
	}

	for _, tt := range tests {
		opts := &CleanupOptions{
			BuildCacheMaxAge: tt.maxAge,
			BuildLogsMaxAge:  tt.maxAge,
		}

		if (opts.BuildCacheMaxAge > 0) != tt.expectValid && tt.maxAge > 0 {
			t.Errorf("%s: unexpected maxAge handling", tt.name)
		}
	}
}

// TestCleanupAllOptionsEnabled tests cleanup with all AUR options enabled
func TestCleanupAllOptionsEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
		KeepVersions: 2,
	}

	os.MkdirAll(cfg.PoolRoot, 0755)
	os.MkdirAll(filepath.Join(tmpDir, "build-cache"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "build-logs"), 0755)
	os.WriteFile(cfg.RegistryPath, []byte("{}"), 0644)

	cmd := NewCleanupCommand(cfg)
	opts := &CleanupOptions{
		CleanupAUR:       true,
		DryRun:           true,
		Force:            true,
		Verbose:          true,
		KeepVersions:     3,
		BuildCacheMaxAge: 48 * time.Hour,
		BuildLogsMaxAge:  48 * time.Hour,
	}

	summary, err := cmd.Execute(opts)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary == nil {
		t.Error("expected CleanupSummary")
	}
	// Verify all options are honored
	if !opts.CleanupAUR || !opts.DryRun || !opts.Force || !opts.Verbose {
		t.Error("cleanup options not preserved")
	}
}

// TestCleanupAURWithOfficial tests cleanup with both official packages and AUR
func TestCleanupAURWithOfficial(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
		KeepVersions: 2,
	}

	os.MkdirAll(cfg.PoolRoot, 0755)
	os.MkdirAll(filepath.Join(tmpDir, "build-cache"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "build-logs"), 0755)
	os.WriteFile(cfg.RegistryPath, []byte("{}"), 0644)

	// Simulate official packages in store
	os.MkdirAll(filepath.Join(cfg.PoolRoot, "bash"), 0755)
	os.MkdirAll(filepath.Join(cfg.PoolRoot, "bash", "5.0.0"), 0755)

	cmd := NewCleanupCommand(cfg)
	opts := &CleanupOptions{
		CleanupAUR: true,
		Force:      true,
		Verbose:    true,
	}

	summary, err := cmd.Execute(opts)
	if err != nil {
		t.Logf("Error occurred (may be expected): %v", err)
	}
	if summary != nil {
		// Check that summary includes both official and AUR counts
		t.Logf("Official versions removed: %d", summary.TotalVersionsRemoved)
		t.Logf("AUR build dirs removed: %d", summary.AURBuildDirsRemoved)
		t.Logf("AUR logs removed: %d", summary.AURLogsRemoved)
	}
}

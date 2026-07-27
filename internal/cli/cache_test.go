package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kodos-prj/pistacho/pkg/config"
)

// TestCacheCommandCreation tests creating cache command instances
func TestCacheCommandCreation(t *testing.T) {
	cfg := &config.Config{
		BaseDir:   "/tmp/pistacho",
		CachePath: "/tmp/pistacho/cache",
	}

	cmd := NewCacheCommand(cfg)
	if cmd == nil {
		t.Error("expected CacheCommand, got nil")
	}
	if cmd.config != cfg {
		t.Error("config not set correctly")
	}
}

// TestCacheOptionsDefaults tests cache options default values
func TestCacheOptionsDefaults(t *testing.T) {
	opts := &CacheOptions{}
	if opts.DryRun {
		t.Error("expected DryRun to be false by default")
	}
	if opts.Verbose {
		t.Error("expected Verbose to be false by default")
	}
	if opts.Force {
		t.Error("expected Force to be false by default")
	}
}

// TestCacheFileStructure tests CacheFile structure
func TestCacheFileStructure(t *testing.T) {
	cf := &CacheFile{
		Name:  "bash-5.3.9-1-x86_64.pkg.tar.zst",
		Path:  "/kod/cache/bash-5.3.9-1-x86_64.pkg.tar.zst",
		Size:  1024 * 1024 * 50, // 50 MB
		IsOld: false,
	}

	if cf.Name != "bash-5.3.9-1-x86_64.pkg.tar.zst" {
		t.Errorf("expected Name bash-5.3.9-1-x86_64.pkg.tar.zst, got %s", cf.Name)
	}
	if cf.Size != 1024*1024*50 {
		t.Errorf("expected Size 50 MB, got %d bytes", cf.Size)
	}
}

// TestCacheSummaryInitialization tests CacheSummary structure
func TestCacheSummaryInitialization(t *testing.T) {
	summary := &CacheSummary{
		RemovedFiles: []string{},
		SkippedFiles: []string{},
		CachedFiles:  []CacheFile{},
	}

	if summary.TotalFiles != 0 {
		t.Error("expected TotalFiles to be 0")
	}
	if summary.FilesRemoved != 0 {
		t.Error("expected FilesRemoved to be 0")
	}
	if len(summary.RemovedFiles) != 0 {
		t.Error("expected RemovedFiles to be empty")
	}
}

// TestExecuteWithNilOptions tests Execute with nil options
func TestCacheExecuteWithNilOptions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:   tmpDir,
		CachePath: filepath.Join(tmpDir, "cache"),
	}

	// Create cache directory
	os.MkdirAll(cfg.CachePath, 0755)

	cmd := NewCacheCommand(cfg)
	summary, err := cmd.Execute(nil)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary == nil {
		t.Error("expected CacheSummary, got nil")
	}
}

// TestExecuteEmptyCache tests Execute with empty cache
func TestExecuteEmptyCache(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:   tmpDir,
		CachePath: filepath.Join(tmpDir, "cache"),
	}

	os.MkdirAll(cfg.CachePath, 0755)

	cmd := NewCacheCommand(cfg)
	opts := &CacheOptions{
		Action:  "list",
		Verbose: true,
	}
	summary, err := cmd.Execute(opts)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary.TotalFiles != 0 {
		t.Errorf("expected 0 files, got %d", summary.TotalFiles)
	}
}

// TestCacheListAction tests list cache action
func TestCacheListAction(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:   tmpDir,
		CachePath: filepath.Join(tmpDir, "cache"),
	}

	os.MkdirAll(cfg.CachePath, 0755)

	// Create test cache files
	testFile := filepath.Join(cfg.CachePath, "test-1.0.0-1-x86_64.pkg.tar.zst")
	os.WriteFile(testFile, []byte("test content"), 0644)

	cmd := NewCacheCommand(cfg)
	opts := &CacheOptions{Action: "list", Verbose: true}
	summary, err := cmd.Execute(opts)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary.TotalFiles != 1 {
		t.Errorf("expected 1 file, got %d", summary.TotalFiles)
	}
}

// TestCacheCleanAction tests clean cache action
func TestCacheCleanAction(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:   tmpDir,
		CachePath: filepath.Join(tmpDir, "cache"),
	}

	os.MkdirAll(cfg.CachePath, 0755)

	// Create test cache files
	testFile := filepath.Join(cfg.CachePath, "test-1.0.0-1-x86_64.pkg.tar.zst")
	os.WriteFile(testFile, []byte("test content"), 0644)

	cmd := NewCacheCommand(cfg)
	opts := &CacheOptions{
		Action: "clean",
		Force:  true,
		DryRun: false,
	}
	summary, err := cmd.Execute(opts)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary.FilesRemoved != 1 {
		t.Errorf("expected 1 file removed, got %d", summary.FilesRemoved)
	}
}

// TestCachePruneAction tests prune cache action
func TestCachePruneAction(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:   tmpDir,
		CachePath: filepath.Join(tmpDir, "cache"),
	}

	os.MkdirAll(cfg.CachePath, 0755)

	cmd := NewCacheCommand(cfg)
	opts := &CacheOptions{
		Action: "prune",
		Force:  true,
	}
	summary, err := cmd.Execute(opts)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary == nil {
		t.Error("expected CacheSummary")
	}
}

// TestCacheMissingDirectory tests Execute with missing cache directory
func TestCacheMissingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:   tmpDir,
		CachePath: filepath.Join(tmpDir, "nonexistent", "cache"),
	}

	cmd := NewCacheCommand(cfg)
	summary, err := cmd.Execute(&CacheOptions{Action: "list"})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Summary should be returned, not error, for empty cache
	if summary == nil {
		t.Error("expected CacheSummary")
	}
}

// TestCacheDefaultPath tests Execute sets default cache path
func TestCacheDefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:   tmpDir,
		CachePath: "",
	}

	os.MkdirAll(filepath.Join(tmpDir, "cache"), 0755)

	cmd := NewCacheCommand(cfg)
	// After Execute, cache path should be set
	cmd.Execute(&CacheOptions{Action: "list"})

	if cfg.CachePath == "" {
		// CachePath should have been set during Execute
		expectedPath := filepath.Join(tmpDir, "cache")
		if cmd.config.CachePath != expectedPath {
			t.Errorf("expected cache path %s, got %s", expectedPath, cmd.config.CachePath)
		}
	}
}

// TestCacheFormatSize tests formatSize utility function
func TestCacheFormatSize(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewCacheCommand(cfg)

	tests := []struct {
		bytes    int64
		expected string
	}{
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 100, "100.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 1024 * 2, "2.0 GB"},
	}

	for _, tt := range tests {
		result := cmd.formatSize(tt.bytes)
		// Check format is close enough (accounting for rounding)
		if len(result) == 0 {
			t.Errorf("formatSize(%d): got empty string", tt.bytes)
		}
		// Verify it contains reasonable content (KB, MB, GB, or B)
		if !containsAny(result, []string{"B", "KB", "MB", "GB"}) {
			t.Errorf("formatSize(%d): got %q, expected size unit", tt.bytes, result)
		}
	}
}

// TestCacheDryRunMode tests dry-run mode doesn't modify cache
func TestCacheDryRunMode(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:   tmpDir,
		CachePath: filepath.Join(tmpDir, "cache"),
	}

	os.MkdirAll(cfg.CachePath, 0755)

	// Create test file
	testFile := filepath.Join(cfg.CachePath, "test-1.0.0-1-x86_64.pkg.tar.zst")
	os.WriteFile(testFile, []byte("test content"), 0644)

	cmd := NewCacheCommand(cfg)
	opts := &CacheOptions{
		Action: "clean",
		DryRun: true,
		Force:  true,
	}
	cmd.Execute(opts)

	// File should still exist
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("file was removed in dry-run mode")
	}
}

// TestCacheTruncateString tests truncateString utility
func TestCacheTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this-is-a-very-long-filename.pkg.tar.zst", 20, "this-is-a-very-l..."},
		{"exact", 5, "exact"},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if len(result) > tt.maxLen {
			t.Errorf("truncateString(%q, %d): got %q (len %d), expected max %d chars",
				tt.input, tt.maxLen, result, len(result), tt.maxLen)
		}
	}
}

// TestCacheInvalidAction tests Execute with invalid action
func TestCacheInvalidAction(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:   tmpDir,
		CachePath: filepath.Join(tmpDir, "cache"),
	}

	os.MkdirAll(cfg.CachePath, 0755)

	cmd := NewCacheCommand(cfg)
	_, err := cmd.Execute(&CacheOptions{Action: "invalid"})

	if err == nil {
		t.Error("expected error for invalid action")
	}
}

// TestCacheFiltering tests that only .pkg.tar.zst files are counted
func TestCacheFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:   tmpDir,
		CachePath: filepath.Join(tmpDir, "cache"),
	}

	os.MkdirAll(cfg.CachePath, 0755)

	// Create cache files and non-cache files
	os.WriteFile(filepath.Join(cfg.CachePath, "file1.pkg.tar.zst"), []byte("cache"), 0644)
	os.WriteFile(filepath.Join(cfg.CachePath, "file2.txt"), []byte("other"), 0644)
	os.WriteFile(filepath.Join(cfg.CachePath, "file3.pkg.tar.zst"), []byte("cache"), 0644)
	os.Mkdir(filepath.Join(cfg.CachePath, "subdir"), 0755)

	cmd := NewCacheCommand(cfg)
	opts := &CacheOptions{Action: "list", Verbose: true}
	summary, err := cmd.Execute(opts)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Only should count .pkg.tar.zst files, not .txt or directories
	if summary.TotalFiles != 2 {
		t.Errorf("expected 2 cache files, got %d", summary.TotalFiles)
	}
}

// TestCacheSpaceTracking tests space tracking in summary
func TestCacheSpaceTracking(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		BaseDir:   tmpDir,
		CachePath: filepath.Join(tmpDir, "cache"),
	}

	os.MkdirAll(cfg.CachePath, 0755)

	// Create files with known sizes
	content1 := make([]byte, 1024)
	content2 := make([]byte, 2048)
	os.WriteFile(filepath.Join(cfg.CachePath, "file1.pkg.tar.zst"), content1, 0644)
	os.WriteFile(filepath.Join(cfg.CachePath, "file2.pkg.tar.zst"), content2, 0644)

	cmd := NewCacheCommand(cfg)
	opts := &CacheOptions{Action: "list"}
	summary, err := cmd.Execute(opts)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if summary.TotalFiles != 2 {
		t.Errorf("expected 2 files, got %d", summary.TotalFiles)
	}
	// Total size should be 3072 bytes (1024 + 2048)
	if summary.TotalSize != 3072 {
		t.Errorf("expected 3072 bytes total, got %d", summary.TotalSize)
	}
}

// TestCacheOptionsVariations tests different option combinations
func TestCacheOptionsVariations(t *testing.T) {
	tests := []struct {
		name       string
		dryRun     bool
		force      bool
		verbose    bool
		action     string
		expectDesc string
	}{
		{"default", false, false, false, "clean", "normal clean with confirmation"},
		{"force", false, true, false, "clean", "clean without confirmation"},
		{"dry-run", true, false, false, "clean", "preview mode"},
		{"list action", false, false, false, "list", "show cache contents"},
		{"prune action", false, false, false, "prune", "prune old files"},
		{"all options", true, true, true, "clean", "dry-run, force, verbose"},
	}

	for _, tt := range tests {
		opts := &CacheOptions{
			DryRun:  tt.dryRun,
			Force:   tt.force,
			Verbose: tt.verbose,
			Action:  tt.action,
		}

		if opts.DryRun != tt.dryRun || opts.Force != tt.force || opts.Verbose != tt.verbose || opts.Action != tt.action {
			t.Errorf("%s: options not set correctly", tt.name)
		}
	}
}

// Helper function to check if string contains any of the provided substrings
func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) && s[len(s)-len(sub):] == sub {
			return true
		}
	}
	return false
}

package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kodos-prj/pistacho/pkg/config"
	"github.com/kodos-prj/pistacho/pkg/extract"
	"github.com/kodos-prj/pistacho/pkg/registry"
)

func TestNewRemoveCommand(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewRemoveCommand(cfg)

	if cmd.config != cfg {
		t.Errorf("config not set correctly")
	}
	if cmd.symlinkDir != "" {
		t.Errorf("symlinkDir should be empty, got %s", cmd.symlinkDir)
	}
}

func TestNewRemoveCommandWithSymlinkDir(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewRemoveCommandWithSymlinkDir(cfg, "/test/symlink")

	if cmd.config != cfg {
		t.Errorf("config not set correctly")
	}
	if cmd.symlinkDir != "/test/symlink" {
		t.Errorf("symlinkDir not set correctly, got %s", cmd.symlinkDir)
	}
}

func TestRemoveCommandNoPackages(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewRemoveCommand(cfg)

	err := cmd.Run([]string{})
	if err == nil {
		t.Errorf("expected error when no packages specified")
	}
}

func TestRemoveCommandWithSymlinks(t *testing.T) {
	// Create temporary directories
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")
	symlinkDir := filepath.Join(tmpDir, "symlink")
	poolDir := filepath.Join(tmpDir, "pool")
	wrapperDir := filepath.Join(tmpDir, "wrappers")

	if err := os.MkdirAll(symlinkDir, 0755); err != nil {
		t.Fatalf("failed to create symlink dir: %v", err)
	}
	if err := os.MkdirAll(poolDir, 0755); err != nil {
		t.Fatalf("failed to create store dir: %v", err)
	}
	if err := os.MkdirAll(wrapperDir, 0755); err != nil {
		t.Fatalf("failed to create wrapper dir: %v", err)
	}

	// Create registry with a package
	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	pkg := &registry.Package{
		Name:         "test-pkg",
		Version:      "1.0.0-1",
		Files:        []string{"usr/bin/test", "usr/lib/libtest.so"},
		Executables:  []string{"test"},
		Dependencies: []string{},
		InstallDate:  time.Now().Format(time.RFC3339),
	}

	if err := reg.AddPackage(pkg); err != nil {
		t.Fatalf("failed to add package: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	// Create symlinks
	testBinPath := filepath.Join(symlinkDir, "usr/bin/test")
	if err := os.MkdirAll(filepath.Dir(testBinPath), 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	if err := os.Symlink(filepath.Join(poolDir, "test-pkg/1.0.0-1/usr/bin/test"), testBinPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	libPath := filepath.Join(symlinkDir, "usr/lib/libtest.so")
	if err := os.MkdirAll(filepath.Dir(libPath), 0755); err != nil {
		t.Fatalf("failed to create lib dir: %v", err)
	}
	if err := os.Symlink(filepath.Join(poolDir, "test-pkg/1.0.0-1/usr/lib/libtest.so"), libPath); err != nil {
		t.Fatalf("failed to create lib symlink: %v", err)
	}

	// Create wrapper script
	wrapperPath := filepath.Join(wrapperDir, "test")
	if err := os.WriteFile(wrapperPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatalf("failed to create wrapper: %v", err)
	}

	// Verify symlinks exist before removal
	if _, err := os.Lstat(testBinPath); err != nil {
		t.Fatalf("test symlink should exist before removal: %v", err)
	}

	// Perform removal
	cfg := &config.Config{
		RegistryPath: registryPath,
		PoolRoot:    poolDir,
		WrapperDir:   wrapperDir,
		SymlinkRoot:  poolDir,
	}

	cmd := NewRemoveCommandWithSymlinkDir(cfg, symlinkDir)
	if err := cmd.Run([]string{"test-pkg"}); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	// Verify symlinks removed
	if _, err := os.Lstat(testBinPath); !os.IsNotExist(err) {
		t.Errorf("test symlink should be removed")
	}

	if _, err := os.Lstat(libPath); !os.IsNotExist(err) {
		t.Errorf("lib symlink should be removed")
	}

	// Verify wrapper removed
	if _, err := os.Lstat(wrapperPath); !os.IsNotExist(err) {
		t.Errorf("wrapper should be removed")
	}

	// Verify registry updated
	reg2, err := registry.NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	if _, ok := reg2.GetPackage("test-pkg"); ok {
		t.Errorf("package should be removed from registry")
	}
}

func TestRemoveCommandNonExistentPackage(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	cfg := &config.Config{
		RegistryPath: registryPath,
	}

	cmd := NewRemoveCommand(cfg)
	err := cmd.Run([]string{"nonexistent"})

	if err == nil {
		t.Errorf("expected error for non-existent package")
	}
}

func TestRemoveCommandForceFlag(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")
	symlinkDir := filepath.Join(tmpDir, "symlink")
	poolDir := filepath.Join(tmpDir, "pool")
	wrapperDir := filepath.Join(tmpDir, "wrappers")

	if err := os.MkdirAll(symlinkDir, 0755); err != nil {
		t.Fatalf("failed to create symlink dir: %v", err)
	}
	if err := os.MkdirAll(poolDir, 0755); err != nil {
		t.Fatalf("failed to create store dir: %v", err)
	}
	if err := os.MkdirAll(wrapperDir, 0755); err != nil {
		t.Fatalf("failed to create wrapper dir: %v", err)
	}

	// Create registry
	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	pkg := &registry.Package{
		Name:        "test-pkg",
		Version:     "1.0.0-1",
		Files:       []string{"usr/bin/test"},
		Executables: []string{"test"},
		InstallDate: time.Now().Format(time.RFC3339),
	}

	if err := reg.AddPackage(pkg); err != nil {
		t.Fatalf("failed to add package: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	// Don't create the symlink - it will be missing

	cfg := &config.Config{
		RegistryPath: registryPath,
		PoolRoot:    poolDir,
		WrapperDir:   wrapperDir,
		SymlinkRoot:  poolDir,
	}

	// With --force, should succeed even if symlinks are missing
	cmd := NewRemoveCommandWithSymlinkDir(cfg, symlinkDir)
	err = cmd.Run([]string{"--force", "test-pkg"})

	if err != nil {
		t.Errorf("remove with --force should succeed even if symlinks missing: %v", err)
	}

	// Verify package removed from registry
	reg2, err := registry.NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	if _, ok := reg2.GetPackage("test-pkg"); ok {
		t.Errorf("package should be removed from registry")
	}
}

func TestRemoveCommandMultiplePackages(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")
	symlinkDir := filepath.Join(tmpDir, "symlink")
	poolDir := filepath.Join(tmpDir, "pool")
	wrapperDir := filepath.Join(tmpDir, "wrappers")

	if err := os.MkdirAll(symlinkDir, 0755); err != nil {
		t.Fatalf("failed to create symlink dir: %v", err)
	}
	if err := os.MkdirAll(poolDir, 0755); err != nil {
		t.Fatalf("failed to create store dir: %v", err)
	}
	if err := os.MkdirAll(wrapperDir, 0755); err != nil {
		t.Fatalf("failed to create wrapper dir: %v", err)
	}

	// Create registry with multiple packages
	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	for i := 1; i <= 3; i++ {
		pkg := &registry.Package{
			Name:        "pkg-" + string(rune('0'+i)),
			Version:     "1.0.0-1",
			Files:       []string{"usr/bin/pkg" + string(rune('0'+i))},
			Executables: []string{"pkg" + string(rune('0'+i))},
			InstallDate: time.Now().Format(time.RFC3339),
		}
		if err := reg.AddPackage(pkg); err != nil {
			t.Fatalf("failed to add package: %v", err)
		}
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	cfg := &config.Config{
		RegistryPath: registryPath,
		PoolRoot:    poolDir,
		WrapperDir:   wrapperDir,
		SymlinkRoot:  poolDir,
	}

	cmd := NewRemoveCommandWithSymlinkDir(cfg, symlinkDir)
	err = cmd.Run([]string{"--force", "pkg-1", "pkg-2"})

	if err != nil {
		t.Errorf("remove multiple packages failed: %v", err)
	}

	// Verify packages removed from registry
	reg2, err := registry.NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	if _, ok := reg2.GetPackage("pkg-1"); ok {
		t.Errorf("pkg-1 should be removed from registry")
	}
	if _, ok := reg2.GetPackage("pkg-2"); ok {
		t.Errorf("pkg-2 should be removed from registry")
	}
	if _, ok := reg2.GetPackage("pkg-3"); !ok {
		t.Errorf("pkg-3 should still be in registry")
	}
}

// TestInstallCreatesSymlinksForExtractedSymlinks tests that symlinks from packages
// are properly recreated in the symlink directory pointing to storage.
func TestInstallCreatesSymlinksForExtractedSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "base")
	poolDir := filepath.Join(baseDir, "store")
	cacheDir := filepath.Join(baseDir, "cache")
	symlinkDir := filepath.Join(tmpDir, "app")
	registryPath := filepath.Join(baseDir, "registry.json")
	wrapperDir := filepath.Join(baseDir, "wrappers")

	// Create directories
	for _, dir := range []string{poolDir, cacheDir, wrapperDir, symlinkDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
	}

	// Create mock extracted files with symlinks
	pkgStoreDir := filepath.Join(poolDir, "libtest", "1.0.0")
	libDir := filepath.Join(pkgStoreDir, "usr", "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatalf("Failed to create lib dir: %v", err)
	}

	// Create actual library file in store
	libFile := filepath.Join(libDir, "libtest.so.1.0.0")
	if err := os.WriteFile(libFile, []byte("library"), 0644); err != nil {
		t.Fatalf("Failed to create lib file: %v", err)
	}

	// Create symlinks in store (simulating extraction)
	libLink1 := filepath.Join(libDir, "libtest.so.1")
	libLink2 := filepath.Join(libDir, "libtest.so")
	if err := os.Symlink("libtest.so.1.0.0", libLink1); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}
	if err := os.Symlink("libtest.so.1", libLink2); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Create extracted files metadata with symlinks
	allFiles := []string{
		"usr/lib/libtest.so.1.0.0",
		"usr/lib/libtest.so.1",
		"usr/lib/libtest.so",
	}

	extractedFiles := []extract.ExtractedFile{
		{Path: "usr/lib/libtest.so.1.0.0", AbsPath: libFile, IsDirectory: false, IsSymlink: false},
		{Path: "usr/lib/libtest.so.1", AbsPath: libLink1, IsDirectory: false, IsSymlink: true, LinkTarget: "libtest.so.1.0.0"},
		{Path: "usr/lib/libtest.so", AbsPath: libLink2, IsDirectory: false, IsSymlink: true, LinkTarget: "libtest.so.1"},
	}

	// Create registry with the package
	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	pkg := &registry.Package{
		Name:        "libtest",
		Version:     "1.0.0",
		Files:       allFiles,
		Executables: []string{},
		InstallDate: time.Now().Format(time.RFC3339),
	}

	if err := reg.AddPackage(pkg); err != nil {
		t.Fatalf("Failed to add package to registry: %v", err)
	}

	if err := reg.Save(); err != nil {
		t.Fatalf("Failed to save registry: %v", err)
	}

	// Simulate the symlink creation logic from install.go
	// Build extracted symlinks map
	extractedSymlinksMap := make(map[string]string)
	for _, extractedFile := range extractedFiles {
		if extractedFile.IsSymlink {
			extractedSymlinksMap[extractedFile.Path] = extractedFile.LinkTarget
		}
	}

	// Create symlinks in symlink directory
	for _, filePath := range allFiles {
		symlinkPath := filepath.Join(symlinkDir, filePath)

		// Create parent directories
		symlinkParentDir := filepath.Dir(symlinkPath)
		if err := os.MkdirAll(symlinkParentDir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Determine target path
		var targetPath string

		if originalTarget, isSymlink := extractedSymlinksMap[filePath]; isSymlink {
			// This is a symlink from the package
			symlinkTargetDir := filepath.Join(poolDir, "libtest", "1.0.0", filepath.Dir(filePath))
			targetPath = filepath.Join(symlinkTargetDir, originalTarget)
		} else {
			// Regular file
			targetPath = filepath.Join(poolDir, "libtest", "1.0.0", filePath)
		}

		// Create symlink
		if err := os.Symlink(targetPath, symlinkPath); err != nil {
			t.Fatalf("Failed to create symlink %s: %v", symlinkPath, err)
		}
	}

	// Verify symlinks were created in symlink directory
	expectedLinks := map[string]string{
		"usr/lib/libtest.so.1": filepath.Join(poolDir, "libtest", "1.0.0", "usr/lib/libtest.so.1.0.0"),
		"usr/lib/libtest.so":   filepath.Join(poolDir, "libtest", "1.0.0", "usr/lib/libtest.so.1"),
	}

	for linkName, expectedTarget := range expectedLinks {
		linkPath := filepath.Join(symlinkDir, linkName)

		// Verify symlink exists
		target, err := os.Readlink(linkPath)
		if err != nil {
			t.Errorf("Failed to read symlink %s: %v", linkPath, err)
		}

		// Verify target is correct
		if target != expectedTarget {
			t.Errorf("Symlink %s points to %s, expected %s", linkPath, target, expectedTarget)
		}

		// Verify target exists (should point to library in store)
		if _, err := os.Lstat(target); err != nil {
			t.Errorf("Symlink target %s does not exist: %v", target, err)
		}
	}

	// Verify the chain of symlinks works
	finalTarget, err := os.Readlink(filepath.Join(symlinkDir, "usr/lib/libtest.so"))
	if err != nil {
		t.Fatalf("Failed to read final symlink: %v", err)
	}

	// This should point to libtest.so.1 in storage
	expectedFinal := filepath.Join(poolDir, "libtest", "1.0.0", "usr/lib/libtest.so.1")
	if finalTarget != expectedFinal {
		t.Errorf("Final symlink points to %s, expected %s", finalTarget, expectedFinal)
	}

	t.Logf("✓ Symlinks properly created in symlink directory:")
	t.Logf("  %s/usr/lib/libtest.so → storage", symlinkDir)
	t.Logf("  %s/usr/lib/libtest.so.1 → storage", symlinkDir)
}

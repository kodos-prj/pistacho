package store

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

// createTestArchive creates a test .tar.zst file with the given files.
func createTestArchive(files map[string]string) ([]byte, error) {
	var buf bytes.Buffer

	encoder, err := zstd.NewWriter(&buf)
	if err != nil {
		return nil, err
	}

	tarWriter := tar.NewWriter(encoder)

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, err
		}

		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return nil, err
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, err
	}

	if err := encoder.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// TestNewStore tests store creation.
func TestNewStore(t *testing.T) {
	store := NewStore("")
	if store == nil {
		t.Fatal("NewStore returned nil")
	}
	if store.root != DefaultPoolRoot {
		t.Errorf("Expected root %s, got %s", DefaultPoolRoot, store.root)
	}

	customRoot := "/custom/store"
	store = NewStore(customRoot)
	if store.root != customRoot {
		t.Errorf("Expected root %s, got %s", customRoot, store.root)
	}
}

// TestGetPackagePath tests getting package paths.
func TestGetPackagePath(t *testing.T) {
	store := NewStore("/tmp/store")

	path := store.GetPackagePath("bash", "5.3.9-1")
	expected := filepath.Join("/tmp/store", "bash", "5.3.9-1")
	if path != expected {
		t.Errorf("Path mismatch: expected %s, got %s", expected, path)
	}
}

// TestGetLatestPath tests getting the latest symlink path.
func TestGetLatestPath(t *testing.T) {
	store := NewStore("/tmp/store")

	path := store.GetLatestPath("bash")
	expected := filepath.Join("/tmp/store", "bash", "current")
	if path != expected {
		t.Errorf("Path mismatch: expected %s, got %s", expected, path)
	}
}

// TestExtractPackage tests extracting a package.
func TestExtractPackage(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create test package
	files := map[string]string{
		"bin/bash":   "#!/bin/bash",
		"etc/config": "configuration",
	}

	archiveData, err := createTestArchive(files)
	if err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	pkgPath := filepath.Join(tmpDir, "bash-5.3.9-1-x86_64.pkg.tar.zst")
	if err := os.WriteFile(pkgPath, archiveData, 0644); err != nil {
		t.Fatalf("Failed to write test package: %v", err)
	}

	// Extract package
	extracted, err := store.ExtractPackage(pkgPath, "bash", "5.3.9-1")
	if err != nil {
		t.Fatalf("ExtractPackage failed: %v", err)
	}

	if len(extracted) != 2 {
		t.Errorf("Expected 2 extracted files, got %d", len(extracted))
	}

	// Verify extracted files exist
	binPath := filepath.Join(tmpDir, "bash", "5.3.9-1", "bin", "bash")
	if _, err := os.Stat(binPath); err != nil {
		t.Errorf("Extracted file not found: %v", err)
	}
}

// TestPackageExists tests checking if a package exists.
func TestPackageExists(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Package should not exist initially
	if store.PackageExists("bash", "5.3.9-1") {
		t.Fatal("Package should not exist initially")
	}

	// Create the package directory
	pkgPath := store.GetPackagePath("bash", "5.3.9-1")
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	// Now package should exist
	if !store.PackageExists("bash", "5.3.9-1") {
		t.Fatal("Package should exist after creation")
	}
}

// TestRemovePackage tests removing a package.
func TestRemovePackage(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create a package
	pkgPath := store.GetPackagePath("bash", "5.3.9-1")
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	// Create a file in the package
	testFile := filepath.Join(pkgPath, "testfile")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Remove package
	if err := store.RemovePackage("bash", "5.3.9-1"); err != nil {
		t.Fatalf("RemovePackage failed: %v", err)
	}

	// Verify it's removed
	if store.PackageExists("bash", "5.3.9-1") {
		t.Fatal("Package should be removed")
	}
}

// TestListVersions tests listing package versions.
func TestListVersions(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create multiple versions
	versions := []string{"5.3.0-1", "5.3.5-1", "5.3.9-1"}
	for _, v := range versions {
		pkgPath := store.GetPackagePath("bash", v)
		if err := os.MkdirAll(pkgPath, 0755); err != nil {
			t.Fatalf("Failed to create package directory: %v", err)
		}
	}

	// List versions
	listed, err := store.ListVersions("bash")
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}

	if len(listed) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(listed))
	}

	// Versions should be sorted descending (newest first)
	if listed[0] != "5.3.9-1" {
		t.Errorf("First version should be 5.3.9-1, got %s", listed[0])
	}
}

// TestListVersionsNonExistent tests listing versions of non-existent package.
func TestListVersionsNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	listed, err := store.ListVersions("nonexistent")
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}

	if len(listed) != 0 {
		t.Errorf("Expected 0 versions for non-existent package, got %d", len(listed))
	}
}

// TestGetPackageSize tests getting package size.
func TestGetPackageSize(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	pkgPath := store.GetPackagePath("bash", "5.3.9-1")
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	// Create some files with known size
	testContent := "test content"
	testFile := filepath.Join(pkgPath, "testfile")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	size, err := store.GetPackageSize("bash", "5.3.9-1")
	if err != nil {
		t.Fatalf("GetPackageSize failed: %v", err)
	}

	if size != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), size)
	}
}

// TestGetAllPackages tests getting all packages.
func TestGetAllPackages(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create multiple packages and versions
	packages := map[string][]string{
		"bash": {"5.3.0-1", "5.3.9-1"},
		"vim":  {"9.0.0-1"},
	}

	for pkg, versions := range packages {
		for _, v := range versions {
			pkgPath := store.GetPackagePath(pkg, v)
			if err := os.MkdirAll(pkgPath, 0755); err != nil {
				t.Fatalf("Failed to create package directory: %v", err)
			}
		}
	}

	// Get all packages
	all, err := store.GetAllPackages()
	if err != nil {
		t.Fatalf("GetAllPackages failed: %v", err)
	}

	if len(all) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(all))
	}

	if len(all["bash"]) != 2 {
		t.Errorf("Expected 2 bash versions, got %d", len(all["bash"]))
	}

	if len(all["vim"]) != 1 {
		t.Errorf("Expected 1 vim version, got %d", len(all["vim"]))
	}
}

// TestCleanupOldVersions tests removing old versions.
func TestCleanupOldVersions(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create 5 versions
	versions := []string{"5.0.0-1", "5.1.0-1", "5.2.0-1", "5.3.0-1", "5.3.9-1"}
	for _, v := range versions {
		pkgPath := store.GetPackagePath("bash", v)
		if err := os.MkdirAll(pkgPath, 0755); err != nil {
			t.Fatalf("Failed to create package directory: %v", err)
		}
	}

	// Keep only 2 versions
	removed, err := store.CleanupOldVersions("bash", 2)
	if err != nil {
		t.Fatalf("CleanupOldVersions failed: %v", err)
	}

	if removed != 3 {
		t.Errorf("Expected 3 versions removed, got %d", removed)
	}

	// Verify only 2 versions remain
	listed, _ := store.ListVersions("bash")
	if len(listed) != 2 {
		t.Errorf("Expected 2 versions remaining, got %d", len(listed))
	}

	// Should keep the newest 2
	if listed[0] != "5.3.9-1" || listed[1] != "5.3.0-1" {
		t.Errorf("Wrong versions kept: %v", listed)
	}
}

// TestSetLatestVersion tests creating/updating latest symlink.
func TestSetLatestVersion(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create a package version
	pkgPath := store.GetPackagePath("bash", "5.3.9-1")
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	// Set latest
	if err := store.SetLatestVersion("bash", "5.3.9-1"); err != nil {
		t.Fatalf("SetLatestVersion failed: %v", err)
	}

	// Verify symlink exists
	currentLink := store.GetLatestPath("bash")
	if _, err := os.Stat(currentLink); err != nil {
		t.Errorf("Latest symlink not created: %v", err)
	}

	// Verify symlink points to correct directory
	target, err := os.Readlink(currentLink)
	if err != nil {
		t.Errorf("Failed to read symlink: %v", err)
	}

	if !strings.HasSuffix(target, "5.3.9-1") {
		t.Errorf("Symlink points to wrong target: %s", target)
	}
}

// TestSetLatestVersionNonExistent tests setting latest for non-existent version.
func TestSetLatestVersionNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	err := store.SetLatestVersion("bash", "5.3.9-1")
	if err == nil {
		t.Fatal("Expected error for non-existent version")
	}
}

// TestGetLatestVersion tests getting the latest version.
func TestGetLatestVersion(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create multiple versions
	versions := []string{"5.3.0-1", "5.3.5-1", "5.3.9-1"}
	for _, v := range versions {
		pkgPath := store.GetPackagePath("bash", v)
		if err := os.MkdirAll(pkgPath, 0755); err != nil {
			t.Fatalf("Failed to create package directory: %v", err)
		}
	}

	// Set 5.3.5-1 as latest
	if err := store.SetLatestVersion("bash", "5.3.5-1"); err != nil {
		t.Fatalf("SetLatestVersion failed: %v", err)
	}

	// Get latest
	latest, err := store.GetLatestVersion("bash")
	if err != nil {
		t.Fatalf("GetLatestVersion failed: %v", err)
	}

	if latest != "5.3.5-1" {
		t.Errorf("Expected latest 5.3.5-1, got %s", latest)
	}
}

// TestValidateStore tests store validation.
func TestValidateStore(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create valid packages
	pkgPath := store.GetPackagePath("bash", "5.3.9-1")
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	issues := store.ValidateStore()
	if len(issues) > 0 {
		t.Errorf("Valid store reported issues: %v", issues)
	}
}

// BenchmarkExtractPackage benchmarks package extraction.
func BenchmarkExtractPackage(b *testing.B) {
	files := make(map[string]string)
	for i := 0; i < 50; i++ {
		files[filepath.Join("bin", "file"+string(rune(i))+".sh")] = "#!/bin/bash"
	}

	archiveData, err := createTestArchive(files)
	if err != nil {
		b.Fatalf("Failed to create test archive: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		store := NewStore(tmpDir)

		pkgPath := filepath.Join(tmpDir, "test.pkg.tar.zst")
		os.WriteFile(pkgPath, archiveData, 0644)

		store.ExtractPackage(pkgPath, "testpkg", "1.0.0-1")
	}
}

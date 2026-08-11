package symlink

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	// Test with explicit symlink root
	m := NewManager("/kod/pool", "/usr")
	if m.storeRoot != "/kod/pool" {
		t.Errorf("Expected storeRoot /kod/pool, got %s", m.storeRoot)
	}
	if m.symlinkRoot != "/usr" {
		t.Errorf("Expected symlinkRoot /usr, got %s", m.symlinkRoot)
	}

	// Test with empty symlink root (should default to /)
	m2 := NewManager("/kod/pool", "")
	if m2.symlinkRoot != "/" {
		t.Errorf("Expected default symlinkRoot /, got %s", m2.symlinkRoot)
	}
}

func TestCreateSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	storeRoot := filepath.Join(tmpDir, "pool")
	symlinkRoot := filepath.Join(tmpDir, "root")

	// Create test directories
	os.MkdirAll(filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin"), 0755)
	os.MkdirAll(filepath.Join(symlinkRoot, "usr", "bin"), 0755)

	// Create fake store files
	storeFile := filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin", "bash")
	os.WriteFile(storeFile, []byte("dummy"), 0644)

	m := NewManager(storeRoot, symlinkRoot)

	// Test creating symlink
	files := []string{"usr/bin/bash"}
	err := m.CreateSymlinks("bash", "5.3.9-1", files)
	if err != nil {
		t.Fatalf("CreateSymlinks failed: %v", err)
	}

	// Verify symlink was created
	symlinkPath := filepath.Join(symlinkRoot, "usr", "bin", "bash")
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}

	expectedTarget := filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin", "bash")
	if target != expectedTarget {
		t.Errorf("Symlink points to %s, expected %s", target, expectedTarget)
	}
}

func TestCreateSymlinksWithExistingSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	storeRoot := filepath.Join(tmpDir, "pool")
	symlinkRoot := filepath.Join(tmpDir, "root")

	// Create test directories
	os.MkdirAll(filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin"), 0755)
	os.MkdirAll(filepath.Join(symlinkRoot, "usr", "bin"), 0755)

	// Create fake store files
	storeFile := filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin", "bash")
	os.WriteFile(storeFile, []byte("dummy"), 0644)

	m := NewManager(storeRoot, symlinkRoot)

	// Create symlink twice
	files := []string{"usr/bin/bash"}
	err := m.CreateSymlinks("bash", "5.3.9-1", files)
	if err != nil {
		t.Fatalf("First CreateSymlinks failed: %v", err)
	}

	// Creating again should skip (symlink already exists)
	err = m.CreateSymlinks("bash", "5.3.9-1", files)
	if err != nil {
		t.Fatalf("Second CreateSymlinks failed: %v", err)
	}
}

func TestCreateSymlinksReplacesExistingSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	storeRoot := filepath.Join(tmpDir, "pool")
	symlinkRoot := filepath.Join(tmpDir, "root")

	// Create test directories
	os.MkdirAll(filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin"), 0755)
	os.MkdirAll(filepath.Join(symlinkRoot, "usr", "bin"), 0755)

	// Create fake store files
	storeFile := filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin", "bash")
	os.WriteFile(storeFile, []byte("dummy"), 0644)

	// Create a symlink pointing elsewhere
	symlinkPath := filepath.Join(symlinkRoot, "usr", "bin", "bash")
	wrongTarget := filepath.Join(tmpDir, "other", "bash")
	if err := os.Symlink(wrongTarget, symlinkPath); err != nil {
		t.Fatalf("Failed to create existing symlink: %v", err)
	}

	m := NewManager(storeRoot, symlinkRoot)

	// Creating should replace the wrong symlink
	files := []string{"usr/bin/bash"}
	err := m.CreateSymlinks("bash", "5.3.9-1", files)
	if err != nil {
		t.Fatalf("CreateSymlinks failed: %v", err)
	}

	// Verify symlink now points to the correct store location
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("Readlink failed: %v", err)
	}
	expectedTarget := m.GetStorePath("bash", "5.3.9-1", "usr/bin/bash")
	if target != expectedTarget {
		t.Errorf("Symlink points to %s, expected %s", target, expectedTarget)
	}
}

func TestCreateSymlinksWithExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	storeRoot := filepath.Join(tmpDir, "pool")
	symlinkRoot := filepath.Join(tmpDir, "root")

	// Create test directories
	os.MkdirAll(filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin"), 0755)
	os.MkdirAll(filepath.Join(symlinkRoot, "usr", "bin"), 0755)

	// Create fake store file
	storeFile := filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin", "bash")
	os.WriteFile(storeFile, []byte("dummy"), 0644)

	// Create regular file at symlink location
	symlinkPath := filepath.Join(symlinkRoot, "usr", "bin", "bash")
	os.WriteFile(symlinkPath, []byte("existing"), 0644)

	m := NewManager(storeRoot, symlinkRoot)

	// Try to create symlink - should skip existing file
	files := []string{"usr/bin/bash"}
	err := m.CreateSymlinks("bash", "5.3.9-1", files)
	// Should succeed (skipping existing file is not an error)
	if err != nil {
		t.Fatalf("CreateSymlinks failed: %v", err)
	}

	// Verify it's still a regular file
	stat, _ := os.Lstat(symlinkPath)
	if stat.Mode()&os.ModeSymlink != 0 {
		t.Error("File was converted to symlink, should have been skipped")
	}
}

func TestRemoveSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	storeRoot := filepath.Join(tmpDir, "pool")
	symlinkRoot := filepath.Join(tmpDir, "root")

	// Create test directories
	os.MkdirAll(filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin"), 0755)
	os.MkdirAll(filepath.Join(symlinkRoot, "usr", "bin"), 0755)

	// Create fake store file
	storeFile := filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin", "bash")
	os.WriteFile(storeFile, []byte("dummy"), 0644)

	m := NewManager(storeRoot, symlinkRoot)

	// Create symlink
	files := []string{"usr/bin/bash"}
	m.CreateSymlinks("bash", "5.3.9-1", files)

	// Remove symlink
	err := m.RemoveSymlinks(files)
	if err != nil {
		t.Fatalf("RemoveSymlinks failed: %v", err)
	}

	// Verify symlink was removed
	symlinkPath := filepath.Join(symlinkRoot, "usr", "bin", "bash")
	_, err = os.Lstat(symlinkPath)
	if !os.IsNotExist(err) {
		t.Error("Symlink still exists after removal")
	}
}

func TestRemoveSymlinksWithRegularFile(t *testing.T) {
	tmpDir := t.TempDir()
	storeRoot := filepath.Join(tmpDir, "pool")
	symlinkRoot := filepath.Join(tmpDir, "root")

	os.MkdirAll(filepath.Join(symlinkRoot, "usr", "bin"), 0755)

	// Create regular file (not a symlink)
	filePath := filepath.Join(symlinkRoot, "usr", "bin", "bash")
	os.WriteFile(filePath, []byte("data"), 0644)

	m := NewManager(storeRoot, symlinkRoot)

	// Try to remove - should skip regular files
	files := []string{"usr/bin/bash"}
	err := m.RemoveSymlinks(files)
	// Should succeed (skipping is not an error)
	if err != nil {
		t.Fatalf("RemoveSymlinks failed: %v", err)
	}

	// Verify file still exists
	if _, err := os.Stat(filePath); err != nil {
		t.Error("Regular file was removed, should have been skipped")
	}
}

func TestRemoveSymlinksNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(filepath.Join(tmpDir, "pool"), filepath.Join(tmpDir, "root"))

	// Try to remove non-existent symlinks - should succeed (skip)
	files := []string{"usr/bin/nonexistent"}
	err := m.RemoveSymlinks(files)
	if err != nil {
		t.Fatalf("RemoveSymlinks failed: %v", err)
	}
}

func TestVerifySymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	storeRoot := filepath.Join(tmpDir, "pool")
	symlinkRoot := filepath.Join(tmpDir, "root")

	// Create test directories
	os.MkdirAll(filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin"), 0755)
	os.MkdirAll(filepath.Join(symlinkRoot, "usr", "bin"), 0755)

	// Create fake store file
	storeFile := filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin", "bash")
	os.WriteFile(storeFile, []byte("dummy"), 0644)

	m := NewManager(storeRoot, symlinkRoot)

	// Create symlink
	files := []string{"usr/bin/bash"}
	m.CreateSymlinks("bash", "5.3.9-1", files)

	// Verify should pass
	err := m.VerifySymlinks("bash", "5.3.9-1", files)
	if err != nil {
		t.Fatalf("VerifySymlinks failed: %v", err)
	}
}

func TestVerifySymlinksPointingWrong(t *testing.T) {
	tmpDir := t.TempDir()
	storeRoot := filepath.Join(tmpDir, "pool")
	symlinkRoot := filepath.Join(tmpDir, "root")

	// Create test directories
	os.MkdirAll(filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin"), 0755)
	os.MkdirAll(filepath.Join(storeRoot, "bash", "5.3.8-1", "usr", "bin"), 0755)
	os.MkdirAll(filepath.Join(symlinkRoot, "usr", "bin"), 0755)

	// Create store files
	os.WriteFile(filepath.Join(storeRoot, "bash", "5.3.9-1", "usr", "bin", "bash"), []byte("v1"), 0644)
	os.WriteFile(filepath.Join(storeRoot, "bash", "5.3.8-1", "usr", "bin", "bash"), []byte("v2"), 0644)

	m := NewManager(storeRoot, symlinkRoot)

	// Create symlink to old version
	oldPath := filepath.Join(storeRoot, "bash", "5.3.8-1", "usr", "bin", "bash")
	symlinkPath := filepath.Join(symlinkRoot, "usr", "bin", "bash")
	os.Symlink(oldPath, symlinkPath)

	// Verify should fail (points to wrong version)
	files := []string{"usr/bin/bash"}
	err := m.VerifySymlinks("bash", "5.3.9-1", files)
	if err == nil {
		t.Error("VerifySymlinks should have failed for wrong target")
	}
}

func TestVerifySymlinksNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(filepath.Join(tmpDir, "pool"), filepath.Join(tmpDir, "root"))

	// Verify non-existent symlinks should fail
	files := []string{"usr/bin/nonexistent"}
	err := m.VerifySymlinks("bash", "5.3.9-1", files)
	if err == nil {
		t.Error("VerifySymlinks should fail for non-existent symlinks")
	}
}

func TestGetSymlinkPath(t *testing.T) {
	m := NewManager("/kod/pool", "/usr")
	path := m.GetSymlinkPath("bin/bash")
	expected := "/usr/bin/bash"
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestGetStorePath(t *testing.T) {
	m := NewManager("/kod/pool", "/usr")
	path := m.GetStorePath("bash", "5.3.9-1", "bin/bash")
	expected := "/kod/pool/bash/5.3.9-1/bin/bash"
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestCreateSymlinksMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	storeRoot := filepath.Join(tmpDir, "pool")
	symlinkRoot := filepath.Join(tmpDir, "root")

	// Create test directories and files
	storeDir := filepath.Join(storeRoot, "bash", "5.3.9-1")
	os.MkdirAll(filepath.Join(storeDir, "usr", "bin"), 0755)
	os.MkdirAll(filepath.Join(storeDir, "usr", "share", "man", "man1"), 0755)
	os.MkdirAll(filepath.Join(symlinkRoot, "usr", "bin"), 0755)
	os.MkdirAll(filepath.Join(symlinkRoot, "usr", "share", "man", "man1"), 0755)

	// Create store files
	os.WriteFile(filepath.Join(storeDir, "usr", "bin", "bash"), []byte("bin"), 0644)
	os.WriteFile(filepath.Join(storeDir, "usr", "share", "man", "man1", "bash.1.gz"), []byte("doc"), 0644)

	m := NewManager(storeRoot, symlinkRoot)

	// Create multiple symlinks
	files := []string{"usr/bin/bash", "usr/share/man/man1/bash.1.gz"}
	err := m.CreateSymlinks("bash", "5.3.9-1", files)
	if err != nil {
		t.Fatalf("CreateSymlinks failed: %v", err)
	}

	// Verify both symlinks exist
	for _, file := range files {
		symlinkPath := filepath.Join(symlinkRoot, file)
		_, err := os.Lstat(symlinkPath)
		if err != nil {
			t.Errorf("Symlink %s not created: %v", file, err)
		}
	}
}

func TestEmptyFileList(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(filepath.Join(tmpDir, "pool"), filepath.Join(tmpDir, "root"))

	// Empty list should not error
	err := m.CreateSymlinks("bash", "5.3.9-1", []string{})
	if err != nil {
		t.Fatalf("CreateSymlinks with empty list failed: %v", err)
	}

	err = m.RemoveSymlinks([]string{})
	if err != nil {
		t.Fatalf("RemoveSymlinks with empty list failed: %v", err)
	}

	err = m.VerifySymlinks("bash", "5.3.9-1", []string{})
	if err != nil {
		t.Fatalf("VerifySymlinks with empty list failed: %v", err)
	}
}

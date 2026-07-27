package wrapper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewGenerator tests generator creation.
func TestNewGenerator(t *testing.T) {
	gen := NewGenerator("/store", "/wrappers", "/")
	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if gen.storeRoot != "/store" {
		t.Errorf("Expected storeRoot /store, got %s", gen.storeRoot)
	}
	if gen.wrapperRoot != "/wrappers" {
		t.Errorf("Expected wrapperRoot /wrappers, got %s", gen.wrapperRoot)
	}

	// Test default symlinkRoot
	gen = NewGenerator("/store", "/wrappers", "")
	if gen.symlinkRoot != "/" {
		t.Errorf("Expected default symlinkRoot /, got %s", gen.symlinkRoot)
	}
}

// TestDiscoverLibraries tests discovering shared libraries in a package.
func TestDiscoverLibraries(t *testing.T) {
	tmpDir := t.TempDir()
	storeDir := filepath.Join(tmpDir, "pool")
	pkgDir := filepath.Join(storeDir, "bash", "5.3.9-1")

	// Create package structure with libraries
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("Failed to create package dir: %v", err)
	}

	// Create lib directory with shared libraries
	libDir := filepath.Join(pkgDir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		t.Fatalf("Failed to create lib dir: %v", err)
	}

	// Create some .so files
	libFiles := []string{"libc.so.6", "libm.so.6", "libdl.so.2"}
	for _, lib := range libFiles {
		libPath := filepath.Join(libDir, lib)
		if err := os.WriteFile(libPath, []byte("fake library"), 0644); err != nil {
			t.Fatalf("Failed to create lib file: %v", err)
		}
	}

	// Create lib64 directory with more libraries
	lib64Dir := filepath.Join(pkgDir, "lib64")
	if err := os.MkdirAll(lib64Dir, 0755); err != nil {
		t.Fatalf("Failed to create lib64 dir: %v", err)
	}
	lib64File := filepath.Join(lib64Dir, "ld-linux-x86-64.so.2")
	if err := os.WriteFile(lib64File, []byte("fake library"), 0644); err != nil {
		t.Fatalf("Failed to create lib64 file: %v", err)
	}

	// Create a regular file without .so
	binDir := filepath.Join(pkgDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin dir: %v", err)
	}
	bashFile := filepath.Join(binDir, "bash")
	if err := os.WriteFile(bashFile, []byte("fake executable"), 0755); err != nil {
		t.Fatalf("Failed to create bash file: %v", err)
	}

	gen := NewGenerator(storeDir, filepath.Join(tmpDir, "wrappers"), "/")
	libraries, err := gen.DiscoverLibraries("bash", "5.3.9-1")

	if err != nil {
		t.Fatalf("DiscoverLibraries failed: %v", err)
	}

	// Check that we found libraries in lib and lib64
	if len(libraries) == 0 {
		t.Fatal("Expected to find libraries, got none")
	}

	// Verify lib directory has the right libraries
	if libs, ok := libraries["lib"]; ok {
		if len(libs) != 3 {
			t.Errorf("Expected 3 libraries in lib, got %d", len(libs))
		}
	} else {
		t.Error("Expected to find lib directory")
	}

	// Verify lib64 directory has libraries
	if libs, ok := libraries["lib64"]; ok {
		if len(libs) != 1 {
			t.Errorf("Expected 1 library in lib64, got %d", len(libs))
		}
	} else {
		t.Error("Expected to find lib64 directory")
	}

	// Verify bin directory was not included (no .so files)
	if _, ok := libraries["bin"]; ok {
		t.Error("Unexpected bin directory in libraries (should only include .so files)")
	}
}

// TestDiscoverLibrariesNotFound tests discovering libraries in non-existent package.
func TestDiscoverLibrariesNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storeDir := filepath.Join(tmpDir, "pool")

	gen := NewGenerator(storeDir, filepath.Join(tmpDir, "wrappers"), "/")
	_, err := gen.DiscoverLibraries("nonexistent", "1.0.0")

	if err == nil {
		t.Fatal("Expected error for non-existent package, got nil")
	}
}

// TestGenerateWrapper tests wrapper script generation.
func TestGenerateWrapper(t *testing.T) {
	tmpDir := t.TempDir()
	storeDir := filepath.Join(tmpDir, "pool")
	wrapperDir := filepath.Join(tmpDir, "wrappers")

	gen := NewGenerator(storeDir, wrapperDir, "/")

	libDirs := []string{"lib", "lib64"}
	err := gen.GenerateWrapper("vim", "vim", "9.0.1", libDirs)

	if err != nil {
		t.Fatalf("GenerateWrapper failed: %v", err)
	}

	// Check that wrapper file was created
	wrapperPath := filepath.Join(wrapperDir, "vim")
	if _, err := os.Stat(wrapperPath); err != nil {
		t.Fatalf("Wrapper file not created: %v", err)
	}

	// Check wrapper content
	content, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("Failed to read wrapper file: %v", err)
	}

	scriptContent := string(content)

	// Verify script contains expected elements
	if !strings.Contains(scriptContent, "#!/bin/bash") {
		t.Error("Wrapper script missing shebang")
	}

	if !strings.Contains(scriptContent, "LD_LIBRARY_PATH") {
		t.Error("Wrapper script missing LD_LIBRARY_PATH")
	}

	if !strings.Contains(scriptContent, "vim") {
		t.Error("Wrapper script missing command name")
	}

	if !strings.Contains(scriptContent, "exec") {
		t.Error("Wrapper script missing exec statement")
	}

	// Check executable bit
	info, err := os.Stat(wrapperPath)
	if err != nil {
		t.Fatalf("Failed to stat wrapper: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("Wrapper script is not executable")
	}
}

// TestGenerateWrapperCreatesDirectory tests that wrapper directory is created.
func TestGenerateWrapperCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	storeDir := filepath.Join(tmpDir, "pool")
	wrapperDir := filepath.Join(tmpDir, "nonexistent", "wrappers")

	gen := NewGenerator(storeDir, wrapperDir, "/")
	err := gen.GenerateWrapper("bash", "bash", "5.3.9-1", []string{"lib"})

	if err != nil {
		t.Fatalf("GenerateWrapper failed: %v", err)
	}

	// Check that directory was created
	if _, err := os.Stat(wrapperDir); err != nil {
		t.Fatalf("Wrapper directory not created: %v", err)
	}
}

// TestRemoveWrapper tests wrapper script removal.
func TestRemoveWrapper(t *testing.T) {
	tmpDir := t.TempDir()
	wrapperDir := filepath.Join(tmpDir, "wrappers")
	os.MkdirAll(wrapperDir, 0755)

	// Create a wrapper file
	wrapperPath := filepath.Join(wrapperDir, "bash")
	if err := os.WriteFile(wrapperPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create wrapper: %v", err)
	}

	gen := NewGenerator("/store", wrapperDir, "/")
	err := gen.RemoveWrapper("bash")

	if err != nil {
		t.Fatalf("RemoveWrapper failed: %v", err)
	}

	// Check that wrapper was removed
	if _, err := os.Stat(wrapperPath); err == nil {
		t.Fatal("Wrapper still exists after removal")
	}
}

// TestRemoveWrapperNotFound tests removing non-existent wrapper.
func TestRemoveWrapperNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	wrapperDir := filepath.Join(tmpDir, "wrappers")

	gen := NewGenerator("/store", wrapperDir, "/")
	err := gen.RemoveWrapper("nonexistent")

	// Should not error for non-existent file
	if err != nil {
		t.Fatalf("RemoveWrapper should not error for non-existent file: %v", err)
	}
}

// TestGetWrapperPath tests getting wrapper path.
func TestGetWrapperPath(t *testing.T) {
	gen := NewGenerator("/store", "/wrappers", "/")

	path := gen.GetWrapperPath("vim")
	expected := "/wrappers/vim"

	if path != expected {
		t.Errorf("Expected path %s, got %s", expected, path)
	}
}

// TestBuildWrapperScript tests wrapper script content generation.
func TestBuildWrapperScript(t *testing.T) {
	gen := NewGenerator("/kod/pool", "/kod/wrappers", "/")

	libDirs := []string{"/kod/pool/vim/9.0.1/lib", "/kod/pool/vim/9.0.1/lib64"}
	script := gen.buildWrapperScript("vim", "vim", "9.0.1", libDirs)

	// Verify script structure
	if !strings.Contains(script, "#!/bin/bash") {
		t.Error("Wrapper script missing shebang for non-bash package")
	}

	if !strings.Contains(script, "LD_LIBRARY_PATH") {
		t.Error("Missing LD_LIBRARY_PATH")
	}

	if !strings.Contains(script, "export") {
		t.Error("Missing export statement")
	}

	if !strings.Contains(script, "exec") {
		t.Error("Missing exec statement")
	}

	// Verify command path includes symlinkRoot
	if !strings.Contains(script, "/usr/bin/vim") {
		t.Error("Missing command path")
	}

	// Verify library paths are included
	if !strings.Contains(script, "/kod/pool/vim/9.0.1/lib") {
		t.Error("Missing lib path")
	}

	if !strings.Contains(script, "/kod/pool/vim/9.0.1/lib64") {
		t.Error("Missing lib64 path")
	}
}

// TestBuildWrapperScriptBashShebang tests that bash wrappers use the bash from store.
func TestBuildWrapperScriptBashShebang(t *testing.T) {
	gen := NewGenerator("/kod/pool", "/kod/wrappers", "/")

	libDirs := []string{"/kod/pool/bash/5.3.9-1/lib"}
	script := gen.buildWrapperScript("bash", "bash", "5.3.9-1", libDirs)

	// Verify shebang uses store bash with "current" symlink
	expectedShebang := "#!/kod/pool/bash/current/usr/bin/bash"
	if !strings.Contains(script, expectedShebang) {
		t.Errorf("Expected shebang %q in bash wrapper, got:\n%s", expectedShebang, script)
	}

	// Verify other elements are still present
	if !strings.Contains(script, "LD_LIBRARY_PATH") {
		t.Error("Missing LD_LIBRARY_PATH in bash wrapper")
	}

	if !strings.Contains(script, "export") {
		t.Error("Missing export statement in bash wrapper")
	}

	if !strings.Contains(script, "exec") {
		t.Error("Missing exec statement in bash wrapper")
	}
}

// TestBuildWrapperScriptBashWithPrefixStripping tests bash shebang with prefix stripping.
func TestBuildWrapperScriptBashWithPrefixStripping(t *testing.T) {
	gen := NewGeneratorWithPrefix("/kod/pool", "/kod/wrappers", "/", "/tmp/demo")

	libDirs := []string{"/tmp/demo/kod/pool/bash/5.3.9-1/lib"}
	script := gen.buildWrapperScript("bash", "bash", "5.3.9-1", libDirs)

	// Verify shebang is stripped (no /tmp/demo prefix)
	expectedShebang := "#!/kod/pool/bash/current/usr/bin/bash"
	if !strings.Contains(script, expectedShebang) {
		t.Errorf("Expected stripped shebang %q in bash wrapper, got:\n%s", expectedShebang, script)
	}

	// Verify the shebang doesn't contain the prefix
	if strings.Contains(script, "#!/tmp/demo/") {
		t.Error("Shebang should not contain stripped prefix in bash wrapper")
	}

	// Verify other elements still work with prefix stripping
	if !strings.Contains(script, "LD_LIBRARY_PATH") {
		t.Error("Missing LD_LIBRARY_PATH in bash wrapper with prefix stripping")
	}

	if !strings.Contains(script, "exec") {
		t.Error("Missing exec statement in bash wrapper with prefix stripping")
	}
}

// TestGenerateWrapperMultipleLibDirs tests wrapper with multiple library directories.
func TestGenerateWrapperMultipleLibDirs(t *testing.T) {
	tmpDir := t.TempDir()
	storeDir := filepath.Join(tmpDir, "pool")
	wrapperDir := filepath.Join(tmpDir, "wrappers")

	gen := NewGenerator(storeDir, wrapperDir, "/")

	libDirs := []string{"lib", "lib64", "lib32"}
	err := gen.GenerateWrapper("complex", "complex", "1.0.0", libDirs)

	if err != nil {
		t.Fatalf("GenerateWrapper failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(wrapperDir, "complex"))
	if err != nil {
		t.Fatalf("Failed to read wrapper: %v", err)
	}

	script := string(content)

	// Verify all library paths are in LD_LIBRARY_PATH
	if !strings.Contains(script, "/lib") {
		t.Error("Missing lib path")
	}
	if !strings.Contains(script, "/lib64") {
		t.Error("Missing lib64 path")
	}
	if !strings.Contains(script, "/lib32") {
		t.Error("Missing lib32 path")
	}
}

// TestDiscoverLibrariesEmpty tests discovering libraries in package with no .so files.
func TestDiscoverLibrariesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	storeDir := filepath.Join(tmpDir, "pool")
	pkgDir := filepath.Join(storeDir, "bash", "5.3.9-1")

	// Create package structure without .so files
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("Failed to create package dir: %v", err)
	}

	// Create bin directory with executables (no .so)
	binDir := filepath.Join(pkgDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin dir: %v", err)
	}
	bashFile := filepath.Join(binDir, "bash")
	if err := os.WriteFile(bashFile, []byte("fake executable"), 0755); err != nil {
		t.Fatalf("Failed to create bash file: %v", err)
	}

	gen := NewGenerator(storeDir, filepath.Join(tmpDir, "wrappers"), "/")
	libraries, err := gen.DiscoverLibraries("bash", "5.3.9-1")

	if err != nil {
		t.Fatalf("DiscoverLibraries failed: %v", err)
	}

	if len(libraries) != 0 {
		t.Errorf("Expected no libraries, got %d directories", len(libraries))
	}
}

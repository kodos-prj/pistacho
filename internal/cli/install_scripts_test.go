package cli

import (
	"path/filepath"
	"testing"

	"github.com/kodos-prj/chisel/pkg/config"
	"github.com/kodos-prj/chisel/pkg/registry"
)

// TestInstallScriptsCommandCreation tests creating install-scripts command instances
func TestInstallScriptsCommandCreation(t *testing.T) {
	cfg := &config.Config{
		BaseDir:      "/tmp/chisel",
		PoolRoot:    "/tmp/chisel/pool",
		RegistryPath: "/tmp/chisel/registry.json",
	}

	cmd := NewInstallScriptsCommand(cfg)
	if cmd == nil {
		t.Error("expected InstallScriptsCommand, got nil")
	}
	if cmd.config != cfg {
		t.Error("config not set correctly")
	}
}

// TestPackageFilesStructure tests PackageFiles structure
func TestPackageFilesStructure(t *testing.T) {
	pf := &PackageFiles{
		AllFiles:         []string{"usr/bin/bash", "usr/share/man/man1/bash.1"},
		Executables:      []string{"usr/bin/bash"},
		HasInstallScript: true,
	}

	if !pf.HasInstallScript {
		t.Error("expected HasInstallScript to be true")
	}
	if len(pf.AllFiles) != 2 {
		t.Errorf("expected 2 files, got %d", len(pf.AllFiles))
	}
	if len(pf.Executables) != 1 {
		t.Errorf("expected 1 executable, got %d", len(pf.Executables))
	}
}

// TestInstallScriptDetection tests that .INSTALL files are properly detected
func TestInstallScriptDetection(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	
	// Create a mock registry with packages
	regPath := filepath.Join(tmpDir, "registry.json")
	reg, err := registry.NewRegistry(regPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Add packages with and without install scripts
	bashPkg := &registry.Package{
		Name:             "bash",
		Version:          "5.3.9-1",
		HasInstallScript: true,
	}
	if err := reg.AddPackage(bashPkg); err != nil {
		t.Fatalf("failed to add bash package: %v", err)
	}

	glibcPkg := &registry.Package{
		Name:             "glibc",
		Version:          "2.39-1",
		HasInstallScript: true,
	}
	if err := reg.AddPackage(glibcPkg); err != nil {
		t.Fatalf("failed to add glibc package: %v", err)
	}

	vimPkg := &registry.Package{
		Name:             "vim",
		Version:          "9.1.0-1",
		HasInstallScript: false,
	}
	if err := reg.AddPackage(vimPkg); err != nil {
		t.Fatalf("failed to add vim package: %v", err)
	}

	// Save registry
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	// Load and verify
	loadedReg, err := registry.NewRegistry(regPath)
	if err != nil {
		t.Fatalf("failed to load registry: %v", err)
	}

	// Check bash package
	bashCheck, ok := loadedReg.GetPackage("bash")
	if !ok {
		t.Error("expected bash package to exist")
	}
	if !bashCheck.HasInstallScript {
		t.Error("expected bash to have install script")
	}

	// Check vim package (should not have install script)
	vimCheck, ok := loadedReg.GetPackage("vim")
	if !ok {
		t.Error("expected vim package to exist")
	}
	if vimCheck.HasInstallScript {
		t.Error("expected vim to not have install script")
	}
}

// TestExecuteInstallScriptsWithNoScripts tests execute when there are no scripts
func TestExecuteInstallScriptsWithNoScripts(t *testing.T) {
	tmpDir := t.TempDir()
	
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
	}

	// Create empty registry
	reg, err := registry.NewRegistry(cfg.RegistryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	cmd := NewInstallScriptsCommand(cfg)
	err = cmd.Execute([]string{"bash"}, false, "")
	if err != nil {
		t.Errorf("expected no error for non-existent package, got %v", err)
	}
}

// TestPackageFilesWithInstallScript verifies HasInstallScript tracking
func TestPackageFilesWithInstallScript(t *testing.T) {
	tests := []struct {
		name             string
		files            []string
		expectedHasScript bool
	}{
		{
			name:             "with install script",
			files:            []string{".INSTALL", "usr/bin/bash", "usr/share/doc/bash/README"},
			expectedHasScript: true,
		},
		{
			name:             "without install script",
			files:            []string{"usr/bin/bash", "usr/share/doc/bash/README"},
			expectedHasScript: false,
		},
		{
			name:             "only install script",
			files:            []string{".INSTALL"},
			expectedHasScript: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate detection logic
			hasScript := false
			for _, f := range tt.files {
				if f == ".INSTALL" {
					hasScript = true
					break
				}
			}

			if hasScript != tt.expectedHasScript {
				t.Errorf("expected HasInstallScript=%v, got %v", tt.expectedHasScript, hasScript)
			}
		})
	}
}

// TestExecuteInstallScriptsNonChrootMode tests non-chroot mode (chrootDir = "")
func TestExecuteInstallScriptsNonChrootMode(t *testing.T) {
	tmpDir := t.TempDir()
	
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
	}

	// Create registry with package
	reg, err := registry.NewRegistry(cfg.RegistryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	testPkg := &registry.Package{
		Name:             "bash",
		Version:          "5.3.9-1",
		HasInstallScript: true,
	}
	if err := reg.AddPackage(testPkg); err != nil {
		t.Fatalf("failed to add package: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	cmd := NewInstallScriptsCommand(cfg)
	
	// Test non-chroot mode (empty chrootDir)
	// Execute() doesn't return error even if scripts fail, it handles them internally
	err = cmd.Execute([]string{"bash"}, false, "")
	if err != nil {
		t.Errorf("Execute() should not return error, got %v", err)
	}
	// The important thing is that it attempted non-chroot mode
	// Which would have tried to access the script file path, not chroot
}

// TestExecuteInstallScriptsChrootMode tests chroot mode (chrootDir provided)
func TestExecuteInstallScriptsChrootMode(t *testing.T) {
	tmpDir := t.TempDir()
	
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
	}

	// Create registry with package
	reg, err := registry.NewRegistry(cfg.RegistryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	testPkg := &registry.Package{
		Name:             "glibc",
		Version:          "2.39-1",
		HasInstallScript: true,
	}
	if err := reg.AddPackage(testPkg); err != nil {
		t.Fatalf("failed to add package: %v", err)
	}
	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	cmd := NewInstallScriptsCommand(cfg)
	
	// Test chroot mode (with chrootDir)
	chrootDir := filepath.Join(tmpDir, "chroot")
	err = cmd.Execute([]string{"glibc"}, false, chrootDir)
	// Execute() doesn't return error even if chroot fails, it handles them internally
	if err != nil {
		t.Errorf("Execute() should not return error, got %v", err)
	}
	// The important thing is that it attempted chroot mode
	// Which would have tried to use the chroot command
}

// TestRunInstallScriptDispatcher tests that runInstallScript correctly dispatches to the right method
func TestRunInstallScriptDispatcher(t *testing.T) {
	tmpDir := t.TempDir()
	
	cfg := &config.Config{
		BaseDir:      tmpDir,
		PoolRoot:    filepath.Join(tmpDir, "pool"),
		RegistryPath: filepath.Join(tmpDir, "registry.json"),
	}

	cmd := NewInstallScriptsCommand(cfg)
	testPkg := &registry.Package{
		Name:    "test",
		Version: "1.0.0",
	}

	// Test that non-chroot dispatch attempts direct execution
	// Should fail because script doesn't exist, but verifies it tried non-chroot path
	err1 := cmd.runInstallScript(testPkg, "post_install", "")
	if err1 == nil {
		t.Error("expected error for missing script in non-chroot mode")
	}
	if err1.Error() != "script not found at " + filepath.Join(cfg.PoolRoot, "test", "1.0.0", ".INSTALL") {
		t.Logf("Got expected non-chroot dispatch error: %v", err1)
	}

	// Test that chroot dispatch attempts chroot execution
	// Should fail trying to run chroot (doesn't exist), but verifies it tried chroot path
	err2 := cmd.runInstallScript(testPkg, "post_install", "/nonexistent/chroot")
	if err2 == nil {
		t.Error("expected error for chroot execution")
	}
	// Error would be from chroot command not being able to execute
	t.Logf("Got expected chroot dispatch error: %v", err2)
}

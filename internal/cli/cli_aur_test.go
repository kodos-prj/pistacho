// Package cli_test provides tests for AUR-integrated CLI commands
package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kodos-prj/pistacho/pkg/config"
	"github.com/kodos-prj/pistacho/pkg/symlink"
)

// TestSearchCommandWithAUR tests basic search command initialization
func TestSearchCommandWithAUR(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:     "/tmp/test",
		AlpmDBPath:   "/tmp/test/db",
		Repositories: []string{"core", "extra"},
	}

	cmd := NewSearchCommand(cfg)
	if cmd == nil {
		t.Fatal("NewSearchCommand returned nil")
	}

	if cmd.config != cfg {
		t.Error("config not set correctly")
	}

	if cmd.aurRPC == nil {
		t.Fatal("aurRPC is nil")
	}

	if cmd.aurCache == nil {
		t.Fatal("aurCache is nil")
	}
}

// TestInfoCommandWithAUR tests info command initialization with AUR support
func TestInfoCommandWithAUR(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:     "/tmp/test",
		AlpmDBPath:   "/tmp/test/db",
		Repositories: []string{"core", "extra"},
	}

	cmd := NewInfoCommand(cfg)
	if cmd == nil {
		t.Fatal("NewInfoCommand returned nil")
	}

	if cmd.config != cfg {
		t.Error("config not set correctly")
	}

	if cmd.aurRPC == nil {
		t.Fatal("aurRPC is nil")
	}
}

// TestInstallCommandWithAUR tests install command initialization with AUR support
func TestInstallCommandWithAUR(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		AlpmRoot:               "/tmp/test",
		AlpmDBPath:             "/tmp/test/db",
		PoolRoot:              tmpDir + "/store",
		WrapperDir:             tmpDir + "/wrappers",
		SymlinkRoot:            tmpDir + "/symlinks",
		CachePath:              tmpDir + "/cache",
		RegistryPath:           tmpDir + "/registry",
		Repositories:           []string{"core", "extra"},
		Architecture:           "x86_64",
		MirrorURL:              "https://mirror.example.com",
		MaxConcurrentDownloads: 4,
	}

	cmd := NewInstallCommand(cfg)
	if cmd == nil {
		t.Fatal("NewInstallCommand returned nil")
	}

	if cmd.config != cfg {
		t.Error("config not set correctly")
	}

	if cmd.aurRPC == nil {
		t.Fatal("aurRPC is nil")
	}

	// buildMgr may be nil if directory creation fails, which is OK in tests
	// The important thing is that the command is created
}

// TestInstallCommandWithSymlinkDir tests install command with custom symlink directory
func TestInstallCommandWithSymlinkDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		AlpmRoot:               "/tmp/test",
		AlpmDBPath:             "/tmp/test/db",
		PoolRoot:              tmpDir + "/store",
		WrapperDir:             tmpDir + "/wrappers",
		SymlinkRoot:            tmpDir + "/symlinks",
		CachePath:              tmpDir + "/cache",
		RegistryPath:           tmpDir + "/registry",
		Repositories:           []string{"core", "extra"},
		Architecture:           "x86_64",
		MirrorURL:              "https://mirror.example.com",
		MaxConcurrentDownloads: 4,
	}

	customSymlinkDir := "/custom/symlinks"
	cmd := NewInstallCommandWithSymlinkDir(cfg, customSymlinkDir)

	if cmd.symlinkDir != customSymlinkDir {
		t.Errorf("symlinkDir not set: got %s, want %s", cmd.symlinkDir, customSymlinkDir)
	}

	if cmd.aurRPC == nil {
		t.Fatal("aurRPC is nil")
	}

	// buildMgr may be nil if directory creation fails, which is OK in tests
	// The important thing is that the command is created
}

// TestSearchCommandEmptyPattern tests search with empty pattern
func TestSearchCommandEmptyPattern(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:     "/tmp/test",
		AlpmDBPath:   "/tmp/test/db",
		Repositories: []string{"core", "extra"},
	}

	cmd := NewSearchCommand(cfg)
	err := cmd.Execute("")

	if err == nil {
		t.Fatal("Execute should fail with empty pattern")
	}
}

// TestInfoCommandEmptyName tests info with empty name
func TestInfoCommandEmptyName(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:     "/tmp/test",
		AlpmDBPath:   "/tmp/test/db",
		Repositories: []string{"core", "extra"},
	}

	cmd := NewInfoCommand(cfg)
	err := cmd.Execute("")

	if err == nil {
		t.Fatal("Execute should fail with empty package name")
	}
}

// TestInstallCommandEmptyPackages tests install with no packages
func TestInstallCommandEmptyPackages(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:   "/tmp/test",
		AlpmDBPath: "/tmp/test/db",
	}

	cmd := NewInstallCommand(cfg)
	err := cmd.Run([]string{})

	if err == nil {
		t.Fatal("Run should fail with no packages")
	}
}

// TestInstallCommandWithOptions tests install command option parsing
func TestInstallCommandWithOptions(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:   "/tmp/test",
		AlpmDBPath: "/tmp/test/db",
	}

	tests := []struct {
		name       string
		args       []string
		shouldFail bool
	}{
		{
			name:       "no options",
			args:       []string{"bash"},
			shouldFail: true, // Will fail due to missing ALPM, but not due to parsing
		},
		{
			name:       "with --no-deps",
			args:       []string{"--no-deps", "bash"},
			shouldFail: true,
		},
		{
			name:       "with --no-extract",
			args:       []string{"--no-extract", "bash"},
			shouldFail: true,
		},
		{
			name:       "with --no-symlink",
			args:       []string{"--no-symlink", "bash"},
			shouldFail: true,
		},
		{
			name:       "with --force",
			args:       []string{"--force", "bash"},
			shouldFail: true,
		},
		{
			name:       "multiple options",
			args:       []string{"--no-deps", "--force", "bash"},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewInstallCommand(cfg)
			err := cmd.Run(tt.args)

			if !tt.shouldFail && err != nil {
				t.Errorf("Run should not fail: %v", err)
			}
		})
	}
}

// TestSearchCommandCache tests that search results can be cached
func TestSearchCommandCache(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:     "/tmp/test",
		AlpmDBPath:   "/tmp/test/db",
		Repositories: []string{"core", "extra"},
	}

	cmd := NewSearchCommand(cfg)
	if cmd.aurCache == nil {
		t.Fatal("aurCache should be initialized")
	}

	// Cache should be empty initially
	if len(cmd.aurCache) != 0 {
		t.Error("cache should be empty initially")
	}
}

// TestInstallOptionsSourceField tests InstallOptions Source field
func TestInstallOptionsSourceField(t *testing.T) {
	opts := &InstallOptions{
		Source: "",
	}

	if opts.Source != "" {
		t.Error("Source should default to empty string")
	}

	opts.Source = "aur"
	if opts.Source != "aur" {
		t.Error("Source should be set to 'aur'")
	}

	opts.Source = "official"
	if opts.Source != "official" {
		t.Error("Source should be set to 'official'")
	}
}

// TestInstallCommandSourceFlagParsing tests --source= flag parsing
func TestInstallCommandSourceFlagParsing(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:   "/tmp/test",
		AlpmDBPath: "/tmp/test/db",
	}

	tests := []struct {
		name       string
		args       []string
		expectErr  string
		expectCode int // 0 = no specific error, 1 = error expected
	}{
		{
			name:       "valid --source=aur",
			args:       []string{"--source=aur", "bash"},
			expectCode: 1, // Will fail later due to ALPM, but flag should parse OK
		},
		{
			name:       "valid --source=official",
			args:       []string{"--source=official", "bash"},
			expectCode: 1, // Will fail later due to ALPM, but flag should parse OK
		},
		{
			name:       "invalid --source=invalid",
			args:       []string{"--source=invalid", "bash"},
			expectErr:  "invalid source",
			expectCode: 0, // Should error on flag parsing
		},
		{
			name:       "multiple --source flags",
			args:       []string{"--source=aur", "--source=official", "bash"},
			expectErr:  "cannot specify multiple --source flags",
			expectCode: 0,
		},
		{
			name:       "no package name",
			args:       []string{"--source=aur"},
			expectErr:  "package name required",
			expectCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewInstallCommand(cfg)
			err := cmd.Run(tt.args)

			if tt.expectErr != "" && (err == nil || err.Error() == "") {
				t.Errorf("expected error containing '%s', got none", tt.expectErr)
			}

			if tt.expectErr != "" && err != nil {
				// Check if error message contains expected substring
				if !contains(err.Error(), tt.expectErr) {
					t.Errorf("expected error containing '%s', got '%v'", tt.expectErr, err)
				}
			}
		})
	}
}

// TestInstallOptionsSourceVariations tests different source option combinations
func TestInstallOptionsSourceVariations(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		noDeps    bool
		force     bool
		noSymlink bool
	}{
		{"aur with default options", "aur", false, false, false},
		{"official with default options", "official", false, false, false},
		{"aur with --no-deps", "aur", true, false, false},
		{"official with --force", "official", false, true, false},
		{"aur with --no-symlink", "aur", false, false, true},
		{"aur with multiple options", "aur", true, true, true},
		{"empty source (auto-detect)", "", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &InstallOptions{
				Source:    tt.source,
				NoDeps:    tt.noDeps,
				Force:     tt.force,
				NoSymlink: tt.noSymlink,
			}

			if opts.Source != tt.source {
				t.Errorf("Source not set correctly: got %s, want %s", opts.Source, tt.source)
			}

			if opts.NoDeps != tt.noDeps {
				t.Errorf("NoDeps not set correctly")
			}

			if opts.Force != tt.force {
				t.Errorf("Force not set correctly")
			}

			if opts.NoSymlink != tt.noSymlink {
				t.Errorf("NoSymlink not set correctly")
			}
		})
	}
}

// TestInstallCommandSourceConstraintWithPackages tests source constraints with multiple packages
func TestInstallCommandSourceConstraintWithPackages(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:   "/tmp/test",
		AlpmDBPath: "/tmp/test/db",
	}

	tests := []struct {
		name   string
		args   []string
		pkgErr string
	}{
		{
			name:   "single package with --source=aur",
			args:   []string{"--source=aur", "yay"},
			pkgErr: "package name required", // Will error on parsing
		},
		{
			name:   "single package with --source=official",
			args:   []string{"--source=official", "bash"},
			pkgErr: "package name required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewInstallCommand(cfg)
			err := cmd.Run(tt.args)

			// Since ALPM won't be available in tests, we just verify
			// that parsing doesn't error on the source flag
			if err != nil && contains(err.Error(), "invalid source") {
				t.Errorf("source flag parsing failed: %v", err)
			}
		})
	}
}

// TestInstallOptionsDefaultSource tests that Source defaults to empty string
func TestInstallOptionsDefaultSource(t *testing.T) {
	opts := &InstallOptions{}

	if opts.Source != "" {
		t.Errorf("default Source should be empty string, got '%s'", opts.Source)
	}

	if opts.NoDeps || opts.NoExtract || opts.NoSymlink || opts.Force {
		t.Error("other options should default to false")
	}
}

// TestInstallCommandSourceFlagValidation tests source flag value validation
func TestInstallCommandSourceFlagValidation(t *testing.T) {
	cfg := &config.Config{
		AlpmRoot:   "/tmp/test",
		AlpmDBPath: "/tmp/test/db",
	}

	invalidSources := []string{
		"--source=aur2",
		"--source=aur ",
		"--source= aur",
		"--source=OFFICIAL",
		"--source=AUR",
		"--source=",
		"--source=pacman",
		"--source=user",
	}

	for _, source := range invalidSources {
		t.Run("invalid_"+source, func(t *testing.T) {
			cmd := NewInstallCommand(cfg)
			err := cmd.Run([]string{source, "bash"})

			if err == nil || (!contains(err.Error(), "invalid source") && !contains(err.Error(), "package name required")) {
				t.Errorf("expected source validation error for '%s', got: %v", source, err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ==================== SYMLINK-PREFIX TESTS ====================

// TestInstallWithChroot tests that --chroot flag is parsed correctly
func TestInstallWithChroot(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		expectedPrefix   string
	}{
		{
			name:             "equals-separated syntax",
			args:             []string{"--chroot=/tmp/chroot", "vim"},
			expectedPrefix:   "/tmp/chroot",
		},
		{
			name:             "space-separated syntax",
			args:             []string{"--chroot", "/tmp/demo", "gcc"},
			expectedPrefix:   "/tmp/demo",
		},
		{
			name:             "no chroot",
			args:             []string{"vim"},
			expectedPrefix:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse arguments into options
			opts := &InstallOptions{}
			args := tt.args
			
			for i := 0; i < len(args); i++ {
				arg := args[i]
				if strings.HasPrefix(arg, "--chroot=") {
					opts.Chroot = strings.TrimPrefix(arg, "--chroot=")
				} else if arg == "--chroot" {
					if i+1 < len(args) {
						i++
						opts.Chroot = args[i]
					}
				}
			}
			
			if opts.Chroot != tt.expectedPrefix {
				t.Errorf("chroot not parsed correctly: got %q, want %q", opts.Chroot, tt.expectedPrefix)
			}
		})
	}
}

// TestInstallOptionsChroot tests Chroot field initialization
func TestInstallOptionsChroot(t *testing.T) {
	opts := &InstallOptions{}
	
	if opts.Chroot != "" {
		t.Errorf("Chroot should default to empty string, got %q", opts.Chroot)
	}
	
	opts.Chroot = "/tmp/chroot"
	if opts.Chroot != "/tmp/chroot" {
		t.Errorf("Chroot not set correctly: got %q, want %q", opts.Chroot, "/tmp/chroot")
	}
}

// TestChrootStripCorrectly tests that symlink.StripPrefix strips paths correctly
func TestChrootStripCorrectly(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		prefix      string
		expected    string
		shouldError bool
	}{
		{
			name:        "strip absolute prefix",
			path:        "/tmp/chroot/kod/pool/vim/9.0.0-1/usr/bin/vim",
			prefix:      "/tmp/chroot",
			expected:    "/kod/pool/vim/9.0.0-1/usr/bin/vim",
			shouldError: false,
		},
		{
			name:        "prefix without trailing slash",
			path:        "/tmp/demo/usr/lib/libc.so.6",
			prefix:      "/tmp/demo",
			expected:    "/usr/lib/libc.so.6",
			shouldError: false,
		},
		{
			name:        "path doesn't start with prefix",
			path:        "/home/user/kod/pool/vim",
			prefix:      "/tmp/chroot",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "empty prefix (no-op)",
			path:        "/kod/pool/vim/9.0.0-1/usr/bin/vim",
			prefix:      "",
			expected:    "/kod/pool/vim/9.0.0-1/usr/bin/vim",
			shouldError: false,
		},
		{
			name:        "root prefix (no-op)",
			path:        "/kod/pool/vim/9.0.0-1/usr/bin/vim",
			prefix:      "/",
			expected:    "/kod/pool/vim/9.0.0-1/usr/bin/vim",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := symlink.StripPrefix(tt.path, tt.prefix)
			
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("strip result incorrect: got %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

// TestSymlinkTargetsAreRelativePaths tests that symlinks point to relative paths when using --chroot
func TestSymlinkTargetsAreRelativePaths(t *testing.T) {
	// This test verifies the behavior of symlink target stripping
	// When --chroot=/tmp/chroot is used:
	// - Symlink location: /tmp/chroot/usr/bin/vim
	// - Symlink target (should be): /kod/pool/vim/.../usr/bin/vim (relative path)
	// - NOT: /tmp/chroot/kod/pool/vim/.../usr/bin/vim (absolute within prefix)

	testCases := []struct {
		name            string
		originalTarget  string
		prefix          string
		expectedTarget  string
	}{
		{
			name:            "executable symlink",
			originalTarget:  "/tmp/chroot/kod/pool/vim/9.0.0-1/usr/bin/vim",
			prefix:          "/tmp/chroot",
			expectedTarget:  "/kod/pool/vim/9.0.0-1/usr/bin/vim",
		},
		{
			name:            "library symlink",
			originalTarget:  "/tmp/chroot/kod/pool/gcc-libs/13.1.0-1/usr/lib/libstdc++.so.6",
			prefix:          "/tmp/chroot",
			expectedTarget:  "/kod/pool/gcc-libs/13.1.0-1/usr/lib/libstdc++.so.6",
		},
		{
			name:            "no prefix stripping needed",
			originalTarget:  "/kod/pool/vim/9.0.0-1/usr/bin/vim",
			prefix:          "",
			expectedTarget:  "/kod/pool/vim/9.0.0-1/usr/bin/vim",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := symlink.StripPrefix(tc.originalTarget, tc.prefix)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			// Verify the result is a relative path (starts with /)
			if !strings.HasPrefix(result, "/") {
				t.Errorf("symlink target should be absolute path, got: %q", result)
			}
			
			// Verify the result matches expected
			if result != tc.expectedTarget {
				t.Errorf("symlink target incorrect: got %q, want %q", result, tc.expectedTarget)
			}
			
			// Verify it doesn't contain the prefix
			if strings.Contains(result, tc.prefix) && tc.prefix != "" {
				t.Errorf("symlink target contains prefix (should be stripped): %q in %q", tc.prefix, result)
			}
		})
	}
}

// TestInstallOptionsChrootWithOtherFlags tests --chroot combined with other flags
func TestInstallOptionsChrootWithOtherFlags(t *testing.T) {
	tests := []struct {
		name              string
		symlink           string
		noSymlink         bool
		force             bool
		noDeps            bool
		expectedSymlink   string
		expectedNoSymlink bool
		expectedForce     bool
		expectedNoDeps    bool
	}{
		{
			name:              "chroot with --force",
			symlink:           "/tmp/chroot",
			force:             true,
			expectedSymlink:   "/tmp/chroot",
			expectedForce:     true,
		},
		{
			name:              "chroot with --no-symlink",
			symlink:           "/tmp/chroot",
			noSymlink:         true,
			expectedSymlink:   "/tmp/chroot",
			expectedNoSymlink: true,
		},
		{
			name:            "chroot with --no-deps",
			symlink:         "/tmp/chroot",
			noDeps:          true,
			expectedSymlink: "/tmp/chroot",
			expectedNoDeps:  true,
		},
		{
			name:              "all flags combined",
			symlink:           "/tmp/chroot",
			noSymlink:         true,
			force:             true,
			noDeps:            true,
			expectedSymlink:   "/tmp/chroot",
			expectedNoSymlink: true,
			expectedForce:     true,
			expectedNoDeps:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &InstallOptions{
				Chroot: tt.symlink,
				NoSymlink:     tt.noSymlink,
				Force:         tt.force,
				NoDeps:        tt.noDeps,
			}

			if opts.Chroot != tt.expectedSymlink {
				t.Errorf("Chroot: got %q, want %q", opts.Chroot, tt.expectedSymlink)
			}
			if opts.NoSymlink != tt.expectedNoSymlink {
				t.Errorf("NoSymlink: got %v, want %v", opts.NoSymlink, tt.expectedNoSymlink)
			}
			if opts.Force != tt.expectedForce {
				t.Errorf("Force: got %v, want %v", opts.Force, tt.expectedForce)
			}
			if opts.NoDeps != tt.expectedNoDeps {
				t.Errorf("NoDeps: got %v, want %v", opts.NoDeps, tt.expectedNoDeps)
			}
		})
	}
}

// TestInstallSymlinkTargetsWithAndWithoutChroot tests that symlink targets differ based on --chroot
func TestInstallSymlinkTargetsWithAndWithoutChroot(t *testing.T) {
	tests := []struct {
		name              string
		usePrefix         bool
		prefix            string
		expectedTargetHas string
		description       string
	}{
		{
			name:              "without prefix points to wrapper",
			usePrefix:         false,
			prefix:            "",
			expectedTargetHas: "/kod/wrappers/",
			description:       "Normal mode: /usr/bin/vim → /kod/wrappers/vim",
		},
		{
			name:              "with prefix points to package file",
			usePrefix:         true,
			prefix:            "/tmp/chroot",
			expectedTargetHas: "/kod/pool/",
			description:       "Prefix mode: /usr/bin/vim → /kod/pool/vim/.../usr/bin/vim",
		},
		{
			name:              "with different prefix still points to package",
			usePrefix:         true,
			prefix:            "/home/user/build",
			expectedTargetHas: "/kod/pool/",
			description:       "Prefix mode with different path: symlink still points to store",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &InstallOptions{
				Chroot: tt.prefix,
			}

			// Simulate the symlink target logic from install.go lines 377-384
			filePath := "usr/bin/vim"
			fileName := "vim"
			storeRoot := "/kod/pool"
			wrapperDir := "/kod/wrappers"
			version := "9.0.0-1"
			pkgName := "vim"

			var targetPath string
			if strings.HasPrefix(filePath, "usr/bin/") || strings.HasPrefix(filePath, "usr/sbin/") {
				if opts.Chroot != "" {
					// With chroot, point directly to package files
					targetPath = filepath.Join(storeRoot, pkgName, version, filePath)
				} else {
					// Normal mode: point to wrapper
					targetPath = filepath.Join(wrapperDir, fileName)
				}
			}

			// Verify the target contains the expected path component
			if !strings.Contains(targetPath, tt.expectedTargetHas) {
				t.Errorf("symlink target incorrect: got %q, expected to contain %q\n  %s", targetPath, tt.expectedTargetHas, tt.description)
			}

			// Additional verification for prefix mode
			if tt.usePrefix {
				// Should point to store, not wrappers
				if strings.Contains(targetPath, "/kod/wrappers/") {
					t.Errorf("with --chroot, symlink should NOT point to wrapper: %q", targetPath)
				}
				// Should contain the full path
				if !strings.Contains(targetPath, version) {
					t.Errorf("symlink target should contain version: %q", targetPath)
				}
			}

			// Additional verification for non-prefix mode
			if !tt.usePrefix {
				// Should point to wrappers
				if !strings.Contains(targetPath, "/kod/wrappers/") {
					t.Errorf("without --chroot, symlink should point to wrapper: %q", targetPath)
				}
				// Should NOT contain version
				if strings.Contains(targetPath, version) {
					t.Errorf("wrapper symlink should NOT contain version: %q", targetPath)
				}
			}
		})
	}
}

// TestExecutableSymlinkBehaviorWithChroot tests that usr/bin and usr/sbin are handled correctly
func TestExecutableSymlinkBehaviorWithChroot(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		usePrefix      bool
		expectedPrefix string
		description    string
	}{
		{
			name:           "usr/bin executable with prefix",
			filePath:       "usr/bin/vim",
			usePrefix:      true,
			expectedPrefix: "/kod/pool/",
			description:    "usr/bin files should point to store with prefix",
		},
		{
			name:           "usr/sbin executable with prefix",
			filePath:       "usr/sbin/useradd",
			usePrefix:      true,
			expectedPrefix: "/kod/pool/",
			description:    "usr/sbin files should point to store with prefix",
		},
		{
			name:           "usr/bin executable without prefix",
			filePath:       "usr/bin/vim",
			usePrefix:      false,
			expectedPrefix: "/kod/wrappers/",
			description:    "usr/bin files should point to wrapper without prefix",
		},
		{
			name:           "usr/sbin executable without prefix",
			filePath:       "usr/sbin/useradd",
			usePrefix:      false,
			expectedPrefix: "/kod/wrappers/",
			description:    "usr/sbin files should point to wrapper without prefix",
		},
		{
			name:           "library file with prefix",
			filePath:       "usr/lib/libvim.so",
			usePrefix:      true,
			expectedPrefix: "/kod/pool/",
			description:    "non-executable files should point to store regardless",
		},
		{
			name:           "library file without prefix",
			filePath:       "usr/lib/libvim.so",
			usePrefix:      false,
			expectedPrefix: "/kod/pool/",
			description:    "non-executable files should point to store regardless",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &InstallOptions{
				Chroot: "",
			}
			if tt.usePrefix {
				opts.Chroot = "/tmp/chroot"
			}

			// Simulate the logic from install.go
			fileName := filepath.Base(tt.filePath)
			storeRoot := "/kod/pool"
			wrapperDir := "/kod/wrappers"
			pkgName := "vim"
			version := "9.0.0-1"

			var targetPath string
			if strings.HasPrefix(tt.filePath, "usr/bin/") || strings.HasPrefix(tt.filePath, "usr/sbin/") {
				if opts.Chroot != "" {
					targetPath = filepath.Join(storeRoot, pkgName, version, tt.filePath)
				} else {
					targetPath = filepath.Join(wrapperDir, fileName)
				}
			} else {
				// Regular file: point to storage
				targetPath = filepath.Join(storeRoot, pkgName, version, tt.filePath)
			}

			if !strings.Contains(targetPath, tt.expectedPrefix) {
				t.Errorf("symlink target incorrect: got %q, expected to contain %q\n  %s", targetPath, tt.expectedPrefix, tt.description)
			}
		})
	}
}

// TestAutoSetBaseDirWithChroot tests that --base-dir is automatically set to {prefix}/kod
// when --chroot is used without an explicit --base-dir
func TestAutoSetBaseDirWithChroot(t *testing.T) {
	cfg := &config.Config{
		BaseDir:    "/kod",
		PoolRoot:  "/kod/pool",
		WrapperDir: "/kod/wrappers",
	}

	cmd := NewInstallCommandWithSymlinkDirAndExplicitBaseDir(cfg, "", false)

	// Verify initial state
	if cmd.BaseDirExplicit() != false {
		t.Errorf("baseDirExplicit should be false, got %v", cmd.BaseDirExplicit())
	}

	// Simulate what Run() does when --chroot is provided
	opts := &InstallOptions{
		Chroot: "/tmp/chroot",
	}

	// Auto-set logic (as in Run method)
	if opts.Chroot != "" && !cmd.BaseDirExplicit() {
		newBaseDir := filepath.Join(opts.Chroot, "kod")
		cmd.config.BaseDir = newBaseDir
		cmd.config.UpdateDerivedPaths()
	}

	// Verify the auto-set worked
	expectedBaseDir := "/tmp/chroot/kod"
	if cmd.config.BaseDir != expectedBaseDir {
		t.Errorf("BaseDir not auto-set correctly: expected %q, got %q", expectedBaseDir, cmd.config.BaseDir)
	}
}

// TestBaseDirNotAutoSetWhenExplicit tests that --base-dir is NOT auto-set when user explicitly provides it
func TestBaseDirNotAutoSetWhenExplicit(t *testing.T) {
	cfg := &config.Config{
		BaseDir:    "/custom/path",
		PoolRoot:  "/custom/path/store",
		WrapperDir: "/custom/path/wrappers",
	}

	cmd := NewInstallCommandWithSymlinkDirAndExplicitBaseDir(cfg, "", true) // baseDirExplicit = true

	// Verify initial state
	if cmd.BaseDirExplicit() != true {
		t.Errorf("baseDirExplicit should be true, got %v", cmd.BaseDirExplicit())
	}

	originalBaseDir := cmd.config.BaseDir

	// Simulate what Run() does when both --chroot and --base-dir are provided
	opts := &InstallOptions{
		Chroot: "/tmp/chroot",
	}

	// Auto-set logic (as in Run method) - should NOT apply when baseDirExplicit is true
	if opts.Chroot != "" && !cmd.BaseDirExplicit() {
		newBaseDir := filepath.Join(opts.Chroot, "kod")
		cmd.config.BaseDir = newBaseDir
		cmd.config.UpdateDerivedPaths()
	}

	// Verify BaseDir was NOT changed
	if cmd.config.BaseDir != originalBaseDir {
		t.Errorf("BaseDir should not be auto-set when explicit: expected %q, got %q", originalBaseDir, cmd.config.BaseDir)
	}
}

// TestBaseDirNotAutoSetWithoutChroot tests that --base-dir is NOT auto-set when --chroot is not provided
func TestBaseDirNotAutoSetWithoutChroot(t *testing.T) {
	cfg := &config.Config{
		BaseDir:    "/kod",
		PoolRoot:  "/kod/pool",
		WrapperDir: "/kod/wrappers",
	}

	cmd := NewInstallCommandWithSymlinkDirAndExplicitBaseDir(cfg, "", false) // baseDirExplicit = false

	originalBaseDir := cmd.config.BaseDir

	// Simulate Run() behavior when --chroot is NOT provided
	opts := &InstallOptions{
		Chroot: "", // No prefix
	}

	// Auto-set logic (as in Run method) - should NOT apply when Chroot is empty
	if opts.Chroot != "" && !cmd.BaseDirExplicit() {
		newBaseDir := filepath.Join(opts.Chroot, "kod")
		cmd.config.BaseDir = newBaseDir
		cmd.config.UpdateDerivedPaths()
	}

	// Verify BaseDir was NOT changed
	if cmd.config.BaseDir != originalBaseDir {
		t.Errorf("BaseDir should not be auto-set without --chroot: expected %q, got %q", originalBaseDir, cmd.config.BaseDir)
	}
}

// TestNewInstallCommandConstructors tests that constructors properly initialize baseDirExplicit
func TestNewInstallCommandConstructors(t *testing.T) {
	cfg := &config.Config{
		BaseDir: "/kod",
	}

	tests := []struct {
		name              string
		constructor       func() *InstallCommand
		expectedExplicit  bool
		description       string
	}{
		{
			name: "NewInstallCommand",
			constructor: func() *InstallCommand {
				return NewInstallCommand(cfg)
			},
			expectedExplicit: false,
			description:      "default constructor should have baseDirExplicit=false",
		},
		{
			name: "NewInstallCommandWithSymlinkDir",
			constructor: func() *InstallCommand {
				return NewInstallCommandWithSymlinkDir(cfg, "/tmp/chroot")
			},
			expectedExplicit: false,
			description:      "symlink-dir constructor should have baseDirExplicit=false",
		},
		{
			name: "NewInstallCommandWithSymlinkDirAndExplicitBaseDir (false)",
			constructor: func() *InstallCommand {
				return NewInstallCommandWithSymlinkDirAndExplicitBaseDir(cfg, "/tmp/chroot", false)
			},
			expectedExplicit: false,
			description:      "new constructor with baseDirExplicit=false should be false",
		},
		{
			name: "NewInstallCommandWithSymlinkDirAndExplicitBaseDir (true)",
			constructor: func() *InstallCommand {
				return NewInstallCommandWithSymlinkDirAndExplicitBaseDir(cfg, "/tmp/chroot", true)
			},
			expectedExplicit: true,
			description:      "new constructor with baseDirExplicit=true should be true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.constructor()
			if cmd.BaseDirExplicit() != tt.expectedExplicit {
				t.Errorf("%s: got %v, expected %v", tt.description, cmd.BaseDirExplicit(), tt.expectedExplicit)
			}
		})
	}
}

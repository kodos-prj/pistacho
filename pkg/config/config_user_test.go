package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetUserConfigPath tests user config path detection
func TestGetUserConfigPath(t *testing.T) {
	// Save original env
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
		os.Setenv("HOME", originalHome)
	}()

	tests := []struct {
		name     string
		xdgHome  string
		home     string
		contains string
	}{
		{
			name:     "XDG_CONFIG_HOME set",
			xdgHome:  "/custom/config",
			home:     "/home/user",
			contains: "/custom/config/pith/config.json",
		},
		{
			name:     "XDG_CONFIG_HOME not set",
			xdgHome:  "",
			home:     "/home/user",
			contains: "/home/user/.config/pith/config.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("XDG_CONFIG_HOME", tt.xdgHome)
			os.Setenv("HOME", tt.home)

			path, err := GetUserConfigPath()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if path != tt.contains {
				t.Errorf("expected %s, got %s", tt.contains, path)
			}
		})
	}
}

// TestGetUserBaseDir tests user base directory detection
func TestGetUserBaseDir(t *testing.T) {
	// Save original env
	originalUserBase := os.Getenv("PITH_USER_BASE_DIR")
	originalXDG := os.Getenv("XDG_DATA_HOME")
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("PITH_USER_BASE_DIR", originalUserBase)
		os.Setenv("XDG_DATA_HOME", originalXDG)
		os.Setenv("HOME", originalHome)
	}()

	tests := []struct {
		name        string
		userBaseDir string
		xdgDataHome string
		home        string
		contains    string
	}{
		{
			name:        "PITH_USER_BASE_DIR set",
			userBaseDir: "/custom/data",
			xdgDataHome: "",
			home:        "/home/user",
			contains:    "/custom/data",
		},
		{
			name:        "XDG_DATA_HOME set",
			userBaseDir: "",
			xdgDataHome: "/custom/data",
			home:        "/home/user",
			contains:    "/custom/data/pith",
		},
		{
			name:        "Default XDG path",
			userBaseDir: "",
			xdgDataHome: "",
			home:        "/home/user",
			contains:    "/home/user/.local/share/pith",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("PITH_USER_BASE_DIR", tt.userBaseDir)
			os.Setenv("XDG_DATA_HOME", tt.xdgDataHome)
			os.Setenv("HOME", tt.home)

			path, err := GetUserBaseDir()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if path != tt.contains {
				t.Errorf("expected %s, got %s", tt.contains, path)
			}
		})
	}
}

// TestLoadUserConfig tests loading user configuration
func TestLoadUserConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create user config directory and file
	userConfigDir := filepath.Join(tmpDir, ".config", "pith")
	os.MkdirAll(userConfigDir, 0755)

	userConfigFile := filepath.Join(userConfigDir, "config.json")
	userConfig := `{
  "base_dir": "` + filepath.Join(tmpDir, ".local", "share", "pith") + `",
  "mirror_url": "https://custom-mirror.example.com/archlinux"
}`
	os.WriteFile(userConfigFile, []byte(userConfig), 0644)

	// Set HOME to tmpDir
	originalHome := os.Getenv("HOME")
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
	}()

	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	cfg, err := LoadUserConfig()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if cfg.MirrorURL != "https://custom-mirror.example.com/archlinux" {
		t.Errorf("expected custom mirror, got %s", cfg.MirrorURL)
	}
}

// TestDefaultUserConfig tests default user configuration
func TestDefaultUserConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Set environment for user paths
	originalHome := os.Getenv("HOME")
	originalXDG := os.Getenv("XDG_DATA_HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_DATA_HOME", originalXDG)
	}()

	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_DATA_HOME", "")

	cfg, err := DefaultUserConfig()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expectedBase := filepath.Join(tmpDir, ".local", "share", "pith")
	if cfg.BaseDir != expectedBase {
		t.Errorf("expected base dir %s, got %s", expectedBase, cfg.BaseDir)
	}

	expectedSymlink := filepath.Join(tmpDir, ".local", "bin")
	if cfg.SymlinkRoot != expectedSymlink {
		t.Errorf("expected symlink root %s, got %s", expectedSymlink, cfg.SymlinkRoot)
	}

	// Verify derived paths are set
	if cfg.PoolRoot != filepath.Join(cfg.BaseDir, "pool") {
		t.Error("pool root not properly derived from base dir")
	}

	if cfg.WrapperDir != filepath.Join(cfg.BaseDir, "wrappers") {
		t.Error("wrapper dir not properly derived from base dir")
	}
}

// TestUserConfigWithEnvironmentOverride tests env var override
func TestUserConfigWithEnvironmentOverride(t *testing.T) {
	tmpDir := t.TempDir()
	customBaseDir := filepath.Join(tmpDir, "custom", "packages")

	// Set environment
	originalUserBase := os.Getenv("PITH_USER_BASE_DIR")
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("PITH_USER_BASE_DIR", originalUserBase)
		os.Setenv("HOME", originalHome)
	}()

	os.Setenv("PITH_USER_BASE_DIR", customBaseDir)
	os.Setenv("HOME", tmpDir)

	baseDir, err := GetUserBaseDir()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if baseDir != customBaseDir {
		t.Errorf("expected %s, got %s", customBaseDir, baseDir)
	}
}

// TestUserConfigPaths tests all derived paths in user config
func TestUserConfigPaths(t *testing.T) {
	tmpDir := t.TempDir()

	originalHome := os.Getenv("HOME")
	originalXDG := os.Getenv("XDG_DATA_HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("XDG_DATA_HOME", originalXDG)
	}()

	os.Setenv("HOME", tmpDir)
	os.Setenv("XDG_DATA_HOME", "")

	cfg, err := DefaultUserConfig()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	baseDir := cfg.BaseDir

	// Check all derived paths
	checks := map[string]string{
		"PoolRoot":     filepath.Join(baseDir, "pool"),
		"RegistryPath": filepath.Join(baseDir, "registry.json"),
		"DBPath":       filepath.Join(baseDir, "db", "sync"),
		"WrapperDir":   filepath.Join(baseDir, "wrappers"),
		"CachePath":    filepath.Join(baseDir, "cache"),
	}

	for name, expected := range checks {
		actual := ""
		switch name {
		case "PoolRoot":
			actual = cfg.PoolRoot
		case "RegistryPath":
			actual = cfg.RegistryPath
		case "DBPath":
			actual = cfg.DBPath
		case "WrapperDir":
			actual = cfg.WrapperDir
		case "CachePath":
			actual = cfg.CachePath
		}

		if actual != expected {
			t.Errorf("%s: expected %s, got %s", name, expected, actual)
		}
	}
}

// TestUserConfigCreation tests that user config can be created and loaded
func TestUserConfigCreation(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "pith")
	os.MkdirAll(configDir, 0755)

	configFile := filepath.Join(configDir, "config.json")

	cfg, err := DefaultUserConfig()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Update base dir for testing
	cfg.BaseDir = filepath.Join(tmpDir, ".local", "share", "pith")
	cfg.UpdateDerivedPaths()

	// Save config
	err = cfg.Save(configFile)
	if err != nil {
		t.Errorf("failed to save config: %v", err)
	}

	// Load config back
	loaded, err := Load(configFile)
	if err != nil {
		t.Errorf("failed to load config: %v", err)
	}

	if loaded.BaseDir != cfg.BaseDir {
		t.Errorf("base dir mismatch: expected %s, got %s", cfg.BaseDir, loaded.BaseDir)
	}
}

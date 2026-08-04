// Package config manages pith configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultConfigPath is the default location for the config file
	DefaultConfigPath = "/etc/pith/config.json"

	// DefaultBaseDir is the default base directory for all pith data
	DefaultBaseDir = "."

	// DefaultSymlinkRoot is the default root for symlinks (system root)
	DefaultSymlinkRoot = "/"
)

// Config represents the pith configuration.
type Config struct {
	// BaseDir is the base directory for all pith data (. by default, current working directory)
	// All paths below are relative to this unless they're absolute
	BaseDir string `json:"base_dir"`

	// SymlinkRoot is the root directory where symlinks are created (/ by default)
	// This is the system root where package files are symlinked to
	SymlinkRoot string `json:"symlink_root"`

	// PoolRoot is the root directory for the package pool
	// Defaults to {BaseDir}/pool
	PoolRoot string `json:"pool_root"`

	// RegistryPath is the path to the registry file
	// Defaults to {BaseDir}/registry.json
	RegistryPath string `json:"registry_path"`

	// AlpmRoot is the root directory for ALPM
	AlpmRoot string `json:"alpm_root"`

	// AlpmDBPath is the database path for ALPM
	AlpmDBPath string `json:"alpm_db_path"`

	// DBPath is the directory for synced Arch databases
	// Defaults to {BaseDir}/db
	DBPath string `json:"db_path"`

	// WrapperDir is the directory for wrapper scripts
	// Defaults to {BaseDir}/wrappers
	WrapperDir string `json:"wrapper_dir"`

	// CachePath is the directory for downloaded package files
	// Defaults to {BaseDir}/cache
	CachePath string `json:"cache_path"`

	// MirrorURL is the Arch Linux mirror base URL
	MirrorURL string `json:"mirror_url"`

	// Architecture is the target architecture (x86_64, aarch64)
	Architecture string `json:"architecture"`

	// Repositories is the list of repositories to sync
	Repositories []string `json:"repositories"`

	// VerifySignatures determines if package signatures should be verified
	VerifySignatures bool `json:"verify_signatures"`

	// MaxConcurrentDownloads is the max number of concurrent downloads
	MaxConcurrentDownloads int `json:"max_concurrent_downloads"`

	// DownloadTimeout is the timeout for downloads in seconds
	DownloadTimeout int `json:"download_timeout"`

	// KeepVersions is the number of old package versions to keep during cleanup
	KeepVersions int `json:"keep_versions"`
}

// DefaultConfig returns a configuration with default values.
func DefaultConfig() *Config {
	baseDir := DefaultBaseDir
	return &Config{
		BaseDir:                baseDir,
		SymlinkRoot:            DefaultSymlinkRoot,
		PoolRoot:              filepath.Join(baseDir, "pool"),
		RegistryPath:           filepath.Join(baseDir, "registry.json"),
		AlpmRoot:               baseDir,                              // Use current directory as ALPM root for cross-distribution
		AlpmDBPath:             filepath.Join(baseDir, "db"),         // Directory containing sync/ subdirectory
		DBPath:                 filepath.Join(baseDir, "db", "sync"), // Actual sync databases location
		WrapperDir:             filepath.Join(baseDir, "wrappers"),
		CachePath:              filepath.Join(baseDir, "cache"), // Package cache
		MirrorURL:              "https://mirror.rackspace.com/archlinux",
		Architecture:           "x86_64",
		Repositories:           []string{"core", "extra"},
		VerifySignatures:       false, // Optional for simplicity
		MaxConcurrentDownloads: 5,
		DownloadTimeout:        300,
		KeepVersions:           3,
	}
}

// Load reads configuration from a file.
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Normalize the config (set defaults for empty fields)
	cfg.Normalize()

	return &cfg, nil
}

// GetUserConfigPath returns the user-level config path following XDG spec.
// Priority: $XDG_CONFIG_HOME/pith/config.json -> ~/.config/pith/config.json
func GetUserConfigPath() (string, error) {
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "pith", "config.json"), nil
	}

	// Fall back to ~/.config/pith/config.json
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "pith", "config.json"), nil
}

// GetUserBaseDir returns the user-level base directory for pith data.
// Priority: $PITH_USER_BASE_DIR -> $XDG_DATA_HOME/pith -> ~/.local/share/pith
func GetUserBaseDir() (string, error) {
	// Check environment variable first
	if userBase := os.Getenv("PITH_USER_BASE_DIR"); userBase != "" {
		return userBase, nil
	}

	// Check XDG_DATA_HOME
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "pith"), nil
	}

	// Fall back to ~/.local/share/pith
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}

	return filepath.Join(homeDir, ".local", "share", "pith"), nil
}

// LoadUserConfig loads configuration with user-level support.
// Tries user config first, then system config, then defaults.
func LoadUserConfig() (*Config, error) {
	// Try user config first
	userConfigPath, err := GetUserConfigPath()
	if err == nil {
		if _, err := os.Stat(userConfigPath); err == nil {
			// User config exists, load it
			cfg, err := Load(userConfigPath)
			if err != nil {
				return nil, err
			}
			return cfg, nil
		}
	}

	// Fall back to system config
	return Load(DefaultConfigPath)
}

// DefaultUserConfig returns a configuration for user-level use.
func DefaultUserConfig() (*Config, error) {
	baseDir, err := GetUserBaseDir()
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	cfg.BaseDir = baseDir
	cfg.UpdateDerivedPaths()

	// For user-level, use home directory instead of system root for symlinks
	homeDir, err := os.UserHomeDir()
	if err == nil {
		// Create a .local/bin for user symlinks
		cfg.SymlinkRoot = filepath.Join(homeDir, ".local", "bin")
	}

	return cfg, nil
}

// Normalize ensures all config fields have valid values.
// It sets defaults for any empty fields based on BaseDir.
func (c *Config) Normalize() {
	// Set BaseDir default if empty
	if c.BaseDir == "" {
		c.BaseDir = DefaultBaseDir
	}

	// Set SymlinkRoot default if empty
	if c.SymlinkRoot == "" {
		c.SymlinkRoot = DefaultSymlinkRoot
	}

	// Set PoolRoot default if empty
	if c.PoolRoot == "" {
		c.PoolRoot = filepath.Join(c.BaseDir, "pool")
	}

	// Set RegistryPath default if empty
	if c.RegistryPath == "" {
		c.RegistryPath = filepath.Join(c.BaseDir, "registry.json")
	}

	// Set AlpmRoot default if empty (use BaseDir for cross-distribution)
	if c.AlpmRoot == "" {
		c.AlpmRoot = c.BaseDir
	}

	// Set AlpmDBPath default if empty (directory containing sync/)
	if c.AlpmDBPath == "" {
		c.AlpmDBPath = filepath.Join(c.BaseDir, "db")
	}

	// Set DBPath default if empty (actual sync databases location)
	if c.DBPath == "" {
		c.DBPath = filepath.Join(c.BaseDir, "db", "sync")
	}

	// Set WrapperDir default if empty
	if c.WrapperDir == "" {
		c.WrapperDir = filepath.Join(c.BaseDir, "wrappers")
	}

	// Set CachePath default if empty (simplified: ./cache)
	if c.CachePath == "" {
		c.CachePath = filepath.Join(c.BaseDir, "cache")
	}

	// Set MirrorURL default if empty
	if c.MirrorURL == "" {
		c.MirrorURL = "https://mirror.rackspace.com/archlinux"
	}

	// Set Architecture default if empty
	if c.Architecture == "" {
		c.Architecture = "x86_64"
	}

	// Set Repositories default if empty
	if len(c.Repositories) == 0 {
		c.Repositories = []string{"core", "extra", "community"}
	}

	// Set MaxConcurrentDownloads default if zero
	if c.MaxConcurrentDownloads == 0 {
		c.MaxConcurrentDownloads = 5
	}

	// Set DownloadTimeout default if zero
	if c.DownloadTimeout == 0 {
		c.DownloadTimeout = 300
	}

	// Set KeepVersions default if zero
	if c.KeepVersions == 0 {
		c.KeepVersions = 3
	}
}

// UpdateDerivedPaths updates all paths that are derived from BaseDir.
// Call this after changing BaseDir to ensure all derived paths are updated.
func (c *Config) UpdateDerivedPaths() {
	c.PoolRoot = filepath.Join(c.BaseDir, "pool")
	c.RegistryPath = filepath.Join(c.BaseDir, "registry.json")
	c.AlpmRoot = c.BaseDir
	c.AlpmDBPath = filepath.Join(c.BaseDir, "db")
	c.DBPath = filepath.Join(c.BaseDir, "db", "sync")
	c.WrapperDir = filepath.Join(c.BaseDir, "wrappers")
	c.CachePath = filepath.Join(c.BaseDir, "cache")
}

// Save writes configuration to a file.
func (c *Config) Save(path string) error {
	if path == "" {
		path = DefaultConfigPath
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

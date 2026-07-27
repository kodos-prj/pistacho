package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kodos-prj/pistacho/pkg/config"
	"github.com/kodos-prj/pistacho/pkg/store"
)

// ExtractCommand handles extracting packages.
type ExtractCommand struct {
	config *config.Config
}

// NewExtractCommand creates a new extract command.
func NewExtractCommand(cfg *config.Config) *ExtractCommand {
	return &ExtractCommand{
		config: cfg,
	}
}

// Run executes the extract command.
// Usage: pith extract <package.pkg.tar.zst> [<package2.pkg.tar.zst>] ...
func (e *ExtractCommand) Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("package file path required")
	}

	// Create store manager
	storeManager := store.NewStore(e.config.PoolRoot)

	var totalExtracted int
	var totalSize int64

	for _, pkgPath := range args {
		// Verify file exists
		if _, err := os.Stat(pkgPath); err != nil {
			fmt.Fprintf(os.Stderr, "✗ Package not found: %s\n", pkgPath)
			continue
		}

		// Extract filename without extension
		baseName := filepath.Base(pkgPath)
		baseName = baseName[:len(baseName)-len(".pkg.tar.zst")] // Remove .pkg.tar.zst

		// Parse package name and version from filename
		// Expected format: name-version-arch.pkg.tar.zst
		// Extract name and version
		var pkgName, pkgVersion string
		parts := pathToPackageParts(baseName)
		if len(parts) >= 2 {
			pkgName = parts[0]
			pkgVersion = parts[1]
		} else {
			fmt.Fprintf(os.Stderr, "✗ Invalid package filename format: %s\n", baseName)
			continue
		}

		fmt.Printf("Extracting %s/%s to store...\n", pkgName, pkgVersion)

		// Extract package to store
		extracted, err := storeManager.ExtractPackage(pkgPath, pkgName, pkgVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Failed to extract %s: %v\n", baseName, err)
			continue
		}

		// Calculate size
		size, _ := storeManager.GetPackageSize(pkgName, pkgVersion)

		fmt.Printf("  ✓ Extracted %d files (%d bytes)\n", len(extracted), size)
		totalExtracted += len(extracted)
		totalSize += size

		// Set as current version
		if err := storeManager.SetLatestVersion(pkgName, pkgVersion); err != nil {
			fmt.Fprintf(os.Stderr, "  ! Warning: Failed to set as current version: %v\n", err)
		} else {
			fmt.Printf("  ✓ Set as current version\n")
		}
	}

	fmt.Printf("\n✓ Extraction complete: %d files extracted (%d bytes)\n", totalExtracted, totalSize)
	return nil
}

// pathToPackageParts parses a package filename to extract name-version parts.
// Example: "bash-5.3.9-1-x86_64" -> ["bash", "5.3.9-1"]
// Format: <name>-<version>-<pkgrel>-<arch>
func pathToPackageParts(pkgName string) []string {
	parts := strings.Split(pkgName, "-")
	
	// Need at least 4 parts: name, version, pkgrel, arch
	if len(parts) < 4 {
		return []string{pkgName}
	}
	
	// Join all but the last 2 parts (arch and pkgrel) as version
	// The rest is the name
	name := strings.Join(parts[:len(parts)-2], "-")
	version := parts[len(parts)-2]
	
	return []string{name, version}
}

// Help returns help text for the extract command.
func (e *ExtractCommand) Help() string {
	return `Extract packages to the store.

Usage:
  pistacho extract <package.pkg.tar.zst> [package2.pkg.tar.zst] ...

Options:
  --no-symlink     Don't create symlinks after extraction
  --skip-current   Don't set as current version
  
Examples:
  pistacho extract bash-5.3.9-1-x86_64.pkg.tar.zst
  pistacho extract /tmp/bash-5.3.9-1-x86_64.pkg.tar.zst
  pistacho extract *.pkg.tar.zst
`
}

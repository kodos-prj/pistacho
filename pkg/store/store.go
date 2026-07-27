// Package store manages the central package pool at /kod/pool.
// It handles package extraction, storage, and cleanup operations.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kodos-prj/chisel/pkg/extract"
)

const (
	// DefaultPoolRoot is the default location for the package pool
	DefaultPoolRoot = "/kod/pool"
)

// Store manages package storage operations.
type Store struct {
	root      string
	extractor *extract.Extractor
}

// NewStore creates a new package store manager.
func NewStore(root string) *Store {
	if root == "" {
		root = DefaultPoolRoot
	}
	return &Store{
		root:      root,
		extractor: extract.NewExtractor(true),
	}
}

// GetPackagePath returns the storage path for a specific package version.
func (s *Store) GetPackagePath(pkgName, version string) string {
	return filepath.Join(s.root, pkgName, version)
}

// GetLatestPath returns the path to the "current" symlink for a package.
// This symlink should point to the latest installed version.
func (s *Store) GetLatestPath(pkgName string) string {
	return filepath.Join(s.root, pkgName, "current")
}

// ExtractPackage extracts a package archive to the store.
// Creates directory structure: /kod/pool/{package}/{version}/
func (s *Store) ExtractPackage(pkgPath, pkgName, version string) ([]extract.ExtractedFile, error) {
	// Create target directory for this version
	destDir := s.GetPackagePath(pkgName, version)

	// Extract the package
	extracted, err := s.extractor.ExtractPackage(pkgPath, destDir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract package: %w", err)
	}

	return extracted, nil
}

// RemovePackage removes a package version from the store.
func (s *Store) RemovePackage(pkgName, version string) error {
	pkgPath := s.GetPackagePath(pkgName, version)

	// Remove the version directory
	if err := os.RemoveAll(pkgPath); err != nil {
		return fmt.Errorf("failed to remove package directory: %w", err)
	}

	return nil
}

// ListVersions returns all stored versions of a package.
func (s *Store) ListVersions(pkgName string) ([]string, error) {
	pkgDir := filepath.Join(s.root, pkgName)

	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // Package doesn't exist, return empty list
		}
		return nil, fmt.Errorf("failed to read package directory: %w", err)
	}

	var versions []string
	for _, entry := range entries {
		// Skip non-directories and the "current" symlink
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "current" {
			continue
		}

		versions = append(versions, entry.Name())
	}

	// Sort versions in descending order (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))

	return versions, nil
}

// PackageExists checks if a package version exists in the store.
func (s *Store) PackageExists(pkgName, version string) bool {
	pkgPath := s.GetPackagePath(pkgName, version)
	info, err := os.Stat(pkgPath)
	return err == nil && info.IsDir()
}

// GetPackageSize returns the total size of a package version in bytes.
func (s *Store) GetPackageSize(pkgName, version string) (int64, error) {
	pkgPath := s.GetPackagePath(pkgName, version)

	var totalSize int64
	err := filepath.Walk(pkgPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}

// GetAllPackages returns information about all packages in the store.
func (s *Store) GetAllPackages() (map[string][]string, error) {
	packages := make(map[string][]string)

	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return packages, nil // Store doesn't exist yet
		}
		return nil, fmt.Errorf("failed to read store directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pkgName := entry.Name()
		versions, err := s.ListVersions(pkgName)
		if err == nil && len(versions) > 0 {
			packages[pkgName] = versions
		}
	}

	return packages, nil
}

// CleanupOldVersions removes old versions of a package, keeping only the specified number.
// Returns the number of versions removed.
func (s *Store) CleanupOldVersions(pkgName string, keepVersions int) (int, error) {
	versions, err := s.ListVersions(pkgName)
	if err != nil {
		return 0, err
	}

	if len(versions) <= keepVersions {
		return 0, nil // Nothing to clean up
	}

	// Remove oldest versions
	removed := 0
	for i := keepVersions; i < len(versions); i++ {
		if err := s.RemovePackage(pkgName, versions[i]); err != nil {
			return removed, fmt.Errorf("failed to remove version %s: %w", versions[i], err)
		}
		removed++
	}

	return removed, nil
}

// SetLatestVersion creates/updates the "current" symlink for a package version.
func (s *Store) SetLatestVersion(pkgName, version string) error {
	if !s.PackageExists(pkgName, version) {
		return fmt.Errorf("package version does not exist: %s/%s", pkgName, version)
	}

	currentLink := s.GetLatestPath(pkgName)
	target := s.GetPackagePath(pkgName, version)

	// Remove existing symlink if it exists
	if _, err := os.Lstat(currentLink); err == nil {
		if err := os.Remove(currentLink); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// Create new symlink
	// Use relative path for better portability
	relTarget, err := filepath.Rel(filepath.Dir(currentLink), target)
	if err != nil {
		// Fall back to absolute path if relative fails
		relTarget = target
	}

	if err := os.Symlink(relTarget, currentLink); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// GetLatestVersion returns the version that "current" points to.
func (s *Store) GetLatestVersion(pkgName string) (string, error) {
	currentLink := s.GetLatestPath(pkgName)

	// Read the symlink target
	target, err := os.Readlink(currentLink)
	if err != nil {
		if os.IsNotExist(err) {
			// No current version set, return latest
			versions, err := s.ListVersions(pkgName)
			if err != nil || len(versions) == 0 {
				return "", fmt.Errorf("no versions found for package %s", pkgName)
			}
			return versions[0], nil // Latest (first after sorting in descending order)
		}
		return "", fmt.Errorf("failed to read symlink: %w", err)
	}

	// Extract version from path (handle both relative and absolute paths)
	parts := strings.Split(filepath.Clean(target), string(filepath.Separator))
	if len(parts) > 0 {
		return parts[len(parts)-1], nil
	}

	return "", fmt.Errorf("invalid symlink target: %s", target)
}

// ValidateStore checks the integrity of stored packages.
// Returns a list of issues found.
func (s *Store) ValidateStore() []string {
	var issues []string

	packages, err := s.GetAllPackages()
	if err != nil {
		issues = append(issues, fmt.Sprintf("Failed to read store: %v", err))
		return issues
	}

	for pkgName, versions := range packages {
		if len(versions) == 0 {
			issues = append(issues, fmt.Sprintf("Package %s has no versions", pkgName))
			continue
		}

		for _, version := range versions {
			if !s.PackageExists(pkgName, version) {
				issues = append(issues, fmt.Sprintf("Package %s/%s directory missing", pkgName, version))
			}
		}

		// Check if current symlink is valid
		currentLink := s.GetLatestPath(pkgName)
		if _, err := os.Stat(currentLink); err != nil {
			if !os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("Package %s current symlink broken: %v", pkgName, err))
			}
		}
	}

	return issues
}

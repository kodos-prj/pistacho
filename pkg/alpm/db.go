package alpm

import (
	"fmt"
	"os"
	"regexp"
)

// NewClient creates a new ALPM client for pure Go package database operations.
// dbPath is the database directory (e.g., "/var/lib/pacman/sync" or "/kod/var/lib/pacman/sync")
// arch is the system architecture for filtering packages (e.g., "x86_64", "aarch64")
func NewGoClient(dbPath, arch string) *Client {
	return &Client{
		DbPath:       dbPath,
		Arch:         arch,
		Cache:        NewDatabaseCache(),
		Databases:    []*Database{},
		DownloadURLs: make(map[string]string),
	}
}

// RegisterSyncDB registers a sync database by loading it from the disk cache.
// Multiple calls register multiple databases; precedence is determined by registration order.
// Examples: "core", "extra", "community", "multilib"
func (c *Client) RegisterSyncDB(repo string) error {
	db, err := c.LoadCachedDatabase(repo)
	if err != nil {
		return fmt.Errorf("failed to load database %s: %w", repo, err)
	}

	c.Databases = append(c.Databases, db)
	c.Cache.AddDatabase(db)

	return nil
}

// RegisterAllSyncDBs registers multiple sync databases at once.
// Databases are registered in order, which determines precedence.
// Databases that don't exist are skipped with a warning instead of failing.
func (c *Client) RegisterAllSyncDBs(repos []string) error {
	var registered []string
	var skipped []string

	for _, repo := range repos {
		if err := c.RegisterSyncDB(repo); err != nil {
			// Check if this is a "file not found" error
			dbPath := fmt.Sprintf("%s/%s.db", c.DbPath, repo)
			if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
				// Database file doesn't exist, skip it
				skipped = append(skipped, repo)
				continue
			}
			// For other errors, return them
			return err
		}
		registered = append(registered, repo)
	}

	// If no databases were registered, that's an error
	if len(registered) == 0 {
		return fmt.Errorf("failed to register any sync databases (checked: %v)", repos)
	}

	return nil
}

// SearchPackage searches for a package by exact name.
// Returns the package if found, or an error if not found.
// Respects repository precedence (first registered = highest priority).
func (c *Client) SearchPackage(name string) (*Package, error) {
	pkg := c.Cache.GetPackage(name, c.Arch)
	if pkg == nil {
		return nil, fmt.Errorf("package %s not found", name)
	}
	return pkg, nil
}

// SearchPackages searches for packages matching a regex pattern in name or description.
// Results are filtered by system architecture.
// Returns all matching packages, or error if pattern is invalid.
func (c *Client) SearchPackages(pattern string) ([]*Package, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	var results []*Package
	seen := make(map[string]bool)

	for _, pkg := range c.Cache.GetAllPackages() {
		// Skip if already seen (handles duplicates from multiple databases)
		if seen[pkg.Name] {
			continue
		}
		seen[pkg.Name] = true

		// Filter by architecture
		if pkg.Architecture != "any" && pkg.Architecture != c.Arch {
			continue
		}

		// Match against name or description
		if re.MatchString(pkg.Name) || re.MatchString(pkg.Description) {
			results = append(results, pkg)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no packages found matching %s", pattern)
	}

	return results, nil
}

// GetPackageInfo returns detailed information about a package.
// Returns PackageInfo struct with all metadata.
func (c *Client) GetPackageInfo(name string) (*PackageInfo, error) {
	pkg, err := c.SearchPackage(name)
	if err != nil {
		return nil, err
	}

	info := &PackageInfo{
		Name:         pkg.Name,
		Version:      pkg.Version,
		Description:  pkg.Description,
		Architecture: pkg.Architecture,
		URL:          pkg.URL,
		Size:         pkg.InstalledSize,
		DownloadSize: pkg.CompressedSize,
		Repository:   pkg.Repository,
		PackageBase:  pkg.PackageBase,
		Licenses:     pkg.Licenses,
		Groups:       pkg.Groups,
		Provides:     pkg.Provides,
		DependsOn:    pkg.DependsOn,
		OptDepends:   pkg.OptDepends,
		Conflicts:    pkg.Conflicts,
		Replaces:     pkg.Replaces,
		Maintainer:   pkg.Maintainer,
	}

	return info, nil
}

// ResolveDependencies resolves all dependencies for a package.
// Returns a list of package names that need to be installed, in dependency order.
// Dependencies come before dependents in the list.
// Returns error if circular dependency is detected or dependency cannot be resolved.
func (c *Client) ResolveDependencies(packageName string) ([]string, error) {
	pkg, err := c.SearchPackage(packageName)
	if err != nil {
		return nil, err
	}

	// Track visited packages to detect cycles
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	var result []string

	err = c.resolveDepsRecursive(pkg, visited, visiting, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// resolveDepsRecursive recursively resolves dependencies with cycle detection.
func (c *Client) resolveDepsRecursive(pkg *Package, visited, visiting map[string]bool, result *[]string) error {
	// Cycle detection: if we're currently visiting this package, we have a cycle
	if visiting[pkg.Name] {
		return &ResolutionError{
			Reason: fmt.Sprintf("circular dependency detected at %s", pkg.Name),
			Cycle:  []string{pkg.Name},
		}
	}

	// Already fully resolved
	if visited[pkg.Name] {
		return nil
	}

	// Mark as currently visiting
	visiting[pkg.Name] = true
	defer delete(visiting, pkg.Name)

	// Resolve all dependencies first (depth-first)
	for _, depStr := range pkg.DependsOn {
		depName, constraint, err := ParseDependency(depStr)
		if err != nil {
			return fmt.Errorf("failed to parse dependency %s: %w", depStr, err)
		}

		// Try to find the dependency package
		depPkg := c.Cache.GetPackage(depName, c.Arch)
		if depPkg == nil {
			// Try to find a package that provides this
			providers := c.Cache.GetProvidingPackages(depName)
			if len(providers) == 0 {
				return &ResolutionError{
					Reason: fmt.Sprintf("dependency %s not found (required by %s)", depName, pkg.Name),
				}
			}
			depPkg = providers[0] // Use first provider
		}

		// Check version constraint
		if !CheckVersionConstraint(depPkg.Version, constraint) {
			return &ResolutionError{
				Reason: fmt.Sprintf("dependency %s version %s does not satisfy constraint %s=%s",
					depName, depPkg.Version, depName, constraint.Value),
			}
		}

		// Recursively resolve
		if err := c.resolveDepsRecursive(depPkg, visited, visiting, result); err != nil {
			return err
		}
	}

	// Mark as visited and add to result
	visited[pkg.Name] = true
	*result = append(*result, pkg.Name)

	return nil
}

// ListSyncDBs returns a list of registered sync database names.
func (c *Client) ListSyncDBs() ([]string, error) {
	var names []string
	for _, db := range c.Databases {
		names = append(names, db.Name)
	}
	return names, nil
}

// SearchPackagesByGroup searches for packages in a given group.
// Returns all packages that belong to the group across all databases.
func (c *Client) SearchPackagesByGroup(groupName string) []*Package {
	return c.Cache.GetPackagesByGroup(groupName)
}

// ListAllGroups returns all package group names from all databases.
func (c *Client) ListAllGroups() []string {
	return c.Cache.ListAllGroups()
}

// Close releases resources (not needed for pure Go implementation but kept for API compatibility).
func (c *Client) Close() error {
	// No resources to close in pure Go implementation
	return nil
}

// GetDownloadURL constructs the download URL for a package.
// Format: {mirror}/archlinux/{repo}/os/{arch}/{pkgname}-{version}-{arch}.pkg.tar.zst
func (c *Client) GetDownloadURL(pkg *Package, mirrorURL string) string {
	arch := pkg.Architecture
	if arch == "any" {
		arch = c.Arch
	}
	return fmt.Sprintf("%s/archlinux/%s/os/%s/%s-%s-%s.pkg.tar.zst",
		mirrorURL,
		pkg.Repository,
		arch,
		pkg.Name,
		pkg.Version,
		pkg.Architecture,
	)
}

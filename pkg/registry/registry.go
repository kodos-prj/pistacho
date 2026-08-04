// Package registry manages the JSON-based package registry.
// It tracks installed packages, their versions, and file lists.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const (
	// DefaultRegistryPath is the default location for the registry file
	DefaultRegistryPath = "./registry.json"
)

// Package represents an installed package in the registry.
type Package struct {
	Name              string   `json:"name"`
	Version           string   `json:"version"`
	Source            string   `json:"source"`     // "official" or "aur"
	Repository        string   `json:"repository"` // e.g., "core", "extra", "aur"
	Files             []string `json:"files"`
	Executables       []string `json:"executables"`
	Dependencies      []string `json:"dependencies"`
	InstallDate       string   `json:"install_date"`
	UpdateDate        string   `json:"update_date,omitempty"` // When the package was last updated
	HasInstallScript  bool     `json:"has_install_script,omitempty"`
}

// Registry manages the package registry.
type Registry struct {
	path     string
	packages map[string]*Package
	mu       sync.RWMutex
}

// NewRegistry creates a new registry manager.
func NewRegistry(path string) (*Registry, error) {
	if path == "" {
		path = DefaultRegistryPath
	}

	r := &Registry{
		path:     path,
		packages: make(map[string]*Package),
	}

	// Load existing registry if it exists
	if err := r.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	return r, nil
}

// Load reads the registry from disk.
func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &r.packages)
}

// Save writes the registry to disk.
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := json.MarshalIndent(r.packages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	return os.WriteFile(r.path, data, 0644)
}

// AddPackage adds or updates a package in the registry.
func (r *Registry) AddPackage(pkg *Package) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.packages[pkg.Name] = pkg
	return nil
}

// RemovePackage removes a package from the registry.
func (r *Registry) RemovePackage(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.packages, name)
	return nil
}

// GetPackage retrieves a package from the registry.
func (r *Registry) GetPackage(name string) (*Package, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pkg, ok := r.packages[name]
	return pkg, ok
}

// ListPackages returns all packages in the registry.
func (r *Registry) ListPackages() []*Package {
	r.mu.RLock()
	defer r.mu.RUnlock()

	packages := make([]*Package, 0, len(r.packages))
	for _, pkg := range r.packages {
		packages = append(packages, pkg)
	}
	return packages
}

// GetAURPackages returns only AUR packages from the registry.
func (r *Registry) GetAURPackages() []*Package {
	r.mu.RLock()
	defer r.mu.RUnlock()

	packages := make([]*Package, 0)
	for _, pkg := range r.packages {
		if pkg.Source == "aur" {
			packages = append(packages, pkg)
		}
	}
	return packages
}

// GetOfficialPackages returns only official repository packages from the registry.
func (r *Registry) GetOfficialPackages() []*Package {
	r.mu.RLock()
	defer r.mu.RUnlock()

	packages := make([]*Package, 0)
	for _, pkg := range r.packages {
		if pkg.Source == "official" {
			packages = append(packages, pkg)
		}
	}
	return packages
}

// UpdatePackageVersion updates only the version and update date for a package.
// Used when upgrading a package to a newer version.
func (r *Registry) UpdatePackageVersion(name, version, updateDate string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pkg, ok := r.packages[name]
	if !ok {
		return fmt.Errorf("package not found: %s", name)
	}

	pkg.Version = version
	pkg.UpdateDate = updateDate
	return nil
}

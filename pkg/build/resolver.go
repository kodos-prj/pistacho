// Package build provides build system integration for Pistacho.
// resolver.go implements unified dependency resolution for official and AUR packages.
package build

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/kodos-prj/pistacho/pkg/alpm"
	"github.com/kodos-prj/pistacho/pkg/aur"
)

// parsePackageName extracts the package name from a dependency string.
// Dependency strings may contain version constraints like ">=", "<=", "=", "<", ">", "=="
// Examples:
//
//	"linux-api-headers>=4.10" → "linux-api-headers"
//	"glibc" → "glibc"
//	"ncurses<5" → "ncurses"
//	"bash>=5.0" → "bash"
func parsePackageName(dep string) string {
	// Match operators: >=, <=, ==, =, >, <
	operatorRegex := regexp.MustCompile(`(>=|<=|==|=|>|<)`)
	parts := operatorRegex.Split(dep, 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return dep
}

// parseVersionConstraint extracts version constraint from a dependency string.
// Returns (operator, version) where operator is >=, <=, =, etc.
// Returns ("", "") if no constraint is present.
func parseVersionConstraint(dep string) (string, string) {
	operatorRegex := regexp.MustCompile(`(>=|<=|==|=|>|<)(.+)`)
	matches := operatorRegex.FindStringSubmatch(dep)
	if len(matches) >= 3 {
		return matches[1], matches[2]
	}
	return "", ""
}

// satisfiesVersionConstraint checks if a version satisfies a constraint.
// Returns true if version satisfies the constraint, or if no constraint is present.
func satisfiesVersionConstraint(version, operator, constraintVersion string) bool {
	if operator == "" {
		// No constraint, always satisfied
		return true
	}

	cmp := alpm.VerCmp(version, constraintVersion)

	switch operator {
	case ">=":
		return cmp >= 0
	case ">":
		return cmp > 0
	case "<=":
		return cmp <= 0
	case "<":
		return cmp < 0
	case "=", "==":
		return cmp == 0
	default:
		// Unknown operator, assume satisfied
		return true
	}
}

// PackageSource represents a package from either official repos or AUR
type PackageSource struct {
	// Package identification
	Name    string
	Version string

	// Source information
	Source string // "official" or "aur"
	Repo   string // "core", "extra", "aur", etc.
	IsAUR  bool   // Convenience flag (true if Source == "aur")

	// For AUR packages only
	PKGBUILD *aur.PKGBUILDInfo

	// Architecture (from official repo desc or PKGBUILD)
	Architecture string

	// Dependency information
	Depends     []string // Runtime dependencies
	MakeDepends []string // Build-time dependencies (AUR only)
	OptDepends  []string // Optional dependencies
}

// MixedResolver resolves dependencies across official repos and AUR
type MixedResolver struct {
	alpClient    *alpm.ALPMClient
	aurRPC       *aur.RPCClient
	gitHandler   *aur.GitHandler
	pkgBuilder   *aur.PKGBUILDParser
	tempBuildDir string // Temporary directory for cloning PKGBUILDs

	// For cycle detection and tracking
	visited    map[string]bool
	resolving  map[string]bool // Currently being resolved (detects cycles)
	resolveErr error
	mu         sync.RWMutex
}

// NewMixedResolver creates a new mixed repository resolver
func NewMixedResolver(alpClient *alpm.ALPMClient, buildCacheDir string) *MixedResolver {
	return &MixedResolver{
		alpClient:    alpClient,
		aurRPC:       aur.NewRPCClient(),
		gitHandler:   aur.NewGitHandler(buildCacheDir),
		pkgBuilder:   aur.NewPKGBUILDParser(),
		tempBuildDir: buildCacheDir,
		visited:      make(map[string]bool),
		resolving:    make(map[string]bool),
	}
}

// ResolveDependencies resolves all dependencies for a package
// Parameters:
//   - pkgName: name of the package to resolve
//   - isRootPackage: whether this is the root package (true) or a dependency (false)
//   - sourceConstraint: "" for auto-detect, "aur" for AUR only, "official" for official only
//
// Returns packages in build order (dependencies first, requested package last)
// Note: sourceConstraint is only applied to root packages; dependencies always auto-detect
func (mr *MixedResolver) ResolveDependencies(pkgName string, isRootPackage bool, sourceConstraint string) ([]PackageSource, error) {
	// Reset state for new resolution
	mr.mu.Lock()
	mr.visited = make(map[string]bool)
	mr.resolving = make(map[string]bool)
	mr.resolveErr = nil
	mr.mu.Unlock()

	var result []PackageSource
	err := mr.resolveDependenciesRecursive(pkgName, &result, isRootPackage, sourceConstraint)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// resolveDependenciesRecursive is the recursive helper for dependency resolution
func (mr *MixedResolver) resolveDependenciesRecursive(pkgName string, result *[]PackageSource, isRootPackage bool, sourceConstraint string) error {
	mr.mu.Lock()

	// Check if already visited (successfully resolved)
	if mr.visited[pkgName] {
		mr.mu.Unlock()
		return nil
	}

	// Check for cycles (package trying to resolve itself)
	if mr.resolving[pkgName] {
		mr.mu.Unlock()
		// Cycle detected - this is allowed in real Arch databases
		// Skip to prevent infinite loops
		return nil
	}

	// Mark as currently resolving
	mr.resolving[pkgName] = true
	mr.mu.Unlock()

	defer func() {
		mr.mu.Lock()
		mr.resolving[pkgName] = false
		mr.visited[pkgName] = true
		mr.mu.Unlock()
	}()

	// For root packages with sourceConstraint, enforce the constraint
	// For dependencies, always auto-detect (no constraint)
	if isRootPackage && sourceConstraint != "" {
		return mr.resolveSingleSource(pkgName, result, sourceConstraint)
	}

	// Auto-detect: try official first, then AUR
	// Try to find package in official repos first
	officialPkg, err := mr.findOfficialPackage(pkgName)
	if err == nil && officialPkg != nil {
		// Found in official repos, resolve its dependencies
		for _, dep := range officialPkg.Depends {
			// Parse the package name from the dependency string (strip version constraints)
			depName := parsePackageName(dep)
			// Dependencies always auto-detect (isRootPackage=false, sourceConstraint="")
			if err := mr.resolveDependenciesRecursive(depName, result, false, ""); err != nil {
				return err
			}
		}

		// Add the package itself
		*result = append(*result, *officialPkg)
		return nil
	}

	// Not in official repos, try AUR
	aurPkg, err := mr.findAURPackage(pkgName)
	if err != nil {
		return fmt.Errorf("package not found: %s (not in official repos or AUR): %w", pkgName, err)
	}

	if aurPkg == nil {
		return fmt.Errorf("package not found: %s", pkgName)
	}

	// Resolve AUR dependencies (both runtime and build-time)
	allDeps := append(aurPkg.Depends, aurPkg.MakeDepends...)
	for _, dep := range allDeps {
		// Parse the package name from the dependency string (strip version constraints)
		depName := parsePackageName(dep)
		// Dependencies always auto-detect (isRootPackage=false, sourceConstraint="")
		if err := mr.resolveDependenciesRecursive(depName, result, false, ""); err != nil {
			return err
		}
	}

	// Add the AUR package itself
	*result = append(*result, *aurPkg)
	return nil
}

// resolveSingleSource resolves a package from a specific source only
func (mr *MixedResolver) resolveSingleSource(pkgName string, result *[]PackageSource, source string) error {
	mr.mu.Lock()
	// Check if already visited
	if mr.visited[pkgName] {
		mr.mu.Unlock()
		return nil
	}
	// Mark as currently resolving
	mr.resolving[pkgName] = true
	mr.mu.Unlock()

	defer func() {
		mr.mu.Lock()
		mr.resolving[pkgName] = false
		mr.visited[pkgName] = true
		mr.mu.Unlock()
	}()

	if source == "official" {
		// Only search official repos
		officialPkg, err := mr.findOfficialPackage(pkgName)
		if err != nil {
			return err
		}
		if officialPkg == nil {
			return fmt.Errorf("package not found in official repositories: %s", pkgName)
		}

		// Resolve dependencies (always auto-detect for dependencies)
		for _, dep := range officialPkg.Depends {
			// Parse the package name from the dependency string (strip version constraints)
			depName := parsePackageName(dep)
			if err := mr.resolveDependenciesRecursive(depName, result, false, ""); err != nil {
				return err
			}
		}

		// Add the package itself
		*result = append(*result, *officialPkg)
		return nil

	} else if source == "aur" {
		// Only search AUR
		aurPkg, err := mr.findAURPackage(pkgName)
		if err != nil {
			return err
		}
		if aurPkg == nil {
			return fmt.Errorf("package not found in AUR: %s", pkgName)
		}

		// Resolve dependencies (always auto-detect for dependencies)
		allDeps := append(aurPkg.Depends, aurPkg.MakeDepends...)
		for _, dep := range allDeps {
			// Parse the package name from the dependency string (strip version constraints)
			depName := parsePackageName(dep)
			if err := mr.resolveDependenciesRecursive(depName, result, false, ""); err != nil {
				return err
			}
		}

		// Add the AUR package itself
		*result = append(*result, *aurPkg)
		return nil

	}

	return fmt.Errorf("unknown source constraint: %s", source)
}

// findOfficialPackage looks up a package in official repositories
// It first tries direct name match, then checks if any package provides the requested name
// This handles both regular packages and virtual packages (e.g., libncursesw.so provided by ncurses)
func (mr *MixedResolver) findOfficialPackage(pkgName string) (*PackageSource, error) {
	if mr.alpClient == nil {
		// No ALPM client available, package not in official repos
		return nil, nil
	}

	pkgInfo, err := mr.alpClient.GetPackageInfo(pkgName)
	if err == nil && pkgInfo != nil {
		return &PackageSource{
			Name:         pkgInfo.Name,
			Version:      pkgInfo.Version,
			Source:       "official",
			Repo:         pkgInfo.Repository,
			IsAUR:        false,
			Architecture: pkgInfo.Architecture,
			Depends:      pkgInfo.DependsOn,
			OptDepends:   pkgInfo.OptDepends,
		}, nil
	}

	// Direct package not found, check if any package provides this virtual package
	providingPkgs := mr.alpClient.GetProvidingPackages(pkgName)
	if len(providingPkgs) > 0 {
		// Use the first providing package
		pkg := providingPkgs[0]
		return &PackageSource{
			Name:         pkg.Name,
			Version:      pkg.Version,
			Source:       "official",
			Repo:         pkg.Repository,
			IsAUR:        false,
			Architecture: pkg.Architecture,
			Depends:      pkg.DependsOn,
			OptDepends:   pkg.OptDepends,
		}, nil
	}

	// Package not found in official repos
	return nil, nil
}

// findAURPackage looks up a package in the AUR
func (mr *MixedResolver) findAURPackage(pkgName string) (*PackageSource, error) {
	// Get package info from AUR RPC
	aurPkg, err := mr.aurRPC.GetPackage(pkgName)
	if err != nil {
		return nil, err
	}

	if aurPkg == nil {
		return nil, fmt.Errorf("package not found in AUR: %s", pkgName)
	}

	// Clone PKGBUILD to parse it
	pkgbuildPath, err := mr.gitHandler.ClonePKGBUILD(pkgName, mr.tempBuildDir)
	if err != nil {
		return nil, fmt.Errorf("failed to clone PKGBUILD for %s: %w", pkgName, err)
	}

	// Verify PKGBUILD exists
	if err := mr.gitHandler.VerifyPKGBUILD(pkgbuildPath); err != nil {
		return nil, fmt.Errorf("invalid PKGBUILD for %s: %w", pkgName, err)
	}

	// Parse PKGBUILD
	pkgbuildInfo, err := mr.pkgBuilder.Parse(mr.gitHandler.GetPKGBUILDPath(pkgbuildPath))
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKGBUILD for %s: %w", pkgName, err)
	}

	// Validate parsed PKGBUILD
	if err := mr.pkgBuilder.ValidatePKGBUILD(pkgbuildInfo); err != nil {
		return nil, fmt.Errorf("invalid PKGBUILD metadata for %s: %w", pkgName, err)
	}

	arch := "x86_64"
	if len(pkgbuildInfo.Architecture) > 0 {
		arch = pkgbuildInfo.Architecture[0]
	}

	return &PackageSource{
		Name:         aurPkg.Name,
		Version:      aurPkg.Version,
		Source:       "aur",
		Repo:         "aur",
		IsAUR:        true,
		Architecture: arch,
		PKGBUILD:     pkgbuildInfo,
		Depends:      pkgbuildInfo.Depends,
		MakeDepends:  pkgbuildInfo.MakeDepends,
		OptDepends:   pkgbuildInfo.OptDepends,
	}, nil
}

// ResolveDependenciesWithPreference resolves dependencies with preference for official repos
// This is the standard flow where official packages are preferred over AUR
// Note: This method does not use source constraints (all packages auto-detect)
func (mr *MixedResolver) ResolveDependenciesWithPreference(pkgNames []string, preferOfficial bool) ([]PackageSource, error) {
	var allResults []PackageSource
	visited := make(map[string]bool)

	for _, pkgName := range pkgNames {
		// Call with isRootPackage=true, sourceConstraint="" for auto-detect
		results, err := mr.ResolveDependencies(pkgName, true, "")
		if err != nil {
			return nil, err
		}

		// Add resolved packages, avoiding duplicates
		for _, pkg := range results {
			if !visited[pkg.Name] {
				allResults = append(allResults, pkg)
				visited[pkg.Name] = true
			}
		}
	}

	return allResults, nil
}

// buildCyclePath builds a string representation of the dependency cycle for error messages
func (mr *MixedResolver) buildCyclePath(pkgName string) string {
	// Simple cycle representation - in a full implementation,
	// we'd track the actual resolution path
	return fmt.Sprintf("[cycle involving %s]", pkgName)
}

// IsPackageInOfficial checks if a package exists in official repositories
func (mr *MixedResolver) IsPackageInOfficial(pkgName string) bool {
	if mr.alpClient == nil {
		return false
	}
	_, err := mr.alpClient.GetPackageInfo(pkgName)
	return err == nil
}

// IsPackageInAUR checks if a package exists in the AUR
func (mr *MixedResolver) IsPackageInAUR(pkgName string) bool {
	_, err := mr.aurRPC.GetPackage(pkgName)
	return err == nil
}

// GetPackageSource returns which source a package is available from
// Returns "official", "aur", "both", or "none"
func (mr *MixedResolver) GetPackageSource(pkgName string) string {
	inOfficial := mr.IsPackageInOfficial(pkgName)
	inAUR := mr.IsPackageInAUR(pkgName)

	switch {
	case inOfficial && inAUR:
		return "both"
	case inOfficial:
		return "official"
	case inAUR:
		return "aur"
	default:
		return "none"
	}
}

// Close closes the resolver and releases resources
func (mr *MixedResolver) Close() error {
	// Clean up RPC client cache
	mr.aurRPC.ClearCache()
	return nil
}

// ResolvePackageVersion gets the version of a specific package
// Prefers official repos over AUR if found in both
func (mr *MixedResolver) ResolvePackageVersion(pkgName string) (string, string, error) {
	// Try official first (if client available)
	if mr.alpClient != nil {
		if officialPkg, err := mr.alpClient.GetPackageInfo(pkgName); err == nil {
			return officialPkg.Version, "official", nil
		}
	}

	// Try AUR
	if aurPkg, err := mr.aurRPC.GetPackage(pkgName); err == nil {
		return aurPkg.Version, "aur", nil
	}

	return "", "", fmt.Errorf("package not found: %s", pkgName)
}

// GetOfficialPackageDependencies gets runtime dependencies for an official package
func (mr *MixedResolver) GetOfficialPackageDependencies(pkgName string) ([]string, error) {
	if mr.alpClient == nil {
		return nil, fmt.Errorf("ALPM client not available")
	}

	pkgInfo, err := mr.alpClient.GetPackageInfo(pkgName)
	if err != nil {
		return nil, fmt.Errorf("package not found in official repos: %s", pkgName)
	}
	return pkgInfo.DependsOn, nil
}

// GetAURPackageDependencies gets dependencies for an AUR package
func (mr *MixedResolver) GetAURPackageDependencies(pkgName string) ([]string, []string, error) {
	// Get package from AUR
	aurPkg, err := mr.aurRPC.GetPackage(pkgName)
	if err != nil {
		return nil, nil, fmt.Errorf("package not found in AUR: %s", pkgName)
	}

	// Try to get makedepends from PKGBUILD
	pkgbuildPath, err := mr.gitHandler.ClonePKGBUILD(pkgName, mr.tempBuildDir)
	if err != nil {
		// If we can't get PKGBUILD, return what we have from RPC
		return aurPkg.Depends, aurPkg.MakeDepends, nil
	}

	if err := mr.gitHandler.VerifyPKGBUILD(pkgbuildPath); err != nil {
		return aurPkg.Depends, aurPkg.MakeDepends, nil
	}

	pkgbuildInfo, err := mr.pkgBuilder.Parse(mr.gitHandler.GetPKGBUILDPath(pkgbuildPath))
	if err != nil {
		return aurPkg.Depends, aurPkg.MakeDepends, nil
	}

	return pkgbuildInfo.Depends, pkgbuildInfo.MakeDepends, nil
}

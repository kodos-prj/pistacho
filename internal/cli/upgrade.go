// Package cli implements command-line interface commands for pith.
package cli

import (
	"fmt"
	"sort"
	"sync"

	pistachoalpm "github.com/kodos-prj/pistacho/pkg/alpm"
	"github.com/kodos-prj/pistacho/pkg/aur"
	"github.com/kodos-prj/pistacho/pkg/config"
	"github.com/kodos-prj/pistacho/pkg/download"
	"github.com/kodos-prj/pistacho/pkg/registry"
	"github.com/kodos-prj/pistacho/pkg/store"
)

// UpgradeCommand implements the 'pith upgrade' command.
// It upgrades installed packages to their latest versions from repositories (official and AUR).
type UpgradeCommand struct {
	config     *config.Config
	symlinkDir string
	aurRPC     *aur.RPCClient
}

// NewUpgradeCommand creates a new upgrade command instance.
func NewUpgradeCommand(cfg *config.Config) *UpgradeCommand {
	return &UpgradeCommand{
		config:     cfg,
		symlinkDir: "",
		aurRPC:     aur.NewRPCClient(),
	}
}

// NewUpgradeCommandWithSymlinkDir creates a new upgrade command with a symlink directory.
func NewUpgradeCommandWithSymlinkDir(cfg *config.Config, symlinkDir string) *UpgradeCommand {
	return &UpgradeCommand{
		config:     cfg,
		symlinkDir: symlinkDir,
		aurRPC:     aur.NewRPCClient(),
	}
}

// UpgradeOptions holds command-line options for upgrade.
type UpgradeOptions struct {
	DryRun   bool
	Verbose  bool
	Packages []string // Empty = all packages
}

// UpgradeCandidate represents a package that can be upgraded.
type UpgradeCandidate struct {
	PackageName      string
	InstalledVersion string
	AvailableVersion string
	PackageInfo      *download.PackageInfo
	IsAutoAdded      bool // True if auto-added due to dependencies
}

// UpgradeResult represents the result of an upgrade operation.
type UpgradeResult struct {
	PackageName string
	OldVersion  string
	NewVersion  string
	Success     bool
	Error       error
	TimeSeconds int
}

// UpgradeSummary represents overall upgrade statistics.
type UpgradeSummary struct {
	Total              int
	Successful         int
	Failed             int
	SkippedNoUpdate    int
	SkippedNotFound    int
	AutoAddedCount     int
	OldVersionsCleaned int
	SpaceFreed         int64
}

// Execute runs the upgrade command.
func (u *UpgradeCommand) Execute(options *UpgradeOptions) (*UpgradeSummary, error) {
	if options == nil {
		options = &UpgradeOptions{}
	}

	summary := &UpgradeSummary{}

	// Initialize ALPM client using new pure Go wrapper
	client, err := pistachoalpm.NewClient(u.config.AlpmRoot, u.config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(u.config.Repositories); err != nil {
		return nil, fmt.Errorf("failed to register sync databases: %w", err)
	}

	// Load registry
	reg, err := registry.NewRegistry(u.config.RegistryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Get installed packages
	installedPkgs := reg.ListPackages()
	if len(installedPkgs) == 0 {
		fmt.Println("No packages installed.")
		return summary, nil
	}

	if options.Verbose {
		fmt.Println("Checking for updates...")
	}

	// Find upgrade candidates
	candidates, err := u.findCandidates(client, installedPkgs, options.Packages)
	if err != nil {
		return nil, fmt.Errorf("failed to find upgrade candidates: %w", err)
	}

	if len(candidates) == 0 {
		if options.Verbose {
			fmt.Println("All packages are up to date.")
		}
		return summary, nil
	}

	summary.Total = len(candidates)

	// Show plan
	u.showPlan(candidates, options.Verbose)

	// If dry-run, exit here
	if options.DryRun {
		if options.Verbose {
			fmt.Println("\n(No changes made)")
		}
		return summary, nil
	}

	// Execute upgrades
	results := u.executeUpgrades(candidates, options.Verbose, u.config, u.symlinkDir)

	// Count results
	for _, result := range results {
		if result.Success {
			summary.Successful++
		} else {
			summary.Failed++
		}
	}

	// Cleanup old versions if configured
	if u.config.KeepVersions > 0 {
		cleaned, freed, err := u.cleanupOldVersions()
		if err == nil {
			summary.OldVersionsCleaned = cleaned
			summary.SpaceFreed = freed
		}
	}

	// Show results
	u.showResults(candidates, results, summary, options.Verbose)

	return summary, nil
}

// findCandidates identifies packages that have newer versions available.
func (u *UpgradeCommand) findCandidates(
	client *pistachoalpm.ALPMClient,
	installedPkgs []*registry.Package,
	selectedPkgs []string,
) ([]UpgradeCandidate, error) {
	var candidates []UpgradeCandidate
	selectedMap := make(map[string]bool)

	// Build map of selected packages if provided
	for _, pkg := range selectedPkgs {
		selectedMap[pkg] = true
	}

	// Check each installed package
	for _, pkg := range installedPkgs {
		// Skip if selective upgrade and not selected
		if len(selectedPkgs) > 0 && !selectedMap[pkg.Name] {
			continue
		}

		// First, try to search in official repositories
		repoPkg, err := client.SearchPackage(pkg.Name)
		if err == nil {
			// Found in official repo - check for version update
			if pistachoalpm.VerCmp(pkg.Version, repoPkg.Version) < 0 {
				pkgInfo := &download.PackageInfo{
					Name:    repoPkg.Name,
					Version: repoPkg.Version,
					Repo:    repoPkg.Repository,
				}

				candidates = append(candidates, UpgradeCandidate{
					PackageName:      pkg.Name,
					InstalledVersion: pkg.Version,
					AvailableVersion: repoPkg.Version,
					PackageInfo:      pkgInfo,
					IsAutoAdded:      false,
				})
			}
			continue
		}

		// Not in official repos - if package source is AUR, check AUR for updates
		if pkg.Source == "aur" {
			aurPkg, err := u.aurRPC.GetPackage(pkg.Name)
			if err != nil {
				continue // Package not found in AUR either
			}

			// Compare versions using our pure Go version comparison
			if pistachoalpm.VerCmp(pkg.Version, aurPkg.Version) < 0 {
				pkgInfo := &download.PackageInfo{
					Name:    aurPkg.Name,
					Version: aurPkg.Version,
					Repo:    "aur",
				}

				candidates = append(candidates, UpgradeCandidate{
					PackageName:      pkg.Name,
					InstalledVersion: pkg.Version,
					AvailableVersion: aurPkg.Version,
					PackageInfo:      pkgInfo,
					IsAutoAdded:      false,
				})
			}
		}
	}

	return candidates, nil
}

// showPlan displays the upgrade plan.
func (u *UpgradeCommand) showPlan(candidates []UpgradeCandidate, verbose bool) {
	if !verbose {
		fmt.Printf("✓ %d packages can be upgraded\n\n", len(candidates))
		return
	}

	// Sort by name
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].PackageName < candidates[j].PackageName
	})

	fmt.Println("\nUpgrade Plan:")
	fmt.Println("┌─────────────────┬──────────────┬──────────────┬──────────┐")
	fmt.Println("│ Package         │ Current      │ Available    │ Type     │")
	fmt.Println("├─────────────────┼──────────────┼──────────────┼──────────┤")

	for _, pkg := range candidates {
		pkgType := ""
		if pkg.IsAutoAdded {
			pkgType = "[auto]"
		}
		fmt.Printf("│ %-15s │ %-12s │ %-12s │ %-8s │\n",
			pkg.PackageName, pkg.InstalledVersion, pkg.AvailableVersion, pkgType)
	}

	fmt.Println("└─────────────────┴──────────────┴──────────────┴──────────┘")

	// Count auto-added
	autoCount := 0
	for _, pkg := range candidates {
		if pkg.IsAutoAdded {
			autoCount++
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  %d packages to upgrade\n", len(candidates))
	if autoCount > 0 {
		fmt.Printf("  %d auto-added (dependencies)\n", autoCount)
	}
}

// executeUpgrades executes the upgrade for each candidate package.
func (u *UpgradeCommand) executeUpgrades(
	candidates []UpgradeCandidate,
	verbose bool,
	cfg *config.Config,
	symlinkDir string,
) []UpgradeResult {
	var results []UpgradeResult
	var resultsMu sync.Mutex

	// Create install command to reuse its logic
	installCmd := NewInstallCommandWithSymlinkDir(cfg, symlinkDir)

	if verbose {
		fmt.Println("\nProceeding with upgrade...")
	}

	// Upgrade each package
	for i, candidate := range candidates {
		if verbose {
			fmt.Printf("[%d/%d] %s (%s → %s)\n",
				i+1, len(candidates),
				candidate.PackageName,
				candidate.InstalledVersion,
				candidate.AvailableVersion)
		}

		result := UpgradeResult{
			PackageName: candidate.PackageName,
			OldVersion:  candidate.InstalledVersion,
			NewVersion:  candidate.AvailableVersion,
		}

		// Use install logic: download → extract → wrap → symlink
		pkgArgs := []string{candidate.PackageName}
		if err := installCmd.Run(pkgArgs); err != nil {
			result.Success = false
			result.Error = err
			if verbose {
				fmt.Printf("  ✗ Upgrade failed: %v\n", err)
			}
		} else {
			result.Success = true
			if verbose {
				fmt.Printf("  ✓ Upgraded successfully\n")
			}
		}

		resultsMu.Lock()
		results = append(results, result)
		resultsMu.Unlock()
	}

	return results
}

// cleanupOldVersions removes old versions of upgraded packages.
func (u *UpgradeCommand) cleanupOldVersions() (int, int64, error) {
	storeManager := store.NewStore(u.config.PoolRoot)
	var totalCleaned int
	var totalSpaceFreed int64

	// Get all packages in store
	allPkgs, err := storeManager.GetAllPackages()
	if err != nil {
		return 0, 0, err
	}

	for pkgName := range allPkgs {
		removed, err := storeManager.CleanupOldVersions(pkgName, u.config.KeepVersions)
		if err != nil {
			continue
		}

		totalCleaned += removed

		// Calculate space freed (approximate)
		if removed > 0 {
			size, _ := storeManager.GetPackageSize(pkgName, "")
			totalSpaceFreed += int64(removed) * size
		}
	}

	return totalCleaned, totalSpaceFreed, nil
}

// showResults displays the upgrade results.
func (u *UpgradeCommand) showResults(
	candidates []UpgradeCandidate,
	results []UpgradeResult,
	summary *UpgradeSummary,
	verbose bool,
) {
	if verbose {
		fmt.Println("\n✓ Upgrade completed")

		if summary.Successful > 0 {
			fmt.Printf("✓ %d packages upgraded successfully\n", summary.Successful)
		}

		if summary.Failed > 0 {
			fmt.Printf("✗ %d packages failed to upgrade\n", summary.Failed)
		}

		if summary.SpaceFreed > 0 {
			fmt.Printf("✓ Freed %.2f MB by removing old versions\n", float64(summary.SpaceFreed)/(1024*1024))
		}
	} else {
		// Minimal output
		if summary.Successful > 0 {
			fmt.Printf("✓ %d packages upgraded successfully\n", summary.Successful)
		}
		if summary.SpaceFreed > 0 {
			fmt.Printf("✓ Freed %.2f MB\n", float64(summary.SpaceFreed)/(1024*1024))
		}
	}
}

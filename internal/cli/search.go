// Package cli implements command-line interface commands for chisel.
package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kodos-prj/pistacho/pkg/alpm"
	"github.com/kodos-prj/pistacho/pkg/aur"
	"github.com/kodos-prj/pistacho/pkg/config"
)

// SearchCommand implements the 'chisel search' command.
// It searches for packages in the synced databases and AUR.
type SearchCommand struct {
	config   *config.Config
	aurRPC   *aur.RPCClient
	aurCache map[string][]aur.AURPackage // Query -> results cache
}

// NewSearchCommand creates a new search command instance.
func NewSearchCommand(cfg *config.Config) *SearchCommand {
	return &SearchCommand{
		config:   cfg,
		aurRPC:   aur.NewRPCClient(),
		aurCache: make(map[string][]aur.AURPackage),
	}
}

// Execute runs the search command.
// pattern is the package name or pattern to search for.
// Searches official repos first, then falls back to AUR.
func (s *SearchCommand) Execute(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("search pattern cannot be empty")
	}

	// Initialize ALPM client
	// Note: We pass DBPath (the sync database directory), not AlpmDBPath (the parent)
	client, err := alpm.NewClient(s.config.AlpmRoot, s.config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(s.config.Repositories); err != nil {
		return fmt.Errorf("failed to register sync databases: %w", err)
	}

	// Search in official repositories
	officialPackages, err := client.SearchPackages(pattern)
	if err != nil {
		// If official search fails, that's OK - we'll search AUR instead
		// Only if both fail will we return an error
		officialPackages = []*alpm.Package{}
	}

	// Display official results
	if len(officialPackages) > 0 {
		fmt.Printf("Official Repositories (%d found):\n\n", len(officialPackages))
		for _, pkg := range officialPackages {
			fmt.Printf("%s/%s %s\n", pkg.Repository, pkg.Name, pkg.Version)
			if pkg.Description != "" {
				fmt.Printf("    %s\n", pkg.Description)
			}
			fmt.Println()
		}
	}

	// Search in AUR
	aurResults, err := s.aurRPC.SearchPackages(pattern, 50) // Limit to 50 results
	if err != nil {
		// Don't fail if AUR search fails, just warn user
		fmt.Fprintf(os.Stderr, "Warning: AUR search failed: %v\n", err)
	}

	if len(aurResults) > 0 {
		if len(officialPackages) > 0 {
			fmt.Println("\n" + strings.Repeat("-", 50))
		}
		fmt.Printf("AUR Packages (%d found):\n\n", len(aurResults))
		for _, pkg := range aurResults {
			fmt.Printf("aur/%s %s\n", pkg.Name, pkg.Version)
			if pkg.Description != "" {
				fmt.Printf("    %s\n", pkg.Description)
			}
			// Show AUR-specific info
			fmt.Printf("    Maintainer: %s\n", pkg.Maintainer)
			if pkg.OutOfDate > 0 {
				fmt.Printf("    ⚠ OUT OF DATE\n")
			}
			fmt.Println()
		}
		s.aurCache[pattern] = aurResults
	}

	// Summary
	if len(officialPackages) == 0 && len(aurResults) == 0 {
		fmt.Printf("No packages found matching '%s'\n", pattern)
	} else if len(officialPackages) == 0 {
		fmt.Printf("\nFound %d package(s) in AUR\n", len(aurResults))
	} else if len(aurResults) == 0 {
		fmt.Printf("\nFound %d package(s) in official repositories\n", len(officialPackages))
	} else {
		fmt.Printf("\nFound %d official and %d AUR package(s)\n", len(officialPackages), len(aurResults))
	}

	return nil
}

// ExactSearch searches for an exact package name in official repos, then AUR.
func (s *SearchCommand) ExactSearch(name string) error {
	if name == "" {
		return fmt.Errorf("package name cannot be empty")
	}

	// Initialize ALPM client
	// Note: We pass DBPath (the sync database directory), not AlpmDBPath (the parent)
	client, err := alpm.NewClient(s.config.AlpmRoot, s.config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(s.config.Repositories); err != nil {
		return fmt.Errorf("failed to register sync databases: %w", err)
	}

	// Try to find in official repos first
	pkg, err := client.SearchPackage(name)
	if err == nil {
		// Found in official repo
		fmt.Printf("%s/%s %s\n", pkg.Repository, pkg.Name, pkg.Version)
		if pkg.Description != "" {
			fmt.Printf("    %s\n", pkg.Description)
		}
		return nil
	}

	// Not in official repos, try AUR
	aurPkg, err := s.aurRPC.GetPackage(name)
	if err != nil {
		return fmt.Errorf("package not found in official repositories or AUR")
	}

	fmt.Printf("aur/%s %s\n", aurPkg.Name, aurPkg.Version)
	if aurPkg.Description != "" {
		fmt.Printf("    %s\n", aurPkg.Description)
	}
	fmt.Printf("    Maintainer: %s\n", aurPkg.Maintainer)
	if aurPkg.OutOfDate > 0 {
		fmt.Printf("    ⚠ OUT OF DATE\n")
	}

	return nil
}

// SearchGroup searches for packages in a given group.
// Displays all packages that belong to the group.
func (s *SearchCommand) SearchGroup(groupName string) error {
	if groupName == "" {
		return fmt.Errorf("group name cannot be empty")
	}

	// Initialize ALPM client
	client, err := alpm.NewClient(s.config.AlpmRoot, s.config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(s.config.Repositories); err != nil {
		return fmt.Errorf("failed to register sync databases: %w", err)
	}

	// Search for packages in the group
	packages := client.SearchPackagesByGroup(groupName)
	if len(packages) == 0 {
		fmt.Printf("Group '%s' not found or contains no packages\n", groupName)
		return nil
	}

	// Display results
	fmt.Printf("Group '%s' (%d packages):\n\n", groupName, len(packages))
	for _, pkg := range packages {
		fmt.Printf("%s/%s %s\n", pkg.Repository, pkg.Name, pkg.Version)
		if pkg.Description != "" {
			fmt.Printf("    %s\n", pkg.Description)
		}
		// Show groups this package belongs to
		if len(pkg.Groups) > 0 {
			fmt.Printf("    Groups: %s\n", strings.Join(pkg.Groups, ", "))
		}
		fmt.Println()
	}

	return nil
}

// ListGroups returns all available package groups.
func (s *SearchCommand) ListGroups() error {
	// Initialize ALPM client
	client, err := alpm.NewClient(s.config.AlpmRoot, s.config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(s.config.Repositories); err != nil {
		return fmt.Errorf("failed to register sync databases: %w", err)
	}

	// Get all groups
	groups := client.ListAllGroups()
	if len(groups) == 0 {
		fmt.Println("No groups found")
		return nil
	}

	// Sort groups for consistent output
	sort.Strings(groups)
	fmt.Printf("Available package groups (%d total):\n\n", len(groups))
	for _, group := range groups {
		packages := client.SearchPackagesByGroup(group)
		fmt.Printf("%s (%d packages)\n", group, len(packages))
	}

	return nil
}

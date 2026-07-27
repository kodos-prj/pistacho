// Package cli implements command-line interface commands for pith.
package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kodos-prj/pistacho/pkg/config"
	"github.com/kodos-prj/pistacho/pkg/registry"
)

// ListCommand implements the 'pith list' command.
// It lists all installed packages from the registry.
type ListCommand struct {
	config *config.Config
}

// NewListCommand creates a new list command instance.
func NewListCommand(cfg *config.Config) *ListCommand {
	return &ListCommand{
		config: cfg,
	}
}

// Execute runs the list command.
func (l *ListCommand) Execute(verbose bool) error {
	// Load registry
	reg, err := registry.NewRegistry(l.config.RegistryPath)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Get all packages
	packages := reg.ListPackages()

	if len(packages) == 0 {
		if verbose {
			// Display configuration header even when no packages
			fmt.Println("Configuration:")
			fmt.Printf("  Base Directory: %s\n", l.config.BaseDir)
			fmt.Printf("  Registry Path:  %s\n", l.config.RegistryPath)
			fmt.Printf("  Store Root:     %s\n", l.config.PoolRoot)
			fmt.Println()
		}
		fmt.Println("No packages installed.")
		return nil
	}

	// Sort packages by name
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Name < packages[j].Name
	})

	// Display packages
	if verbose {
		l.displayVerbose(packages)
	} else {
		l.displayCompact(packages)
	}

	return nil
}

// displayCompact shows a compact table view of installed packages
func (l *ListCommand) displayCompact(packages []*registry.Package) {
	fmt.Printf("Installed packages (%d total):\n\n", len(packages))

	// Calculate column widths
	maxNameLen := 0
	maxVersionLen := 0
	for _, pkg := range packages {
		if len(pkg.Name) > maxNameLen {
			maxNameLen = len(pkg.Name)
		}
		if len(pkg.Version) > maxVersionLen {
			maxVersionLen = len(pkg.Version)
		}
	}

	// Ensure minimum widths
	if maxNameLen < 20 {
		maxNameLen = 20
	}
	if maxVersionLen < 10 {
		maxVersionLen = 10
	}

	// Print header
	fmt.Printf("%-*s  %-*s  %-15s  %s\n",
		maxNameLen, "NAME",
		maxVersionLen, "VERSION",
		"INSTALL DATE", "FILES")
	fmt.Println(strings.Repeat("-", maxNameLen+maxVersionLen+40))

	// Print packages
	for _, pkg := range packages {
		installDate := pkg.InstallDate
		if len(installDate) > 10 {
			installDate = installDate[:10] // Show only date part
		}

		fileCount := len(pkg.Files)
		fmt.Printf("%-*s  %-*s  %-15s  %d\n",
			maxNameLen, pkg.Name,
			maxVersionLen, pkg.Version,
			installDate, fileCount)
	}
}

// displayVerbose shows detailed information for each package
func (l *ListCommand) displayVerbose(packages []*registry.Package) {
	// Display configuration header
	fmt.Println("Configuration:")
	fmt.Printf("  Base Directory: %s\n", l.config.BaseDir)
	fmt.Printf("  Registry Path:  %s\n", l.config.RegistryPath)
	fmt.Printf("  Store Root:     %s\n", l.config.PoolRoot)
	fmt.Println()

	fmt.Printf("Installed packages (%d total):\n\n", len(packages))

	for i, pkg := range packages {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("Package: %s\n", pkg.Name)
		fmt.Printf("  Version:      %s\n", pkg.Version)
		fmt.Printf("  Install Date: %s\n", pkg.InstallDate)
		fmt.Printf("  Files:        %d\n", len(pkg.Files))
		fmt.Printf("  Executables:  %d\n", len(pkg.Executables))

		if len(pkg.Dependencies) > 0 {
			fmt.Printf("  Dependencies: %s\n", strings.Join(pkg.Dependencies, ", "))
		} else {
			fmt.Printf("  Dependencies: none\n")
		}

		if len(pkg.Executables) > 0 {
			fmt.Printf("  Executables:\n")
			for _, exe := range pkg.Executables {
				fmt.Printf("    - %s\n", exe)
			}
		}
	}
}

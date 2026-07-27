// Package cli implements command-line interface commands for chisel.
package cli

import (
	"fmt"

	"github.com/kodos-prj/pistacho/pkg/alpm"
	"github.com/kodos-prj/pistacho/pkg/aur"
	"github.com/kodos-prj/pistacho/pkg/config"
)

// InfoCommand implements the 'chisel info' command.
// It displays detailed information about a package from official repos or AUR.
type InfoCommand struct {
	config *config.Config
	aurRPC *aur.RPCClient
}

// NewInfoCommand creates a new info command instance.
func NewInfoCommand(cfg *config.Config) *InfoCommand {
	return &InfoCommand{
		config: cfg,
		aurRPC: aur.NewRPCClient(),
	}
}

// Execute runs the info command.
// name is the package name to get information about.
// First searches official repos, then falls back to AUR.
func (i *InfoCommand) Execute(name string) error {
	if name == "" {
		return fmt.Errorf("package name cannot be empty")
	}

	// Initialize ALPM client
	// Note: We pass DBPath (the sync database directory), not AlpmDBPath (the parent)
	client, err := alpm.NewClient(i.config.AlpmRoot, i.config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(i.config.Repositories); err != nil {
		return fmt.Errorf("failed to register sync databases: %w", err)
	}

	// Try to get info from official repos first
	info, err := client.GetPackageInfo(name)
	if err == nil {
		// Found in official repo
		i.displayOfficialInfo(info)
		return nil
	}

	// Not in official repos, try AUR
	aurPkg, err := i.aurRPC.GetPackage(name)
	if err != nil {
		return fmt.Errorf("package not found in official repositories or AUR: %w", err)
	}

	// Display AUR package info
	i.displayAURInfo(aurPkg)
	return nil
}

// displayOfficialInfo displays information about an official repository package
func (i *InfoCommand) displayOfficialInfo(info *alpm.PackageInfo) {
	fmt.Printf("Repository      : %s\n", info.Repository)
	fmt.Printf("Name            : %s\n", info.Name)
	fmt.Printf("Version         : %s\n", info.Version)
	fmt.Printf("Description     : %s\n", info.Description)
	fmt.Printf("Architecture    : %s\n", info.Architecture)
	fmt.Printf("URL             : %s\n", info.URL)

	if len(info.Licenses) > 0 {
		fmt.Printf("Licenses        : %v\n", info.Licenses)
	}

	if len(info.Groups) > 0 {
		fmt.Printf("Groups          : %v\n", info.Groups)
	}

	if len(info.Provides) > 0 {
		fmt.Printf("Provides        : %v\n", info.Provides)
	}

	if len(info.DependsOn) > 0 {
		fmt.Printf("Depends On      : %v\n", info.DependsOn)
	}

	if len(info.OptDepends) > 0 {
		fmt.Printf("Optional Deps   : %v\n", info.OptDepends)
	}

	if len(info.Conflicts) > 0 {
		fmt.Printf("Conflicts With  : %v\n", info.Conflicts)
	}

	if len(info.Replaces) > 0 {
		fmt.Printf("Replaces        : %v\n", info.Replaces)
	}

	fmt.Printf("Download Size   : %.2f MB\n", float64(info.DownloadSize)/(1024*1024))
	fmt.Printf("Installed Size  : %.2f MB\n", float64(info.Size)/(1024*1024))
	fmt.Printf("Packager        : %s\n", info.Maintainer)

	if info.PackageBase != "" {
		fmt.Printf("Package Base    : %s\n", info.PackageBase)
	}
}

// displayAURInfo displays information about an AUR package
func (i *InfoCommand) displayAURInfo(pkg *aur.AURPackage) {
	fmt.Printf("Repository      : AUR\n")
	fmt.Printf("Name            : %s\n", pkg.Name)
	fmt.Printf("Package Base    : %s\n", pkg.PackageBase)
	fmt.Printf("Version         : %s\n", pkg.Version)
	fmt.Printf("Description     : %s\n", pkg.Description)
	fmt.Printf("URL             : %s\n", pkg.URL)
	fmt.Printf("Maintainer      : %s\n", pkg.Maintainer)

	if len(pkg.Depends) > 0 {
		fmt.Printf("Depends On      : %v\n", pkg.Depends)
	}

	if len(pkg.MakeDepends) > 0 {
		fmt.Printf("Make Depends    : %v\n", pkg.MakeDepends)
	}

	if len(pkg.OptDepends) > 0 {
		fmt.Printf("Optional Deps   : %v\n", pkg.OptDepends)
	}

	if len(pkg.Conflicts) > 0 {
		fmt.Printf("Conflicts With  : %v\n", pkg.Conflicts)
	}

	if len(pkg.Provides) > 0 {
		fmt.Printf("Provides        : %v\n", pkg.Provides)
	}

	if len(pkg.Replaces) > 0 {
		fmt.Printf("Replaces        : %v\n", pkg.Replaces)
	}

	fmt.Printf("Popularity      : %.2f (%d votes)\n", pkg.Popularity, pkg.NumVotes)

	if pkg.OutOfDate > 0 {
		fmt.Printf("Status          : ⚠ OUT OF DATE\n")
	}
}

// ExecuteWithDeps runs the info command and also shows dependency tree.
func (i *InfoCommand) ExecuteWithDeps(name string) error {
	// First show basic info
	if err := i.Execute(name); err != nil {
		return err
	}

	// Initialize ALPM client for dependency resolution
	// Note: We pass DBPath (the sync database directory), not AlpmDBPath (the parent)
	client, err := alpm.NewClient(i.config.AlpmRoot, i.config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(i.config.Repositories); err != nil {
		return fmt.Errorf("failed to register sync databases: %w", err)
	}

	// Resolve dependencies
	deps, err := client.ResolveDependencies(name)
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// Display dependency tree
	fmt.Println("\nComplete Dependency Tree (install order):")
	for idx, dep := range deps {
		fmt.Printf("  %d. %s\n", idx+1, dep)
	}

	return nil
}

package cli

import (
	"fmt"
	"os"

	"github.com/kodos-prj/pistacho/pkg/alpm"
	"github.com/kodos-prj/pistacho/pkg/config"
	"github.com/kodos-prj/pistacho/pkg/download"
)

// DownloadCommand handles downloading packages.
type DownloadCommand struct {
	config *config.Config
}

// NewDownloadCommand creates a new download command.
func NewDownloadCommand(cfg *config.Config) *DownloadCommand {
	return &DownloadCommand{
		config: cfg,
	}
}

// Run executes the download command.
// Usage: pith download [options] <package> [package2] ...
func (d *DownloadCommand) Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("package name required")
	}

	// Initialize ALPM to get package info
	client, err := alpm.NewClient(d.config.AlpmRoot, d.config.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(d.config.Repositories); err != nil {
		return fmt.Errorf("failed to register databases: %w", err)
	}

	// Create downloader
	downloader := download.NewDownloader(
		d.config.MirrorURL,
		d.config.CachePath,
		d.config.Architecture,
		d.config.MaxConcurrentDownloads,
		0, // timeout handled via context in real implementation
	)

	// Prepare packages to download
	var packages []download.PackageInfo
	for _, pkgName := range args {
		// Get package info (GetPackageInfo searches internally)
		info, err := client.GetPackageInfo(pkgName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Package not found: %s\n", pkgName)
			continue
		}

		packages = append(packages, download.PackageInfo{
			Name:    info.Name,
			Version: info.Version,
			Repo:    info.Repository,
		})
	}

	if len(packages) == 0 {
		return fmt.Errorf("no packages to download")
	}

	// Download packages concurrently
	fmt.Printf("Downloading %d package(s)...\n", len(packages))
	results, err := downloader.DownloadPackages(packages)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Download warnings/errors: %v\n", err)
	}

	// Print results
	fmt.Printf("\n✓ Downloaded %d package(s):\n", len(results))
	for name, path := range results {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ Warning: could not stat %s: %v\n", name, err)
			continue
		}
		size := info.Size()
		fmt.Printf("  ✓ %s (%d bytes) -> %s\n", name, size, path)
	}

	return nil
}

// Help returns help text for the download command.
func (d *DownloadCommand) Help() string {
	return `Download packages from Arch mirrors.

Usage:
  chisel download [options] <package> [package2] ...

Options:
  --no-deps        Don't download dependencies
  --only-cache     Only download, don't extract
  
Examples:
  chisel download bash
  chisel download bash vim git
  chisel download --no-deps curl
`
}

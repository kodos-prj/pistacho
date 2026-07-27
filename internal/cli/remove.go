package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kodos-prj/pistacho/pkg/config"
	"github.com/kodos-prj/pistacho/pkg/registry"
	"github.com/kodos-prj/pistacho/pkg/wrapper"
)

// RemoveCommand handles removing packages and cleaning up symlinks.
type RemoveCommand struct {
	config     *config.Config
	symlinkDir string
}

// NewRemoveCommand creates a new remove command.
func NewRemoveCommand(cfg *config.Config) *RemoveCommand {
	return &RemoveCommand{
		config:     cfg,
		symlinkDir: "",
	}
}

// NewRemoveCommandWithSymlinkDir creates a new remove command with a symlink directory.
func NewRemoveCommandWithSymlinkDir(cfg *config.Config, symlinkDir string) *RemoveCommand {
	return &RemoveCommand{
		config:     cfg,
		symlinkDir: symlinkDir,
	}
}

// RemoveOptions holds command-line options for remove.
type RemoveOptions struct {
	Force  bool // Force removal even if symlinks don't exist
	NoDeps bool // Skip dependency checking
}

// Run executes the remove command.
// Usage: pith remove [options] <package> [package2] ...
func (r *RemoveCommand) Run(args []string) error {
	opts := &RemoveOptions{}

	// Parse command-line options
	packageNames := []string{}
	for _, arg := range args {
		switch arg {
		case "--force":
			opts.Force = true
		case "--no-deps":
			opts.NoDeps = true
		case "--help", "-h":
			r.showHelp()
			return nil
		default:
			packageNames = append(packageNames, arg)
		}
	}

	if len(packageNames) == 0 {
		fmt.Fprintln(os.Stderr, "Error: package name required")
		fmt.Fprintln(os.Stderr, "Usage: pith remove [options] <package> [package2] ...")
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  --force     Force removal even if symlinks don't exist")
		fmt.Fprintln(os.Stderr, "  --no-deps   Skip checking for dependent packages")
		return fmt.Errorf("no packages specified")
	}

	// Load registry
	reg, err := registry.NewRegistry(r.config.RegistryPath)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Verify all packages exist before removing anything
	toRemove := []*registry.Package{}
	for _, pkgName := range packageNames {
		pkg, ok := reg.GetPackage(pkgName)
		if !ok {
			fmt.Fprintf(os.Stderr, "Warning: package %s not found in registry\n", pkgName)
			continue
		}
		toRemove = append(toRemove, pkg)
	}

	if len(toRemove) == 0 {
		return fmt.Errorf("no packages found to remove")
	}

	// Check for packages that depend on the ones being removed
	if !opts.NoDeps {
		fmt.Println("\nChecking for dependent packages...")
		allPackages := reg.ListPackages()
		dependentMap := make(map[string][]string) // pkgName -> list of packages that depend on it

		// Build the dependency map
		for _, pkg := range allPackages {
			for _, dep := range pkg.Dependencies {
				dependentMap[dep] = append(dependentMap[dep], pkg.Name)
			}
		}

		// Check if any packages being removed have dependents that are NOT being removed
		for _, pkg := range toRemove {
			if dependents, exists := dependentMap[pkg.Name]; exists {
				var stillInstalled []string
				toRemoveMap := make(map[string]bool)
				for _, p := range toRemove {
					toRemoveMap[p.Name] = true
				}

				for _, depPkg := range dependents {
					if !toRemoveMap[depPkg] {
						stillInstalled = append(stillInstalled, depPkg)
					}
				}

				if len(stillInstalled) > 0 {
					fmt.Printf("  ⚠ Warning: %s is a dependency of: %v\n", pkg.Name, stillInstalled)
					if !opts.Force {
						fmt.Fprintf(os.Stderr, "  ! To remove %s anyway, use --force\n", pkg.Name)
						return fmt.Errorf("packages still depend on %s", pkg.Name)
					}
				}
			}
		}
	}

	// Determine symlink directory
	symlinkDir := r.symlinkDir
	if symlinkDir == "" {
		symlinkDir = r.config.SymlinkRoot
	}

	// Remove symlinks if symlink directory is set and not system root
	if symlinkDir != "" && symlinkDir != "." && symlinkDir != "/" {
		fmt.Println("\nRemoving symlinks...")
		for _, pkg := range toRemove {
			if err := r.removeSymlinks(pkg, opts, symlinkDir); err != nil {
				fmt.Fprintf(os.Stderr, "  ! Warning: Failed to remove symlinks for %s: %v\n", pkg.Name, err)
				if !opts.Force {
					return err
				}
			}
		}
		fmt.Printf("✓ Removed symlinks\n")
	}

	// Remove wrapper scripts
	fmt.Println("\nRemoving wrapper scripts...")
	wrapperGen := wrapper.NewGenerator(r.config.PoolRoot, r.config.WrapperDir, r.config.SymlinkRoot)
	for _, pkg := range toRemove {
		for _, exec := range pkg.Executables {
			if err := wrapperGen.RemoveWrapper(exec); err != nil {
				fmt.Fprintf(os.Stderr, "  ! Warning: Failed to remove wrapper for %s: %v\n", exec, err)
				if !opts.Force {
					return err
				}
			}
		}
	}
	fmt.Printf("✓ Removed wrapper scripts\n")

	// Update registry
	fmt.Println("\nUpdating registry...")
	for _, pkg := range toRemove {
		if err := reg.RemovePackage(pkg.Name); err != nil {
			return fmt.Errorf("failed to remove %s from registry: %w", pkg.Name, err)
		}
	}

	if err := reg.Save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	fmt.Printf("✓ Removal complete! (Removed %d package(s))\n", len(toRemove))
	return nil
}

// removeSymlinks removes all symlinks for a package from the symlink directory.
func (r *RemoveCommand) removeSymlinks(pkg *registry.Package, opts *RemoveOptions, symlinkDir string) error {
	for _, filePath := range pkg.Files {
		symlinkPath := filepath.Join(symlinkDir, filePath)

		// Check if symlink exists
		stat, err := os.Lstat(symlinkPath)
		if err != nil {
			if os.IsNotExist(err) {
				// Symlink doesn't exist, skip
				if !opts.Force {
					fmt.Fprintf(os.Stderr, "  ! Warning: Symlink does not exist at %s\n", symlinkPath)
				}
				continue
			}
			return fmt.Errorf("failed to stat %s: %w", symlinkPath, err)
		}

		// Only remove if it's a symlink (not a regular file)
		if stat.Mode()&os.ModeSymlink == 0 {
			if !opts.Force {
				return fmt.Errorf("not a symlink: %s", symlinkPath)
			}
			fmt.Fprintf(os.Stderr, "  ! Warning: Not a symlink: %s (skipping)\n", symlinkPath)
			continue
		}

		// Remove the symlink
		if err := os.Remove(symlinkPath); err != nil {
			return fmt.Errorf("failed to remove symlink %s: %w", symlinkPath, err)
		}
	}

	// Clean up empty directories
	r.removeEmptyDirectories(symlinkDir)

	return nil
}

// removeEmptyDirectories removes empty directories in the symlink directory tree.
func (r *RemoveCommand) removeEmptyDirectories(symlinkDir string) {
	// Walk the directory tree from deepest to shallowest, removing empty dirs
	filepath.Walk(symlinkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == symlinkDir {
			return nil
		}

		// Try to remove the directory (only succeeds if empty)
		_ = os.Remove(path)
		return nil
	})
}

// showHelp displays help for the remove command.
func (r *RemoveCommand) showHelp() {
	fmt.Println("Remove packages and clean up symlinks")
	fmt.Println("")
	fmt.Println("Usage: pith remove [options] <package> [package2] ...")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --force     Force removal even if symlinks are missing or in unexpected state")
	fmt.Println("  --no-deps   Skip checking for dependent packages")
	fmt.Println("  -h, --help  Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # Remove a single package")
	fmt.Println("  pistacho remove vim")
	fmt.Println("")
	fmt.Println("  # Remove multiple packages")
	fmt.Println("  pistacho remove btop curl git")
	fmt.Println("")
	fmt.Println("  # Force removal even if symlinks are missing")
	fmt.Println("  pistacho remove --force btop")
	fmt.Println("")
	fmt.Println("  # Skip dependency checking")
	fmt.Println("  pistacho remove --no-deps vim")
}

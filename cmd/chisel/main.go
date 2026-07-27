package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/kodos-prj/chisel/internal/cli"
	"github.com/kodos-prj/chisel/pkg/config"
)

const version = "0.3.2"

var (
	configPath      string
	baseDir         string
	baseDirExplicit bool // Track if --base-dir was explicitly provided
	mirrorURL       string
	symlinkDir      string
)

func main() {
	// Define global flags
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.StringVar(&configPath, "c", "", "Path to configuration file (shorthand)")
	flag.StringVar(&baseDir, "base-dir", "", "Base directory for chisel data (overrides config)")
	flag.StringVar(&mirrorURL, "mirror", "", "Arch mirror URL (overrides config)")
	flag.StringVar(&symlinkDir, "symlink-dir", "", "Directory to create symlink hierarchy (optional)")

	// Parse flags before checking commands
	flag.Parse()

	// Track if --base-dir was explicitly provided (before loading config)
	baseDirExplicit = (baseDir != "")

	args := flag.Args()
	if len(args) < 1 {
		showUsage()
		os.Exit(1)
	}

	command := args[0]

	switch command {
	case "version":
		fmt.Printf("chisel version %s\n", version)
		return

	case "sync":
		handleSync(args[1:])

	case "search":
		handleSearch(args[1:])

	case "info":
		handleInfo(args[1:])

	case "download":
		handleDownload(args[1:])

	case "extract":
		handleExtract(args[1:])

	case "install":
		handleInstall(args[1:])

	case "install-scripts":
		handleInstallScripts(args[1:])

	case "remove":
		handleRemove(args[1:])

	case "list", "query":
		handleList(args[1:])

	case "upgrade":
		handleUpgrade(args[1:])

	case "cleanup":
		handleCleanup(args[1:])

	case "cache":
		handleCache(args[1:])

	case "help", "--help", "-h":
		showUsage()

	default:
		fmt.Printf("Unknown command: %s\n", command)
		showUsage()
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Println("chisel - Cross-Distribution Package Manager")
	fmt.Println("Brings Arch Linux packages to any Linux distribution")
	fmt.Println("")
	fmt.Println("Usage: chisel [global-options] <command> [options]")
	fmt.Println("")
	fmt.Println("Global Options:")
	fmt.Println("  -c, --config <path>   Path to configuration file (default: /etc/chisel/config.json)")
	fmt.Println("  --base-dir <path>     Base directory for chisel data (default: /kod)")
	fmt.Println("  --mirror <url>        Arch mirror URL (overrides config)")
	fmt.Println("")
	fmt.Println("Available Commands (Phase 1):")
	fmt.Println("  sync              Sync package databases from Arch mirrors")
	fmt.Println("  sync --status     Show database sync status")
	fmt.Println("  search <pattern>  Search for packages")
	fmt.Println("  search --group    Search packages in a group")
	fmt.Println("  search --groups   List all available package groups")
	fmt.Println("  info <package>    Show detailed package information")
	fmt.Println("  info --deps <pkg> Show package info with dependency tree")
	fmt.Println("")
	fmt.Println("Available Commands (Phase 2):")
	fmt.Println("  download <pkg>    Download packages from Arch mirrors")
	fmt.Println("  extract <file>    Extract packages to the store")
	fmt.Println("")
	fmt.Println("Available Commands (Phase 3):")
	fmt.Println("  install <pkg>     Install a package (with dependencies, symlinks, wrappers)")
	fmt.Println("  install-scripts   Execute post_install/post_upgrade scripts (chroot context)")
	fmt.Println("  remove <pkg>      Remove a package")
	fmt.Println("  list              List installed packages")
	fmt.Println("  list --verbose    List packages with detailed information")
	fmt.Println("")
	fmt.Println("Available Commands (Phase 4):")
	fmt.Println("  upgrade           Upgrade all packages")
	fmt.Println("  upgrade <pkg>     Upgrade specific packages")
	fmt.Println("  upgrade --dry-run Preview upgrades without making changes")
	fmt.Println("")
	fmt.Println("Available Commands (Phase 5):")
	fmt.Println("  cleanup           Remove old package versions")
	fmt.Println("  cleanup --dry-run Preview cleanup without making changes")
	fmt.Println("  cleanup --force   Remove versions without confirmation")
	fmt.Println("  cache              Clean downloaded package cache")
	fmt.Println("  cache --list       Show cache contents without removing")
	fmt.Println("  cache --dry-run    Preview cache clean without making changes")
	fmt.Println("")
	fmt.Println("Other:")
	fmt.Println("  version           Show version information")
	fmt.Println("  help              Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # Use custom config file")
	fmt.Println("  chisel -c /home/user/.chisel.json sync")
	fmt.Println("")
	fmt.Println("  # Use custom base directory for testing")
	fmt.Println("  chisel --base-dir /tmp/chisel-test sync")
	fmt.Println("")
	fmt.Println("  # Use different mirror")
	fmt.Println("  chisel --mirror https://mirrors.kernel.org/archlinux sync")
	fmt.Println("")
	fmt.Println("  # Combine options")
	fmt.Println("  chisel -c myconfig.json --base-dir /tmp/test search vim")
	fmt.Println("")
	fmt.Println("Configuration: /etc/chisel/config.json (or specify with --config)")
	fmt.Println("Environment variables: CHISEL_CONFIG, CHISEL_BASE_DIR")
	fmt.Println("See documentation at: https://github.com/kodos-prj/chisel")
}

func loadConfig() *config.Config {
	// Priority order:
	// 1. Command line flags (--config, --base-dir, --mirror, --generation)
	// 2. Environment variables (CHISEL_CONFIG, CHISEL_BASE_DIR, CHISEL_GENERATION)
	// 3. Default config file (/etc/chisel/config.json)
	// 4. Built-in defaults

	// Determine config path
	cfgPath := configPath
	if cfgPath == "" {
		// Check environment variable
		if envConfig := os.Getenv("CHISEL_CONFIG"); envConfig != "" {
			cfgPath = envConfig
		} else {
			cfgPath = config.DefaultConfigPath
		}
	}

	// Check if config file exists
	var cfg *config.Config
	if _, err := os.Stat(cfgPath); err == nil {
		// Config file exists, load it
		cfg, err = config.Load(cfgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load config from %s: %v\n", cfgPath, err)
			fmt.Fprintf(os.Stderr, "Using user-level defaults\n")
			userCfg, _ := config.DefaultUserConfig()
			cfg = userCfg
		}
	} else {
		// Config file doesn't exist, use user-level defaults
		userCfg, err := config.DefaultUserConfig()
		if err != nil {
			// Fall back to system defaults if user config fails
			fmt.Fprintf(os.Stderr, "Warning: Failed to get user config: %v\n", err)
			fmt.Fprintf(os.Stderr, "Using built-in defaults\n")
			cfg = config.DefaultConfig()
		} else {
			cfg = userCfg
		}
	}
	cfg.Normalize()

	// Apply command-line overrides
	if baseDir != "" {
		cfg.BaseDir = baseDir
		// Update all derived paths to use the new base directory
		cfg.UpdateDerivedPaths()
	} else if envBaseDir := os.Getenv("CHISEL_BASE_DIR"); envBaseDir != "" {
		cfg.BaseDir = envBaseDir
		cfg.UpdateDerivedPaths()
	}

	if mirrorURL != "" {
		cfg.MirrorURL = mirrorURL
	}

	// Handle symlink directory from environment variable if not set via flag
	if symlinkDir == "" {
		if envSymlink := os.Getenv("CHISEL_SYMLINK_DIR"); envSymlink != "" {
			symlinkDir = envSymlink
		}
	}

	return cfg
}

func handleSync(args []string) {
	cfg := loadConfig()

	// Check for --status flag
	if len(args) > 0 && args[0] == "--status" {
		cmd := cli.NewSyncCommand(cfg)
		if err := cmd.ShowStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Check for --force flag
	force := false
	if len(args) > 0 && args[0] == "--force" {
		force = true
	}

	cmd := cli.NewSyncCommand(cfg)
	var err error
	if force {
		err = cmd.ExecuteWithForce()
	} else {
		err = cmd.Execute()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleSearch(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: search pattern or --group <name> required")
		fmt.Fprintln(os.Stderr, "Usage: chisel search <pattern>")
		fmt.Fprintln(os.Stderr, "       chisel search --group <group-name>")
		fmt.Fprintln(os.Stderr, "       chisel search --groups (list all groups)")
		os.Exit(1)
	}

	cfg := loadConfig()
	cmd := cli.NewSearchCommand(cfg)

	// Check for group-related flags
	if args[0] == "--group" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: group name required after --group")
			os.Exit(1)
		}
		groupName := args[1]
		if err := cmd.SearchGroup(groupName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if args[0] == "--groups" {
		if err := cmd.ListGroups(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Regular pattern search
	pattern := args[0]
	if err := cmd.Execute(pattern); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleInfo(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: package name required")
		fmt.Fprintln(os.Stderr, "Usage: chisel info <package>")
		fmt.Fprintln(os.Stderr, "       chisel info --deps <package>")
		os.Exit(1)
	}

	cfg := loadConfig()

	// Check for --deps flag
	if args[0] == "--deps" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: package name required after --deps")
			os.Exit(1)
		}
		packageName := args[1]
		cmd := cli.NewInfoCommand(cfg)
		if err := cmd.ExecuteWithDeps(packageName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	packageName := args[0]
	cmd := cli.NewInfoCommand(cfg)
	if err := cmd.Execute(packageName); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleDownload(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: package name required")
		fmt.Fprintln(os.Stderr, "Usage: chisel download <package> [package2] ...")
		os.Exit(1)
	}

	cfg := loadConfig()
	cmd := cli.NewDownloadCommand(cfg)
	if err := cmd.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleExtract(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: package file path required")
		fmt.Fprintln(os.Stderr, "Usage: chisel extract <package.pkg.tar.zst> [package2.pkg.tar.zst] ...")
		os.Exit(1)
	}

	cfg := loadConfig()
	cmd := cli.NewExtractCommand(cfg)
	if err := cmd.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleInstall(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: package name required")
		fmt.Fprintln(os.Stderr, "Usage: chisel install [options] <package> [package2] ...")
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  --source=aur       Install from AUR only (skip official repositories)")
		fmt.Fprintln(os.Stderr, "  --source=official  Install from official repositories only (skip AUR)")
		fmt.Fprintln(os.Stderr, "  --no-deps          Skip dependency resolution")
		fmt.Fprintln(os.Stderr, "  --no-extract       Skip extraction (assume already in store)")
		fmt.Fprintln(os.Stderr, "  --no-symlink       Skip symlink creation")
		fmt.Fprintln(os.Stderr, "  --force            Force overwrite of existing symlinks")
		fmt.Fprintln(os.Stderr, "  --chroot           Chroot directory path for self-contained installation")
		os.Exit(1)
	}

	cfg := loadConfig()
	cmd := cli.NewInstallCommandWithSymlinkDirAndExplicitBaseDir(cfg, symlinkDir, baseDirExplicit)
	if err := cmd.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleInstallScripts(args []string) {
	// Show help if --help flag is provided
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			fmt.Fprintln(os.Stderr, "Usage: chisel install-scripts [options] [package ...]")
			fmt.Fprintln(os.Stderr, "Execute post_install/post_upgrade scripts for packages")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Options:")
			fmt.Fprintln(os.Stderr, "  --chroot <dir>  Chroot base directory (optional - if omitted, executes in current context)")
			fmt.Fprintln(os.Stderr, "  --verbose       Show detailed script execution information")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Examples:")
			fmt.Fprintln(os.Stderr, "  # Execute scripts directly in current system context")
			fmt.Fprintln(os.Stderr, "  chisel install-scripts bash")
			fmt.Fprintln(os.Stderr, "  chisel install-scripts bash glibc")
			fmt.Fprintln(os.Stderr, "  chisel install-scripts  # Run all packages with scripts")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "  # Execute scripts in chroot context")
			fmt.Fprintln(os.Stderr, "  chisel install-scripts --chroot /tmp/chroot bash")
			fmt.Fprintln(os.Stderr, "  chisel install-scripts --chroot /tmp/chroot bash glibc")
			os.Exit(0)
		}
	}

	cfg := loadConfig()
	cmd := cli.NewInstallScriptsCommand(cfg)

	// Parse install-scripts specific flags
	verbose := false
	chrootDir := ""
	var packages []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--verbose", "-v":
			verbose = true
		case "--chroot":
			// Get next argument as chroot directory
			if i+1 < len(args) {
				chrootDir = args[i+1]
				i++ // Skip next argument
			}
		default:
			// Treat as package name (skip if it looks like a flag)
			if !startswith(arg, "--") {
				packages = append(packages, arg)
			}
		}
	}

	if err := cmd.Execute(packages, verbose, chrootDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func startswith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func handleRemove(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: package name required")
		fmt.Fprintln(os.Stderr, "Usage: chisel remove [options] <package> [package2] ...")
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  --force  Force removal even if symlinks are missing or cannot be accessed")
		os.Exit(1)
	}

	cfg := loadConfig()
	cmd := cli.NewRemoveCommandWithSymlinkDir(cfg, symlinkDir)
	if err := cmd.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleList(args []string) {
	// Show help if --help flag is provided
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			fmt.Fprintln(os.Stderr, "Usage: chisel list [options]")
			fmt.Fprintln(os.Stderr, "Options:")
			fmt.Fprintln(os.Stderr, "  --verbose  Show detailed package information including configuration")
			os.Exit(0)
		}
	}

	// Check for --verbose flag
	verbose := false
	for _, arg := range args {
		if arg == "--verbose" || arg == "-v" {
			verbose = true
			break
		}
	}

	cfg := loadConfig()
	cmd := cli.NewListCommand(cfg)
	if err := cmd.Execute(verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleUpgrade(args []string) {
	// Show help if --help flag is provided
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			fmt.Fprintln(os.Stderr, "Usage: chisel upgrade [options] [package ...]")
			fmt.Fprintln(os.Stderr, "Options:")
			fmt.Fprintln(os.Stderr, "  --dry-run  Preview upgrades without making changes")
			fmt.Fprintln(os.Stderr, "  --verbose  Show detailed upgrade information")
			os.Exit(0)
		}
	}

	cfg := loadConfig()

	// Parse upgrade-specific flags
	dryRun := false
	verbose := false
	var packages []string

	for _, arg := range args {
		switch arg {
		case "--dry-run":
			dryRun = true
		case "--verbose", "-v":
			verbose = true
		default:
			// Treat as package name
			packages = append(packages, arg)
		}
	}

	// Create upgrade command and execute
	cmd := cli.NewUpgradeCommandWithSymlinkDir(cfg, symlinkDir)
	summary, err := cmd.Execute(&cli.UpgradeOptions{
		DryRun:   dryRun,
		Verbose:  verbose,
		Packages: packages,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// If summary indicates failures, exit with error code
	if summary != nil && summary.Failed > 0 {
		os.Exit(1)
	}
}

func handleCleanup(args []string) {
	// Show help if --help flag is provided
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			fmt.Fprintln(os.Stderr, "Usage: chisel cleanup [options]")
			fmt.Fprintln(os.Stderr, "Options:")
			fmt.Fprintln(os.Stderr, "  --keep <N>   Keep N recent versions (default: 3)")
			fmt.Fprintln(os.Stderr, "  --dry-run    Preview cleanup without making changes")
			fmt.Fprintln(os.Stderr, "  --verbose    Show detailed cleanup information")
			fmt.Fprintln(os.Stderr, "  --force      Remove versions without confirmation")
			os.Exit(0)
		}
	}

	cfg := loadConfig()

	// Parse cleanup-specific flags
	dryRun := false
	verbose := false
	force := false
	keepVersions := -1

	for i, arg := range args {
		switch arg {
		case "--dry-run":
			dryRun = true
		case "--verbose", "-v":
			verbose = true
		case "--force":
			force = true
		case "--keep":
			if i+1 < len(args) {
				// Parse next argument as keep count
				if v, err := strconv.Atoi(args[i+1]); err == nil {
					keepVersions = v
				}
			}
		}
	}

	// Create cleanup command and execute
	cmd := cli.NewCleanupCommandWithSymlinkDir(cfg, symlinkDir)
	summary, err := cmd.Execute(&cli.CleanupOptions{
		DryRun:       dryRun,
		Verbose:      verbose,
		Force:        force,
		KeepVersions: keepVersions,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// If summary indicates issues, exit with appropriate code
	if summary != nil && summary.VersionsSkipped > 0 && !dryRun {
		// Print detailed info when there are skipped versions
		if verbose {
			fmt.Printf("\nNote: %d versions were skipped (still in use)\n", summary.VersionsSkipped)
		}
	}
}

func handleCache(args []string) {
	// Show help if --help flag is provided
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			fmt.Fprintln(os.Stderr, "Usage: chisel cache [options]")
			fmt.Fprintln(os.Stderr, "Options:")
			fmt.Fprintln(os.Stderr, "  --list     List cache contents without removing")
			fmt.Fprintln(os.Stderr, "  --dry-run  Preview cache cleanup without making changes")
			fmt.Fprintln(os.Stderr, "  --verbose  Show detailed cache information")
			fmt.Fprintln(os.Stderr, "  --force    Force removal of all cache files")
			os.Exit(0)
		}
	}

	cfg := loadConfig()

	// Parse cache-specific flags
	verbose := false
	force := false
	dryRun := false
	action := "clean"

	for _, arg := range args {
		switch arg {
		case "--list":
			action = "list"
		case "--verbose", "-v":
			verbose = true
		case "--force":
			force = true
		case "--dry-run":
			dryRun = true
		case "--prune":
			action = "prune"
		}
	}

	// Create cache command and execute
	cmd := cli.NewCacheCommand(cfg)
	summary, err := cmd.Execute(&cli.CacheOptions{
		Action:  action,
		DryRun:  dryRun,
		Verbose: verbose,
		Force:   force,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// If summary indicates errors, exit with error code
	if summary != nil && summary.FilesWithError > 0 {
		os.Exit(1)
	}
}

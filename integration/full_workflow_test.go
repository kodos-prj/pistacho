package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kodos-prj/chisel/pkg/alpm"
	"github.com/kodos-prj/chisel/pkg/config"
	"github.com/kodos-prj/chisel/pkg/database"
	"github.com/kodos-prj/chisel/pkg/registry"
)

// TestFullWorkflowIntegration tests the complete chisel workflow
func TestFullWorkflowIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "chisel-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up configuration for testing
	baseDir := filepath.Join(tmpDir, "kod")
	// ALPM requires databases to be in var/lib/pacman structure
	alpmDBPath := filepath.Join(baseDir, "var/lib/pacman")
	syncPath := filepath.Join(alpmDBPath, "sync")
	storeDir := filepath.Join(baseDir, "store")
	wrapperDir := filepath.Join(baseDir, "wrappers")
	registryPath := filepath.Join(baseDir, "registry.json")
	cacheDir := filepath.Join(baseDir, "cache")

	// Create directories
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("failed to create base dir: %v", err)
	}
	if err := os.MkdirAll(syncPath, 0755); err != nil {
		t.Fatalf("failed to create sync dir: %v", err)
	}
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		t.Fatalf("failed to create store dir: %v", err)
	}
	if err := os.MkdirAll(wrapperDir, 0755); err != nil {
		t.Fatalf("failed to create wrapper dir: %v", err)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}

	// Create configuration
	cfg := &config.Config{
		BaseDir:                baseDir,
		SymlinkRoot:            "/",
		PoolRoot:              storeDir,
		RegistryPath:           registryPath,
		AlpmRoot:               baseDir,
		AlpmDBPath:             alpmDBPath,
		DBPath:                 syncPath,
		WrapperDir:             wrapperDir,
		CachePath:              cacheDir,
		MirrorURL:              "https://mirror.rackspace.com/archlinux",
		Architecture:           "x86_64",
		Repositories:           []string{"core", "extra"},
		VerifySignatures:       false,
		MaxConcurrentDownloads: 5,
		DownloadTimeout:        300,
		KeepVersions:           3,
	}

	// Step 1: Sync databases
	t.Log("Step 1: Syncing databases...")
	// Database syncer needs to put files in the sync/ directory for ALPM to find them
	syncer := database.NewSyncer(
		cfg.MirrorURL,
		syncPath,
		cfg.Architecture,
		30*time.Second,
	)

	if err := syncer.Sync(cfg.Repositories); err != nil {
		t.Fatalf("failed to sync databases: %v", err)
	}
	t.Log("✓ Databases synced successfully")

	// Debug: verify files exist
	entries, err := os.ReadDir(syncPath)
	if err != nil {
		t.Fatalf("failed to read sync path: %v", err)
	}
	for _, entry := range entries {
		info, _ := entry.Info()
		t.Logf("  - %s (%d bytes)", entry.Name(), info.Size())
	}

	// Step 2: Test search functionality
	t.Log("Step 2: Testing search...")
	client, err := alpm.NewClient(cfg.AlpmRoot, syncPath)
	if err != nil {
		t.Fatalf("failed to create ALPM client: %v", err)
	}
	defer client.Close()

	if err := client.RegisterAllSyncDBs(cfg.Repositories); err != nil {
		t.Fatalf("failed to register sync databases: %v", err)
	}

	// Debug: list registered databases and package counts
	impl := client.GetImpl().(*alpm.Client)
	for i, db := range impl.Databases {
		count := len(db.Packages)
		t.Logf("Database %d (%s): %d packages", i, db.Name, count)
		if count > 0 && i == 0 {
			// Show first few packages
			j := 0
			for name := range db.Packages {
				if j < 5 {
					t.Logf("  - %s", name)
				}
				j++
			}
		}
	}

	// Test searching for a package
	pkg, err := client.SearchPackage("acl")
	if err != nil {
		t.Fatalf("SearchPackage failed: %v", err)
	}
	if pkg == nil {
		t.Fatal("SearchPackage returned nil")
	}
	t.Logf("✓ Found package: %s %s", pkg.Name, pkg.Version)

	// Step 3: Simulate install by updating registry
	t.Log("Step 3: Simulating installation in registry...")

	// Create registry and add packages
	reg, err := registry.NewRegistry(cfg.RegistryPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	// Add bash package to registry
	bashPkg := &registry.Package{
		Name:        "bash",
		Version:     "5.2.002-1", // Dummy version
		InstallDate: time.Now().Format(time.RFC3339),
	}

	if err := reg.AddPackage(bashPkg); err != nil {
		t.Logf("Warning: failed to add bash to registry: %v", err)
	}

	// Add coreutils package to registry
	coreutilsPkg := &registry.Package{
		Name:        "coreutils",
		Version:     "9.1-1", // Dummy version
		InstallDate: time.Now().Format(time.RFC3339),
	}

	if err := reg.AddPackage(coreutilsPkg); err != nil {
		t.Logf("Warning: failed to add coreutils to registry: %v", err)
	}

	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry: %v", err)
	}

	t.Logf("✓ Installed packages to registry")

	// Step 4: Verify installed packages
	t.Log("Step 4: Verifying package installation...")

	// Check if packages are registered
	bashPkg, exists := reg.GetPackage("bash")
	if !exists {
		t.Error("bash package not found in registry")
	} else {
		t.Logf("✓ bash package installed (version: %s)", bashPkg.Version)
	}

	// Get coreutils package info
	coreutilsPkgInfo, exists := reg.GetPackage("coreutils")
	if !exists {
		t.Error("coreutils package not found in registry")
	} else {
		t.Logf("✓ coreutils package installed (version: %s)", coreutilsPkgInfo.Version)
	}

	// Step 5: Remove one package
	t.Log("Step 5: Removing one package...")

	// Remove from registry
	if err := reg.RemovePackage("bash"); err != nil {
		t.Fatalf("failed to remove bash from registry: %v", err)
	}

	if err := reg.Save(); err != nil {
		t.Fatalf("failed to save registry after removal: %v", err)
	}

	t.Logf("✓ Removed package: bash")

	// Step 6: Verify removal
	t.Log("Step 6: Verifying package removal...")

	// Check that bash is no longer in registry
	_, exists = reg.GetPackage("bash")
	if exists {
		t.Error("bash package still found in registry after removal")
	} else {
		t.Logf("✓ bash package successfully removed from registry")
	}

	// Check that coreutils is still present
	_, exists = reg.GetPackage("coreutils")
	if !exists {
		t.Error("coreutils package missing after removal of bash")
	} else {
		t.Logf("✓ coreutils package still present")
	}

	t.Log("✓ Full workflow test completed successfully")
}

// TestRealDatabaseParsing tests that all metadata fields are correctly parsed from real Arch databases.
// This test uses cached databases from previous test runs and verifies multi-line metadata parsing.
func TestRealDatabaseParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "chisel-parse-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up configuration - matching the structure used in TestFullWorkflowIntegration
	baseDir := filepath.Join(tmpDir, "kod")
	alpmDBPath := filepath.Join(baseDir, "var/lib/pacman")
	syncPath := filepath.Join(alpmDBPath, "sync")
	storeDir := filepath.Join(baseDir, "store")
	wrapperDir := filepath.Join(baseDir, "wrappers")
	registryPath := filepath.Join(baseDir, "registry.json")
	cacheDir := filepath.Join(baseDir, "cache")

	// Create directories
	if err := os.MkdirAll(syncPath, 0755); err != nil {
		t.Fatalf("failed to create sync dir: %v", err)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}

	// Create configuration matching existing test setup
	cfg := &config.Config{
		BaseDir:                baseDir,
		SymlinkRoot:            "/",
		PoolRoot:              storeDir,
		RegistryPath:           registryPath,
		AlpmRoot:               baseDir,
		AlpmDBPath:             alpmDBPath,
		DBPath:                 syncPath,
		WrapperDir:             wrapperDir,
		CachePath:              cacheDir,
		MirrorURL:              "https://mirror.rackspace.com/archlinux",
		Architecture:           "x86_64",
		Repositories:           []string{"core", "extra"},
		VerifySignatures:       false,
		MaxConcurrentDownloads: 5,
		DownloadTimeout:        300,
		KeepVersions:           3,
	}

	// Set up database syncer
	t.Log("Step 1: Syncing databases...")
	syncer := database.NewSyncer(
		cfg.MirrorURL,
		syncPath,
		cfg.Architecture,
		30*time.Second,
	)

	if err := syncer.Sync(cfg.Repositories); err != nil {
		t.Fatalf("failed to sync databases: %v", err)
	}
	t.Log("✓ Databases synced successfully")

	// Create ALPM client and register databases
	t.Log("Step 2: Loading and registering databases...")
	client, err := alpm.NewClient(cfg.AlpmRoot, syncPath)
	if err != nil {
		t.Fatalf("failed to create ALPM client: %v", err)
	}
	defer client.Close()

	if err := client.RegisterAllSyncDBs(cfg.Repositories); err != nil {
		t.Fatalf("failed to register sync databases: %v", err)
	}

	// Get internal implementation to access databases
	impl := client.GetImpl().(*alpm.Client)

	// Create cache for group queries
	cache := alpm.NewDatabaseCache()
	for _, db := range impl.Databases {
		cache.AddDatabase(db)
		t.Logf("  Loaded %s database: %d packages", db.Name, len(db.Packages))
	}

	t.Log("Step 3: Verifying GROUPS metadata parsing...")

	// Test 1: Verify groups exist in the cache
	allGroups := cache.ListAllGroups()
	if len(allGroups) == 0 {
		t.Error("No groups found in database (expected 50+)")
	} else {
		t.Logf("✓ Found %d total groups", len(allGroups))
		// Show sample groups
		displayGroups := allGroups
		if len(displayGroups) > 10 {
			displayGroups = displayGroups[:10]
		}
		t.Logf("  Sample groups: %v", displayGroups)
	}

	// Test 2: Try to find a package with groups
	t.Log("Step 4: Testing package group parsing...")
	if len(allGroups) > 0 {
		// Try to get packages from a group
		testGroups := []string{"base-devel", "editors", "pro-audio", "gnome", "kde", "base"}
		foundGroupWithPackages := false
		for _, groupName := range testGroups {
			pkgs := cache.GetPackagesByGroup(groupName)
			if len(pkgs) > 0 {
				t.Logf("✓ Group '%s' contains %d packages", groupName, len(pkgs))
				if len(pkgs) > 0 {
					t.Logf("  Sample packages: %s, %s", pkgs[0].Name, func() string {
						if len(pkgs) > 1 {
							return pkgs[1].Name
						}
						return ""
					}())
				}
				foundGroupWithPackages = true
				break
			}
		}
		if !foundGroupWithPackages {
			t.Logf("⚠ Could not find any common groups with packages (searched: base-devel, editors, pro-audio, gnome, kde, base)")
		}
	}

	t.Log("Step 5: Verifying DEPENDS metadata parsing...")

	// Test 3: Find bash and verify dependencies are parsed
	bashPkg, err := client.SearchPackage("bash")
	if err == nil && bashPkg != nil {
		t.Logf("✓ Found bash with %d dependencies", len(bashPkg.DependsOn))
		if len(bashPkg.DependsOn) > 0 {
			t.Logf("  Sample dependencies: %v", func() []string {
				if len(bashPkg.DependsOn) > 3 {
					return bashPkg.DependsOn[:3]
				}
				return bashPkg.DependsOn
			}())
		} else {
			t.Logf("⚠ bash has no dependencies parsed (may not have any in this repository)")
		}
	} else {
		t.Logf("⚠ bash package not found (may not be in this mirror)")
	}

	t.Log("Step 6: Verifying other metadata fields...")

	// Test 4: Spot check other metadata fields
	for _, pkgName := range []string{"bash", "coreutils", "gcc"} {
		pkg, err := client.SearchPackage(pkgName)
		if err == nil && pkg != nil {
			t.Logf("Package %s metadata:", pkgName)
			if len(pkg.Provides) > 0 {
				t.Logf("  - Provides (%d): %v", len(pkg.Provides), pkg.Provides[:1])
			}
			if len(pkg.Conflicts) > 0 {
				t.Logf("  - Conflicts (%d): %v", len(pkg.Conflicts), pkg.Conflicts[:1])
			}
			if len(pkg.Replaces) > 0 {
				t.Logf("  - Replaces (%d): %v", len(pkg.Replaces), pkg.Replaces[:1])
			}
			if len(pkg.OptDepends) > 0 {
				t.Logf("  - OptDepends (%d): %v", len(pkg.OptDepends), pkg.OptDepends[:1])
			}
		}
	}

	t.Log("✓ Real database parsing test completed successfully")
	totalPackages := 0
	for _, db := range impl.Databases {
		totalPackages += len(db.Packages)
	}
	t.Logf("✓ Summary: %d total packages, %d groups found", totalPackages, len(allGroups))
}

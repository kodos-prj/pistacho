// Package build provides build system integration for Pistacho.
// resolver_test.go contains tests for mixed dependency resolution.
package build

import (
	"testing"

	"github.com/kodos-prj/pistacho/pkg/aur"
)

// TestNewMixedResolver tests resolver initialization
func TestNewMixedResolver(t *testing.T) {
	tmpDir := t.TempDir()

	// We'll create a minimal ALPM client
	// For testing, we just need to verify the resolver initializes correctly
	resolver := NewMixedResolver(nil, tmpDir)

	if resolver == nil {
		t.Fatal("NewMixedResolver returned nil")
	}

	if resolver.aurRPC == nil {
		t.Fatal("aurRPC is nil")
	}

	if resolver.gitHandler == nil {
		t.Fatal("gitHandler is nil")
	}

	if resolver.pkgBuilder == nil {
		t.Fatal("pkgBuilder is nil")
	}

	if resolver.visited == nil {
		t.Fatal("visited map is nil")
	}

	if resolver.resolving == nil {
		t.Fatal("resolving map is nil")
	}
}

// TestPackageSource tests PackageSource initialization and fields
func TestPackageSource(t *testing.T) {
	official := PackageSource{
		Name:       "bash",
		Version:    "5.3.9-1",
		Source:     "official",
		Repo:       "core",
		IsAUR:      false,
		Depends:    []string{"libc", "ncurses"},
		OptDepends: []string{"bash-completion"},
	}

	if official.Name != "bash" {
		t.Errorf("expected name 'bash', got %q", official.Name)
	}

	if official.IsAUR {
		t.Error("IsAUR should be false for official package")
	}

	if official.Source != "official" {
		t.Errorf("expected source 'official', got %q", official.Source)
	}

	// Test AUR package source
	aurPkg := PackageSource{
		Name:        "vim-aur",
		Version:     "9.0.0-1",
		Source:      "aur",
		Repo:        "aur",
		IsAUR:       true,
		MakeDepends: []string{"gcc", "make"},
	}

	if !aurPkg.IsAUR {
		t.Error("IsAUR should be true for AUR package")
	}

	if len(aurPkg.MakeDepends) != 2 {
		t.Errorf("expected 2 make dependencies, got %d", len(aurPkg.MakeDepends))
	}
}

// TestResolverStateReset tests that resolver state is properly reset
func TestResolverStateReset(t *testing.T) {
	tmpDir := t.TempDir()
	resolver := NewMixedResolver(nil, tmpDir)

	// Simulate marking something as visited
	resolver.mu.Lock()
	resolver.visited["test-pkg"] = true
	resolver.mu.Unlock()

	// Now when we call ResolveDependencies, state should be reset
	// (We expect an error since we have no real ALPM client, but we can check the state reset)
	_, _ = resolver.ResolveDependencies("nonexistent-pkg", true, "")

	// After resolution attempt, the previous visited state should be cleared
	resolver.mu.RLock()
	if len(resolver.visited) == 1 && resolver.visited["test-pkg"] {
		t.Error("visited state was not properly reset")
	}
	resolver.mu.RUnlock()
}

// TestIsPackageInOfficial tests official package detection
func TestIsPackageInOfficial(t *testing.T) {
	tmpDir := t.TempDir()
	resolver := NewMixedResolver(nil, tmpDir)

	// With nil ALPM client, this will panic or return false
	// In a real test, we'd mock the ALPM client
	result := resolver.IsPackageInOfficial("nonexistent")
	if result {
		t.Error("expected false for non-existent package")
	}
}

// TestGetPackageSource tests package source detection
func TestGetPackageSource(t *testing.T) {
	tmpDir := t.TempDir()
	resolver := NewMixedResolver(nil, tmpDir)

	// With no real ALPM or AUR client, all packages should return "none"
	source := resolver.GetPackageSource("nonexistent")
	if source != "none" {
		t.Errorf("expected 'none', got %q", source)
	}
}

// TestResolverClose tests cleanup
func TestResolverClose(t *testing.T) {
	tmpDir := t.TempDir()
	resolver := NewMixedResolver(nil, tmpDir)

	err := resolver.Close()
	if err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}

// TestResolvePackageVersion tests version resolution
func TestResolvePackageVersion(t *testing.T) {
	tmpDir := t.TempDir()
	resolver := NewMixedResolver(nil, tmpDir)

	// With nil ALPM client, this should error
	version, source, err := resolver.ResolvePackageVersion("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent package")
	}
	if version != "" {
		t.Errorf("expected empty version, got %q", version)
	}
	if source != "" {
		t.Errorf("expected empty source, got %q", source)
	}
}

// TestMixedResolverWithMockALP tests resolver with mock functionality
func TestMixedResolverIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create resolver without real ALPM/AUR connections
	resolver := NewMixedResolver(nil, tmpDir)

	if resolver == nil {
		t.Fatal("resolver is nil")
	}

	// Test that internal structures are properly initialized
	if resolver.visited == nil {
		t.Fatal("visited map not initialized")
	}

	if resolver.resolving == nil {
		t.Fatal("resolving map not initialized")
	}

	// Test state management
	resolver.mu.Lock()
	resolver.visited["test"] = true
	resolver.mu.Unlock()

	resolver.mu.RLock()
	if !resolver.visited["test"] {
		t.Error("failed to mark package as visited")
	}
	resolver.mu.RUnlock()
}

// TestDependencyCycleDetection tests cycle detection logic
func TestDependencyCycleDetection(t *testing.T) {
	tmpDir := t.TempDir()
	resolver := NewMixedResolver(nil, tmpDir)

	// Manually set up a cycle scenario
	resolver.mu.Lock()
	resolver.resolving["pkg-a"] = true
	resolver.mu.Unlock()

	// Try to resolve the same package again (simulates cycle)
	// In real usage, this would be caught during recursive resolution
	// Cycles are now allowed (not an error) to handle real Arch database circular deps
	err := resolver.resolveDependenciesRecursive("pkg-a", &[]PackageSource{}, false, "")
	if err != nil {
		t.Errorf("cycle resolution returned error: %v (expected nil)", err)
	}
}

// TestPackageSourceFields tests various PackageSource field combinations
func TestPackageSourceFields(t *testing.T) {
	tests := []struct {
		name     string
		pkg      PackageSource
		wantAUR  bool
		wantRepo string
	}{
		{
			name: "official core repo",
			pkg: PackageSource{
				Name:   "bash",
				Source: "official",
				Repo:   "core",
				IsAUR:  false,
			},
			wantAUR:  false,
			wantRepo: "core",
		},
		{
			name: "aur package",
			pkg: PackageSource{
				Name:   "yay",
				Source: "aur",
				Repo:   "aur",
				IsAUR:  true,
			},
			wantAUR:  true,
			wantRepo: "aur",
		},
		{
			name: "official extra repo",
			pkg: PackageSource{
				Name:   "vim",
				Source: "official",
				Repo:   "extra",
				IsAUR:  false,
			},
			wantAUR:  false,
			wantRepo: "extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pkg.IsAUR != tt.wantAUR {
				t.Errorf("IsAUR: got %v, want %v", tt.pkg.IsAUR, tt.wantAUR)
			}
			if tt.pkg.Repo != tt.wantRepo {
				t.Errorf("Repo: got %q, want %q", tt.pkg.Repo, tt.wantRepo)
			}
			if tt.pkg.Source == "aur" && !tt.pkg.IsAUR {
				t.Error("AUR packages should have IsAUR=true")
			}
		})
	}
}

// TestResolverWithPKGBUILDInfo tests resolver with PKGBUILD info
func TestResolverWithPKGBUILDInfo(t *testing.T) {
	tmpDir := t.TempDir()
	_ = NewMixedResolver(nil, tmpDir) // Initialize resolver

	pkgInfo := &aur.PKGBUILDInfo{
		Name:        "test-pkg",
		Version:     "1.0.0",
		Depends:     []string{"bash", "coreutils"},
		MakeDepends: []string{"gcc", "make"},
		OptDepends:  []string{"git"},
	}

	pkg := PackageSource{
		Name:        pkgInfo.Name,
		Version:     pkgInfo.Version,
		Source:      "aur",
		Repo:        "aur",
		IsAUR:       true,
		PKGBUILD:    pkgInfo,
		Depends:     pkgInfo.Depends,
		MakeDepends: pkgInfo.MakeDepends,
		OptDepends:  pkgInfo.OptDepends,
	}

	if pkg.PKGBUILD == nil {
		t.Error("PKGBUILD should not be nil")
	}

	if pkg.PKGBUILD.Name != "test-pkg" {
		t.Errorf("expected package name test-pkg, got %q", pkg.PKGBUILD.Name)
	}

	if len(pkg.MakeDepends) != 2 {
		t.Errorf("expected 2 make dependencies, got %d", len(pkg.MakeDepends))
	}

	if len(pkg.Depends) != 2 {
		t.Errorf("expected 2 runtime dependencies, got %d", len(pkg.Depends))
	}
}

// TestResolverEdgeCases tests edge cases
func TestResolverEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	resolver := NewMixedResolver(nil, tmpDir)

	tests := []struct {
		name      string
		pkgName   string
		wantError bool
	}{
		{
			name:      "empty package name",
			pkgName:   "",
			wantError: true,
		},
		{
			name:      "invalid package name",
			pkgName:   "@invalid#name",
			wantError: true,
		},
		{
			name:      "nonexistent package",
			pkgName:   "this-package-definitely-does-not-exist",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolver.ResolveDependencies(tt.pkgName, true, "")
			if (err != nil) != tt.wantError {
				t.Errorf("error: got %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

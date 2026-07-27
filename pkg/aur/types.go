// Package aur provides Arch User Repository (AUR) support for Pistacho.
// It handles querying AUR packages, downloading PKGBUILDs, and parsing build metadata.
package aur

import (
	"time"
)

// AURPackage represents an AUR package with its metadata from the AUR RPC API v5.
// This struct maps directly to the JSON responses from https://aur.archlinux.org/rpc/v5/
type AURPackage struct {
	// Package identification
	ID          int    `json:"ID"`
	Name        string `json:"Name"`
	PackageBase string `json:"PackageBase"`
	Version     string `json:"Version"`
	Description string `json:"Description"`
	URL         string `json:"URL"`

	// Maintainer information
	Maintainer string `json:"Maintainer"`

	// Runtime dependencies
	Depends []string `json:"Depends"`

	// Build-time dependencies
	MakeDepends []string `json:"MakeDepends"`

	// Optional dependencies
	OptDepends []string `json:"OptDepends"`

	// Conflict information
	Conflicts []string `json:"Conflicts"`

	// Virtual packages this package provides
	Provides []string `json:"Provides"`

	// Packages this package replaces
	Replaces []string `json:"Replaces"`

	// Submission and modification timestamps (Unix time)
	FirstSubmitted int64 `json:"FirstSubmitted"`
	LastModified   int64 `json:"LastModified"`

	// Out-of-date status (0 = current, non-zero = unix timestamp of when marked outdated)
	OutOfDate int `json:"OutOfDate"`

	// Popularity metrics
	Popularity float64 `json:"Popularity"`
	NumVotes   int     `json:"NumVotes"`
}

// PKGBUILDInfo contains extracted metadata from a PKGBUILD file.
// This is parsed from the shell script to get build and dependency information.
type PKGBUILDInfo struct {
	// Package name and version
	Name    string
	Version string

	// Architecture support (e.g., x86_64, any, aarch64)
	Architecture []string

	// Dependencies
	Depends      []string
	MakeDepends  []string
	OptDepends   []string
	CheckDepends []string

	// Conflict information
	Conflicts []string
	Provides  []string
	Replaces  []string

	// Build options
	Options []string

	// Checksums (SHA256, MD5)
	SHA256Sums []string
	MD5Sums    []string

	// Source files
	Sources []string
}

// RPCSearchResult represents the response from an AUR RPC search query
type RPCSearchResult struct {
	Version     int          `json:"version"`
	Type        string       `json:"type"`
	ResultCount int          `json:"resultcount"`
	Results     []AURPackage `json:"results"`
}

// RPCInfoResult represents the response from an AUR RPC info query
type RPCInfoResult struct {
	Version     int          `json:"version"`
	Type        string       `json:"type"`
	ResultCount int          `json:"resultcount"`
	Results     []AURPackage `json:"results"`
}

// CachedAURPackage wraps an AUR package with cache metadata
type CachedAURPackage struct {
	Package   *AURPackage
	CachedAt  time.Time
	ExpiresAt time.Time
}

// IsCacheValid checks if the cached package is still valid
func (c *CachedAURPackage) IsCacheValid() bool {
	return time.Now().Before(c.ExpiresAt)
}

// Dependency represents a parsed dependency string with optional version constraint
// Example: "go>=1.21", "bash", "python>=3.10"
type Dependency struct {
	Name       string
	Constraint string // ">", "<", ">=", "<=", "=", or empty
	Version    string
}

// ParseDependency parses a dependency string into name and version constraint
// Examples:
//
//	"bash" → Dependency{Name: "bash"}
//	"go>=1.21" → Dependency{Name: "go", Constraint: ">=", Version: "1.21"}
//	"python<4.0" → Dependency{Name: "python", Constraint: "<", Version: "4.0"}
func ParseDependency(depString string) Dependency {
	// Handle empty string
	if depString == "" {
		return Dependency{}
	}

	// Check for version constraints (check longer ones first to avoid matching >= as >)
	constraints := []string{">=", "<=", "==", ">", "<", "="}
	for _, constraint := range constraints {
		idx := findConstraintIndexInString(depString, constraint)
		if idx > 0 {
			return Dependency{
				Name:       depString[:idx],
				Constraint: constraint,
				Version:    depString[idx+len(constraint):],
			}
		}
	}

	// No constraint found, just package name
	return Dependency{Name: depString}
}

// findConstraintIndexInString finds the index of constraint in str
func findConstraintIndexInString(str, constraint string) int {
	for i := 0; i < len(str); i++ {
		if i+len(constraint) <= len(str) && str[i:i+len(constraint)] == constraint {
			return i
		}
	}
	return -1
}

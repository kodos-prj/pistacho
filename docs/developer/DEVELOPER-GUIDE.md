# Developer Guide: Pure Go ALPM Implementation

## Overview

Pistacho has migrated from `go-alpm/v2` (CGO-based wrapper around libalpm) to a pure Go implementation of the ALPM (Arch Linux Package Manager) protocol. This document guides developers on the architecture, usage, and migration patterns.

## Architecture

### Pure Go ALPM Package (`pkg/alpm/`)

The new implementation provides a pure Go package that directly handles:

- **Database Parsing** (`parse.go`): Parses tar.gz ALPM databases from Arch mirrors
- **Version Comparison** (`version.go`): Implements RPM version comparison algorithm (VerCmp)
- **Database API** (`db.go`): Manages package databases and syncing
- **Dependency Resolution** (`deps.go`): Resolves package dependencies and conflicts
- **Caching** (`cache.go`): In-memory package cache for performance
- **GPG Verification** (`gpg.go`): Wrapper for GPG signature verification

### Wrapper Layer (`alpm.go`)

The `ALPMClient` struct provides backward compatibility with code that expected the go-alpm interface:

```go
type ALPMClient struct {
    impl     interface{}    // *Client (pure Go implementation)
    isGoImpl  bool
    root     string         // Root directory for installation
    dbPath   string         // Database directory path
}
```

## Migration Guide for Developers

### Scenario 1: Using ALPM for Package Search

#### Old (go-alpm):
```go
import "github.com/jguer/go-alpm/v2"

client, err := alpm.NewHandle("/", "/var/lib/pacman")
if err != nil {
    return err
}

// Search for package
pkg, err := client.GetPackage("bash")
```

#### New (Pure Go):
```go
import "github.com/kodos-prj/pistacho/pkg/alpm"

client, err := alpm.NewClient("/", "/var/lib/pacman/sync")
if err != nil {
    return err
}

// Search for package
pkg, err := client.SearchPackage("bash")
```

**Key Differences:**
- `dbPath` should point to the `sync` directory (where downloaded databases are stored)
- Method names changed (e.g., `GetPackage` → `SearchPackage`)
- Pure Go implementation doesn't require libalpm to be installed

### Scenario 2: Resolving Dependencies

#### Old (go-alpm):
```go
deps, err := client.ResolveDependencies([]string{"linux", "bash"})
```

#### New (Pure Go):
```go
deps, err := client.ResolveDependencies([]string{"linux", "bash"})
```

**Compatibility Note:** The interface is the same! The pure Go implementation handles:
- Multiple dependency types (Required, Optional, etc.)
- Version constraints (e.g., `package>=1.0`)
- Virtual packages and provides
- Circular dependency detection

### Scenario 3: Registering Databases

#### Old (go-alpm):
```go
client.RegisterSyncDBs()  // Auto-discovered
```

#### New (Pure Go):
```go
client.RegisterAllSyncDBs([]string{"core", "extra", "community"})
```

**Key Differences:**
- Databases must be explicitly registered
- Order determines precedence (first registered = highest priority)
- Databases are loaded from `dbPath` directory

## Data Structures

### Package Struct

```go
type Package struct {
    Name           string
    Version        string
    Description    string
    Architecture   string    // "x86_64", "aarch64", "any"
    Repository     string    // "core", "extra", "community"
    URL            string
    Licenses       []string
    Groups         []string
    Provides       []string
    DependsOn      []string
    OptDepends     []string
    Conflicts      []string
    Replaces       []string
    CompressedSize int64
    InstalledSize  int64
    PackageBase    string
    BuildDate      string
    Maintainer     string
    MD5Sum         string
    SHA256Sum      string
}
```

### Client Methods

```go
// Search for a package by exact name
func (c *Client) SearchPackage(name string) (*Package, error)

// Search packages matching a pattern
func (c *Client) SearchPackages(pattern string) ([]*Package, error)

// Resolve dependencies for given packages
func (c *Client) ResolveDependencies(pkgNames []string) ([]*Package, error)

// Get package information
func (c *Client) GetPackageInfo(name string) (*PackageInfo, error)

// Register sync databases
func (c *Client) RegisterSyncDB(repo string) error
func (c *Client) RegisterAllSyncDBs(repos []string) error

// Get download URL for a package
func (c *Client) GetDownloadURL(pkg *Package, mirrorURL string) string

// Close client and cleanup
func (c *Client) Close() error
```

## Database Format

ALPM databases are tar.gz archives containing package directories:

```
core.db.tar.gz/
├── acl-2.3.2-1/
│   ├── desc
│   ├── depends
│   ├── files (optional)
│   └── optdepends (optional)
├── bash-5.3.9-1/
│   ├── desc
│   ├── depends
│   └── ...
└── ...
```

### Metadata Format

Each `desc` file contains metadata in the format:

```
%FILENAME%
acl-2.3.2-1-x86_64.pkg.tar.zst

%NAME%
acl

%VERSION%
2.3.2-1

%DESC%
Access control list utilities, libraries and headers

%ARCH%
x86_64

%CSIZE%
141091

%ISIZE%
337902
```

## Version Comparison

The pure Go implementation includes an RPM-compatible version comparison algorithm:

```go
// Returns: <0 if v1 < v2, 0 if v1 == v2, >0 if v1 > v2
result := alpm.VerCmp("1.2.3-1", "1.2.4-1")  // Returns -1

// Supports epochs
alpm.VerCmp("2:1.0", "1:2.0")  // Returns 1 (epochs compared first)
```

## Dependency Resolution

The resolver handles:

```go
// Simple dependency
"bash"

// Version constraint
"glibc>=2.31"

// Alternative (OR dependency)
"xdotool|wmctrl"

// Optional dependency with description
"python: for some optional feature"
```

## Testing

Run tests with:

```bash
# All tests
go test ./...

# ALPM-specific tests
go test ./pkg/alpm -v

# Integration tests (downloads real databases)
go test ./integration -v

# With CGO disabled (recommended)
CGO_ENABLED=0 go test ./...
```

## Common Issues and Solutions

### Issue: "Database not found"
**Cause:** Wrong `dbPath` or database not synced
**Solution:** 
- Ensure `dbPath` points to the sync directory
- Verify databases are downloaded: `ls /var/lib/pacman/sync/`
- Check file permissions

### Issue: "Package not found" (but exists in database)
**Cause:** Database not registered or architecture mismatch
**Solution:**
- Call `RegisterAllSyncDBs()` with correct repository names
- Verify package architecture matches system: `uname -m`

### Issue: Version comparison unexpected results
**Cause:** Epoch not recognized
**Solution:** Versions with epochs use format: `epoch:version-release`
```
2:1.0-1 (epoch 2)
1:2.0-1 (epoch 1)
// First one is newer despite lower numeric version
```

### Issue: Build fails with CGO error
**Cause:** Trying to use go-alpm wrapper
**Solution:** Use pure Go implementation directly
```go
// Don't do this:
client, _ := alpm.NewHandle(...)

// Do this instead:
client := alpm.NewGoClient("/", "/var/lib/pacman/sync")
```

## Performance Notes

- **Caching:** Database queries are cached in memory after first load
- **Parsing:** Tar.gz parsing is optimized but slower than libalpm (written in C)
- **Memory:** In-memory cache uses ~20-50MB for core+extra databases
- **Startup:** First database load takes ~100-200ms, subsequent searches are instant

## Limitations and Workarounds

### Current Limitations:

1. **Limited GPG verification:** Wrapper calls system `gpg` command
2. **No file lists:** Only `desc` files are parsed, not full package file listings
3. **No local package queries:** Only sync databases, not local installed packages
4. **Single mirror:** No automatic fallback to alternate mirrors

### Workarounds:

1. **GPG verification:** Manual verification before using packages
2. **File lists:** Store separately or fetch from package website
3. **Local packages:** Implement separate local database
4. **Mirror fallback:** Implement at application level

## Migration Checklist

When migrating code from go-alpm to pure Go:

- [ ] Replace imports from `github.com/jguer/go-alpm/v2` to `github.com/kodos-prj/pistacho/pkg/alpm`
- [ ] Update `NewHandle()` to `NewClient()` with correct dbPath
- [ ] Update method calls: `GetPackage()` → `SearchPackage()`, etc.
- [ ] Ensure `RegisterAllSyncDBs()` is called before searches
- [ ] Remove libalpm installation from build scripts
- [ ] Update documentation (README, docs, etc.)
- [ ] Test with `CGO_ENABLED=0` build flag
- [ ] Run integration tests with real databases
- [ ] Benchmark and compare performance if needed

## Examples

### Example 1: Simple Package Search

```go
package main

import (
    "fmt"
    "github.com/kodos-prj/pistacho/pkg/alpm"
)

func main() {
    client, err := alpm.NewClient("/", "/var/lib/pacman/sync")
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    if err := client.RegisterAllSyncDBs([]string{"core", "extra"}); err != nil {
        panic(err)
    }
    
    pkg, err := client.SearchPackage("bash")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found: %s v%s (%s)\n", pkg.Name, pkg.Version, pkg.Repository)
}
```

### Example 2: Dependency Resolution

```go
func main() {
    client, _ := alpm.NewClient("/", "/var/lib/pacman/sync")
    defer client.Close()
    client.RegisterAllSyncDBs([]string{"core", "extra"})
    
    // Resolve what needs to be installed to get these packages
    deps, err := client.ResolveDependencies([]string{"linux", "base-devel"})
    if err != nil {
        panic(err)
    }
    
    for _, pkg := range deps {
        fmt.Println(pkg.Name, pkg.Version)
    }
}
```

## Contributing

When contributing to the pure Go ALPM implementation:

1. **Add tests** for new functionality
2. **Follow Go conventions** (package names, naming, documentation)
3. **Benchmark** changes if affecting performance-critical paths
4. **Update docs** (this guide) for user-facing API changes
5. **Test with CGO_ENABLED=0** to ensure pure Go compatibility

## References

- [ALPM Database Format](https://wiki.archlinux.org/title/Pacman/Pacman_database_format)
- [RPM Version Comparison](https://linux.die.net/man/8/rpm)
- [Package Dependencies](https://wiki.archlinux.org/title/Pacman#Viewing_package_dependencies)

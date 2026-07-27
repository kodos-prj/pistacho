# Registry Reference

Complete reference for Pistacho's package registry system that tracks all installed packages and their metadata.

---

## Overview

The registry is a JSON file that tracks all packages installed by Pistacho, maintaining metadata about each package including version, files, executables, dependencies, and installation date.

**Location**: `/kod/registry.json` (configurable via `Config.RegistryPath`)

**Format**: JSON map of package names to package structures

---

## Data Structure

### Registry Package

```go
type Package struct {
    Name         string   // Package name
    Version      string   // Version string (e.g., "5.3.9-1")
    Source       string   // "official" or "aur"
    Repository   string   // "core", "extra", "aur", etc.
    Files        []string // All extracted files (relative paths)
    Executables  []string // Executables in usr/bin, usr/sbin
    Dependencies []string // Tracked dependencies
    InstallDate  string   // RFC3339 timestamp
    UpdateDate   string   // RFC3339 timestamp (optional)
}
```

### Example Registry File

```json
{
  "bash": {
    "name": "bash",
    "version": "5.3.9-1",
    "source": "official",
    "repository": "core",
    "files": [
      "usr/bin/bash",
      "usr/share/doc/bash/README",
      "usr/share/man/man1/bash.1.gz"
    ],
    "executables": [
      "usr/bin/bash"
    ],
    "dependencies": [
      "glibc",
      "ncurses",
      "readline"
    ],
    "install_date": "2024-01-15T10:30:00Z",
    "update_date": "2024-03-21T14:20:00Z"
  },
  "glibc": {
    "name": "glibc",
    "version": "2.37-1",
    "source": "official",
    "repository": "core",
    "files": [
      "usr/lib/libc.so.6",
      "usr/lib/ld-linux-x86-64.so.2"
    ],
    "executables": [],
    "dependencies": [
      "linux-api-headers"
    ],
    "install_date": "2024-01-15T10:29:00Z"
  }
}
```

---

## Current Usage Patterns

### I/O Operations

| Operation | Method | Details |
|-----------|--------|---------|
| Load | `NewRegistry()` | Reads entire JSON from disk into memory |
| Save | `Save()` | Marshals entire map to JSON and writes to disk |
| Add | `AddPackage()` | Adds/updates in-memory map, no immediate disk write |
| Remove | `RemovePackage()` | Removes from in-memory map, no immediate disk write |
| Query | `GetPackage()`, `ListPackages()` | Reads from in-memory map |

### Command Operation Patterns

**Install Command**:
- Reads registry once at start
- Calls `AddPackage()` for each installed package (memory operations)
- Calls `Save()` once at end
- Pattern: **Read → Cache → Batch Updates → Single Write**

**Remove Command**:
- Reads registry once at start
- Checks dependencies against all installed packages
- Calls `RemovePackage()` for each removal
- Calls `Save()` once at end
- Pattern: **Read → Modify-In-Memory → Single Write**

**Upgrade Command**:
- Reads registry once at start
- Compares versions for outdated packages
- Re-uses InstallCommand for each upgrade
- Calls `Save()` inside InstallCommand
- Pattern: **Multiple Read-Modify-Write cycles**

**List Command**:
- Reads registry once at start
- Calls `ListPackages()` (memory operation)
- No writes
- Pattern: **Read-Only**

**Cleanup Command**:
- Reads registry once at start
- Identifies active versions from registry
- No writes to registry (filesystem operations only)
- Pattern: **Read-Only**

### Disk I/O Summary

- **Most operations**: 1 disk read at start, 1 disk write at end (efficient)
- **Upgrade command**: Multiple reads/writes across multiple invocations
- **Frequency**: Registry accessed on every package operation
- **Typical workflow**: Install (1 write) → Upgrade (N writes) → List (1 read) → Remove (1 write)

---

## Thread & Process Safety

### Current Locking

**In-Process**: `sync.RWMutex` for thread-safe concurrent access within same process
- Protects against goroutine race conditions
- Allows multiple readers or single writer

**File-Level**: NO inter-process locking
- Different `pistacho` invocations don't coordinate access
- Relies on atomic `os.WriteFile` behavior

### Concurrency Issues

**Multi-User Scenario**:
```
User A: Reads registry, installs pkg1, writes registry
User B: Reads registry (gets old version), installs pkg2, writes registry
Result: User B overwrites User A's changes (lost update problem)
```

**Upgrade Crash Scenario**:
```
Upgrade starts, installs pkg1, registers it
Upgrade crashes before writing registry
Registry state inconsistent with filesystem state
```

**Current Mitigation**:
- Single-user assumption (all operations as same user or root)
- Design accepts registry-filesystem divergence
- No rollback mechanism on failure

### Recommendations

For multi-user systems:
1. Implement file-level locking using `flock` or similar
2. Add atomic read-compare-write operations
3. Implement transaction log for crash recovery
4. Add pre-flight validation of registry vs. filesystem state

---

## Registry Operations API

### NewRegistry(path string) *Registry

Creates or loads a registry from the given path.

```go
reg := registry.NewRegistry("/kod/registry.json")
```

### AddPackage(pkg *Package) error

Adds or updates a package in the registry (in-memory).

```go
reg.AddPackage(&registry.Package{
    Name: "bash",
    Version: "5.3.9-1",
    Files: []string{"usr/bin/bash", ...},
    Dependencies: []string{"glibc", "ncurses"},
    InstallDate: time.Now().Format(time.RFC3339),
})
```

### RemovePackage(name string) error

Removes a package from the registry (in-memory).

```go
reg.RemovePackage("bash")
```

### GetPackage(name string) *Package

Retrieves a package from the registry.

```go
if pkg := reg.GetPackage("bash"); pkg != nil {
    fmt.Printf("%s %s\n", pkg.Name, pkg.Version)
}
```

### ListPackages() []*Package

Returns all installed packages.

```go
packages := reg.ListPackages()
for _, pkg := range packages {
    fmt.Printf("%s %s\n", pkg.Name, pkg.Version)
}
```

### Save() error

Writes the registry to disk as JSON.

```go
if err := reg.Save(); err != nil {
    log.Fatalf("Failed to save registry: %v", err)
}
```

---

## File Format

### JSON Schema

```json
{
  "package-name": {
    "name": "string",
    "version": "string",
    "source": "official | aur",
    "repository": "string",
    "files": ["string"],
    "executables": ["string"],
    "dependencies": ["string"],
    "install_date": "RFC3339 timestamp",
    "update_date": "RFC3339 timestamp (optional)"
  }
}
```

### Field Descriptions

- **name**: Package name (matches key)
- **version**: Version string following Arch versioning (e.g., "5.3.9-1")
- **source**: Package source - "official" for Arch repos, "aur" for AUR
- **repository**: Repository name ("core", "extra", "community", "aur", etc.)
- **files**: List of all extracted files (relative to store root)
- **executables**: List of executable files (in usr/bin or usr/sbin)
- **dependencies**: List of direct dependencies (for tracking purposes)
- **install_date**: When the package was installed (RFC3339 format)
- **update_date**: When the package was last updated (optional)

---

## Integration with Package Management

### During Installation

```
install.go:483-528
├─ reg := registry.NewRegistry(registryPath)  [Load from disk]
├─ For each package in toInstall:
│  ├─ regPkg := &registry.Package{...}
│  └─ reg.AddPackage(regPkg)                 [Update in memory]
└─ reg.Save()                                [Write to disk]
```

### During Removal

```
remove.go:72-176
├─ reg := registry.NewRegistry(registryPath)  [Load from disk]
├─ For each package in toRemove:
│  ├─ Check dependencies: reg.ListPackages()
│  └─ reg.RemovePackage(pkgName)             [Update in memory]
└─ reg.Save()                                [Write to disk]
```

### During Listing

```
list.go:29
├─ reg := registry.NewRegistry(registryPath)  [Load from disk]
├─ packages := reg.ListPackages()            [Query in memory]
└─ Display packages                          [No write]
```

### Dependency Checking

The registry is used to identify:
- **Installed status**: Prevent re-installing
- **Dependencies**: When removing, check what depends on this package
- **Orphaned files**: Track which files belong to which package
- **Version tracking**: Which version is currently installed

---

## Consistency Guarantees

### Read Consistency

- **Strong**: All reads see latest committed writes from previous operations
- **Note**: Assumes no concurrent writes from different processes

### Write Consistency

- **Atomic**: `os.WriteFile` is atomic on most filesystems
- **No rollback**: Failed partial writes leave file in unknown state
- **No transactions**: Multiple packages written in single JSON file

### Cross-Invocation Consistency

- **Persistent**: Changes survive process termination
- **Sequential**: Each invocation sees results of previous invocations
- **No ordering guarantees**: Multiple concurrent invocations may lose updates

---

## Future Enhancements

### Immediate (Recommended)

1. **File Locking**: Implement `flock` for multi-user safety
2. **Validation**: Add registry-filesystem consistency checks
3. **Logging**: Track all registry modifications to audit log

### Medium Term

1. **Generation-Based Paths**: Move to per-version registry files
2. **Atomic Transactions**: Use temporary files + rename pattern
3. **Crash Recovery**: Transaction log for recovery from crashes

### Long Term

1. **Database Backend**: Replace JSON with SQLite for performance
2. **Replication**: Support multi-system registry sync
3. **Version Control**: Git-backed registry for full history

---

## Quick Reference

| Task | Method | Pattern |
|------|--------|---------|
| Load registry | `NewRegistry(path)` | Read entire JSON |
| Add package | `AddPackage(pkg)` | Update in-memory map |
| Remove package | `RemovePackage(name)` | Update in-memory map |
| Find package | `GetPackage(name)` | Query in-memory map |
| List all | `ListPackages()` | Get all packages |
| Persist | `Save()` | Write entire map to JSON |
| Check installed | `GetPackage(name) != nil` | Simple existence check |
| Find dependents | Manual iteration over `ListPackages()` | Search dependencies array |

---

## See Also

- [ARCHITECTURE.md](../architecture/ARCHITECTURE.md) - Complete system architecture
- [CONFIGURATION.md](CONFIGURATION.md) - Configuration reference
- User guides in `/docs/user-guides/`

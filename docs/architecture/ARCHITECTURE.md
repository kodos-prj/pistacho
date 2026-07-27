# Pistacho Architecture

Complete architecture documentation for Chisel, a cross-distribution package manager that brings Arch Linux packages to any Linux distribution.

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Directory Structure](#directory-structure)
3. [Core Components](#core-components)
4. [Dependency Resolution](#dependency-resolution)
5. [Package Installation Flow](#package-installation-flow)
6. [Download & Extract System](#download--extract-system)
7. [Wrapper & Symlink Management](#wrapper--symlink-management)
8. [CLI Command Structure](#cli-command-structure)
9. [Data Structures](#data-structures)
10. [AUR Extension Points](#aur-extension-points)
11. [Error Handling](#error-handling)

---

## System Overview

Chisel is organized into modular packages that handle specific responsibilities:

- **Base Directory**: `/kod` (configurable)
- **Store**: `/kod/store/{package}/{version}/` - Extracted packages in isolation
- **Registry**: `/kod/registry.json` - Metadata of installed packages
- **Databases**: `/kod/db/sync/{repo}.db` - Arch package metadata
- **Cache**: `/kod/cache/` - Downloaded `.pkg.tar.zst` files
- **Wrappers**: `/kod/wrappers/` - Shell scripts that set `LD_LIBRARY_PATH` for executables

---

## Directory Structure

```
chisel/
├── cmd/
│   └── chisel/
│       └── main.go                    # CLI entry point (542 lines)
├── pkg/                               # 11 public packages (~11,200 LOC)
│   ├── alpm/                          # Pure Go package management (92 KB)
│   │   ├── alpm.go, types.go, parse.go, db.go
│   │   ├── version.go, deps.go, cache.go, gpg.go
│   │   └── *_test.go (7 test files)
│   ├── aur/                           # AUR support (48 KB)
│   ├── build/                         # AUR builder system (52 KB)
│   ├── config/                        # Configuration management (28 KB)
│   ├── database/                      # Sync databases (16 KB)
│   ├── download/                      # Download packages (20 KB)
│   ├── extract/                       # Extract archives (28 KB)
│   ├── registry/                      # Track installed packages (20 KB)
│   ├── store/                         # Manage package store (20 KB)
│   ├── symlink/                       # Symlink management (20 KB)
│   └── wrapper/                       # Generate wrapper scripts (28 KB)
└── internal/cli/                      # CLI commands (5,981 LOC)
    ├── sync.go, search.go, install.go, remove.go
    ├── list.go, upgrade.go, cleanup.go, cache.go
    └── *_test.go (8 test files)
```

---

## Core Components

### 1. ALPM Package Management (`pkg/alpm/`)

Pure Go implementation of Arch Linux Package Management with no external dependencies.

**Key Files**:
- `alpm.go` - Main client interface
- `types.go` - Package, Database, DatabaseCache, Dependency structures
- `parse.go` - Database parsing from tar.gz archives
- `db.go` - Database API and dependency resolution
- `version.go` - RPM version comparison algorithm (VerCmp)
- `deps.go` - Dependency resolution with circular dependency handling
- `cache.go` - In-memory caching with repository precedence

**Repository Precedence**:
```
core (0)      - Highest priority
extra (1)
community (2)
multilib (3)  - Lowest priority
```

### 2. Configuration (`pkg/config/`)

JSON-based configuration management supporting both system-level and user-level modes.

**Default Locations**:
- System: `/etc/chisel/config.json`
- User: `~/.config/chisel/config.json`

**Key Configuration**:
```json
{
  "base_dir": "/kod",
  "symlink_root": "/",
  "store_root": "/kod/store",
  "registry_path": "/kod/registry.json",
  "db_path": "/kod/db/sync",
  "cache_path": "/kod/cache",
  "wrapper_dir": "/kod/wrappers",
  "mirror_url": "https://mirror.rackspace.com/archlinux",
  "architecture": "x86_64",
  "repositories": ["core", "extra", "community"],
  "max_concurrent_downloads": 5,
  "download_timeout": 300,
  "keep_versions": 3
}
```

### 3. Registry (`pkg/registry/`)

Tracks installed packages and their metadata in JSON format.

**Location**: `/kod/registry.json`

**Fields per Package**:
```json
{
  "bash": {
    "name": "bash",
    "version": "5.3.9-1",
    "files": ["usr/bin/bash", "usr/share/..."],
    "executables": ["usr/bin/bash"],
    "dependencies": ["glibc", "ncurses"],
    "install_date": "2024-01-15T10:30:00Z"
  }
}
```

### 4. Download Manager (`pkg/download/`)

Manages concurrent downloads of `.pkg.tar.zst` files from Arch mirrors.

**URL Format**: `{mirror}/{repo}/os/{arch}/{name}-{version}-{arch}.pkg.tar.zst`

**Download Process**:
1. Create cache directory if missing
2. HTTP GET with configurable timeout
3. Write to temporary file: `{cache}/{filename}.tmp`
4. Atomic rename to final: `{cache}/{filename}`
5. Concurrent downloads with configurable semaphore (default 5)

### 5. Extraction (`pkg/extract/`)

Extracts `.pkg.tar.zst` packages to the store with security checks.

**Archive Format**: Zstd compression + tar format

**Extraction Process**:
1. Decompress zstd stream
2. Read tar entries
3. Create directories, files, symlinks
4. Preserve permissions
5. Prevent path traversal attacks

**Output**: `[]ExtractedFile` with metadata (path, size, mode, symlink info)

### 6. Symlink & Wrapper Management

**Two-Tier Symlink Strategy**:
- **Executables** (usr/bin/*, usr/sbin/*) → wrapper shell scripts
- **Other files** → direct symlinks to store location

**Wrapper Scripts**:
- Located in `/kod/wrappers/{cmdName}`
- Set `LD_LIBRARY_PATH` for dependencies
- Execute binary from store with isolation

---

## Dependency Resolution

### Algorithm Overview

Uses depth-first search with cycle detection to resolve dependencies.

**Entry Point**: `Client.ResolveDependencies(packageName)` → `pkg/alpm/db.go:130`

**Algorithm**:
1. Maintain `visited` set (completed packages)
2. Maintain `visiting` set (currently processing - for cycle detection)
3. For each dependency, recursively resolve with DFS
4. Detect cycles when visiting package already in `visiting` set
5. Return ordered list: dependencies before dependents

### Version Constraint Handling

**Constraint Types**:
- `ConstraintNone` - Any version acceptable
- `ConstraintEqual` - Exact version match (e.g., `vim=8.2`)
- `ConstraintGreaterEqual` - `vim>=8.2`
- `ConstraintGreater` - `vim>8.2`
- `ConstraintLessEqual` - `vim<=8.2`
- `ConstraintLess` - `vim<8.2`

### Version Comparison (VerCmp)

**Location**: `pkg/alpm/version.go:9-34`

Implements exact RPM version scheme:
1. Parse epochs (prefix before `:`)
2. Split release and revision (suffix after last `-`)
3. Compare using segment-based tokenization
4. Handles "1.0" < "1.0.1" cases correctly

**Example**: 
- "5.3.9-1" < "5.3.10-1" ✓
- "2:1.0" > "1:2.0" ✓

### Dependency Resolution Flow

```
client.ResolveDependencies("bash")
  ↓
Find package in cache
  ↓
Recursively resolve each dependency (DFS)
  ├─ Detect cycles via "visiting" set
  ├─ Parse dependency constraints
  ├─ Validate version meets requirement
  ├─ Fall back to virtual packages if needed
  └─ Append in correct order
  ↓
Return: [linux-api-headers, zlib, glibc, ncurses, bash]
```

---

## Package Installation Flow

Complete end-to-end installation with 6 stages.

### Stage 1: Dependency Resolution

```
resolveDependenciesWithMap(client, pkgNames, skipDeps)
  ├─ Load all sync databases
  ├─ For each package: ResolveDependencies()
  ├─ Skip already installed (registry check)
  └─ Build toInstall list + dependency map
```

### Stage 2: Download

```
downloader.DownloadPackages(toInstall)
  ├─ Create semaphore(maxConcurrent=5)
  └─ For each package (concurrent):
    ├─ HTTP GET from mirror
    ├─ Write to .tmp file
    └─ Atomic rename to final location
```

### Stage 3: Extract

```
storeManager.ExtractPackage(cachePath, pkgName, version)
  ├─ Create destDir: /kod/store/{pkgName}/{version}
  ├─ Decompress .pkg.tar.zst
  ├─ Extract tar entries
  ├─ Preserve permissions
  └─ Update "current" symlink
```

### Stage 4: Symlink Creation

```
For each package's files:
  ├─ Skip metadata (.PKGINFO, .BUILDINFO, .MTREE, .INSTALL)
  ├─ If executable (usr/bin/* or usr/sbin/*):
  │  └─ Point to wrapper script
  └─ Otherwise:
     └─ Point directly to store location
```

### Stage 5: Wrapper Generation

```
wrapperGen.GenerateWrapperWithDeps(cmdName, pkgName, version, dependencies)
  ├─ Create shell script at /kod/wrappers/{cmdName}
  ├─ Set LD_LIBRARY_PATH with dependency lib paths
  ├─ Exec /kod/store/{pkgName}/{version}/{execPath}
  └─ Create symlink: /usr/bin/{cmdName} → /kod/wrappers/{cmdName}
```

### Stage 6: Registry Update

```
reg.AddPackage(regPkg)
  ├─ Record name, version, files, executables
  ├─ Record dependencies and install date
  └─ Save to /kod/registry.json (JSON format)
```

---

## Download & Extract System

### Download Implementation

**Location**: `pkg/download/download.go`

**Downloader Struct**:
```go
type Downloader struct {
    mirrorURL string
    cachePath string
    arch string
    maxConcurrent int
    downloadTimeout time.Duration
    httpClient *http.Client
}
```

**URL Construction**:
```
{mirrorURL}/{repo}/os/{arch}/{name}-{version}-{arch}.pkg.tar.zst
Example: https://mirror.rackspace.com/archlinux/core/os/x86_64/bash-5.3.9-1-x86_64.pkg.tar.zst
```

**Download Process**:
1. Create cache directory if missing
2. HTTP GET with configurable timeout
3. Check status code (must be 200 OK)
4. Write to temporary file
5. Atomic rename to final location
6. Use semaphore for concurrent control

### Extraction Implementation

**Location**: `pkg/extract/extract.go`

**Extractor Struct**:
```go
type Extractor struct {
    preservePerms bool  // Preserve original file permissions
}
```

**Archive Format**: `.pkg.tar.zst` (zstd compression + tar)

**Extraction Process**:
1. Open and decompress `.pkg.tar.zst`
2. Create destination directory
3. For each tar entry:
   - **Regular files**: Create and copy content
   - **Directories**: Create with `os.MkdirAll`
   - **Symlinks**: Create symlink with target
   - **Hard links**: Create hard link
4. Prevent path traversal via `HasPrefix` check
5. Return `[]ExtractedFile` with metadata

**Extracted File Metadata**:
```go
type ExtractedFile struct {
    Path string         // Relative: "usr/bin/bash"
    AbsPath string      // Absolute: "/kod/store/bash/5.3.9-1/usr/bin/bash"
    IsDirectory bool
    IsSymlink bool
    LinkTarget string
    Size int64
    Mode os.FileMode
}
```

---

## Wrapper & Symlink Management

### Wrapper Script Generation

**Purpose**: Dynamically set `LD_LIBRARY_PATH` for executable isolation

**Example Wrapper for `bash`**:
```bash
#!/bin/bash
export LD_LIBRARY_PATH="/kod/store/glibc/2.37-1/usr/lib:/kod/store/ncurses/6.4-1/usr/lib:$LD_LIBRARY_PATH"
exec /kod/store/bash/5.3.9-1/usr/bin/bash "$@"
```

**Symlink Hierarchy**:
```
/usr/bin/bash → /kod/wrappers/bash
/usr/share/doc/bash/README → /kod/store/bash/5.3.9-1/usr/share/doc/bash/README
/usr/lib/libc.so.6 → /kod/store/glibc/2.37-1/usr/lib/libc.so.6
```

### Symlink Conflict Resolution

For each file to symlink:
1. Check if symlink already exists
   - If points to correct location: skip
   - If points elsewhere: skip with warning (unless `--force`)
   - If regular file exists: skip with warning
2. Create parent directories
3. Create symlink

---

## CLI Command Structure

### Main Router

**File**: `cmd/chisel/main.go`

**Commands**:
```
sync       - Synchronize Arch databases
search     - Search for packages
info       - Display package information
download   - Download packages
extract    - Extract packages to store
install    - Full installation flow
remove     - Remove installed packages
list       - List installed packages
query      - Search installed packages
upgrade    - Upgrade installed packages
cleanup    - Remove old package versions
cache      - Manage download cache
```

### Command Handler Pattern

Each command handler:
1. Parses command-line flags
2. Calls `loadConfig()` to get configuration
3. Creates command object from `internal/cli/`
4. Calls `Run()` or `Execute()` method
5. Handles errors and exit codes

### Configuration Loading

**Priority Order**:
1. Command-line flags
2. Environment variables (CHISEL_CONFIG, CHISEL_BASE_DIR, etc.)
3. Config file (/etc/chisel/config.json)
4. Built-in defaults

---

## Data Structures

### ALPM Package

```go
type Package struct {
    Name, Version, Description, Architecture string
    URL string
    Licenses []string
    Groups []string
    Provides []string         // Virtual packages
    DependsOn, OptDepends []string
    Conflicts, Replaces []string
    CompressedSize, InstalledSize int64
    Repository, PackageBase string
    BuildDate, Maintainer string
    MD5Sum, SHA256Sum string
}
```

### Database

```go
type Database struct {
    Name string                           // Repository name
    Path string                           // Disk location
    Packages map[string]*Package          // Name → Package
    Provides map[string][]*Package        // Virtual → Packages
    Arch string                           // Target architecture
}
```

### Registry Package

```go
type Package struct {
    Name, Version string
    Files, Executables []string
    Dependencies []string
    InstallDate string  // RFC3339 format
}
```

---

## AUR Extension Points

### Proposed New Commands

```go
case "aur-install": handleAURInstall()
case "aur-search": handleAURSearch()
case "aur-info": handleAURInfo()
case "aur-upgrade": handleAURUpgrade()
```

### New Packages Needed

- `pkg/aur/client.go` - AUR API queries and PKGBUILD fetching
- `pkg/build/builder.go` - makepkg wrapper for building
- `internal/cli/aur_install.go` - CLI integration for AUR

### Extended Package Metadata

```go
// Add to Package struct:
IsAUR bool
AURPackageBase string
AURMaintainer string
PKGBUILDRef string
AURBuildDeps []string
```

### Build Flow Integration

```
aur-install package-name
  ↓
Search AUR for package
  ↓
Clone PKGBUILD from AUR git
  ↓
Run makepkg build process
  ↓
Move output to cache as .pkg.tar.zst
  ↓
Continue with normal install flow (stages 3-6)
```

**No changes needed** to extract or subsequent logic - same `.pkg.tar.zst` format!

---

## Error Handling

### Circular Dependency Detection

Maintains `visiting` set during DFS:
- If current package already in `visiting`: cycle detected
- Return `ResolutionError` with cycle information
- Propagate error up to CLI

### Missing Dependency

When package lookup fails:
1. Try exact match: `Cache.GetPackage(name)`
2. Fall back to virtual: `Cache.GetProvidingPackages(name)`
3. If both fail: return `ResolutionError` with package name

### Download Failures

Handle partial download success:
- Collect errors for failed packages
- Continue with successfully downloaded packages
- Warn user of failures but don't abort

### Path Traversal Prevention

In extraction:
```go
targetPath := filepath.Join(destDir, header.Name)
if !filepath.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir)) {
    // Archive contains path outside destination - reject
}
```

Prevents attacks like `../../../../etc/passwd` in archives.

---

## Testing

### Test Coverage

- **Dependency Resolution Tests** (`pkg/alpm/deps_test.go`)
  - Simple chains
  - Multiple dependencies
  - Transitive dependencies
  - Circular dependency detection
  - Version constraints
  - Virtual packages

- **Version Comparison Tests** (`pkg/alpm/version_test.go`)
  - Epoch handling
  - Release/revision parsing
  - Complex version strings

- **Database Tests** (`pkg/alpm/db_test.go`)
  - Database loading
  - Package search
  - Cache merging

- **Integration Tests** (`integration/full_workflow_test.go`)
  - End-to-end installation workflows

---

## File Reference Index

| Component | Files | Key Lines |
|-----------|-------|-----------|
| **CLI Router** | cmd/chisel/main.go | 22-87, 152-202 |
| **Dependency Resolution** | pkg/alpm/db.go | 126-207 |
| **Version Comparison** | pkg/alpm/version.go | 9-34 |
| **ALPM Public API** | pkg/alpm/alpm.go | 38-206 |
| **Database Parsing** | pkg/alpm/parse.go | 14-250 |
| **Database Caching** | pkg/alpm/cache.go | 3-95 |
| **Database Sync** | pkg/database/sync.go | 34-116 |
| **Download Manager** | pkg/download/download.go | 24-177 |
| **Extraction** | pkg/extract/extract.go | 39-272 |
| **Store Management** | pkg/store/store.go | 26-273 |
| **Symlink Management** | pkg/symlink/symlink.go | 12-206 |
| **Registry** | pkg/registry/registry.go | 35-116 |
| **Install Command** | internal/cli/install.go | 56-524 |
| **Wrapper Generation** | pkg/wrapper/wrapper.go | - |
| **Config** | pkg/config/config.go | 22-200+ |

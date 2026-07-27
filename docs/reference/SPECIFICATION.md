# Pistacho Specification

## Overview

Pistacho is a cross-distribution package manager that brings Arch Linux packages to any Linux distribution (Ubuntu, Fedora, Debian, etc.) with complete dependency isolation.

## Core Problem

Stable LTS distributions (Ubuntu 22.04, Debian 12) ship with outdated packages. Pistacho solves this by:
- Running Arch binaries natively on any distribution
- Providing complete dependency isolation (ALL dependencies from Arch, not host)
- Creating wrapper scripts that dynamically set `LD_LIBRARY_PATH`
- Using a custom ALPM root at `/kod/` instead of `/`

## Architecture

### Directory Structure

```
/kod/
├── store/           # Package storage: /kod/store/<package>/<version>/
├── wrappers/        # Wrapper scripts: /kod/wrappers/<package>
├── db/              # Arch databases: /kod/db/<repo>/
├── registry.json    # Package registry (JSON)
└── config.json      # Configuration file
```

### Components

1. **Config** - Configuration management
2. **Registry** - Package tracking and metadata
3. **Database** - Sync and manage Arch package databases
4. **ALPM** - Arch Linux Package Management wrapper
5. **Store** - Package storage and extraction
6. **Download** - Package download manager
7. **Extract** - Package archive extraction (.pkg.tar.zst)
8. **Wrapper** - Shell script generation for library isolation
9. **Symlink** - Two-tier symlink management

## Functionality Specification

### 1. Configuration (`pkg/config`)

**Purpose**: Manage Pistacho configuration

**Data Structure**:
```json
{
  "base_dir": "/kod",
  "db_dir": "/kod/db",
  "store_dir": "/kod/store",
  "wrappers_dir": "/kod/wrappers",
  "symlink_dir": "/usr/bin",
  "registry_file": "/kod/registry.json",
  "mirror_url": "https://geo.mirror.pkgbuild.com",
  "arch": "x86_64",
  "repos": ["core", "extra", "community"]
}
```

**Operations**:
- `DefaultConfig()` - Return default configuration
- `Normalize()` - Validate and normalize configuration paths
- `Save(path string)` - Write config to JSON file
- `Load(path string)` - Read and parse config from JSON file
- `Validate()` - Check configuration validity

### 2. Package Registry (`pkg/registry`)

**Purpose**: Track installed packages and their metadata

**Data Structure** (`registry.json`):
```json
{
  "installed": {
    "vim": {
      "name": "vim",
      "version": "9.0.000",
      "installed_at": "2024-01-15T10:30:00Z",
      "size": 1234567,
      "provides": ["editor", "vi"],
      "depends_on": ["ncurses", "glibc"]
    }
  }
}
```

**Operations**:
- `NewRegistry(path string)` - Initialize registry at path
- `Install(pkg PackageInfo)` - Record package installation
- `Uninstall(name string)` - Remove package from registry
- `Get(name string) (*PackageInfo, error)` - Retrieve package info
- `List() ([]PackageInfo, error)` - List all installed packages
- `Exists(name string) bool` - Check if package is installed
- `UpdateLastUsed(name string)` - Update last-used timestamp

### 3. Database Sync (`pkg/database`)

**Purpose**: Download and manage Arch Linux package databases

**Database Files**:
- `core.db` - Core repository packages
- `extra.db` - Extra repository packages
- `community.db` - Community repository packages

**Operations**:
- `NewSyncer(config *Config)` - Create syncer with configuration
- `DownloadDatabase(repo, url string) (string, error)` - Download single database
- `Sync() error` - Sync all configured repositories
- `DatabaseExists(repo string) bool` - Check if database exists
- `LastSyncTime(repo string) (time.Time, error)` - Get last sync timestamp
- `ListDatabases() ([]string, error)` - List all downloaded databases

**Download URL Format**:
```
{mirror_url}/{repo}/os/{arch}/{filename}.db
```

### 4. ALPM Operations (`pkg/alpm`)

**Purpose**: Pure Go implementation of package management operations

**Client Types**:
- `GoClient` - Pure Go implementation (recommended, no external dependencies)
- `ALPMClient` - Wrapper for backward compatibility (uses go-alpm/v2 if available)

**Operations**:
- `NewClient(root, dbPath string)` - Initialize ALPM handle
- `Close()` - Release ALPM resources
- `RegisterSyncDB(repo string)` - Register a sync database
- `RegisterAllSyncDBs(repos []string)` - Register multiple databases
- `SearchPackage(name string) (IPackage, error)` - Search for single package
- `SearchPackages(pattern string) ([]IPackage, error)` - Pattern search
- `GetPackageInfo(name string) (*PackageInfo, error)` - Get detailed package info
- `ResolveDependencies(packageName string) ([]string, error)` - Resolve all deps
- `ListSyncDBs() ([]string, error)` - List registered databases
- `GetLocalPackages() ([]IPackage, error)` - Get installed packages
- `IsPackageInstalled(name string) (bool, error)` - Check installation status
- `GetDownloadURL(pkg IPackage, mirrorURL, arch string) string` - Generate download URL

**PackageInfo Fields**:
- Name, Version, Description
- Architecture, URL, Licenses
- Groups, Provides
- DependsOn, OptDepends, Conflicts, Replaces
- Size, DownloadSize
- Repository, PackageBase, Maintainer

### 5. Package Store (`pkg/store`)

**Purpose**: Store and manage extracted package contents

**Storage Layout**:
```
/kod/store/<package>/<version>/
├── usr/
├── lib/
└── ... (rest of package filesystem)
```

**Operations**:
- `NewStore(baseDir string)` - Create store instance
- `StorePackage(name, version string, reader io.Reader) (string, error)` - Store package
- `GetPackagePath(name, version string) string` - Get package path
- `ListPackages() ([]Package, error)` - List all stored packages
- `RemovePackage(name, version string) error` - Remove package from store
- `PackageExists(name, version string) bool` - Check package existence

### 6. Package Download (`pkg/download`)

**Purpose**: Download Arch Linux packages (.pkg.tar.zst)

**Download URL Format**:
```
{mirror_url}/{repo}/os/{arch}/{name}-{version}-{arch}.pkg.tar.zst
```

**Operations**:
- `NewDownloader(config *Config)` - Create downloader
- `DownloadPackage(pkg PackageInfo, mirrorURL string) (string, error)` - Download single package
- `DownloadPackages(packages []PackageInfo, mirrorURL string) ([]string, error)` - Download multiple
- `DownloadWithConcurrency(packages []PackageInfo, mirrorURL string, concurrency int) error` - Parallel download
- `PackageExists(localPath string) bool` - Check if package exists locally

**Features**:
- Atomic writes (download to temp, then rename)
- Concurrent downloads with configurable limit
- Progress reporting
- Retry on failure

### 7. Package Extraction (`pkg/extract`)

**Purpose**: Extract .pkg.tar.zst archives

**Archive Format**: Arch Linux package format (tar.zst compressed)

**Operations**:
- `ExtractPackage(packagePath, destDir string) error` - Extract entire package
- `ExtractFile(packagePath, filePath, destDir string) error` - Extract single file
- `GetPackageContents(packagePath) ([]string, error)` - List all files in package
- `VerifyPackage(packagePath) (bool, error)` - Verify archive integrity

**Features**:
- zstd decompression
- Preserve file permissions
- Handle symlinks in archives
- Extract metadata (PKGBUILD info)

### 8. Wrapper Generation (`pkg/wrapper`)

**Purpose**: Generate shell wrapper scripts for library isolation

**Wrapper Script Format**:
```bash
#!/bin/bash
export LD_LIBRARY_PATH="/kod/store/vim/9.0.000/usr/lib:$LD_LIBRARY_PATH"
exec "/kod/store/vim/9.0.000/usr/bin/vim" "$@"
```

**Operations**:
- `NewGenerator(config *Config)` - Create wrapper generator
- `GenerateWrapper(packageName, version string, binaries []BinaryInfo) (string, error)` - Generate wrapper
- `RemoveWrapper(packageName string) error` - Remove wrapper script
- `GetWrapperPath(packageName string) string` - Get wrapper path
- `BuildWrapperScript(binaries []BinaryInfo, libPaths []string) string` - Build wrapper content
- `DiscoverLibraries(packagePath string) ([]Library, error)` - Find .so files in package

**BinaryInfo Fields**:
- Path (relative to package root)
- IsExecutable (boolean)

**Library Fields**:
- Path (full path to .so file)
- Name (filename)

### 9. Symlink Management (`pkg/symlink`)

**Purpose**: Create two-tier symlink structure

**Symlink Structure**:
```
/usr/bin/vim              → /kod/wrappers/vim (symlink)
/kod/wrappers/vim         → /kod/store/vim/9.0.000/usr/bin/vim (symlink)
/kod/store/vim/9.0.000/usr/bin/vim (actual binary)
```

**Operations**:
- `NewManager(config *Config)` - Create symlink manager
- `CreateSymlinks(packagePath, packageName, version string) error` - Create all symlinks
- `RemoveSymlinks(packageName string) error` - Remove all symlinks for package
- `VerifySymlinks(packageName string) (bool, error)` - Verify symlink integrity
- `GetSymlinkPath(packageName, relativePath string) string` - Get symlink target path
- `GetStorePath(packageName, version, relativePath string) string` - Get actual file path

**Features**:
- Handle existing files (replace or skip)
- Atomic symlink creation
- Cleanup on removal
- Verification of symlink chain

### 10. Package Removal (`internal/cli/remove`)

**Purpose**: Remove installed packages and clean up

**Operations**:
- `RemovePackage(packageName string, force bool) error` - Remove package
- Remove symlinks from symlink_dir
- Remove wrapper scripts
- Remove package from store
- Update registry
- Cleanup orphaned libraries

**Force Flag Behavior**:
- Without `--force`: Verify symlinks exist before removal
- With `--force`: Remove even if symlinks don't exist

## CLI Commands

### `pith sync`
Sync Arch Linux package databases from mirrors.

**Options**:
- No options required
- Downloads core, extra, community databases

### `pith install <package> [package2] ...`
Install one or more packages with all dependencies.

**Options**:
- `--force` Force installation even if package exists
- `--no-deps` Skip dependency resolution

**Steps**:
1. Resolve dependencies
2. Download all packages
3. Extract each package to store
4. Generate wrapper scripts
5. Create symlinks
6. Update registry

### `pith remove <package> [package2] ...`
Remove installed packages.

**Options**:
- `--force` Force removal even if symlinks don't exist

**Steps**:
1. Verify package exists in registry
2. Remove symlinks from symlink_dir
3. Remove wrapper scripts
4. Remove package from store
5. Update registry

### `pith list`
List all installed packages.

**Output**: Table format with name, version, size, install date

### `pith search <pattern>`
Search for packages in repositories.

**Output**: List of matching packages with version and description

### `pith info <package>`
Show detailed information about a package.

**Output**: All PackageInfo fields in formatted output

### `pith upgrade`
Upgrade all installed packages to latest versions.

**Steps**:
1. Sync databases
2. Check for updates
3. Download new versions
4. Replace in store
5. Update registry

### `pith cleanup`
Remove old package versions and unused libraries.

**Operations**:
- Remove versions older than configured threshold
- Remove orphaned libraries not used by any package

## Dependency Resolution Algorithm

```
function ResolveDependencies(packageName):
    seen = empty set
    result = empty list
    
    function Resolve(pkg):
        if pkg.name in seen:
            return
        
        seen.add(pkg.name)
        
        for dep in pkg.depends:
            depPkg = SearchPackage(dep.name)
            if depPkg == null:
                depPkg = FindProviding(dep.name)
            
            Resolve(depPkg)
        
        result.append(pkg.name)
    
    Resolve(SearchPackage(packageName))
    return result
```

## Configuration File (`/etc/pistacho/config.json`)

```json
{
  "base_dir": "/kod",
  "db_dir": "/kod/db",
  "store_dir": "/kod/store",
  "wrappers_dir": "/kod/wrappers",
  "symlink_dir": "/usr/bin",
  "registry_file": "/kod/registry.json",
  "mirror_url": "https://geo.mirror.pkgbuild.com",
  "arch": "x86_64",
  "repos": ["core", "extra", "community"],
  "concurrency": 4,
  "keep_versions": 2,
  "auto_cleanup": false
}
```

## Error Handling

### Common Errors
- `PackageNotFound` - Package not found in registry or repository
- `DependencyConflict` - Dependency resolution failed
- `DownloadFailed` - Package download failed after retries
- `ExtractionFailed` - Package extraction failed
- `SymlinkError` - Symlink creation/removal failed
- `RegistryError` - Registry file corrupted or inaccessible

### Recovery
- Atomic operations (temp files, then rename)
- Registry updates after successful operations
- Rollback on failure for multi-package installs

## Implementation Notes

### Pure Go Implementation

Pistacho uses a **pure Go implementation** of libalpm functionality (`pkg/alpm/`), eliminating the need for external C libraries and CGO compilation. This provides several benefits:

- **Zero system dependencies**: No need to install libalpm-dev or libarchive-dev
- **Better portability**: Cross-compilation to different architectures without CGO hassles
- **Single binary**: Fully static builds possible for CI/CD environments
- **Simpler deployment**: No runtime dependency on host system packages

**Key Components**:
- `version.go`: RPM version comparison algorithm (epoch, release, revision handling)
- `parse.go`: Arch database tar.gz parser with metadata extraction
- `db.go`: Database API and cache management
- `deps.go`: Recursive dependency resolution with cycle detection
- `gpg.go`: Optional GPG signature verification using system `gpg` binary

**Architecture**:
- In-memory database caching for performance
- Repository precedence for conflict resolution (core > extra > community > multilib)
- Architecture filtering (x86_64, aarch64, any)
- Latest version selection across all repositories

See `pkg/alpm/README.md` for detailed API documentation and usage examples.

## Performance Considerations

- **Concurrent downloads**: Configurable concurrency limit
- **Package caching**: Store downloaded packages in store
- **Database caching**: Keep databases synced locally
- **Lazy symlink creation**: Create symlinks only for requested binaries

## Security Considerations

- **Signature verification**: Verify package signatures (optional)
- **Checksum validation**: Verify archive integrity on download
- **Isolated execution**: All packages run with LD_LIBRARY_PATH isolation
- **No host contamination**: Host libraries never mixed with package libraries

## Filesystem Requirements

- **Supported filesystems**: ext4, xfs, btrfs, any POSIX filesystem
- **Symlink support**: Required for two-tier symlink structure
- **Disk space**: 2-3x package size (due to isolation)
- **Permissions**: Write access to base_dir and symlink_dir

## Cross-Distribution Compatibility

**Supported Distributions**:
- Ubuntu 20.04+
- Debian 11+
- Fedora 35+
- Any systemd-based distribution with glibc

**Requirements**:
- glibc (for running binaries)
- zstd (for decompression, or use Go's decompressor)
- gpg (optional, for signature verification)

## Known Limitations and Edge Cases

### ALPM Implementation

#### 1. Database Format Handling

**Issue:** Go's `http.Client` auto-decompresses based on `Content-Encoding` header.

**Behavior:** 
- Arch mirrors serve `.db` files with `Content-Encoding: x-gzip`
- Some HTTP clients decompress automatically
- The pure Go parser auto-detects and handles both formats

**Workaround:** Already implemented - parsePackageDatabase checks for gzip magic bytes (1f 8b)

#### 2. GPG Signature Verification

**Current:** Calls system `gpg` command via wrapper

**Limitation:** Requires GPG to be installed on the system

**Security Note:** For production use, consider implementing pure Go GPG verification

#### 3. Package File Listings

**Not Supported:** `.files` database files are not parsed

**Impact:** Cannot query complete file listings for packages

**Workaround:** Store file listings separately or fetch from package metadata

#### 4. Local Database Queries

**Not Supported:** Only sync databases are supported, not local installed packages

**Use Case:** Current implementation focuses on remote package resolution

**Future:** Could implement local database support if needed

#### 5. Mirror Fallback

**Not Implemented:** No automatic failover to alternate mirrors

**Workaround:** Implement at application level:
```go
mirrors := []string{
    "https://mirror.rackspace.com/archlinux",
    "https://mirrors.kernel.org/archlinux",
}
// Try each mirror in sequence
```

#### 6. Virtual Package Resolution

**Supported:** Virtual packages (those provided by multiple packages)

**Algorithm:** 
- Finds all packages providing the virtual package
- Selects first match in database order
- May not match pacman's heuristics exactly

**Note:** Complex provider selection edge cases may behave differently

### Dependency Resolution Edge Cases

#### 1. Alternative Dependencies (OR Dependencies)

**Format:** `package1|package2`

**Behavior:** Selects first available package

**Note:** Different from pacman's package selection heuristics

#### 2. Circular Dependencies

**Detected:** Yes, prevents infinite loops

**Behavior:** Returns error when circular dependency detected

```
package A depends on B
package B depends on C
package C depends on A  // <- Circular, error returned
```

#### 3. Missing Dependencies

**Behavior:** Returns error with missing package name

**No Automatic Resolution:** Cannot install packages with missing dependencies

#### 4. Optional Dependencies

**Behavior:** Included in dependency list for information

**Not Required:** Installation can proceed without them

#### 5. Version Constraints

**Supported Formats:**
- `=` exact version
- `>=` greater than or equal
- `<=` less than or equal
- `>` greater than
- `<` less than

**Known Issue:** Complex constraints like `>=1.0 <2.0` not fully supported

**Workaround:** Current implementation checks last constraint only

### Performance Edge Cases

#### 1. Large Dependency Trees

**Performance:** O(n) where n = number of packages

**Tested With:** core (286 packages) + extra (14,082 packages)

**Typical Resolution Time:** < 100ms for most packages

**Worst Case:** Complex dependencies may take 200-300ms

#### 2. In-Memory Cache Size

**Memory Usage:** ~20-50MB for core + extra databases

**Scaling:** Linear with package count

**Note:** No cache eviction - all packages loaded until client closes

#### 3. First Load Latency

**Initial Parse:** 100-200ms per database

**Cached Access:** < 1ms for subsequent queries

#### 4. Search Performance

**Exact Match:** O(1) hash lookup - < 1ms

**Pattern Match:** O(n) linear scan - < 50ms for extra database

### Version Comparison Edge Cases

#### 1. Epoch Handling

**Format:** `[epoch:]version[-release]`

**Behavior:** Epochs always take precedence

```
2:0.9 > 1:10.0  // True - epoch 2 > epoch 1
0.9 < 10.0      // True - no epochs compared numerically
```

#### 2. Pre-release Versions

**Format:** Version strings with letters mixed with numbers

**Behavior:** Alphabetic characters get special sorting

```
1.0_beta < 1.0_rc < 1.0 < 1.0.1
```

#### 3. Separator Handling

**Supported:** Hyphens, underscores, dots as separators

**Algorithm:** Each segment compared separately

```
1.2.3 > 1.2_3   // False - different separator, same comparison
```

### Concurrency Limitations

#### 1. Thread Safety

**Current:** Not thread-safe

**Note:** Client should be used from single goroutine

**Workaround:** Synchronize access or create per-goroutine clients

#### 2. Shared Cache

**Issue:** In-memory cache not synchronized

**Solution:** Create separate client instances for concurrent access

### Arch-Specific Limitations

#### 1. Architecture Filtering

**Implemented:** x86_64, aarch64, any

**Behavior:** Filters packages by architecture match

**Issue:** Some packages may claim multiple architectures - only supports single

#### 2. Repository Precedence

**Implemented:** First registered = highest priority

**Behavior:** When package exists in multiple repos, first match is used

**Limitation:** Cannot query all matches, only first

#### 3. Split Packages

**Not Supported:** Pkgbase with multiple split packages

**Current:** Treats each as independent package

**Impact:** May miss relationships between related packages

### Integration Points

#### 1. Database Sync

**Dependency:** External `database.Syncer`

**Assumption:** Databases available at expected path

**Error Handling:** Returns error if database not found

#### 2. GPG Verification

**Dependency:** System `gpg` command

**Requirement:** Must be in PATH

**Fallback:** Can be disabled in configuration

#### 3. Download Manager

**Dependency:** HTTP client for package download

**Assumption:** Mirror URLs are valid and accessible

**Timeout:** Configurable per package

## Testing Coverage

### Unit Tests: 42+ tests
- Version comparison (20+ tests)
- Dependency resolution (9+ tests)
- Database parsing (14+ tests)

### Integration Tests: 1 comprehensive test
- Full workflow from database sync to package search
- Real Arch databases (core + extra)
- Registry operations

### Known Test Gaps
- GPG verification (system-dependent)
- Mirror fallback scenarios
- Extreme version numbers
- Very large dependency trees (1000+ packages)

## Future Improvements

### High Priority
1. Thread-safe concurrent access
2. Pure Go GPG verification
3. Cache eviction/TTL
4. Mirror failover support

### Medium Priority
1. Local database support
2. Package file listing support
3. Improved virtual package selection
4. Complex version constraint support

### Low Priority
1. Performance optimization (benchmarking)
2. Support for split packages
3. Pkgbuild execution
4. Custom package sources


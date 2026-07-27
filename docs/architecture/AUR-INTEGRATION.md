# AUR Integration

Complete guide for Arch User Repository (AUR) integration in Chisel, including architecture, usage, and implementation details.

---

## Overview

Chisel supports installing packages from the Arch User Repository (AUR) in addition to official Arch repositories. AUR packages are built from source using `makepkg` and integrated seamlessly with official packages in Chisel's store and registry.

### Key Features

- **Automatic AUR Fallback**: Official repositories checked first; AUR used as fallback
- **Mixed Dependency Resolution**: Automatic resolution across both official and AUR sources
- **Build Management**: Integrated build system using `makepkg`
- **Version Tracking**: Track package source (official vs. AUR) and version history
- **Transparent Operation**: Works with existing commands (install, search, upgrade)
- **Build Caching**: Persistent build cache and logs for efficiency

---

## Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────────┐
│                    CLI Layer (internal/cli)               │
│  InstallCommand, SearchCommand, UpgradeCommand, etc.      │
└────────────┬────────────────────────────────────────────┘
             │
┌────────────▼────────────────────────────────────────────┐
│              Build & Resolution (pkg/build)               │
│  ├─ MixedResolver: Dependency across sources             │
│  └─ BuildManager: makepkg execution & caching            │
└────────────┬────────────────────────────────────────────┘
             │
┌────────────▼────────────────────────────────────────────┐
│            AUR Client & Git (pkg/aur)                     │
│  ├─ RPC Client: AUR API v5 communication                 │
│  ├─ Git Handler: PKGBUILD repository cloning             │
│  └─ PKGBUILD Parser: Metadata extraction                 │
└────────────┬────────────────────────────────────────────┘
             │
┌────────────▼────────────────────────────────────────────┐
│         Package Storage & Registry (pkg/*)                │
│  Standard install/extract/symlink/registry flow           │
└─────────────────────────────────────────────────────────┘
```

### Module Structure

```
pkg/aur/
  ├─ types.go              # AUR package types
  ├─ rpc.go                # AUR API v5 client
  ├─ git.go                # PKGBUILD download via git
  ├─ pkgbuild.go           # PKGBUILD parsing
  └─ *_test.go             # Tests

pkg/build/
  ├─ resolver.go           # Mixed repo dependency resolver
  ├─ builder.go            # makepkg build execution
  └─ *_test.go             # Tests
```

---

## AUR Module (`pkg/aur/`)

### RPC Client

**File**: `pkg/aur/rpc.go`

Communicates with official AUR RPC interface (API v5).

**Features**:
- Caches results with 24-hour TTL to reduce API calls
- Implements rate limiting (4000 requests per day maximum)
- Handles network errors and timeouts gracefully
- Supports multi-package queries

**API Methods**:
```go
type Client interface {
    // Search for packages by name/keyword
    Search(query string) ([]*Package, error)
    
    // Get detailed info for single package
    Info(packageName string) (*Package, error)
    
    // Get info for multiple packages
    MultiInfo(packageNames []string) ([]*Package, error)
}
```

**AUR Package Metadata**:
```go
type Package struct {
    Name            string    // Package name
    Version         string    // Current version
    Description     string    // Short description
    URL             string    // Project homepage
    Maintainer      string    // AUR maintainer
    OutOfDate       bool      // Flagged as outdated?
    SubmittedTime   int64     // Submission timestamp
    ModifiedTime    int64     // Last modification timestamp
    DependsOn       []string  // Dependencies
    MakeDepends     []string  // Build-time dependencies
    OptDepends      []string  // Optional dependencies
    Conflicts       []string  // Conflicting packages
    Provides        []string  // Virtual packages provided
    Keywords        []string  // Search keywords
}
```

### Git Handler

**File**: `pkg/aur/git.go`

Manages PKGBUILD repository operations.

**Operations**:
- Clone AUR repositories to build cache
- Verify repository integrity
- Update existing clones for rebuilds

**Directories**:
- Build directory: `/kod/build-cache/{package-name}/`
- Log directory: `/kod/build-logs/`

### PKGBUILD Parser

**File**: `pkg/aur/pkgbuild.go`

Extracts metadata from PKGBUILD files.

**Capabilities**:
- Parse bash array syntax
- Extract dependencies, build dependencies
- Handle conditional logic
- Identify executable files
- Extract version and release information

**Example PKGBUILD**:
```bash
pkgname=mypackage
pkgver=1.0.0
pkgrel=1
pkgdesc="My awesome package"
depends=('glibc' 'ncurses')
makedepends=('gcc' 'make')
build() {
    cd "$pkgname-$pkgver"
    ./configure
    make
}
```

---

## Build System (`pkg/build/`)

### BuildManager

**File**: `pkg/build/builder.go`

Orchestrates AUR package compilation.

**Operations**:

1. **Build Execution**
   - Clone PKGBUILD from AUR git
   - Execute `makepkg` with safe defaults
   - Output: `.pkg.tar.zst` file in cache

2. **Cache Management**
   - Persist build directory between builds
   - Reuse cached sources and partial builds
   - Location: `/kod/build-cache/`

3. **Log Management**
   - Save all build output to logs
   - Location: `/kod/build-logs/`
   - Delete logs on cleanup (optional)

**Build Flow**:
```
BuildManager.Build(packageName)
  ├─ Check if already built and cached
  ├─ If not cached:
  │  ├─ Clone PKGBUILD from AUR
  │  ├─ Parse PKGBUILD for dependencies
  │  ├─ Resolve all build-time dependencies
  │  ├─ Execute: makepkg -s -r -C
  │  │  (-s: install deps, -r: remove deps after, -C: clean)
  │  └─ Save output to /kod/cache/{name}-{version}-{arch}.pkg.tar.zst
  └─ Return path to built package
```

**makepkg Options**:
- `-s` (--syncdeps): Install missing dependencies
- `-r` (--rmdeps): Remove dependencies after build
- `-C` (--clean): Remove build directory after
- `-i` (--install): Install after build (disabled, we handle install)

### MixedResolver

**File**: `pkg/build/resolver.go`

Resolves dependencies across both official and AUR sources.

**Resolution Priority**:
1. Check official repositories first (core > extra > community)
2. Fall back to AUR if not found in official
3. Detect circular dependencies
4. Build dependency chain in correct order

**Key Features**:
- Handles both build-time (`makedepends`) and runtime (`depends`) dependencies
- Tracks which packages from which source
- Returns ordered list for installation

**Resolution Algorithm**:
```
ResolveDependencies(packageName, includeAUR=true)
  ├─ Check official repos: ALPM cache
  │  └─ If found, use official version
  └─ If not found, check AUR: RPC client
     └─ If found, mark for building
     └─ Recursively resolve AUR dependencies
```

---

## Usage Examples

### Search AUR

```bash
# Search AUR for packages
pith search --aur "programming"

# Show AUR-only packages
pith search --aur --exact vim
```

### Install from AUR

```bash
# Install an AUR package (builds from source)
sudo pith install --aur vim-plug

# Install with build dependencies
sudo pith install --aur --build-deps vim-plug

# Force rebuild even if cached
sudo pith install --aur --rebuild vim-plug
```

### Get Package Info

```bash
# Get AUR package info
chisel info --aur vim-plug

# Compare official vs AUR
chisel info vim       # Official version
chisel info --aur vim # AUR version (if available)
```

### Upgrade with AUR

```bash
# Upgrade all packages (checks both sources)
sudo pith upgrade

# Upgrade only official packages
sudo pith upgrade --no-aur

# Rebuild outdated AUR packages
sudo pith upgrade --aur --rebuild-outdated
```

### List and Cleanup

```bash
# List installed packages, showing source
pith list --verbose

# Clean AUR build cache
pith cleanup --aur-cache

# Remove build logs
pith cleanup --aur-logs
```

---

## Integration with Existing Flow

### Modified Installation Flow

```
install --aur package-name
  │
  ├─ [NEW] Check if AUR package
  │  ├─ If AUR: Resolve build dependencies
  │  ├─ Build with makepkg (outputs .pkg.tar.zst)
  │  └─ Move to cache
  │
  ├─ [STANDARD] Dependency Resolution
  │  ├─ Use MixedResolver (official + AUR)
  │  └─ Build installation order
  │
  ├─ [STANDARD] Download packages
  ├─ [STANDARD] Extract to store
  ├─ [STANDARD] Create symlinks
  ├─ [STANDARD] Generate wrappers
  └─ [MODIFIED] Update registry with source
```

### Registry Extension

Registry tracks AUR packages:

```json
{
  "vim-plug": {
    "name": "vim-plug",
    "version": "1.13.0-1",
    "source": "aur",
    "repository": "aur",
    "files": [...],
    "executables": [],
    "dependencies": ["vim"],
    "install_date": "2024-01-15T10:30:00Z"
  }
}
```

---

## Implementation Details

### New CLI Flags

**--aur**: Include AUR in search/install operations
```bash
pith install --aur package-name
pith search --aur pattern
```

**--rebuild**: Force rebuild of AUR package
```bash
pith install --aur --rebuild package-name
```

**--build-deps**: Install build dependencies (usually automatic)
```bash
pith install --aur --build-deps package-name
```

### Build Environment

**Isolated Build**:
- Build occurs in `/kod/build-cache/{package}/`
- Dependencies mounted from `/kod/store/`
- No pollution of host system
- Build artifacts cleaned after

**Environment Variables**:
```bash
SRCDEST=/kod/build-cache/{package}/src
PKGDEST=/kod/cache
LOGDEST=/kod/build-logs
```

### Error Handling

**Build Failures**:
- Capture stdout/stderr to log
- Return build error with helpful message
- Preserve build directory for debugging

**Missing Dependencies**:
- Detect unresolvable dependencies
- Return error listing missing packages
- Suggest official package as alternative

**Version Conflicts**:
- Detect conflicting version constraints
- Report which packages conflict
- Suggest version constraint modification

---

## Performance Considerations

### Build Caching

**First Build**: Slower (download sources, compile)
```
pith install --aur package-name  [Slow first time]
```

**Incremental Build**: Faster (reuse sources if PKGBUILD unchanged)
```
pith install --aur --rebuild package-name  [Faster with cached sources]
```

**Cache Location**: `/kod/build-cache/`

### Dependency Caching

**AUR RPC Cache**: 24-hour TTL on queries
- Reduces API calls
- Speeds up repeated searches
- Respects "out of date" flags

**Registry Integration**: Tracks installed AUR versions
- Quick lookup for already-installed packages
- Speeds up upgrade checks

---

## Limitations

### Current

1. **No Signature Verification**: Doesn't verify PKGBUILD signatures
2. **No Sandboxing**: User must trust PKGBUILD content
3. **Single Build**: One package building at a time
4. **No Binary Cache**: Every install rebuilds from source

### Potential Future

1. **Binary Cache Repository**: Pre-built AUR packages
2. **Signature Verification**: GPG signing for PKGBUILDs
3. **Parallel Builds**: Multiple packages building concurrently
4. **Custom Patches**: Allow local PKGBUILD modifications

---

## Troubleshooting

### Build Failure

1. Check build logs: `/kod/build-logs/{package}.log`
2. Review PKGBUILD for errors
3. Check dependency resolution
4. Try clean rebuild: `pith install --aur --rebuild package`

### Dependency Issues

1. Use `chisel info --aur package` to see dependencies
2. Check if dependencies are available in official repos
3. May need to install AUR dependencies first

### Version Conflicts

1. Check official version: `chisel info package`
2. Check AUR version: `chisel info --aur package`
3. Resolve manually or use official if available

---

## See Also

- [ARCHITECTURE.md](../architecture/ARCHITECTURE.md) - Complete system architecture
- [REGISTRY.md](REGISTRY.md) - Registry reference
- [USER-GUIDE.md](../user-guides/USER-GUIDE.md) - User guide
- User-level installation guide in `/docs/user-guides/`

# Pistacho Features

This document provides an overview of Chisel's key features.

## Core Features

### Package Management
- ✅ **Complete Arch package support** - All 12,000+ packages from Arch repositories
- ✅ **Dependency resolution** - Automatically installs all required dependencies
- ✅ **Package groups** - Install entire collections like GNOME, KDE, or development tools
- ✅ **Package upgrades** - Keep packages up-to-date with `pith upgrade`
- ✅ **Package removal** - Clean uninstallation with dependency checking
- ✅ **Package cleanup** - Remove old versions to save disk space

### Installation Modes
- ✅ **System-level** - Install with `sudo` for system-wide access
- ✅ **User-level** - Install without `sudo` using `~/.local` directory
- ✅ **Chroot mode** - Install to isolated chroot environments or containers
- ✅ **No-extract mode** - Skip extraction if packages already extracted

### Package Discovery
- ✅ **Search packages** - Find packages by name or pattern
- ✅ **Package info** - View detailed package metadata
- ✅ **Dependency tree** - Show complete dependency graphs
- ✅ **Group browsing** - List and explore package groups

### Database Management
- ✅ **Database sync** - Download Arch package databases from mirrors
- ✅ **Mirror selection** - Choose alternative Arch mirrors
- ✅ **Cache management** - Clean downloaded packages to free disk space

### Registry & Tracking
- ✅ **Package registry** - Track all installed packages and versions
- ✅ **Version tracking** - Know exactly which version is installed
- ✅ **Installation metadata** - Record install date, source, and dependencies

## Advanced Features

### Install Scripts Support ⭐ *NEW*

Automatically execute post-install and post-upgrade scripts for packages that need additional configuration.

**Dual-mode execution:**
- **Non-chroot mode** - Execute scripts directly in current system context
- **Chroot mode** - Execute scripts in isolated chroot environment

**Example usage:**
```bash
# Auto-execution during install
pith install bash

# Manual execution later
pith install-scripts bash

# Chroot-based execution
pith install --chroot /tmp/chroot bash
pith install-scripts --chroot /tmp/chroot bash
```

**Why it matters:**
- Some packages need post-install setup (bash, glibc, systemd, etc.)
- Scripts run after extraction and symlinks are created
- Automatic operation detection (post_install vs post_upgrade)
- Registry tracks which packages have install scripts
- Non-blocking execution - failures don't stop other packages

[Full documentation](docs/user-guides/INSTALL-SCRIPTS.md)

### Wrapper Scripts

Automatically generate wrapper scripts for executables that set up the runtime environment:

```bash
# When you install vim:
pith install vim

# A wrapper script is created that:
# 1. Sets LD_LIBRARY_PATH to Arch libraries
# 2. Executes the Arch vim binary
# 3. Cleans up environment afterward

# Result: vim works perfectly with Arch dependencies
vim myfile.txt
```

### Symlink Management
- ✅ **Automatic symlink creation** - Install packages to `/kod/store` and symlink to `/usr/local`
- ✅ **Custom symlink directories** - Use `--symlink-dir` for custom locations
- ✅ **Force overwrite** - Use `--force` to overwrite existing symlinks
- ✅ **Chroot-compatible symlinks** - Strip prefixes for container portability

### AUR Support
- ✅ **Build from source** - Install packages from Arch User Repository
- ✅ **Dependency handling** - Resolve AUR package dependencies
- ✅ **Source control** - Clone from Git if needed
- ✅ **Build caching** - Cache build artifacts to speed up rebuilds

## Configuration

### Environment Variables
- `CHISEL_BASE_DIR` - Override base directory
- `CHISEL_CONFIG` - Override config file path
- `CHISEL_SYMLINK_DIR` - Override symlink directory
- `CHISEL_USER_BASE_DIR` - User-level installation directory

### Configuration File
```json
{
  "baseDir": "/kod",
  "mirror": "https://mirrors.kernel.org/archlinux",
  "storeRoot": "/kod/store",
  "wrapperDir": "/kod/wrappers",
  "symlink_root": "/usr/local",
  "databaseRoot": "/kod/db",
  "cachePath": "/kod/cache",
  "registryPath": "/kod/registry.json"
}
```

## Performance Features

### Caching
- ✅ **Download cache** - Reuse already-downloaded packages
- ✅ **Build cache** - Cache AUR build results
- ✅ **Database cache** - Cache synced package databases

### Parallel Operations
- ✅ **Concurrent downloads** - Download multiple packages simultaneously
- ✅ **Dependency resolution optimization** - Fast graph traversal
- ✅ **Bulk operations** - Install multiple packages efficiently

## Reliability Features

### Error Handling
- ✅ **Non-blocking failures** - If one script fails, others continue
- ✅ **Rollback support** - Remove packages if installation fails
- ✅ **Dry-run mode** - Preview operations without making changes
- ✅ **Verbose logging** - Detailed execution information

### Data Integrity
- ✅ **Registry consistency** - Track package state accurately
- ✅ **Version tracking** - Know exactly what's installed
- ✅ **Dependency verification** - Ensure no broken installations
- ✅ **Symlink validation** - Verify symlinks point to correct locations

## Quality & Testing

### Test Coverage
- ✅ **Unit tests** - Comprehensive test coverage
- ✅ **Integration tests** - End-to-end workflow testing
- ✅ **Command tests** - All CLI commands tested
- ✅ **Registry tests** - Package tracking validated

### Documentation
- ✅ **User guides** - Step-by-step instructions
- ✅ **Command reference** - Complete CLI documentation
- ✅ **Architecture docs** - System design and flows
- ✅ **Examples** - Real-world usage scenarios

## Compatibility

### Operating Systems
- ✅ Ubuntu 22.04 LTS, 24.04
- ✅ Debian 12, 13
- ✅ Fedora 39, 40
- ✅ Other systemd-based distributions
- ✅ WSL2 on Windows
- ✅ Container environments

### Architectures
- ✅ x86_64 (primary)
- ✅ ARM64 (experimental)

## Roadmap & Future Features

### Planned
- [ ] Generation-based rollback (immutable snapshots)
- [ ] Enhanced AUR integration
- [ ] Binary cache support
- [ ] Additional architecture support (i686, ARM)

## Summary

Chisel provides a complete package management solution for installing Arch Linux packages on any Linux distribution with:

1. **Complete dependency isolation** - No host contamination
2. **Flexible installation modes** - System, user, or chroot-based
3. **Automatic configuration** - Scripts run when needed
4. **Easy management** - Install, upgrade, remove packages
5. **No learning curve** - Familiar Arch package management experience

All wrapped in a simple, reliable, and well-tested CLI tool.

For detailed information, see the [documentation](docs/INDEX.md).

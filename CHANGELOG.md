# Changelog

All notable changes to Pith will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.4] - 2026-08-11

### Fixed
- **Conflicting symlinks**: `install` now replaces an existing symlink that points elsewhere instead of skipping it, so the correct package file is linked

## [0.4.3] - 2026-08-06

### Fixed
- **Architecture detection**: Use `runtime.GOARCH` instead of hardcoded `x86_64` for system architecture detection
- **Per-package architecture in download URLs**: Download URLs now use each package's architecture from the database `%ARCH%` field (e.g. `any`, `x86_64`, `aarch64`) instead of the system architecture for the URL path and filename
  - `any` packages download from `/os/any/` with `-any` suffix (e.g. `ca-certificates-20240618-1-any.pkg.tar.zst`)
  - `aarch64` packages download from `/os/aarch64/` with `-aarch64` suffix
  - Falls back to `x86_64` if architecture is not specified

## [0.4.2] - 2026-06-18

### Fixed
- **Symlink creation with pool outside chroot**: Allow symlink creation when package pool directory is located outside the chroot environment

### Changed
- **Default base-dir**: Changed default base directory from `/kod` to current working directory for easier local testing

## [0.4.1] - 2026-06-12

### Fixed
- **PoolRoot path**: Corrected PoolRoot path to use `pool` directory instead of `store`

### Changed
- **Project rebranding**: Completed renaming from `chisel` to `pistacho` and CLI binary from `chisel` to `pith`
  - Updated all internal references, configuration, and release workflows
  - Removed archived `/old/` directory with legacy documentation

## [0.4.0] - 2026-06-10

### Added
- **Project rename to pistacho**: Renamed project from `chisel` to `pistacho` and CLI binary from `chisel` to `pith`
  - Updated all internal references, configuration paths, and release workflows
  - Renamed user-level scripts from `chisel-user` to `pith-user`

### Changed
- **Project branding**: Updated CHANGELOG, README, and all documentation to reflect new project name

## [0.3.2] - 2026-05-15

### Added
- **Package storage migration**: Migrated package storage from `store` to `pool` directory structure
  - New directory layout: `{baseDir}/pool/{pkgName}/{version}/`
  - Improved organization and separation of concerns

- **Install scripts support**: Added support for executing post_install/post_upgrade scripts from Arch packages
  - `pith install-scripts` command for executing scripts in local or chroot context
  - Automatic detection of `.INSTALL` files in extracted packages
  - Proper handling of `post_install` vs `post_upgrade` operations

- **Chroot flag enhancements**: Improved `--chroot` flag behavior
  - Renamed `--symlink-prefix` to `--chroot` for clarity
  - Auto-set `--base-dir` to `{chroot}/kod` when `--chroot` is specified
  - Create symlinks directly to installed files in chroot mode

### Fixed
- **Verbose output**: Fixed various output formatting issues
- **Install scripts command**: Fixed command execution and parameter handling

## [0.3.1] - 2026-05-08

### Fixed
- **Optional dependencies display**: Strip descriptions from optional dependencies for cleaner output
  - Example: `aspell: spelling corrections` → `aspell`
- **CI workflow**: Updated test workflow to use valid `ubuntu-latest` runner
- **README accuracy**: Updated README to accurately reflect v0.3.0 features

## [0.3.0] - 2026-04-06

### Added
- **Package Group Support**: Install entire package collections with a single command
  - `pith-user install gnome` to install all GNOME desktop packages
  - `pith-user search --groups` to list all available package groups
  - `pith-user search --group gnome` to view packages in a specific group
  - Support for mixed group and individual package installation
  - Automatic dependency resolution for all packages in groups

- **Database Parsing Enhancement**:
  - Added `%GROUPS%` metadata parsing from Arch package databases
  - Group data indexed for fast lookups and installation
  - One-to-many package-to-groups relationships supported

- **Search Command Enhancements**:
  - New `--groups` flag to list all available package groups with package counts
  - New `--group <name>` flag to search and display packages within a group

- **Enhanced User Documentation**:
  - New "Installing Package Groups" section in USER-GUIDE.md
  - Examples for common use cases: desktop environments, development tools
  - Workflow examples showing how to discover and install groups

### Changed
- Group expansion now happens before dependency resolution in install command
- Clearer visual feedback during installation showing which packages are from groups

## [0.2.0] - 2026-04-06

### Added
- **Symlink-Prefix Stripping for Chroot Support**: New `--chroot` flag enables running packages inside chroot environments
  - Strip path prefixes from symlinks, wrapper scripts, and command paths
  - Support both equals-separated (`--chroot=/tmp/demo`) and space-separated (`--chroot /tmp/demo`) flag syntax
  - Enable package portability across different mount points and containerized environments
  - Comprehensive documentation with use cases and examples

- **Generation-Based Registry Management Specification**: Future architecture design for Kodos integration
  - Per-user registry management with immutable generation snapshots
  - Planned full rollback capabilities and audit trail
  - Specification document for future implementation phases

- **Enhanced User Documentation**:
  - New "Chroot Support with Symlink Prefix Stripping" section in USER-GUIDE.md
  - Troubleshooting guide for symlink prefix issues
  - Advanced usage examples combining flags

### Fixed
- **Feature Branch Cleanup**: Removed outdated `feature/chroot-stripping` branch after merging into main

### Changed
- **Repository Management**: Removed community repository from default configuration

## [0.1.0] - 2026-04-06

### Added
- **AUR Package Support**: Full support for building and installing AUR (Arch User Repository) packages
  - Automatic dependency resolution for mixed official/AUR packages
  - Real-time compilation output streaming during AUR package builds
  - Copy progress indication with transfer speed and percentage display
  - Extraction progress indication for all packages
  - Seamless integration with official package manager

- **Verbose Installation Output**: Comprehensive progress reporting during installation
  - Stream makepkg output to console in real-time
  - Display copy progress with file size and transfer speed
  - Show extraction progress for each package
  - Clear indication of build stages and file extraction counts

- **Mixed Resolver**: New dependency resolution system that handles both official and AUR packages
  - Automatic detection of package source (official repo vs AUR)
  - Proper ordering of dependencies considering source constraints
  - Support for `--source=official` and `--source=aur` flags to restrict package sources

- **Core package installation functionality**
- **Official Arch Linux repository support**
- **Basic dependency resolution**
- **User-level package management**
- **Symlink creation for package binaries**
- **Registry tracking for installed packages**

### Fixed
- **Database Parser**: Handle package versions with epoch (e.g., `1:1.3.2-3`)
  - Changed key separator from `:` to null byte (`\x00`) to avoid conflicts with epoch in version strings
  - All 286+ core packages now load correctly

- **Repository Field Population**: Fixed missing Repository field in parsed packages
  - Repository name now properly populated for all packages
  - Download URLs now correctly formed: `https://mirror/.../archlinux/core/os/...`

- **Build System**: Improved file copying for AUR builds
  - Replaced shell `cp -r` with Go's `filepath.Walk` for direct file copying
  - Prevents creation of unwanted subdirectories in build cache
  - More portable across different systems

- **AUR Build Integration**: Built packages now properly copied to cache
  - Artifacts from build cache transferred to user cache for extraction
  - Prevents redundant rebuilds on subsequent installations
  - Enables installation of large AUR packages (tested with 1.1GB lmstudio-bin)

### Changed
- **Installation Flow**: Separated AUR and official package handling
  - AUR packages built before downloading official packages
  - Better organization and clearer progress reporting
  - Improved error handling with specific error messages per package

- **GitHub Actions**: Updated to Node.js 24 compatible versions
  - Updated `actions/checkout` from v6 to v4
  - Updated `actions/setup-go` from v6 to v5
  - Updated Go version from 1.25 to 1.26

### Security
- User-level package installation now uses secure paths
  - Cache stored in `~/.local/share/pith/cache/`
  - Build cache in `~/.local/share/pith/build-cache/`
  - Build logs in `~/.local/share/pith/build-logs/`

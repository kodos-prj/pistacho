# Pistacho User-Level Package Manager Guide

This guide explains how to use pith as a regular user to install and manage packages at the user level without requiring root/sudo access.

## Overview

By default, pith stores packages in `/kod/` which requires root access. This guide shows how to configure pith to use user-level directories following the XDG Base Directory specification, allowing any user to independently manage packages.

**Key Benefits:**
- ✅ No root/sudo required
- ✅ Packages isolated per user
- ✅ Follows XDG standards (~/.local, ~/.config)
- ✅ Easy to set up and tear down
- ✅ Compatible with all pith features

## Quick Start

### 1. Initialize User Environment

Run the setup script to automatically configure pith for user-level use:

```bash
./pith-user-init.sh
```

This script:
- Creates user directories (~/.local/share/chisel, ~/.config/chisel, etc.)
- Creates a user configuration file
- Updates your shell configuration (PATH, LD_LIBRARY_PATH, environment variables)

**Output:**
```
✓ Chisel binary found
✓ Creating user-level directories...
✓ Creating user-level configuration...
✓ Updating shell configuration...
```

### 2. Reload Your Shell

```bash
# For bash
source ~/.bashrc

# For zsh
source ~/.zshrc

# For fish
source ~/.config/fish/config.fish
```

### 3. Start Using Chisel

```bash
# Sync package databases
pith-user sync

# Search for a package
pith-user search vim

# Install a package
pith-user install nano

# List installed packages
pith-user list

# Remove a package
pith-user remove nano
```

## User-Level Directory Structure

When you run the setup script, pith creates this directory structure:

```
~/.local/share/chisel/              # Main data directory (CHISEL_USER_BASE_DIR)
├── store/                          # Extracted packages
├── db/                             # Synced package databases
├── wrappers/                       # Package wrapper scripts
├── cache/                          # Downloaded packages
└── registry.json                   # Installed packages registry

~/.config/chisel/                   # Configuration directory
└── config.json                     # User configuration

~/.local/bin/                       # User symlinks (in PATH)
```

## How It Works

### Configuration Priority

Chisel uses this priority order for configuration:

1. **Command-line flags** (`--base-dir`, `--config`)
2. **Environment variables** (`CHISEL_USER_BASE_DIR`, `CHISEL_CONFIG`, `CHISEL_BASE_DIR`)
3. **User config file** (`~/.config/chisel/config.json`)
4. **System config file** (`/etc/chisel/config.json`)
5. **Built-in defaults** (`/kod`, `/etc/chisel/config.json`)

### Using pith-user vs pith

**pith-user** (recommended for users):
- Automatically uses user-level configuration
- Seamless user experience
- Falls back to system config if needed

```bash
pith-user install nano
```

**chisel** (with environment variables):
- Full control
- Flexible for different use cases

```bash
export CHISEL_USER_BASE_DIR=~/.local/share/chisel
export CHISEL_CONFIG=~/.config/chisel/config.json
pith install nano
```

**Direct flag usage:**
```bash
chisel \
  --base-dir ~/.local/share/chisel \
  --config ~/.config/chisel/config.json \
  install nano
```

## Environment Variables

After setup, your shell will have:

```bash
# User-level base directory
export CHISEL_USER_BASE_DIR="$HOME/.local/share/chisel"

# Add user bin to PATH (for package executables)
export PATH="$PATH:$HOME/.local/bin"

# Add user lib to LD_LIBRARY_PATH (for package libraries)
export LD_LIBRARY_PATH="$HOME/.local/lib:$LD_LIBRARY_PATH"
```

You can override these:

```bash
# Use a different base directory
export CHISEL_USER_BASE_DIR=/path/to/packages

# Use a different symlink location
export CHISEL_SYMLINK_DIR=$HOME/custom/bin
```

## Common Operations

### Search and Install

```bash
# Search for packages
pith-user search vim

# Get detailed package information
pith-user info vim

# Install with all dependencies automatically resolved
pith-user install vim

# Install without extracting (if already in store)
pith-user install vim --no-extract
```

### Installing Package Groups

Chisel supports package groups, which are collections of related packages that you can install together. For example, the `gnome` group contains all GNOME desktop packages, `kde` contains KDE Plasma packages, and `base-devel` contains essential development tools.

#### List Available Groups

```bash
# See all available package groups
pith-user search --groups
```

This shows all available groups with the number of packages in each:
```
base (7 packages)
base-devel (24 packages)
development-tools (18 packages)
editors (12 packages)
gnome (85 packages)
kde (102 packages)
xfce (45 packages)
...
```

#### Search Packages in a Group

```bash
# See all packages in a specific group
pith-user search --group gnome

# Search for packages in the development tools group
pith-user search --group base-devel
```

#### Install a Package Group

```bash
# Install all packages from a group
pith-user install gnome

# Install all development tools
pith-user install base-devel

# Install multiple groups together
pith-user install gnome development-tools

# Mix groups with individual packages
pith-user install gnome vim curl
```

When installing a group, pith will:
1. Show you which packages are from the group
2. Resolve all dependencies automatically
3. Download and extract all packages
4. Create symlinks in ~/.local/bin

#### Group Examples

**Desktop Environments:**
```bash
# Install GNOME desktop
pith-user install gnome

# Install KDE Plasma
pith-user install kde

# Install XFCE desktop
pith-user install xfce
```

**Development Setup:**
```bash
# Install essential development tools
pith-user install base-devel development-tools editors

# This installs compilers, build tools, version control, and editors
```

**Terminal User:**
```bash
# Install common terminal applications
pith-user install base editors
```

### Chroot Support with Symlink Prefix Stripping

The `--chroot` flag enables running packages inside a chroot environment by stripping path prefixes from symlinks. This is useful for creating isolated package environments.

```bash
# Install with symlink prefix stripping
pith-user install vim --chroot=/tmp/demo

# Or use space-separated syntax
pith-user install vim --chroot /tmp/demo
```

#### What It Does

When you install a package with `--chroot`, pith modifies the symlink paths by removing the specified prefix. This enables packages to work correctly within a chroot environment.

**Example:**
- Without prefix stripping: `/tmp/demo/kod/store/vim/bin/vim`
- With `--chroot=/tmp/demo`: `/kod/store/vim/bin/vim`

#### Use Cases

1. **Development/Testing**: Create isolated package environments for testing without affecting system packages
2. **Container Support**: Prepare packages for container deployment
3. **CI/CD Pipelines**: Build packages in temporary directories and strip paths for reproducibility

#### How It Works

The prefix stripping affects three key areas:

1. **Symlinks**: Symlink targets have the prefix removed
2. **Wrapper Scripts**: Library paths (LD_LIBRARY_PATH) have the prefix removed
3. **Command Paths**: Absolute paths in wrapper scripts have the prefix removed

#### Flag Syntax

Both of these are equivalent:

```bash
# Equals-separated syntax
pith-user install vim --chroot=/tmp/demo

# Space-separated syntax
pith-user install vim --chroot /tmp/demo
```

#### Advanced Usage

Combine with other flags:

```bash
# Install with prefix stripping and force overwrite
pith-user install vim --chroot=/tmp/demo --force

# Install multiple packages with prefix stripping
pith-user install vim curl wget --chroot=/tmp/demo

# Install from AUR with prefix stripping
pith-user install yay --source=aur --chroot=/tmp/demo
```

### Manage Installed Packages

```bash
# List all packages
pith-user list

# List with detailed information
pith-user list --verbose

# Remove a package
pith-user remove vim

# Remove without confirmation
pith-user remove vim --force
```

### Updates and Cleanup

```bash
# Check for updates
pith-user upgrade --dry-run

# Upgrade all packages
pith-user upgrade

# Upgrade specific packages
pith-user upgrade bash curl

# Remove old versions (keeps 3 most recent)
pith-user cleanup --dry-run

# Remove old versions without confirmation
pith-user cleanup --force

# Preview cleanup in verbose mode
pith-user cleanup --verbose --dry-run
```

### Cache Management

```bash
# Show cache contents
pith-user cache --list

# Preview cache clean
pith-user cache --dry-run

# Clean all cached packages
pith-user cache --force
```

## Troubleshooting

### "Command not found: pith-user"

Make sure you:
1. Ran `./pith-user-init.sh`
2. Reloaded your shell (`source ~/.bashrc`)
3. Have ~/.local/bin in your PATH

Check:
```bash
echo $PATH
ls -la ~/.local/bin/pith-user
```

### Setup script not found

The setup script must be in the same directory as the pith binary:

```bash
ls -la chisel*
# Should show: chisel, pith-user, pith-user-init.sh
```

### Packages not in PATH

After installation, packages should be available in ~/.local/bin:

```bash
# Check if symlinks were created
ls ~/.local/bin/

# Verify PATH includes ~/.local/bin
echo $PATH
```

If packages still aren't available:
1. Reload your shell: `source ~/.bashrc`
2. Check LD_LIBRARY_PATH: `echo $LD_LIBRARY_PATH`
3. Verify package was installed: `pith-user list`

### Database sync fails

Make sure you have internet connectivity:

```bash
# Try syncing with verbose output
pith-user sync --verbose

# Check if you can reach the mirror
curl -I https://mirror.rackspace.com/archlinux/core/os/x86_64/core.db
```

### Installation fails with permission errors

User-level pith should not require sudo. If you get permission errors:

1. Check directory permissions:
```bash
ls -la ~/.local/share/chisel/
```

2. Ensure user owns the directories:
```bash
chown -R $USER ~/.local/share/chisel
chown -R $USER ~/.config/chisel
```

3. Check disk space:
```bash
df -h ~/.local/share/
```

### Configuration conflicts

If you have both user and system configs:

1. User config takes priority (good for customization)
2. To force system config: `chisel --base-dir /kod install vim`
3. To use specific config: `chisel --config /path/to/config.json install vim`

### Symlink prefix stripping issues

If packages don't work correctly with `--chroot`:

1. Verify the prefix matches your chroot path:
```bash
# If you're building in /tmp/demo, use that exact prefix
pith-user install vim --chroot=/tmp/demo
```

2. Check that symlinks were created correctly:
```bash
# List symlinks to see the paths
ls -la ~/.local/bin/
```

3. Verify wrapper scripts have stripped paths:
```bash
# Check the wrapper script for a package
cat ~/.local/share/chisel/wrappers/vim-wrapper.sh

# Look for stripped paths (should not contain the prefix)
```

4. If paths still contain the prefix:
```bash
# Reinstall without the prefix and try again
pith-user remove vim
pith-user install vim --chroot=/tmp/demo
```

## Advanced Usage

### Multiple Users

Each user can run the setup independently:

```bash
# User 1
user1@host:~$ ./pith-user-init.sh

# User 2 (different user)
user2@host:~$ ./pith-user-init.sh

# Each has isolated packages
user1@host:~$ pith-user list
user2@host:~$ pith-user list    # Different packages
```

### Custom Base Directories

Override the user base directory:

```bash
# Use a different location
export CHISEL_USER_BASE_DIR=/tmp/my-packages
pith-user sync

# Or use the wrapper with explicit path
pith-user --base-dir /tmp/my-packages install vim
```

### Temporary Package Testing

Test packages without affecting your main installation:

```bash
# Create temporary directory
mkdir /tmp/chisel-test
export CHISEL_USER_BASE_DIR=/tmp/chisel-test

# Install and test
pith-user sync
pith-user install test-package

# Clean up
rm -rf /tmp/chisel-test
```

### Integration with Other Tools

Since packages are installed to ~/.local/bin, you can:

```bash
# Add to PATH temporarily
export PATH=$PATH:$HOME/.local/bin

# Use in scripts
#!/bin/bash
export PATH=$PATH:$HOME/.local/bin
my-package-command

# Add to systemd user services
# ~/.config/systemd/user/my-service.service
[Service]
Environment="PATH=%h/.local/bin:%h/.local/lib"
ExecStart=%h/.local/bin/my-package
```

## Migration from System-Level

If you have packages installed system-wide and want to migrate to user-level:

```bash
# Initialize user environment
./pith-user-init.sh

# Reinstall packages in user location
pith-user sync
pith-user install package1 package2 package3

# Verify new installation
pith-user list --verbose

# Remove system-level packages (requires sudo)
sudo pith remove package1 package2 package3
```

## Cleanup and Uninstall

### Remove Individual Packages

```bash
pith-user remove package-name
```

### Remove All User Data

```bash
# Remove all pith user data
rm -rf ~/.local/share/chisel
rm -rf ~/.config/chisel

# Remove symlinks
rm -rf ~/.local/bin/*

# Remove from shell config (manually edit ~/.bashrc, ~/.zshrc, etc.)
# Remove lines added by the setup script
```

## Best Practices

### 1. Regular Cleanup

```bash
# Periodically remove old versions
pith-user cleanup --dry-run    # Preview first
pith-user cleanup --force      # Then execute
```

### 2. Cache Management

```bash
# Clean cache after upgrades
pith-user cache --list         # See what's there
pith-user cache --force        # Remove all
```

### 3. Database Updates

```bash
# Keep databases fresh
pith-user sync                # Sync databases regularly
```

### 4. Monitoring Usage

```bash
# Check disk usage
du -sh ~/.local/share/chisel/

# List packages and sizes
pith-user list --verbose
```

## FAQ

**Q: Can multiple users share packages?**
A: By default, each user has isolated packages. To share, set the same CHISEL_USER_BASE_DIR and ensure read permissions.

**Q: Do I need sudo for anything?**
A: No. User-level pith is completely sudo-free. All packages go to user-owned directories.

**Q: Can I use the system pith and pith-user together?**
A: Yes. System pith uses /kod, user-level uses ~/.local/share/chisel. They don't conflict.

**Q: What if ~/.local/bin is not in my PATH?**
A: The setup script adds it. If not, manually add to your ~/.bashrc:
```bash
export PATH="$PATH:$HOME/.local/bin"
```

**Q: Can I move packages after installation?**
A: You can move the entire ~/.local/share/chisel directory. Symlinks will need to be recreated.

**Q: What about LD_LIBRARY_PATH?**
A: Some packages need libraries from their own store. Set LD_LIBRARY_PATH to enable this:
```bash
export LD_LIBRARY_PATH="$HOME/.local/lib:$LD_LIBRARY_PATH"
```

## See Also

- [Chisel Main Documentation](./README.md)
- [Test Workflow Guide](./TEST-WORKFLOW.md)
- [Quick Test Guide](./QUICK-TEST.md)
- [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html)

## Getting Help

```bash
# Show help for pith-user
pith-user --help

# Show help for setup
./pith-user-init.sh --help

# Show pith help
chisel help
```

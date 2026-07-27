# Install Scripts Guide

This guide explains how to use the `pith install-scripts` command to execute post-install and post-upgrade scripts for packages.

## Overview

Many packages require post-installation scripts to properly configure the system. Examples include:

- **bash**: Sets up shell configuration files
- **glibc**: Initializes locale data
- **systemd**: Installs system units and triggers dbus reloads
- **kernel packages**: Updates bootloader configuration

The `pith install-scripts` command provides a way to run these scripts after packages are extracted and symlinks are created.

## Quick Start

### Non-Chroot Mode (Default)

Execute scripts directly in your current system context:

```bash
# Install package and auto-run scripts (in single command)
pith install bash

# Later, re-run scripts if needed
pith install-scripts bash

# Run scripts for multiple packages
pith install-scripts bash glibc

# Run scripts for all packages that have them
pith install-scripts
```

### Chroot Mode

Execute scripts in a chroot environment (for containerized or isolated installations):

```bash
# Install package in chroot (scripts deferred)
pith install --chroot /tmp/chroot bash

# Later, execute scripts in chroot
pith install-scripts --chroot /tmp/chroot bash

# Run all scripts in chroot
pith install-scripts --chroot /tmp/chroot
```

## Command Reference

### Basic Usage

```bash
pith install-scripts [options] [package ...]
```

### Options

| Option | Description | Required | Example |
|--------|-------------|----------|---------|
| `--chroot <dir>` | Chroot base directory for script execution | Optional | `--chroot /tmp/chroot` |
| `--verbose` or `-v` | Show detailed execution information | Optional | `--verbose` |
| `--help` or `-h` | Show help message | Optional | `--help` |

### Package Selection

If no packages are specified, the command runs scripts for **all installed packages** that have install scripts.

```bash
# Specific packages
pith install-scripts bash glibc

# All packages with install scripts
pith install-scripts
```

## Execution Modes

### 1. Non-Chroot Mode (Direct Execution)

**When to use**: Installing packages directly in your system or user environment

**Execution context**: Current system filesystem

**Command format**:
```bash
pith install-scripts bash
```

**What happens internally**:
```bash
cd /kod/store/bash/5.3.9-1 && source ./.INSTALL && post_install
```

**Example output**:
```
Running install scripts (current system context) for 1 package(s)...
Running post_install for bash/5.3.9-1...
✓ bash: post_install completed
✓ 1 install script(s) executed
```

### 2. Chroot Mode (Containerized Execution)

**When to use**: Installing packages in a chroot environment, container, or isolated filesystem

**Execution context**: Specified chroot directory

**Command format**:
```bash
pith install-scripts --chroot /tmp/chroot bash
```

**What happens internally**:
```bash
chroot /tmp/chroot bash -c "cd /kod/store/bash/5.3.9-1 && source ./.INSTALL && post_install"
```

**Example output**:
```
Running install scripts (chroot /tmp/chroot) for 1 package(s)...
Running post_install for bash/5.3.9-1...
✓ bash: post_install completed
✓ 1 install script(s) executed
```

## Integration with `pith install`

### Non-Chroot Installation

When you use `pith install` without `--chroot`, scripts **automatically execute** after symlinks are created:

```bash
pith install bash
```

**What happens**:
1. Download package
2. Extract to `/kod/store/bash/5.3.9-1`
3. Create symlinks
4. **Auto-execute** `post_install` script
5. Generate wrapper scripts
6. Done!

### Chroot Installation

When you use `pith install --chroot /tmp/chroot`, scripts are **deferred** for manual execution:

```bash
pith install --chroot /tmp/chroot bash
```

**Output**:
```
Note: Install scripts must be executed in chroot context.
Run the following command to execute install scripts:
  pith install-scripts --chroot /tmp/chroot
```

**What you need to do next**:
```bash
pith install-scripts --chroot /tmp/chroot bash
```

## Operation Detection

The command automatically determines whether to run `post_install` or `post_upgrade` based on the registry:

### New Package Installation
- First time a package is installed
- Operation: `post_install`
- Example: Fresh bash installation

### Package Upgrade
- Package version changes in registry
- Operation: `post_upgrade`
- Example: bash 5.2.0 → bash 5.3.9

### Script Idempotency

**Important**: Install scripts should be **idempotent** — they can be safely run multiple times without causing problems.

If you run the same script twice:
```bash
pith install-scripts bash
pith install-scripts bash  # Safe to run again
```

Both executions should succeed or produce the same result.

## Common Scenarios

### Scenario 1: Quick Installation with Auto-Execution

```bash
# Install package with automatic script execution
pith install bash

# Verification
pith list --verbose bash
```

**Result**: Package installed, scripts executed, ready to use.

---

### Scenario 2: Deferred Execution in Chroot

```bash
# Install in chroot (scripts skipped)
pith install --chroot /tmp/chroot bash glibc

# Later, execute scripts
pith install-scripts --chroot /tmp/chroot bash glibc
```

**Result**: Packages installed with scripts executed in chroot context.

---

### Scenario 3: Re-Running Scripts

```bash
# Install package (scripts auto-run)
pith install vim

# Later, re-run scripts if configuration changed
pith install-scripts vim
```

**Result**: Script runs again in non-chroot context.

---

### Scenario 4: Batch Processing

```bash
# Install many packages at once
pith install bash glibc grep sed awk

# Later, execute all scripts
pith install-scripts
```

**Result**: All packages with scripts have scripts executed.

---

### Scenario 5: Verbose Debugging

```bash
# Run scripts with detailed output
pith install-scripts bash --verbose

# In chroot with verbose
pith install-scripts --chroot /tmp/chroot bash --verbose
```

**Result**: Detailed execution information for debugging.

## Troubleshooting

### "Package not found in registry"

**Error**:
```
⚠ Warning: Package bash not found in registry
```

**Cause**: Package hasn't been installed yet

**Solution**:
```bash
# First install the package
pith install bash

# Then run scripts
pith install-scripts bash
```

---

### "No install script"

**Output**:
```
ℹ vim: No install script
```

**Meaning**: Package doesn't have a `.INSTALL` file in the extracted archive

**Action**: No scripts to run, installation is complete

---

### "Install script failed"

**Error**:
```
✗ bash: Install script failed (post_install): error message
```

**Cause**: Script execution failed but other packages continue

**Solution**:
1. Check the error message
2. Verify the chroot/system is in correct state
3. Re-run the script:
   ```bash
   pith install-scripts bash --verbose
   ```

---

### "Script not found"

**Error**:
```
script not found at /kod/store/bash/5.3.9-1/.INSTALL
```

**Cause**: 
- Package extraction failed or `.INSTALL` file wasn't extracted
- Package doesn't have a `.INSTALL` file

**Solution**:
```bash
# Re-install the package
pith install --no-symlink bash

# Then run scripts
pith install-scripts bash
```

---

### Chroot-specific Issues

**Issue**: `chroot: cannot change root directory to '/tmp/chroot': No such file or directory`

**Cause**: Chroot directory doesn't exist

**Solution**:
```bash
# Create chroot directory first
mkdir -p /tmp/chroot

# Then run chroot installation
pith install --chroot /tmp/chroot bash
pith install-scripts --chroot /tmp/chroot bash
```

---

**Issue**: Scripts work in non-chroot but fail in chroot

**Cause**: Environment differences (PATH, dependencies, etc.) between chroot and system

**Solution**:
1. Verify chroot environment has all required dependencies
2. Check script execution context and paths
3. Use `--verbose` to see full error messages:
   ```bash
   pith install-scripts --chroot /tmp/chroot bash --verbose
   ```

## Script Format

Package install scripts follow this format:

```bash
#!/bin/bash

# post_install() function - runs when package is first installed
post_install() {
    echo "Setting up bash..."
    # Configuration commands here
}

# post_upgrade() function - runs when package version changes
post_upgrade() {
    echo "Upgrading bash..."
    # Upgrade-specific commands here
}

# The install-scripts command calls the appropriate function
```

**Important**:
- Functions must be **idempotent** (safe to run multiple times)
- Use `post_install` for first-time setup
- Use `post_upgrade` for version-specific updates
- Both functions are optional; scripts can have just one

## Performance Considerations

### Serial Execution

Scripts execute **sequentially**, one package at a time:

```bash
pith install-scripts bash glibc grep sed
# Runs: bash → glibc → grep → sed (not in parallel)
```

### Timeout Handling

If a script hangs or times out:
- Use `Ctrl+C` to interrupt
- Run again or use `--verbose` to debug
- Check system resources if installation is slow

### Output Buffering

Script output appears in real-time:
```bash
pith install-scripts bash
# Output from post_install script appears immediately
```

## Advanced Usage

### Custom Script Execution

You can also manually run scripts if needed:

```bash
# Non-chroot manual execution
cd /kod/store/bash/5.3.9-1
source ./.INSTALL
post_install

# Chroot manual execution
chroot /tmp/chroot bash -c "cd /kod/store/bash/5.3.9-1 && source ./.INSTALL && post_install"
```

### Conditional Execution

Some packages have conditional logic in scripts:

```bash
# Run only if specific conditions are met
pith install-scripts glibc --verbose

# The script handles conditions internally
```

### Environment Variables

Scripts run with your current environment:

```bash
# To set environment variables for scripts
export MY_VAR=value
pith install-scripts bash
```

## See Also

- [USER-GUIDE.md](USER-GUIDE.md) - General user guide
- [REGISTRY.md](../reference/REGISTRY.md) - How package registry works
- [SPECIFICATION.md](../reference/SPECIFICATION.md) - Technical specification
- `pith install --help` - Install command help
- `pith install-scripts --help` - Install-scripts help

## Questions?

For more help:

```bash
# Show help
pith install-scripts --help

# Verbose output for debugging
pith install-scripts bash --verbose

# Check installed packages
pith list --verbose
```

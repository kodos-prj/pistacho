# Pistacho Use Cases

This document provides comprehensive coverage of Pistacho's three main use cases with real-world scenarios, benefits, and implementation details.

## Overview

Pistacho is a cross-distribution package manager that brings Arch Linux packages to any systemd-based Linux distribution. It solves three primary use cases:

1. **Development & Tooling** - Get cutting-edge development tools on stable systems
2. **Per-User Package Management** - Install packages without sudo access
3. **Container/Chroot & CI/CD** - Portable, reproducible package environments

---

## Use Case 1: Development & Tooling

### Problem

Stable Linux distributions (Ubuntu 22.04 LTS, Debian 12) have outdated packages. Developers need:
- Latest GCC, LLVM, Go, Rust toolchains
- Newest development tools and libraries
- Desktop environments with modern features
- Specialized tools (audio production, graphics, etc.)

**The Challenge:** Traditional solutions are cumbersome:
- ❌ PPAs often break or fall out of maintenance
- ❌ Containers add unnecessary overhead
- ❌ Building from source is time-consuming and error-prone
- ❌ Distribution backports lag behind upstream releases

### Solution: Pistacho

Pistacho installs Arch packages directly with complete dependency isolation. All dependencies come from Arch, not the host system.

### Scenarios

#### Scenario 1: Developer on Ubuntu 22.04 LTS
**Goal:** Set up a modern development environment

```bash
# Install essential development tools
sudo pith install gcc git vim neovim make cmake

# Install language runtimes
sudo pith install python nodejs golang rust

# Install development libraries
sudo pith install libssl-dev libcurl4-openssl-dev libxml2-dev
```

**Result:**
- Latest GCC compiler (not Ubuntu 22.04's GCC 11.4)
- Newest Python, Node.js, Go, Rust versions
- All dependencies automatically resolved from Arch
- System remains unchanged; all packages isolated in `/kod/`

#### Scenario 2: Desktop Environment Upgrade
**Goal:** Install GNOME or KDE Plasma on Debian 12

```bash
# Install complete GNOME desktop (50+ packages)
sudo pith install gnome

# Or KDE Plasma
sudo pith install kde-applications

# Or lightweight XFCE
sudo pith install xfce
```

**Result:**
- Latest desktop environment with all dependencies
- Newer than host distribution's official packages
- Works alongside existing desktop (if any)
- Complete isolation from host system

#### Scenario 3: Pro-Audio Production
**Goal:** Set up audio production tools on Ubuntu

```bash
# Install pro-audio production suite
sudo pith install pro-audio

# Or specific audio tools
sudo pith install audacity ardour jack pulseaudio
```

**Result:**
- Latest audio production software
- All audio libraries and dependencies from Arch
- No conflicts with host system's audio configuration
- Professional-grade audio tools with latest features

#### Scenario 4: Multiple Language Toolchains
**Goal:** Maintain multiple versions of development tools

```bash
# Install Rust with multiple toolchains
sudo pith install rust rustup

# Install Go development
sudo pith install golang

# Install Python with pip and virtualenv
sudo pith install python python-pip python-virtualenv

# Install Node.js LTS and latest
sudo pith install nodejs nodejs-lts
```

**Result:**
- Multiple versions coexist without conflicts
- Each tool has complete dependency isolation
- Can upgrade tools independently
- Host system's build tools untouched

### Benefits

| Benefit | Details |
|---------|---------|
| **Latest Software** | Access bleeding-edge development tools without waiting for distribution backports |
| **Reliable** | No PPA maintenance issues or broken dependencies |
| **Isolated** | Host system remains clean; all packages in `/kod/` |
| **Easy Setup** | Single command installs tools with all dependencies |
| **No Containers** | Direct execution without container overhead |
| **Cross-Distribution** | Same packages work on Ubuntu, Fedora, Debian, Arch |

### Who Benefits

- **Software Developers** - Need latest compilers, runtimes, frameworks
- **Data Scientists** - Want newest ML/AI libraries (TensorFlow, PyTorch, etc.)
- **System Administrators** - Need to maintain modern tools on LTS systems
- **DevOps Engineers** - Require cutting-edge infrastructure tools
- **Power Users** - Want access to specialized software not in main repos

---

## Use Case 2: Per-User Package Management

### Problem

System-wide package installation requires root/sudo access. In many scenarios, users need:
- Personal development environments without admin access
- Isolated package installations per user
- Easy setup and cleanup
- Following standard XDG directory conventions

**The Challenge:**
- ❌ Users without sudo can't install packages
- ❌ Multi-user systems can't have per-user package isolation
- ❌ Non-interactive environments (HPC, shared servers) prevent global installation
- ❌ Development environments need quick provisioning and cleanup

### Solution: Pistacho User-Level Management

Pistacho provides `pith-user` for per-user package management without requiring root access. Each user has their own isolated installation in `~/.local/share/pistacho/`.

### Scenarios

#### Scenario 1: Developer Without Root Access
**Goal:** Install development tools on a shared server

```bash
# One-time setup
./pith-user-init.sh

# Reload shell
source ~/.bashrc

# Install packages without sudo
pith-user sync
pith-user install gcc git vim neovim

# Use installed tools
vim --version    # Works directly
gcc --version    # Works directly

# List installed packages
pith-user list

# Upgrade packages
pith-user upgrade

# Clean old versions
pith-user cleanup
```

**Result:**
- User can install packages independently
- No root access required
- Packages in `~/.local/share/pistacho/` (user owns directory)
- Executables in `~/.local/bin/` (in PATH)
- Complete isolation from other users

#### Scenario 2: Multi-User Development Server
**Goal:** Multiple developers with independent package installations

```bash
# User 1 setup
user1@server:~$ ./pith-user-init.sh
user1@server:~$ pith-user install rust golang nodejs
user1@server:~$ pith-user list
# rust, golang, nodejs installed for user1

# User 2 setup (independent)
user2@server:~$ ./pith-user-init.sh
user2@server:~$ pith-user install python java ruby
user2@server:~$ pith-user list
# python, java, ruby installed for user2 (different from user1)
```

**Result:**
- Each user has isolated packages
- No conflicts between users
- Each user manages their own tools
- System administrator doesn't need to manage individual packages

#### Scenario 3: Temporary Project Environment
**Goal:** Quick development environment for a project

```bash
# Setup for project
./pith-user-init.sh

# Install project-specific tools
pith-user install nodejs npm python pip

# Develop the project
npm install
python -m venv .venv
source .venv/bin/activate

# When done, cleanup
pith-user cleanup
rm -rf ~/.local/share/pistacho
```

**Result:**
- Quick setup without system modifications
- Project-specific tools installed
- Easy cleanup when project ends
- No impact on other users or system

#### Scenario 4: HPC Cluster User
**Goal:** Personal tools on shared cluster without admin access

```bash
# Load module (if available)
module load pith

# Personal setup
./pith-user-init.sh

# Install scientific tools
pith-user install gcc gfortran openmpi hdf5

# Run computations with personal tools
pith-user search gnuplot
pith-user install gnuplot

# Visualize results
gnuplot script.gnuplot
```

**Result:**
- Users can customize their environment independently
- Cluster administrators don't manage per-user tools
- Scientific tools with all dependencies
- Reproducible results across cluster

### Directory Structure

```bash
~/.local/share/pistacho/              # Main data directory
├── store/                          # Extracted packages
│   ├── gcc/5.3.0-1/usr/bin/...
│   ├── vim/9.0.0-1/usr/bin/...
│   └── ...
├── db/                             # Synced package databases
│   ├── core.db
│   ├── extra.db
│   └── ...
├── wrappers/                       # Wrapper scripts
│   ├── gcc-wrapper.sh
│   ├── vim-wrapper.sh
│   └── ...
├── cache/                          # Downloaded packages
│   ├── gcc-*.pkg.tar.zst
│   ├── vim-*.pkg.tar.zst
│   └── ...
└── registry.json                   # Installed packages registry

~/.config/pistacho/                   # User configuration
└── config.json

~/.local/bin/                       # User symlinks (in PATH)
├── gcc -> ~/.local/share/pistacho/wrappers/gcc-wrapper.sh
├── vim -> ~/.local/share/pistacho/wrappers/vim-wrapper.sh
└── ...
```

### Benefits

| Benefit | Details |
|---------|---------|
| **No Root Required** | Users install packages independently |
| **Per-User Isolation** | Each user's packages don't affect others |
| **XDG Standards** | Follows XDG Base Directory specification |
| **Easy Setup** | One-time `pith-user-init.sh` |
| **Easy Cleanup** | Just remove `~/.local/share/pistacho/` |
| **Flexible** | Works on shared servers, HPC clusters, workstations |

### Who Benefits

- **Regular Users** - Want to install packages without admin assistance
- **Developers** - Need personal development environments on shared systems
- **HPC Researchers** - Install scientific tools on clusters without admin access
- **Multi-Tenant Servers** - Multiple users need independent environments
- **System Administrators** - Reduce support tickets for per-user tools

### Comparison: System vs User-Level

| Feature | System (`sudo pistacho`) | User-Level (`pith-user`) |
|---------|----------------------|----------------------------|
| **Permissions** | Requires sudo/root | No root required |
| **Location** | `/kod/` (global) | `~/.local/share/pistacho/` (per-user) |
| **Isolation** | Shared across all users | Per-user isolated |
| **Use Case** | Servers, shared tools | Personal development |
| **Setup** | Direct use | `pith-user-init.sh` once |
| **Number of Users** | Single administrator | Multiple independent users |

---

## Use Case 3: Container/Chroot & CI/CD Environments

### Problem

Modern development requires:
- Portable, reproducible package environments
- Cross-distribution compatibility (Ubuntu, Fedora, Debian, etc.)
- Containerization and CI/CD pipeline support
- Consistent build environments across different systems

**The Challenge:**
- ❌ Docker/containers add overhead and complexity
- ❌ Different distributions have different package versions
- ❌ Chroot environments require careful path management
- ❌ CI/CD pipelines need reproducible, portable builds

### Solution: Pistacho with Symlink Prefix Stripping

Pistacho's `--chroot` feature enables portable package environments by stripping path prefixes from symlinks and wrapper scripts. Packages work identically in containers, chroots, and different mount points.

### Scenarios

#### Scenario 1: Container Image Build
**Goal:** Build container images with Arch packages

```bash
# Build environment setup
mkdir /tmp/buildroot
cd /tmp/buildroot

# Install packages with symlink prefix stripping
pith install \
  --chroot=/tmp/buildroot \
  gcc git vim curl base-devel

# Create Dockerfile referencing the build
FROM ubuntu:22.04
COPY buildroot/kod /kod
ENV LD_LIBRARY_PATH=/kod/store/...
ENV PATH=/kod/wrappers:$PATH

# Container has all packages with correct paths
```

**Result:**
- Container image includes Arch packages
- Symlinks correctly reference `/kod/` in container
- No path conflicts with host system
- Portable across container registries

#### Scenario 2: CI/CD Pipeline with Multiple Runners
**Goal:** Consistent build environment across different CI runners

```bash
# GitHub Actions workflow
name: Build with Pistacho Packages
on: [push]
jobs:
  build:
    runs-on: [ubuntu-latest, ubuntu-20.04]
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Pistacho packages
        run: |
          sudo pith install \
            --chroot=/home/runner/work/build \
            gcc cmake ninja make
      
      - name: Build project
        run: |
          export PATH=/home/runner/work/build/kod/wrappers:$PATH
          cmake .
          make -j4
```

**Result:**
- Same build environment on all runners
- Consistent compiler versions across CI platforms
- No "works on my machine" issues
- Reproducible builds

#### Scenario 3: Development Chroot
**Goal:** Isolated development environment in chroot

```bash
# Setup chroot directory
mkdir /tmp/dev-chroot
cd /tmp/dev-chroot

# Install development tools with symlink prefix
pith install \
  --chroot=/tmp/dev-chroot \
  gcc git make cmake python

# Enter chroot
sudo chroot /tmp/dev-chroot /bin/bash

# Inside chroot, tools work correctly
gcc --version
git --version
python3 --version

# Exit chroot
exit

# Cleanup
rm -rf /tmp/dev-chroot
```

**Result:**
- Isolated development environment
- Tools available inside chroot with correct paths
- Easy setup and cleanup
- No host system modifications

#### Scenario 4: Cross-Distribution CI/CD
**Goal:** Test builds on multiple distributions

```bash
# CI configuration supporting multiple distros
matrix:
  os: [ubuntu-22.04, ubuntu-20.04, debian-12, fedora-39]
  
  steps:
    - name: Setup
      run: |
        if [ "$RUNNER_OS" == "Linux" ]; then
          sudo pith sync
          sudo pith install --chroot=/tmp/build \
            gcc cmake nodejs
        fi
    
    - name: Build
      run: |
        export PATH=/tmp/build/kod/wrappers:$PATH
        ./configure
        make
```

**Result:**
- Identical build environment on all distributions
- Same compiler, build tools, libraries everywhere
- No distribution-specific workarounds needed
- Truly reproducible builds

#### Scenario 5: Development Workspace Setup
**Goal:** Quick reproducible development environment

```bash
# Setup workspace
mkdir ~/workspace && cd ~/workspace

# Install workspace tools
pith install \
  --chroot=~/workspace \
  gcc git vim docker docker-compose

# Create portable environment
# ~/workspace/kod contains all packages with correct paths

# Export workspace for sharing
tar czf workspace-backup.tar.gz ~/workspace

# Later, restore on different machine
tar xzf workspace-backup.tar.gz
cd workspace
export PATH=~/workspace/kod/wrappers:$PATH
export LD_LIBRARY_PATH=~/workspace/kod/store/...
gcc --version  # Works on any Linux distribution
```

**Result:**
- Portable development environment
- Can be packaged and shared
- Works on any systemd-based distribution
- Quick setup on new machines

### How Symlink Prefix Stripping Works

The `--chroot` flag accomplishes two critical tasks:

#### 1. Creates Symlinks INSIDE the Prefix Directory

**Without `--chroot`:**
```bash
# Symlinks created on host system
/usr/bin/vim → /kod/store/vim/.../usr/bin/vim
/usr/bin/git → /kod/store/git/.../usr/bin/git
```

**With `--chroot=/tmp/chroot`:**
```bash
# Symlinks created INSIDE the prefix directory
/tmp/chroot/usr/bin/vim → /kod/store/vim/.../usr/bin/vim
/tmp/chroot/usr/bin/git → /kod/store/git/.../usr/bin/git
```

#### 2. Strips Prefix from Symlink Targets

**Without prefix stripping:**
```bash
# Installed with --chroot=/tmp/demo
Symlink target: /tmp/demo/kod/store/vim/9.0.0-1/usr/bin/vim
```

**With prefix stripping:**
```bash
# Installed with --chroot=/tmp/demo
Symlink target: /kod/store/vim/9.0.0-1/usr/bin/vim  # Prefix stripped!
```

#### Real-World Flow

```bash
# Command
sudo pith install --chroot=/tmp/chroot gcc

# What gets created
Step 1: Extract package
  Location: /tmp/chroot/kod/store/gcc/12.0.0-1/usr/bin/gcc

Step 2: Create symlink LOCATION
  Symlink path = /tmp/chroot + usr/bin/gcc = /tmp/chroot/usr/bin/gcc

Step 3: Create symlink TARGET (strip prefix)
  Target = /tmp/chroot/kod/store/gcc/.../usr/bin/gcc
  After stripping /tmp/chroot: /kod/store/gcc/.../usr/bin/gcc

Step 4: Final result
  /tmp/chroot/usr/bin/gcc → /kod/store/gcc/.../usr/bin/gcc

# Inside chroot
sudo chroot /tmp/chroot /bin/bash
$ gcc --version  # ✓ Works! Symlink resolves to /kod/store/gcc/...
```

#### Why This Matters

**Inside the chroot directory `/tmp/chroot/`:**
- When gcc runs and looks for libraries
- It resolves `/kod/store/gcc/.../lib/libstdc++.so`
- Since we're inside the chroot, `/` is actually `/tmp/chroot/`
- So it finds: `/tmp/chroot/kod/store/gcc/.../lib/libstdc++.so` ✓
- This works even if `/tmp/chroot` is moved to a different location!

**The Magic:**
All paths are relative to the prefix directory, making it truly portable across:
- Different mount points
- Container registries
- CI/CD runners
- Different machines

### Benefits

| Benefit | Details |
|---------|---------|
| **Portable** | Packages work in containers, chroots, different mount points |
| **Reproducible** | Consistent environment across CI runners and distributions |
| **Cross-Distribution** | Ubuntu, Fedora, Debian, Arch all get identical packages |
| **Easy Cleanup** | Just remove the prefix directory |
| **No Containers** | Direct execution in chroots; optional Docker integration |
| **Minimal Overhead** | Faster than Docker, lighter than VMs |
| **Self-Contained** | Symlinks point directly to package files; no wrapper scripts needed |

### Who Benefits

- **DevOps Engineers** - Need portable, reproducible CI/CD environments
- **Container Builders** - Want efficient base images with Arch packages
- **System Administrators** - Maintain consistent builds across infrastructure
- **Open Source Maintainers** - Ensure builds work on multiple distributions
- **Development Teams** - Share reproducible development environments

### Real-World Example: Open Source Project

```bash
# Project repository includes build script
#!/bin/bash
set -e

# Initialize Pistacho packages
pith install --chroot=$PWD/build \
  gcc cmake ninja python3 doxygen

export PATH=$PWD/build/kod/wrappers:$PATH
export LD_LIBRARY_PATH=$PWD/build/kod/store/...

# Build the project
cmake -B build -S . -DCMAKE_BUILD_TYPE=Release
cmake --build build -j$(nproc)

# Run tests
ctest --test-dir build

# Generate documentation
cd docs && doxygen Doxyfile && cd ..

# Result: Same build succeeds on Ubuntu, Fedora, Debian, Arch
```

---

## Comparison Matrix

| Feature | Use Case 1: Development | Use Case 2: Per-User | Use Case 3: Container/CI |
|---------|------------------------|----------------------|--------------------------|
| **Requires Sudo** | Yes (system-wide) | No (user-only) | Varies (may be containerized) |
| **Location** | `/kod/` global | `~/.local/share/pistacho/` | Configurable with `--chroot` |
| **Isolation** | System-wide | Per-user | Per-environment/container |
| **Key Feature** | Latest tools | No-root access | Portable paths |
| **Primary Users** | Developers | Regular users | DevOps/CI/CD |
| **Scale** | Single machine | Single machine | Infrastructure-wide |
| **Reproducibility** | Good | Good | Excellent |

---

## Getting Started

### For Development & Tooling
```bash
# Quick start
go build -o pith ./cmd/pistacho
sudo ./pith sync
sudo ./pith install gcc git vim
```

**See:** [README.md](../README.md#quick-start)

### For Per-User Management
```bash
# One-time setup
./pith-user-init.sh
source ~/.bashrc

# Start using
pith-user install vim
pith-user list
```

**See:** [USER-GUIDE.md](./user-guides/USER-GUIDE.md)

### For Container/CI/CD
```bash
# Install with prefix stripping
sudo pith install --chroot=/tmp/build gcc cmake

# Use in CI pipeline
export PATH=/tmp/build/kod/wrappers:$PATH
```

**See:** [USER-GUIDE.md - Chroot Support](./user-guides/USER-GUIDE.md#chroot-support-with-chroot-stripping)

---

## Frequently Asked Questions

**Q: Can I use multiple use cases together?**
A: Yes! You can combine them. Example: Use development tools (Use Case 1) in a container build (Use Case 3) for per-user CI/CD pipelines (Use Case 2).

**Q: Which use case should I choose?**
A: Pick based on your needs:
- Need latest tools on your dev machine? → Use Case 1
- Don't have root access? → Use Case 2
- Building containers/CI pipelines? → Use Case 3

**Q: Can I switch between use cases?**
A: Yes. You can use system-wide pith for some projects and user-level for others.

**Q: What if my use case doesn't fit these three?**
A: Pistacho is designed for these three primary scenarios. For other needs, consider standard package managers or containers.

---

## See Also

- [README.md](../README.md) - Main documentation
- [USER-GUIDE.md](./user-guides/USER-GUIDE.md) - User-level package management guide
- [DEVELOPER-GUIDE.md](./developer/DEVELOPER-GUIDE.md) - Development setup and architecture
- [AUR-INTEGRATION.md](./architecture/AUR-INTEGRATION.md) - AUR support details


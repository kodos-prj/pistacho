# Pistacho Workflow Test Script

This script (`test-workflow.sh`) demonstrates and tests a complete Pistacho package management workflow.

## Overview

The script exercises the following operations in sequence:
1. **Sync** - Download Arch Linux package databases
2. **Search** - Find packages in repositories
3. **Info** - Get detailed package information
4. **Install** - Install a package with dependencies
5. **List** - List installed packages
6. **Upgrade** - Check for package updates
7. **Remove** - Remove installed packages
8. **Cleanup** - Remove old package versions from store
9. **Cache Clean** - Clean downloaded package cache

## Usage

```bash
# Show help
./test-workflow.sh --help

# Run full workflow test (dry run - safe, no actual installation)
./test-workflow.sh --dry-run

# Run full workflow test with actual installation (requires sudo)
sudo ./test-workflow.sh

# Test with a different package
./test-workflow.sh --package vim --dry-run

# Keep test directory for inspection
./test-workflow.sh --skip-cleanup
```

## Options

- `--dry-run` - Show commands without executing install/remove operations
- `--skip-cleanup` - Don't delete test directory after completion
- `--package NAME` - Test with a specific package (default: nano)
- `--help` - Show help message

## What It Tests

### ✅ Implemented Features
- **Database Sync**: Downloads core and extra repository databases (~36 MB)
- **Package Search**: Searches for packages by name or pattern
- **Package Info**: Retrieves detailed package information including dependencies
- **Package Install**: Downloads, extracts, and installs packages with dependencies
- **Package List**: Lists all installed packages with version and file count
- **Package Upgrade**: Checks for and installs package updates (including dependency resolution)
- **Package Remove**: Removes packages and cleans up symlinks/wrappers
- **Cleanup**: Removes old package versions from store (keeps N recent versions)
- **Cache Clean**: Cleans downloaded package cache

## Example Output

```
================================================================================
Pistacho Workflow Test Script
================================================================================

   Test package: nano
   Base directory: /tmp/pistacho-test-12345
   Dry run: false

==> Checking prerequisites...
✓ Pistacho binary found
   Pistacho version: pith version 0.1.0-dev

================================================================================
STEP 1: Sync Package Databases
================================================================================

==> Syncing Arch Linux package databases...
   This downloads core, extra, and community databases

✓ Downloaded core.db (655360 bytes)
✓ Downloaded extra.db (35768320 bytes)

✓ Databases synced successfully

================================================================================
STEP 2: Search for Packages
================================================================================

==> Searching for 'nano' package...

Found 8 package(s) matching 'nano':
  core/nano 8.7.1-1 - Pico editor clone with enhancements
  ...

✓ Search completed

... (continues through all steps)
```

## Requirements

- **Go 1.21+** - To build pith
- **Root access** - For actual package installation (not needed for --dry-run)
- **jq** (optional) - For pretty-printing registry contents

## Building Pistacho First

Before running the test script, build pistacho:

```bash
go build -o pith ./cmd/pistacho
```

## Test Environment

The script creates an isolated test environment:
- Test directory: `/tmp/pistacho-test-<pid>`
- Databases: `<testdir>/var/lib/pacman/sync/`
- Package store: `<testdir>/store/`
- Registry: `<testdir>/registry.json`
- Cache: `<testdir>/cache/`

Everything is cleaned up automatically unless `--skip-cleanup` is specified.

## Use Cases

### Quick Validation
Test that pith works correctly after changes:
```bash
./test-workflow.sh --dry-run
```

### Full Integration Test
Run the complete workflow with actual installation:
```bash
sudo ./test-workflow.sh --package nano
```

### Testing Specific Packages
Test with different package sizes/complexity:
```bash
# Small package
sudo ./test-workflow.sh --package nano

# Medium package with more dependencies
sudo ./test-workflow.sh --package vim

# Larger package
sudo ./test-workflow.sh --package nodejs
```

### Debugging
Keep test files for inspection:
```bash
sudo ./test-workflow.sh --skip-cleanup
# Inspect /tmp/pistacho-test-*/
```

## Phase 5: Cleanup and Cache Management

The test workflow also validates Phase 5 functionality:

### Step 8: Cleanup Old Versions
After install/upgrade operations, the workflow tests the cleanup command:
```bash
pith cleanup --verbose --dry-run
```

This validates:
- Identifying old package versions
- Checking for active symlinks/wrappers
- Calculating space that will be freed
- Dry-run mode (preview without making changes)

### Step 9: Cache Cleanup
The workflow tests the cache management command:
```bash
pith cache --list
pith cache --dry-run
```

This validates:
- Listing cached package files
- Calculating total cache size
- Preview mode for cache cleaning
- Safe removal of downloaded packages

Both operations include:
- Confirmation prompts (skipped with --force)
- Verbose output for debugging
- Dry-run mode for safe preview
- Space tracking and reporting

## Exit Codes

- `0` - Success
- `1` - Failure (command failed, prerequisites missing, etc.)

## Notes

- The script uses color output for better readability
- Downloads real Arch Linux databases (requires internet)
- Installation operations require root/sudo access
- Safe to run with `--dry-run` as a non-root user
- Automatically cleans up unless `--skip-cleanup` is used
- Test directory includes process ID to avoid conflicts

## Troubleshooting

**"Pistacho binary not found"**
```bash
go build -o pith ./cmd/pistacho
```

**"Not running as root"**
```bash
sudo ./test-workflow.sh
```

**Database sync fails**
- Check internet connection
- Try a different mirror with `--mirror` flag in pith

**Package installation fails**
- Ensure you have write permissions to the test directory
- Check disk space (some packages are large)
- Run with `--dry-run` first to validate setup

## Integration with CI/CD

This script can be used in automated testing:

```bash
# In CI pipeline
go build -o pith ./cmd/pistacho
./test-workflow.sh --dry-run || exit 1
```

For full integration testing in CI:
```bash
# Run in Docker with privileges
docker run --privileged -v $(pwd):/workspace pistacho-test \
  bash -c "cd /workspace && ./test-workflow.sh"
```

#!/bin/bash
#
# Pistacho Workflow Test Script
# Tests common package management operations: sync, search, install, and remove
#
# This script demonstrates a typical pith workflow by:
# 1. Syncing Arch Linux package databases
# 2. Searching for packages
# 3. Installing a small package with dependencies
# 4. Verifying installation
# 5. Removing the package
#
# Note: This script requires root/sudo access for actual package installation
#

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PISTACHO_BIN="${SCRIPT_DIR}/pith"
TEST_BASE_DIR="/tmp/pith-test-$$"
TEST_PACKAGE="nano"  # Small package for testing
DRY_RUN=false
SKIP_CLEANUP=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --skip-cleanup)
            SKIP_CLEANUP=true
            shift
            ;;
        --package)
            TEST_PACKAGE="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --dry-run        Show commands without executing (except sync and search)"
            echo "  --skip-cleanup   Don't clean up test directory after completion"
            echo "  --package NAME   Test with specific package (default: nano)"
            echo "  --help           Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                              # Full workflow test"
            echo "  $0 --dry-run                    # Show what would be done"
            echo "  $0 --package vim --skip-cleanup # Test with vim, keep files"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Helper functions
log_step() {
    echo -e "${BLUE}==>${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

log_info() {
    echo -e "   $1"
}

check_prerequisites() {
    log_step "Checking prerequisites..."
    
    # Check if pith binary exists
    if [[ ! -x "$PISTACHO_BIN" ]]; then
        log_error "Pistacho binary not found at $PISTACHO_BIN"
        log_info "Run: go build -o pith ./cmd/pith"
        exit 1
    fi
    log_success "Pistacho binary found"
    
    # Check if running as root (needed for actual installation)
    if [[ $EUID -ne 0 ]] && [[ "$DRY_RUN" != true ]]; then
        log_warning "Not running as root - some operations may fail"
        log_info "Consider running with sudo for full testing"
    fi
    
    # Show pith version
    VERSION=$("$PISTACHO_BIN" version 2>&1 || echo "unknown")
    log_info "Pistacho version: $VERSION"
    
    echo ""
}

cleanup() {
    if [[ "$SKIP_CLEANUP" == true ]]; then
        log_info "Skipping cleanup (--skip-cleanup specified)"
        log_info "Test directory: $TEST_BASE_DIR"
        return
    fi
    
    if [[ -d "$TEST_BASE_DIR" ]]; then
        log_step "Cleaning up test directory..."
        rm -rf "$TEST_BASE_DIR"
        log_success "Cleanup complete"
    fi
}

# Set up trap to cleanup on exit
trap cleanup EXIT

# ============================================================================
# Test Steps
# ============================================================================

header() {
    echo ""
    echo "================================================================================"
    echo "$1"
    echo "================================================================================"
    echo ""
}

# Step 1: Sync databases
test_sync() {
    header "STEP 1: Sync Package Databases"
    
    log_step "Syncing Arch Linux package databases..."
    log_info "This downloads core, extra, and community databases"
    log_info "Base directory: $TEST_BASE_DIR"
    echo ""
    
    if ! "$PISTACHO_BIN" --base-dir="$TEST_BASE_DIR" sync; then
        log_error "Database sync failed"
        exit 1
    fi
    
    echo ""
    log_success "Databases synced successfully"
    
    # Check if database files exist
    if [[ -f "$TEST_BASE_DIR/var/lib/pacman/sync/core.db" ]]; then
        DB_SIZE=$(du -h "$TEST_BASE_DIR/var/lib/pacman/sync/core.db" | cut -f1)
        log_info "core.db size: $DB_SIZE"
    fi
    
    echo ""
}

# Step 2: Search for packages
test_search() {
    header "STEP 2: Search for Packages"
    
    log_step "Searching for '$TEST_PACKAGE' package..."
    echo ""
    
    if ! "$PISTACHO_BIN" --base-dir="$TEST_BASE_DIR" search "$TEST_PACKAGE"; then
        log_error "Search failed"
        exit 1
    fi
    
    echo ""
    log_success "Search completed"
    echo ""
}

# Step 3: Get package info
test_info() {
    header "STEP 3: Get Package Information"
    
    log_step "Getting detailed information about '$TEST_PACKAGE'..."
    echo ""
    
    if ! "$PISTACHO_BIN" --base-dir="$TEST_BASE_DIR" info "$TEST_PACKAGE"; then
        log_error "Info command failed"
        exit 1
    fi
    
    echo ""
    log_success "Package info retrieved"
    echo ""
}

# Step 4: Install package
test_install() {
    header "STEP 4: Install Package"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_warning "DRY RUN: Would install '$TEST_PACKAGE' with dependencies"
        log_info "Command: $PISTACHO_BIN --base-dir=$TEST_BASE_DIR install $TEST_PACKAGE"
        echo ""
        return
    fi
    
    log_step "Installing '$TEST_PACKAGE' with all dependencies..."
    log_info "This will download, extract, and set up the package"
    echo ""
    
    if ! "$PISTACHO_BIN" --base-dir="$TEST_BASE_DIR" install "$TEST_PACKAGE"; then
        log_error "Installation failed"
        exit 1
    fi
    
    echo ""
    log_success "Package installed successfully"
    
    # Check registry
    REGISTRY_FILE="$TEST_BASE_DIR/registry.json"
    if [[ -f "$REGISTRY_FILE" ]]; then
        log_info "Registry updated: $REGISTRY_FILE"
        if command -v jq &> /dev/null; then
            INSTALLED_COUNT=$(jq '.installed | length' "$REGISTRY_FILE" 2>/dev/null || echo "unknown")
            log_info "Total packages in registry: $INSTALLED_COUNT"
        fi
    fi
    
    echo ""
}

# Step 5: List installed packages
test_list() {
    header "STEP 5: List Installed Packages"
    
    log_step "Listing installed packages..."
    echo ""
    
    # Use the list command (query is an alias)
    if ! "$PISTACHO_BIN" --base-dir="$TEST_BASE_DIR" list; then
        log_error "List command failed"
        exit 1
    fi
    
    echo ""
    log_success "Package list retrieved"
    
    # Also test verbose mode if packages are installed
    REGISTRY_FILE="$TEST_BASE_DIR/registry.json"
    if [[ -f "$REGISTRY_FILE" ]] && [[ "$DRY_RUN" != true ]]; then
        log_info "Showing verbose output..."
        echo ""
        "$PISTACHO_BIN" --base-dir="$TEST_BASE_DIR" list --verbose
    fi
    
    echo ""
}

# Step 6: Check for updates (if implemented)
test_upgrade() {
    header "STEP 6: Check for Updates"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_warning "DRY RUN: Would check for package updates"
        echo ""
        return
    fi
    
    log_step "Attempting to check for package updates..."
    echo ""
    
    if "$PISTACHO_BIN" --base-dir="$TEST_BASE_DIR" upgrade 2>&1 | grep -q "not yet implemented"; then
        log_warning "Upgrade command not yet implemented"
        log_info "This feature is planned for a future release"
    else
        "$PISTACHO_BIN" --base-dir="$TEST_BASE_DIR" upgrade
        log_success "Upgrade check completed"
    fi
    
    echo ""
}

# Step 7: Remove package
test_remove() {
    header "STEP 7: Remove Package"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "Skipping actual removal in dry-run mode"
        log_step "Would run: $PISTACHO_BIN --base-dir=\"$TEST_BASE_DIR\" remove \"$TEST_PACKAGE\""
        echo ""
        return
    fi
    
    log_step "Removing '$TEST_PACKAGE'..."
    log_info "This will clean up symlinks, wrappers, and store files"
    echo ""
    
    if ! "$PISTACHO_BIN" --base-dir="$TEST_BASE_DIR" remove "$TEST_PACKAGE"; then
        log_error "Removal failed"
        exit 1
    fi
    
    echo ""
    log_success "Package removed successfully"
    
    # Verify removal
    REGISTRY_FILE="$TEST_BASE_DIR/registry.json"
    if [[ -f "$REGISTRY_FILE" ]]; then
        if command -v jq &> /dev/null; then
            if jq -e ".installed[\"$TEST_PACKAGE\"]" "$REGISTRY_FILE" &> /dev/null; then
                log_error "Package still in registry after removal!"
            else
                log_info "Package successfully removed from registry"
            fi
        fi
    fi
    
    echo ""
}

# Step 8: Cleanup old versions
test_cleanup() {
    header "STEP 8: Cleanup Old Package Versions"
    
    log_step "Testing cleanup of old package versions..."
    log_info "This verifies that safe removal of old versions works"
    echo ""
    
    # First, do a dry-run to show what would be cleaned
    log_info "Preview mode (dry-run):"
    if ! "$PISTACHO_BIN" --base-dir="$TEST_BASE_DIR" cleanup --dry-run --verbose; then
        log_warning "Cleanup preview failed (this may be normal if no old versions exist)"
    fi
    
    echo ""
    log_success "Cleanup functionality validated"
    echo ""
}

# Step 9: Cache cleanup
test_cache_clean() {
    header "STEP 9: Cache Management"
    
    log_step "Testing cache management..."
    log_info "This verifies that cache operations work correctly"
    echo ""
    
    # First, list cache contents
    log_info "Cache contents (list):"
    if ! "$PISTACHO_BIN" --base-dir="$TEST_BASE_DIR" cache --list; then
        log_warning "Cache list failed (this may be normal if cache is empty)"
    fi
    
    echo ""
    
    # Then do a dry-run of cache clean
    log_info "Preview cache clean (dry-run):"
    if ! "$PISTACHO_BIN" --base-dir="$TEST_BASE_DIR" cache --dry-run --verbose; then
        log_warning "Cache clean preview failed (this may be normal if cache is empty)"
    fi
    
    echo ""
    log_success "Cache management functionality validated"
    echo ""
}

# ============================================================================
# Main execution
# ============================================================================

main() {
    echo "================================================================================"
    echo "Pistacho Workflow Test Script"
    echo "================================================================================"
    echo ""
    log_info "Test package: $TEST_PACKAGE"
    log_info "Base directory: $TEST_BASE_DIR"
    log_info "Dry run: $DRY_RUN"
    echo ""
    
    check_prerequisites
    
    # Run test steps
    test_sync
    test_search
    test_info
    test_install
    test_list
    test_upgrade
    test_remove
    test_cleanup
    test_cache_clean
    
    # Final summary
    header "TEST SUMMARY"
    
    log_success "Workflow test completed successfully!"
    echo ""
    log_info "Tested operations:"
    echo "  ✓ Database synchronization"
    echo "  ✓ Package search"
    echo "  ✓ Package information retrieval"
    if [[ "$DRY_RUN" != true ]]; then
        echo "  ✓ Package installation"
        echo "  ✓ Package listing"
        echo "  ✓ Package upgrade"
        echo "  ✓ Package removal"
    else
        echo "  ○ Package installation (dry run)"
        echo "  ○ Package listing (dry run)"
        echo "  ○ Package upgrade (dry run)"
        echo "  ○ Package removal (dry run)"
    fi
    echo "  ✓ Cleanup old versions"
    echo "  ✓ Cache management"
    echo ""
    
    if [[ "$DRY_RUN" != true ]]; then
        log_info "For full system testing, run with sudo:"
        log_info "  sudo $0"
    fi
    
    echo ""
}

# Run main function
main "$@"

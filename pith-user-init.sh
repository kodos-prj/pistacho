#!/bin/bash
#
# Pistacho User-Level Setup Script
# Initializes pith for user-level package management
#
# This script sets up:
# - User-level pith directories (~/.local/share/pith/)
# - User configuration file (~/.config/pith/config.json)
# - User symlink directory (~/.local/bin/)
# - Shell configuration for PATH and LD_LIBRARY_PATH
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PITH_BIN="${SCRIPT_DIR}/pith"

# Determine user directories following XDG spec
XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
XDG_DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}"
PITH_CONFIG_DIR="${XDG_CONFIG_HOME}/pith"
PITH_DATA_DIR="${XDG_DATA_HOME}/pith"
PITH_BIN_DIR="${HOME}/.local/bin"
PITH_CONFIG_FILE="${PITH_CONFIG_DIR}/config.json"

# Parse arguments
FORCE=false
SHELL_CONFIG_ONLY=false
while [[ $# -gt 0 ]]; do
  case $1 in
  --force)
    FORCE=true
    shift
    ;;
  --shell-config-only)
    SHELL_CONFIG_ONLY=true
    shift
    ;;
  --help)
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --force                 Overwrite existing configuration"
    echo "  --shell-config-only     Only update shell configuration (don't create dirs)"
    echo "  --help                  Show this help message"
    echo ""
    echo "This script initializes pith for user-level package management."
    echo "It creates:"
    echo "  - ${PITH_DATA_DIR}"
    echo "  - ${PITH_CONFIG_FILE}"
    echo "  - ${PITH_BIN_DIR}"
    exit 0
    ;;
  *)
    echo "Unknown option: $1"
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

# Check prerequisites
check_prerequisites() {
  log_step "Checking prerequisites..."

  if [[ ! -x "$PITH_BIN" ]]; then
    log_error "Pistacho binary not found at $PITH_BIN"
    log_info "Run: go build -o pith ./cmd/pith"
    exit 1
  fi

  log_success "Pistacho binary found"

  # Check if pith supports user-level config
  if ! "$PITH_BIN" --help 2>&1 | grep -q "Phase 5"; then
    log_warning "Pistacho might not have latest user-level support"
  fi

  echo ""
}

# Create directories
create_directories() {
  log_step "Creating user-level directories..."

  # Create config directory
  if [[ ! -d "$PITH_CONFIG_DIR" ]]; then
    mkdir -p "$PITH_CONFIG_DIR"
    log_info "Created $PITH_CONFIG_DIR"
  else
    log_info "$PITH_CONFIG_DIR already exists"
  fi

  # Create data directory
  if [[ ! -d "$PITH_DATA_DIR" ]]; then
    mkdir -p "$PITH_DATA_DIR"
    log_info "Created $PITH_DATA_DIR"
  else
    log_info "$PITH_DATA_DIR already exists"
  fi

  # Create bin directory
  if [[ ! -d "$PITH_BIN_DIR" ]]; then
    mkdir -p "$PITH_BIN_DIR"
    log_info "Created $PITH_BIN_DIR"
  else
    log_info "$PITH_BIN_DIR already exists"
  fi

  # Create subdirectories in PITH_DATA_DIR
  for dir in store db wrappers cache; do
    if [[ ! -d "$PITH_DATA_DIR/$dir" ]]; then
      mkdir -p "$PITH_DATA_DIR/$dir"
      log_info "Created $PITH_DATA_DIR/$dir"
    fi
  done

  echo ""
}

# Create configuration file
create_config() {
  log_step "Creating user-level configuration..."

  if [[ -f "$PITH_CONFIG_FILE" ]] && [[ "$FORCE" != true ]]; then
    log_warning "Configuration already exists at $PITH_CONFIG_FILE"
    log_info "Use --force to overwrite"
    echo ""
    return
  fi

  # Create config JSON
  cat >"$PITH_CONFIG_FILE" <<EOF
{
  "base_dir": "$PITH_DATA_DIR",
  "symlink_root": "$PITH_BIN_DIR",
  "mirror_url": "https://mirror.rackspace.com/archlinux",
  "architecture": "x86_64",
  "repositories": ["core", "extra", "community"],
  "verify_signatures": false,
  "max_concurrent_downloads": 5,
  "download_timeout": 300,
  "keep_versions": 3
}
EOF

  log_success "Created configuration file"
  log_info "Location: $PITH_CONFIG_FILE"
  log_info "Note: base_dir and symlink_root are automatically configured"
  echo ""
}

# Add to shell configuration
update_shell_config() {
  log_step "Updating shell configuration..."

  # Detect shell
  SHELL_NAME=$(basename "$SHELL")
  case "$SHELL_NAME" in
  bash)
    SHELL_RC="$HOME/.bashrc"
    ;;
  zsh)
    SHELL_RC="$HOME/.zshrc"
    ;;
  fish)
    SHELL_RC="$HOME/.config/fish/config.fish"
    ;;
  *)
    log_warning "Unsupported shell: $SHELL_NAME"
    log_info "Manually add the following to your shell configuration:"
    print_shell_config
    echo ""
    return
    ;;
  esac

  # Check if already configured
  if grep -q "PITH_USER_BASE_DIR" "$SHELL_RC" 2>/dev/null; then
    log_warning "Shell already configured for pith"
    log_info "Skipping shell configuration update"
    echo ""
    return
  fi

  # Add configuration to shell rc
  cat >>"$SHELL_RC" <<EOF

# Pistacho user-level configuration
export PITH_USER_BASE_DIR="$PITH_DATA_DIR"
export PATH="\$PATH:\$HOME/.local/bin"
export LD_LIBRARY_PATH="\$HOME/.local/lib:\$LD_LIBRARY_PATH"
EOF

  log_success "Updated $SHELL_RC"
  log_info "Added PITH_USER_BASE_DIR, PATH, and LD_LIBRARY_PATH"
  echo ""
}

# Print shell configuration snippet
print_shell_config() {
  echo ""
  echo "Add the following to your shell configuration file (~/.bashrc, ~/.zshrc, etc.):"
  echo ""
  echo "# Pistacho user-level configuration"
  echo "export PITH_USER_BASE_DIR=\"$PITH_DATA_DIR\""
  echo "export PATH=\"\$PATH:\$HOME/.local/bin\""
  echo "export LD_LIBRARY_PATH=\"\$HOME/.local/lib:\$LD_LIBRARY_PATH\""
  echo ""
}

# Main execution
main() {
  echo "================================================================================"
  echo "Pistacho User-Level Setup"
  echo "================================================================================"
  echo ""

  if [[ "$SHELL_CONFIG_ONLY" == true ]]; then
    update_shell_config
    return
  fi

  check_prerequisites
  create_directories
  create_config
  update_shell_config

  # Print summary
  echo "================================================================================"
  echo "Setup Complete!"
  echo "================================================================================"
  echo ""
  log_info "User-level configuration:"
  echo "  Config:   $PITH_CONFIG_FILE"
  echo "  Data:     $PITH_DATA_DIR"
  echo "  Symlinks: $PITH_BIN_DIR"
  echo ""
  log_info "Next steps:"
  echo "  1. Reload your shell: source $SHELL_RC"
  echo "  2. Install packages: pith install <package>"
  echo "  3. List packages:    pith list"
  echo "  4. Remove packages:  pith remove <package>"
  echo ""
  log_info "To use pith:"
  echo "  pith --help"
  echo ""
}

main "$@"

#!/bin/sh
# gtk-ai installer
# Usage: curl -sSL https://raw.githubusercontent.com/jmeiracorbal/gtk-ai/main/install.sh | sh
#
# Agent selection (default: auto-detect installed compatible agents):
#   sh -s -- --agent=auto
#   sh -s -- --agent=claudecode
#   sh -s -- --agent=cursor
#   sh -s -- --agent=codex
#   sh -s -- --agent=opencode
#   sh -s -- --agent=all
#
# To skip binary install (configure agents only):
#   GTKAI_CLAUDE_ONLY=1 sh install.sh
#   GTKAI_SKIP_BINARY=1 sh install.sh -- --agent=cursor
#
# Environment:
#   GTKAI_AGENT=cursor
#   GTKAI_SCRIPTS_DIR=/path/to/scripts
#   GTKAI_INSTALL_DIR=$HOME/.local/bin
#   GTKAI_DRY_RUN=true

set -e

REPO="jmeiracorbal/gtk-ai"
BINARY="gtkai"
INSTALL_DIR="${GTKAI_INSTALL_DIR:-$HOME/.local/bin}"
SKIP_BINARY="${GTKAI_SKIP_BINARY:-${GTKAI_CLAUDE_ONLY:-}}"
DRY_RUN="${GTKAI_DRY_RUN:-false}"
AGENT="${GTKAI_AGENT:-auto}"
if [ -n "${GTKAI_CLAUDE_ONLY:-}" ] && [ -z "${GTKAI_AGENT:-}" ]; then
  AGENT=claudecode
fi
TMP_DIR=$(mktemp -d)
TMP_SCRIPTS=""
GTKAI_BIN=""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { printf "${BLUE}  →${RESET} %s\n" "$1"; }
success() { printf "${GREEN}  ✓${RESET} %s\n" "$1"; }
warn()    { printf "${YELLOW}  ⚠${RESET} %s\n" "$1"; }
error()   { printf "${RED}  ✗${RESET} %s\n" "$1" >&2; exit 1; }
header()  { printf "\n${BOLD}%s${RESET}\n" "$1"; }

printf "${BOLD}"
cat <<'EOF'
   gtk-ai — rule-based output filtering for coding agents
EOF
printf "${RESET}\n"

header "Detecting system"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       error "Unsupported architecture: $ARCH" ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) error "Unsupported OS: $OS" ;;
esac

info "OS:   $OS"
info "Arch: $ARCH"

header "Checking dependencies"

HAS_GO=false
HAS_CURL=false
HAS_WGET=false

command -v go    >/dev/null 2>&1 && HAS_GO=true   && success "Go found: $(go version | awk '{print $3}')"
command -v curl  >/dev/null 2>&1 && HAS_CURL=true && success "curl found"
command -v wget  >/dev/null 2>&1 && HAS_WGET=true
command -v git   >/dev/null 2>&1 || error "git is required. Install it and retry."

fetch() {
  url="$1"
  dest="$2"
  if $HAS_CURL; then
    curl -sSL "$url" -o "$dest"
  elif $HAS_WGET; then
    wget -q "$url" -O "$dest"
  else
    error "Neither curl nor wget found. Install one and retry."
  fi
}

fetch_stdout() {
  url="$1"
  if $HAS_CURL; then
    curl -sSL "$url"
  elif $HAS_WGET; then
    wget -q "$url" -O -
  else
    error "Neither curl nor wget found. Install one and retry."
  fi
}

probe_url() {
  url="$1"
  if $HAS_CURL; then
    curl -sSfI "$url" >/dev/null 2>&1
  elif $HAS_WGET; then
    wget -q --spider "$url" >/dev/null 2>&1
  else
    return 1
  fi
}

json_merge() {
  patch="$1"
  file="$2"
  printf '%s' "$patch" | "$GTKAI_BIN" json-merge "$file"
}

append_marked_block() {
  dest="$1"
  src="$2"
  start="<!-- gtk-ai:start -->"
  end="<!-- gtk-ai:end -->"
  mkdir -p "$(dirname "$dest")"
  if [ -f "$dest" ] && grep -qF "$start" "$dest"; then
    info "$dest — gtk-ai block already present"
    return
  fi
  {
    [ -f "$dest" ] && cat "$dest"
    printf '\n%s\n' "$start"
    cat "$src"
    printf '%s\n' "$end"
  } > "$dest.tmp"
  mv "$dest.tmp" "$dest"
  success "$dest updated"
}

resolve_binary() {
  if [ -x "$INSTALL_DIR/$BINARY" ]; then
    GTKAI_BIN="$INSTALL_DIR/$BINARY"
    return
  fi
  if command -v "$BINARY" >/dev/null 2>&1; then
    GTKAI_BIN=$(command -v "$BINARY")
    return
  fi
  error "$BINARY not found. Install it first or unset GTKAI_SKIP_BINARY."
}

if [ -n "$SKIP_BINARY" ]; then
  header "Skipping binary install (GTKAI_SKIP_BINARY/GTKAI_CLAUDE_ONLY)"
  resolve_binary
  INSTALLED_VERSION=$("$GTKAI_BIN" version | awk '{print $2}')
  success "$BINARY $INSTALLED_VERSION found ($GTKAI_BIN)"
else
  header "Installing $BINARY"

  mkdir -p "$INSTALL_DIR"

  ASSET_NAME="${BINARY}-${OS}-${ARCH}"
  RELEASE_URL="https://github.com/$REPO/releases/latest/download/$ASSET_NAME"
  CHECKSUM_URL="https://github.com/$REPO/releases/latest/download/${ASSET_NAME}.sha256"

  if $HAS_CURL || $HAS_WGET; then
    info "Trying pre-built binary..."

    HTTP_CODE=0
    if $HAS_CURL; then
      HTTP_CODE=$(curl -sSL -o "$TMP_DIR/$BINARY" -w "%{http_code}" "$RELEASE_URL" 2>/dev/null || echo 0)
    elif $HAS_WGET; then
      wget -q "$RELEASE_URL" -O "$TMP_DIR/$BINARY" 2>/dev/null && HTTP_CODE=200 || HTTP_CODE=0
    fi

    if [ "$HTTP_CODE" = "200" ]; then
      info "Verifying checksum..."
      EXPECTED=$(fetch_stdout "$CHECKSUM_URL" | awk '{print $1}')
      if [ -z "$EXPECTED" ]; then
        if [ "${GTKAI_SKIP_CHECKSUM:-}" = "1" ]; then
          warn "Could not fetch checksum — proceeding because GTKAI_SKIP_CHECKSUM=1"
        else
          error "Could not fetch checksum. Set GTKAI_SKIP_CHECKSUM=1 to bypass."
        fi
      else
        if command -v shasum >/dev/null 2>&1; then
          ACTUAL=$(shasum -a 256 "$TMP_DIR/$BINARY" | awk '{print $1}')
        elif command -v sha256sum >/dev/null 2>&1; then
          ACTUAL=$(sha256sum "$TMP_DIR/$BINARY" | awk '{print $1}')
        else
          error "No SHA256 tool found (shasum/sha256sum). Set GTKAI_SKIP_CHECKSUM=1 to bypass."
        fi

        if [ "$ACTUAL" != "$EXPECTED" ]; then
          error "Checksum mismatch. Expected: $EXPECTED  Got: $ACTUAL"
        fi
        success "Checksum verified"
      fi

      mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
      chmod +x "$INSTALL_DIR/$BINARY"
      success "Binary downloaded from GitHub releases"
    else
      info "No pre-built binary found — building from source"

      if ! $HAS_GO; then
        error "Go is required to build from source. Install it from https://go.dev/dl/ and retry."
      fi

      info "Cloning repository..."
      git clone --depth 1 "https://github.com/$REPO.git" "$TMP_DIR/gtk-ai" >/dev/null 2>&1

      info "Building $BINARY..."
      cd "$TMP_DIR/gtk-ai"
      go build -o "$INSTALL_DIR/$BINARY" ./cmd/gtkai/
      cd - >/dev/null
      success "Built from source"
      if [ -d "$TMP_DIR/gtk-ai/scripts/cursor/hooks" ]; then
        TMP_SCRIPTS="$TMP_DIR/gtk-ai/scripts"
      fi
    fi
  fi

  GTKAI_BIN="$INSTALL_DIR/$BINARY"
  if ! "$GTKAI_BIN" version >/dev/null 2>&1; then
    error "Binary installed but failed to run. Check $GTKAI_BIN"
  fi

  INSTALLED_VERSION=$("$GTKAI_BIN" version | awk '{print $2}')
  success "$BINARY $INSTALLED_VERSION installed to $GTKAI_BIN"

  header "Configuring PATH"

  add_to_path() {
    shell_rc="$1"
    if [ -f "$shell_rc" ]; then
      if ! grep -q "$INSTALL_DIR" "$shell_rc" 2>/dev/null; then
        printf '\n# gtk-ai\nexport PATH="%s:$PATH"\n' "$INSTALL_DIR" >> "$shell_rc"
        success "Added $INSTALL_DIR to PATH in $shell_rc"
      else
        info "$INSTALL_DIR already in $shell_rc"
      fi
    fi
  }

  case "$SHELL" in
    */zsh)  add_to_path "$HOME/.zshrc"  ;;
    */bash) add_to_path "$HOME/.bashrc" ;;
    *)      add_to_path "$HOME/.profile" ;;
  esac

  export PATH="$INSTALL_DIR:$PATH"
fi

resolve_scripts() {
  if [ -n "$TMP_SCRIPTS" ]; then
    return
  fi
  if [ -n "${GTKAI_SCRIPTS_DIR:-}" ]; then
    TMP_SCRIPTS="$GTKAI_SCRIPTS_DIR"
    return
  fi
  if [ -f "$0" ] && [ -d "$(dirname "$0")/scripts/cursor/hooks" ]; then
    TMP_SCRIPTS="$(cd "$(dirname "$0")" && pwd)/scripts"
    return
  fi

  header "Fetching agent scripts"
  archive_url="https://github.com/$REPO/releases/latest/download/gtkai-scripts.tar.gz"
  checksum_url="https://github.com/$REPO/releases/latest/download/gtkai-scripts.tar.gz.sha256"
  if probe_url "$archive_url"; then
    tmp_archive=$(mktemp)
    fetch "$archive_url" "$tmp_archive" || error "Scripts archive download failed"
    expected=$(fetch_stdout "$checksum_url" | awk '{print $1}')
    if [ -z "$expected" ]; then
      error "Could not fetch scripts checksum"
    fi
    if command -v shasum >/dev/null 2>&1; then
      actual=$(shasum -a 256 "$tmp_archive" | awk '{print $1}')
    else
      actual=$(sha256sum "$tmp_archive" | awk '{print $1}')
    fi
    if [ "$actual" != "$expected" ]; then
      error "Scripts checksum mismatch. Expected: $expected  Got: $actual"
    fi
    TMP_SCRIPTS=$(mktemp -d)
    tar -xzf "$tmp_archive" -C "$TMP_SCRIPTS" --strip-components=1
    rm -f "$tmp_archive"
    success "Scripts ready"
    return
  fi

  info "No scripts archive in the latest release — cloning repository"
  git clone --depth 1 "https://github.com/$REPO.git" "$TMP_DIR/gtk-ai-scripts" >/dev/null 2>&1
  TMP_SCRIPTS="$TMP_DIR/gtk-ai-scripts/scripts"
}

setup_claudecode() {
  header "Configuring Claude Code"

  CLAUDE_DIR="$HOME/.claude"
  SETTINGS_FILE="$CLAUDE_DIR/settings.json"
  KNOWN_MARKETPLACES="$CLAUDE_DIR/plugins/known_marketplaces.json"
  MARKETPLACE_DIR="$CLAUDE_DIR/plugins/marketplaces/gtk-ai"
  PROTOCOL_DOC="$CLAUDE_DIR/gtk-ai.md"
  CLAUDE_MD="$CLAUDE_DIR/CLAUDE.md"

  mkdir -p "$CLAUDE_DIR/plugins/marketplaces"

  result=$(json_merge '{"extraKnownMarketplaces":{"gtk-ai":{"source":{"source":"github","repo":"jmeiracorbal/gtk-ai"}}}}' "$SETTINGS_FILE")
  success "$HOME/.claude/settings.json — $result"

  if [ -d "$MARKETPLACE_DIR/.git" ]; then
    info "Marketplace cache exists — updating..."
    git -C "$MARKETPLACE_DIR" pull --ff-only -q 2>/dev/null || warn "Could not update marketplace cache"
    if git -C "$MARKETPLACE_DIR" checkout "v$INSTALLED_VERSION" -q 2>/dev/null; then
      success "$HOME/.claude/plugins/marketplaces/gtk-ai pinned to v$INSTALLED_VERSION"
    else
      warn "Tag v$INSTALLED_VERSION not found — using default branch"
    fi
  else
    info "Cloning marketplace cache..."
    if git clone --depth 1 --branch "v$INSTALLED_VERSION" "https://github.com/$REPO.git" "$MARKETPLACE_DIR" >/dev/null 2>&1; then
      success "$HOME/.claude/plugins/marketplaces/gtk-ai cloned at v$INSTALLED_VERSION"
    elif git clone --depth 1 "https://github.com/$REPO.git" "$MARKETPLACE_DIR" >/dev/null 2>&1; then
      warn "Tag v$INSTALLED_VERSION not found — using default branch"
    else
      error "Failed to clone marketplace repository"
    fi
  fi

  NOW=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")
  result=$(json_merge "$(printf '{"gtk-ai":{"source":{"source":"github","repo":"jmeiracorbal/gtk-ai"},"installLocation":"%s","lastUpdated":"%s"}}' "$MARKETPLACE_DIR" "$NOW")" "$KNOWN_MARKETPLACES")
  success "$HOME/.claude/plugins/known_marketplaces.json — $result"

  cp "$TMP_SCRIPTS/claudecode/gtk-ai.md" "$PROTOCOL_DOC"
  success "$HOME/.claude/gtk-ai.md written"

  if [ -f "$CLAUDE_MD" ]; then
    if grep -q "@gtk-ai.md" "$CLAUDE_MD" 2>/dev/null; then
      info "$HOME/.claude/CLAUDE.md — already up to date"
    else
      printf '\n@gtk-ai.md\n' >> "$CLAUDE_MD"
      success "$HOME/.claude/CLAUDE.md updated"
    fi
  else
    printf '@gtk-ai.md\n' > "$CLAUDE_MD"
    success "$HOME/.claude/CLAUDE.md created"
  fi

  printf "To activate the Claude plugin, run:\n\n"
  printf "  ${BOLD}%s${RESET}\n\n" "claude plugin install -s user gtk-ai@gtk-ai"
  printf "Then restart Claude Code.\n"
}

setup_cursor() {
  header "Configuring Cursor"

  hooks_dir="$HOME/.cursor/hooks"
  hooks_json="$HOME/.cursor/hooks.json"
  rules_dir="$HOME/.cursor/rules"

  mkdir -p "$hooks_dir" "$rules_dir"
  cp "$TMP_SCRIPTS/cursor/hooks/gtkai-pre-tool-use.sh" "$hooks_dir/"
  cp "$TMP_SCRIPTS/cursor/hooks/gtkai-post-tool-use.sh" "$hooks_dir/"
  chmod +x "$hooks_dir/gtkai-pre-tool-use.sh" "$hooks_dir/gtkai-post-tool-use.sh"
  success "Hook scripts installed to ${hooks_dir}"

  patch=$(printf '{"version":1,"hooks":{"preToolUse":[{"command":"%s/gtkai-pre-tool-use.sh","matcher":"Shell"}],"postToolUse":[{"command":"%s/gtkai-post-tool-use.sh","matcher":"MCP:.*"}]}}' "$hooks_dir" "$hooks_dir")
  result=$(json_merge "$patch" "$hooks_json")
  success "$HOME/.cursor/hooks.json — $result"

  cp "$TMP_SCRIPTS/cursor/rules/gtk-ai.mdc" "$rules_dir/gtk-ai.mdc"
  success "$HOME/.cursor/rules/gtk-ai.mdc written"
}

setup_codex() {
  header "Configuring Codex"

  hooks_dir="$HOME/.codex/hooks"
  hooks_json="$HOME/.codex/hooks.json"
  codex_config="$HOME/.codex/config.toml"
  agents_md="$HOME/.codex/AGENTS.md"

  mkdir -p "$hooks_dir"
  cp "$TMP_SCRIPTS/codex/hooks/gtkai-pre-tool-use.sh" "$hooks_dir/"
  chmod +x "$hooks_dir/gtkai-pre-tool-use.sh"
  success "Hook scripts installed to ${hooks_dir}"

  patch=$(printf '{"hooks":{"PreToolUse":[{"matcher":"Bash|shell|local_shell|container_exec|exec_command|shell_command","hooks":[{"type":"command","command":"%s/gtkai-pre-tool-use.sh","statusMessage":"gtkai rewrite","timeout":10}]}]}}' "$hooks_dir")
  result=$(json_merge "$patch" "$hooks_json")
  success "$HOME/.codex/hooks.json — $result"

  mkdir -p "$HOME/.codex"
  touch "$codex_config"
  if grep -q 'codex_hooks' "$codex_config" 2>/dev/null; then
    info "$HOME/.codex/config.toml — codex_hooks already set"
  elif grep -q '^\[features\]' "$codex_config" 2>/dev/null; then
    tmp_cfg=$(mktemp)
    awk '/^\[features\]/{print; print "codex_hooks = true"; next} {print}' "$codex_config" > "$tmp_cfg"
    mv "$tmp_cfg" "$codex_config"
    success "$HOME/.codex/config.toml — enabled [features].codex_hooks"
  else
    tail -c1 "$codex_config" 2>/dev/null | grep -q $'\n' || printf '\n' >> "$codex_config"
    printf '\n[features]\ncodex_hooks = true\n' >> "$codex_config"
    success "$HOME/.codex/config.toml — enabled [features].codex_hooks"
  fi

  append_marked_block "$agents_md" "$TMP_SCRIPTS/codex/AGENTS.md"
}

setup_opencode() {
  header "Configuring OpenCode"

  plugins_dir="$HOME/.config/opencode/plugins"
  agents_md="$HOME/.config/opencode/AGENTS.md"

  mkdir -p "$plugins_dir"
  cp "$TMP_SCRIPTS/opencode/plugins/gtkai.ts" "$plugins_dir/"
  success "Plugin installed to ${plugins_dir}"

  append_marked_block "$agents_md" "$TMP_SCRIPTS/opencode/AGENTS.md"
}

agent_detected() {
  case "$1" in
    claudecode)
      command -v claude >/dev/null 2>&1 || [ -d "$HOME/.claude" ] || [ -f "$HOME/.claude.json" ]
      ;;
    cursor)
      command -v cursor >/dev/null 2>&1 || [ -d "$HOME/.cursor" ]
      ;;
    codex)
      command -v codex >/dev/null 2>&1 || [ -d "$HOME/.codex" ]
      ;;
    opencode)
      command -v opencode >/dev/null 2>&1 || [ -d "$HOME/.config/opencode" ]
      ;;
    *) return 1 ;;
  esac
}

detect_agents() {
  found=""
  for agent in claudecode cursor codex opencode; do
    if agent_detected "$agent"; then
      found="$found $agent"
    fi
  done
  printf '%s' "$found"
}

setup_agent() {
  agent="$1"
  case "$agent" in
    claudecode) setup_claudecode ;;
    cursor) setup_cursor ;;
    codex) setup_codex ;;
    opencode) setup_opencode ;;
    *) error "Unknown agent: ${agent}. Valid options: auto | claudecode | cursor | codex | opencode | all" ;;
  esac
}

warn_rtk() {
  if command -v rtk >/dev/null 2>&1; then
    warn "RTK is installed. To avoid conflicts, remove its hooks from agent settings"
    warn "Look for entries referencing rtk-rewrite.sh or rtk-post-tool-use.sh"
  fi
}

for arg in "$@"; do
  case "$arg" in
    --agent=*) AGENT="${arg#--agent=}" ;;
  esac
done

if [ "$DRY_RUN" = "true" ]; then
  info "Dry-run mode — no agent files will be written"
  info "Would configure agent=${AGENT}"
  success "Done (dry-run)."
  rm -rf "$TMP_DIR"
  exit 0
fi

resolve_scripts

case "$AGENT" in
  auto)
    detected=$(detect_agents)
    if [ -z "$detected" ]; then
      error "No compatible agent detected. Pass --agent=claudecode|cursor|codex|opencode|all"
    fi
    info "Auto-detected agents:${detected}"
    for selected in $detected; do
      setup_agent "$selected"
    done
    ;;
  all)
    for selected in claudecode cursor codex opencode; do
      setup_agent "$selected"
    done
    ;;
  claudecode|cursor|codex|opencode)
    setup_agent "$AGENT"
    ;;
  *)
    error "Unknown agent: ${AGENT}. Valid options: auto | claudecode | cursor | codex | opencode | all"
    ;;
esac

install_official_filters() {
  if [ "${GTKAI_SKIP_FILTERS:-}" = "1" ]; then
    warn "Skipping official filter install (GTKAI_SKIP_FILTERS=1)"
    return
  fi

  header "Installing official filters"

  OFFICIAL_JSON=""
  if [ -f "$(dirname "$0")/filters/official.json" ]; then
    OFFICIAL_JSON="$(cd "$(dirname "$0")" && pwd)/filters/official.json"
  elif [ -n "$TMP_DIR" ] && [ -f "$TMP_DIR/gtk-ai/filters/official.json" ]; then
    OFFICIAL_JSON="$TMP_DIR/gtk-ai/filters/official.json"
  else
    warn "official filters manifest not found — skipping filter install"
    return
  fi

  INSTALL_OFFICIAL_ARGS="--core-version=$INSTALLED_VERSION"

  if "$GTKAI_BIN" filter install-official "$OFFICIAL_JSON" $INSTALL_OFFICIAL_ARGS; then
    success "Official filters installed"
  else
    error "Official filter installation failed"
  fi
}

warn_rtk
install_official_filters
rm -rf "$TMP_DIR"

header "Done"
success "gtk-ai $INSTALLED_VERSION configured for agent=${AGENT}"
printf "Restart the agent after installation.\n\n"

#!/bin/sh
set -eu

PACKAGE_ROOT=$(CDPATH= cd -P -- "$(dirname -- "$0")" && pwd)
BIN_DIR=${HOME:?}/.local/bin
SKILLS_DIR=${CODEX_HOME:-$HOME/.codex}/skills
ADD_TO_PATH=0
REPLACE_SHARED=0
VERIFY_ONLY=0

usage() {
  printf '%s\n' 'Usage: ./install.sh [--bin-dir DIR] [--skills-dir DIR] [--add-to-path] [--replace-shared-skill] [--verify-only]'
}
while [ "$#" -gt 0 ]; do
  case "$1" in
    --bin-dir) [ "$#" -ge 2 ] || { usage; exit 2; }; BIN_DIR=$2; shift 2 ;;
    --skills-dir) [ "$#" -ge 2 ] || { usage; exit 2; }; SKILLS_DIR=$2; shift 2 ;;
    --add-to-path) ADD_TO_PATH=1; shift ;;
    --replace-shared-skill) REPLACE_SHARED=1; shift ;;
    --verify-only) VERIFY_ONLY=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) usage; exit 2 ;;
  esac
done

if command -v sha256sum >/dev/null 2>&1; then hash_tool=sha256sum
elif command -v shasum >/dev/null 2>&1; then hash_tool=shasum
elif command -v openssl >/dev/null 2>&1; then hash_tool=openssl
else printf '%s\n' 'A SHA256 tool (sha256sum, shasum or openssl) is required.' >&2; exit 1
fi

count=0
# Reject symlinks once for the package tree instead of stat-ing every source
# file from the shell; the checksum utility also reports missing files.
if find "$PACKAGE_ROOT" -type l -print | grep -q .; then
  printf '%s\n' 'Package contains a symbolic link; refusing to install.' >&2
  exit 1
fi
while IFS= read -r line || [ -n "$line" ]; do
  [ -n "$line" ] || continue
  expected=${line%%  *}
  relative=${line#*  }
  [ "${#expected}" -eq 64 ] || { printf '%s\n' 'Invalid checksum manifest.' >&2; exit 1; }
  case "$expected" in *[!0-9a-f]*) printf '%s\n' 'Invalid checksum manifest.' >&2; exit 1 ;; esac
  case "$relative" in ''|/*|../*|*/../*|*/..) printf '%s\n' 'Unsafe checksum path.' >&2; exit 1 ;; esac
  file=$PACKAGE_ROOT/$relative
  # Common SHA256 utilities can check the whole manifest in one process. This
  # avoids spawning thousands of subprocesses when full source is included.
  if [ "$hash_tool" = openssl ]; then
    [ -f "$file" ] || { printf 'Missing file: %s\n' "$relative" >&2; exit 1; }
    actual=$(openssl dgst -sha256 "$file" | awk '{print $NF}')
    [ "$actual" = "$expected" ] || { printf 'Checksum mismatch: %s\n' "$relative" >&2; exit 1; }
  fi
  count=$((count + 1))
done < "$PACKAGE_ROOT/SHA256SUMS"
[ "$count" -ge 10 ] || { printf '%s\n' 'Incomplete package manifest.' >&2; exit 1; }
if [ "$hash_tool" = sha256sum ]; then
  (cd "$PACKAGE_ROOT" && sha256sum -c SHA256SUMS >/dev/null) || { printf '%s\n' 'Checksum mismatch.' >&2; exit 1; }
elif [ "$hash_tool" = shasum ]; then
  (cd "$PACKAGE_ROOT" && shasum -a 256 -c SHA256SUMS >/dev/null) || { printf '%s\n' 'Checksum mismatch.' >&2; exit 1; }
fi
printf 'Verified %s package files.\n' "$count"
[ "$VERIFY_ONLY" -eq 0 ] || exit 0

case "$(uname -s)" in Darwin) system=darwin ;; Linux) system=linux ;; *) printf '%s\n' 'Use install.ps1 on Windows. This script supports macOS and Linux.' >&2; exit 1 ;; esac
case "$(uname -m)" in x86_64|amd64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; *) printf '%s\n' 'This kit contains x64 and ARM64 binaries only.' >&2; exit 1 ;; esac
case "$BIN_DIR" in /*) ;; *) printf '%s\n' '--bin-dir must be absolute.' >&2; exit 2 ;; esac
case "$SKILLS_DIR" in /*) ;; *) printf '%s\n' '--skills-dir must be absolute.' >&2; exit 2 ;; esac
mkdir -p "$SKILLS_DIR"
SKILLS_DIR=$(CDPATH= cd -P -- "$SKILLS_DIR" && pwd)
if [ "$system" = darwin ]; then
  command -v python3 >/dev/null 2>&1 || { echo 'Python 3.8+ is required.' >&2; exit 1; }
  python3 -c 'import sys; sys.exit(sys.version_info < (3, 8))' || { echo 'Python 3.8+ is required.' >&2; exit 1; }
  backup=$SKILLS_DIR/.feishu-backups/$(date -u +%Y%m%d-%H%M%S)-$$
else
  [ -f "$PACKAGE_ROOT/bin/$system-$arch/lark-cli" ] || { echo 'Select the Linux package matching this architecture.' >&2; exit 1; }
  [ ! -L "$BIN_DIR/lark-cli" ] && [ ! -L "$BIN_DIR/feishu-credential-export" ] || { printf '%s\n' 'Refusing to replace a symlinked executable.' >&2; exit 1; }
  mkdir -p "$BIN_DIR"
  BIN_DIR=$(CDPATH= cd -P -- "$BIN_DIR" && pwd)
  backup=$BIN_DIR/backups/$(date -u +%Y%m%d-%H%M%S)-$$
  for executable in lark-cli feishu-credential-export; do
    if [ -e "$BIN_DIR/$executable" ]; then
      [ -f "$BIN_DIR/$executable" ] || { printf '%s\n' 'Existing executable is not a regular file.' >&2; exit 1; }
      mkdir -p "$backup"
      cp -p "$BIN_DIR/$executable" "$backup/$executable"
    fi
    cp "$PACKAGE_ROOT/bin/$system-$arch/$executable" "$BIN_DIR/$executable"
    chmod 755 "$BIN_DIR/$executable"
  done
fi

for name in feishu-enterprise-app-setup lark-shared; do
  target=$SKILLS_DIR/$name
  if [ -e "$target" ]; then
    [ -d "$target" ] || { printf 'Skill path is not a directory: %s\n' "$name" >&2; exit 1; }
    if [ "$name" = lark-shared ] && [ "$REPLACE_SHARED" -eq 0 ]; then
      printf '%s\n' 'Keeping the existing lark-shared skill.'
      continue
    fi
    [ ! -L "$target" ] || { printf 'Refusing to update a symlinked skill: %s\n' "$name" >&2; exit 1; }
    mkdir -p "$backup"
    cp -R "$target" "$backup/$name"
  else mkdir -p "$target"
  fi
  cp -R "$PACKAGE_ROOT/skills/$name/." "$target/"
done

if [ "$system" = darwin ]; then
  script=$SKILLS_DIR/feishu-enterprise-app-setup/scripts/feishu_app.py
  python3 -B "$script" preview >/dev/null
  printf 'Skills: %s\nRun: python3 "%s" preview\n' "$SKILLS_DIR" "$script"
  exit 0
fi

# Use a fresh config location for local checks; never bind or authorize an app.
check_dir=$(mktemp -d "${TMPDIR:-/tmp}/feishu-app-kit-check.XXXXXXXX")
run_check() (
  for key in $(env | sed -n 's/=.*//p' | LC_ALL=C grep '^LARKSUITE_CLI_[A-Za-z0-9_]*$'); do unset "$key"; done
  unset HERMES_HOME OPENCLAW_HOME
  export LARKSUITE_CLI_CONFIG_DIR="$check_dir" LARKSUITE_CLI_LOG_DIR="$check_dir/logs"
  export LARKSUITE_CLI_NO_UPDATE_NOTIFIER=1 LARKSUITE_CLI_NO_SKILLS_NOTIFIER=1
  "$BIN_DIR/lark-cli" "$@"
)
version=$(run_check --version)
[ "$version" = 'lark-cli version 1.0.100-docs7.2' ] || { printf 'Unexpected version: %s\n' "$version" >&2; exit 1; }
"$BIN_DIR/feishu-credential-export" --help >/dev/null 2>&1
run_check config init --new --dry-run > "$check_dir/preview.json"
grep -q '"permission_preset": "docs7-managed"' "$check_dir/preview.json" || { printf '%s\n' 'Unexpected permission preset.' >&2; exit 1; }
[ ! -f "$check_dir/config.json" ] || { printf '%s\n' 'The dry-run unexpectedly wrote a binding.' >&2; exit 1; }
rm -f "$check_dir/preview.json"
rmdir "$check_dir" 2>/dev/null || true

quote_shell() { printf "'"; printf '%s' "$1" | sed "s/'/'\\\\''/g"; printf "'"; }
if [ "$ADD_TO_PATH" -eq 1 ]; then
  shell_name=${SHELL:-sh}
  case "${shell_name##*/}" in
    zsh) rc=$HOME/.zshrc ;;
    bash) if [ "$system" = darwin ]; then rc=$HOME/.bash_profile; else rc=$HOME/.bashrc; fi ;;
    sh|dash|'') rc=$HOME/.profile ;;
    *) rc='' ;;
  esac
  if [ -n "$rc" ]; then
    path_line=$(printf 'export PATH=%s:"$PATH"' "$(quote_shell "$BIN_DIR")")
    if [ ! -f "$rc" ] || ! grep -Fqx "$path_line" "$rc"; then
      printf '\n# Feishu App Kit\n%s\n' "$path_line" >> "$rc"
    fi
    printf 'PATH entry added to %s. Open a new terminal or source that file.\n' "$rc"
  else printf 'Add this directory to your shell PATH manually: %s\n' "$BIN_DIR"
  fi
fi
printf '%s\nCLI: %s\nCredential export: %s\nSkills: %s\n' "$version" "$BIN_DIR/lark-cli" "$BIN_DIR/feishu-credential-export" "$SKILLS_DIR"
[ ! -d "$backup" ] || printf 'Previous files backed up: %s\n' "$backup"
printf '%s\n' 'Install checks passed. No app was created and no account was authorized.'
[ "$ADD_TO_PATH" -eq 1 ] || printf '%s\n' 'Use the full CLI path above, or rerun with --add-to-path.'

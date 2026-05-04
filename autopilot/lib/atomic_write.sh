#!/usr/bin/env bash
# atomic_write.sh — atomic file write with .bak rotation.
#
# Usage:
#   atomic_write.sh <target_path> <content_source>
#     content_source: "-" for stdin, OR existing file path
#
# Behaviour:
#   1. Write content to <target>.tmp
#   2. (Caller is responsible for schema validation BEFORE calling this script,
#      OR validation can be plugged via $AUTOPILOT_VALIDATE_CMD env var.)
#   3. If target exists → mv target target.bak
#   4. mv target.tmp target
#
# Rationale: if validation plugged via env, set AUTOPILOT_VALIDATE_CMD="ajv ..."
# Default mode: no validation (caller validates).

set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <target_path> <content_source|->" >&2
  exit 2
fi

target="$1"
src="$2"

target_dir="$(dirname "$target")"
mkdir -p "$target_dir"

tmp="${target}.tmp.$$"
bak="${target}.bak"

cleanup_tmp() { [[ -f "$tmp" ]] && rm -f "$tmp"; }
trap cleanup_tmp EXIT

if [[ "$src" == "-" ]]; then
  cat > "$tmp"
else
  if [[ ! -f "$src" ]]; then
    echo "ERROR: source file not found: $src" >&2
    exit 3
  fi
  cp "$src" "$tmp"
fi

# Optional schema validation hook
if [[ -n "${AUTOPILOT_VALIDATE_CMD:-}" ]]; then
  if ! eval "$AUTOPILOT_VALIDATE_CMD '$tmp'"; then
    echo "ERROR: validation failed for $tmp" >&2
    exit 4
  fi
fi

# Rotate previous version
if [[ -f "$target" ]]; then
  mv -f "$target" "$bak"
fi

# Atomic rename
mv -f "$tmp" "$target"
trap - EXIT

echo "OK ${target} (backup: ${bak} if existed)"

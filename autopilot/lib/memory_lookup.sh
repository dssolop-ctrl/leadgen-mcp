#!/usr/bin/env bash
# memory_lookup.sh — grep по тегам в narrative md.
#
# Usage:
#   memory_lookup.sh <city> <tag_query>
#   memory_lookup.sh omsk "campaign:8765432"
#   memory_lookup.sh omsk "topic:vtorichka"
#   memory_lookup.sh omsk "type:incident"
#
# Tag format в шапке файлов: tags: [campaign:NNN] [topic:T] [channel:C] [city:CITY]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUNTIME_ROOT="${SCRIPT_DIR}/../runtime"

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <city> <tag_query>" >&2
  echo "examples: $0 omsk \"campaign:8765432\"  $0 omsk \"topic:vtorichka\"" >&2
  exit 2
fi

city="$1"
query="$2"

narrative="${RUNTIME_ROOT}/${city}/narrative"
if [[ ! -d "$narrative" ]]; then
  echo "(no narrative dir for ${city} at ${narrative})" >&2
  exit 0
fi

# Escape brackets for grep
pattern="\[${query}\]"

echo "# Files matching tag [${query}] for ${city}:"
grep -rIl --include="*.md" "$pattern" "$narrative" 2>/dev/null || true

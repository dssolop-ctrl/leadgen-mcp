#!/usr/bin/env bash
# telegram_send_doc.sh — отправить файл (HTML report, текстовый файл) в Telegram.
#
# Usage:
#   telegram_send_doc.sh <chat_id> <file_path> [filename] [--caption "text"]
#
# If <file_path> is "-", reads from stdin and uploads as <filename>.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECRETS="${SCRIPT_DIR}/../config/secrets.env"

if [[ -f "$SECRETS" ]]; then
  set -a; source "$SECRETS"; set +a
fi

: "${TELEGRAM_BOT_TOKEN:?TELEGRAM_BOT_TOKEN not set}"
: "${TELEGRAM_ALLOWLIST_CHAT_IDS:?TELEGRAM_ALLOWLIST_CHAT_IDS not set}"
TELEGRAM_HTTP_TIMEOUT="${TELEGRAM_HTTP_TIMEOUT:-30}"

mask_token() { sed "s/${TELEGRAM_BOT_TOKEN}/***MASKED***/g"; }

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <chat_id> <file_path|-> [filename] [--caption text]" >&2
  exit 2
fi

chat_id="$1"
file_arg="$2"
filename="${3:-report.html}"

caption=""
if [[ "${4:-}" == "--caption" ]]; then
  caption="${5:-}"
fi

# Allowlist
if ! echo ",${TELEGRAM_ALLOWLIST_CHAT_IDS}," | grep -q ",${chat_id},"; then
  echo "ERROR: chat_id ${chat_id} not in allowlist" >&2
  exit 3
fi

# Resolve file
tmp_file=""
cleanup() { [[ -n "$tmp_file" && -f "$tmp_file" ]] && rm -f "$tmp_file"; }
trap cleanup EXIT

if [[ "$file_arg" == "-" ]]; then
  tmp_file="$(mktemp -t autopilot-tg-doc.XXXXXX)"
  cat > "$tmp_file"
  upload_path="$tmp_file"
else
  if [[ ! -f "$file_arg" ]]; then
    echo "ERROR: file not found: $file_arg" >&2
    exit 4
  fi
  upload_path="$file_arg"
fi

URL="https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendDocument"

attempt=0
max_attempts=3
backoff=2
while (( attempt < max_attempts )); do
  attempt=$((attempt + 1))
  curl_args=(
    -sS --max-time "$TELEGRAM_HTTP_TIMEOUT"
    -X POST "$URL"
    -F "chat_id=${chat_id}"
    -F "document=@${upload_path};filename=${filename}"
  )
  if [[ -n "$caption" ]]; then
    curl_args+=(-F "caption=${caption}")
  fi

  response="$(curl "${curl_args[@]}" 2>&1 | mask_token || true)"

  if echo "$response" | grep -q '"ok":true'; then
    message_id="$(echo "$response" | sed -n 's/.*"message_id":\([0-9]*\).*/\1/p' | head -1)"
    echo "OK message_id=${message_id}"
    exit 0
  fi

  echo "WARN attempt ${attempt}/${max_attempts}: ${response}" >&2
  if (( attempt < max_attempts )); then
    sleep $((backoff ** attempt))
  fi
done

echo "ERROR: telegram_send_doc failed after ${max_attempts} attempts" >&2
exit 1

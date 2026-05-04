#!/usr/bin/env bash
# telegram_send.sh — отправить текст в Telegram через Bot API.
#
# Usage:
#   telegram_send.sh <chat_id> <text_or_file> [--html|--markdown]
#   echo "hello" | telegram_send.sh <chat_id> -            # text from stdin
#
# Environment:
#   TELEGRAM_BOT_TOKEN          — required (loaded from autopilot/config/secrets.env)
#   TELEGRAM_ALLOWLIST_CHAT_IDS — required, comma-separated; chat_id must be in this list
#   TELEGRAM_HTTP_TIMEOUT       — default 15 sec
#
# Behaviour:
#   - retries 3 times with exponential backoff on transient errors
#   - escapes nothing automatically; caller must pass safe text
#   - if message > 4000 chars, falls back to telegram_send_doc.sh
#   - masks token in any log output

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECRETS="${SCRIPT_DIR}/../config/secrets.env"

if [[ -f "$SECRETS" ]]; then
  set -a; source "$SECRETS"; set +a
fi

: "${TELEGRAM_BOT_TOKEN:?TELEGRAM_BOT_TOKEN not set (check autopilot/config/secrets.env)}"
: "${TELEGRAM_ALLOWLIST_CHAT_IDS:?TELEGRAM_ALLOWLIST_CHAT_IDS not set}"
TELEGRAM_HTTP_TIMEOUT="${TELEGRAM_HTTP_TIMEOUT:-15}"

mask_token() {
  sed "s/${TELEGRAM_BOT_TOKEN}/***MASKED***/g"
}

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <chat_id> <text_or_'-'> [--html|--markdown]" >&2
  exit 2
fi

chat_id="$1"
text_arg="$2"
parse_mode=""
case "${3:-}" in
  --html) parse_mode="HTML" ;;
  --markdown) parse_mode="MarkdownV2" ;;
esac

# Allowlist check
if ! echo ",${TELEGRAM_ALLOWLIST_CHAT_IDS}," | grep -q ",${chat_id},"; then
  echo "ERROR: chat_id ${chat_id} not in allowlist" >&2
  exit 3
fi

# Resolve text source
if [[ "$text_arg" == "-" ]]; then
  text="$(cat)"
elif [[ -f "$text_arg" ]]; then
  text="$(cat "$text_arg")"
else
  text="$text_arg"
fi

# Telegram message limit ~ 4096 chars; leave headroom
text_len=${#text}
if (( text_len > 3900 )); then
  echo "INFO: text >${text_len} chars, falling back to sendDocument" >&2
  exec "${SCRIPT_DIR}/telegram_send_doc.sh" "$chat_id" "$text_arg" "long-message.txt"
fi

URL="https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage"

# Write text to temp file in raw UTF-8 (avoids Git Bash on Windows mangling stdin/argv encoding).
# curl reads file as bytes and posts as multipart form, preserving UTF-8 cleanly.
text_file="$(mktemp -t autopilot-tg-text.XXXXXX)"
trap 'rm -f "$text_file"' EXIT
printf '%s' "$text" > "$text_file"

attempt=0
max_attempts=3
backoff=2
while (( attempt < max_attempts )); do
  attempt=$((attempt + 1))
  curl_args=(
    -sS --max-time "$TELEGRAM_HTTP_TIMEOUT"
    -X POST "$URL"
    -F "chat_id=${chat_id}"
    -F "text=<${text_file}"
    -F "disable_web_page_preview=true"
  )
  if [[ -n "$parse_mode" ]]; then
    curl_args+=(-F "parse_mode=${parse_mode}")
  fi
  response="$(curl "${curl_args[@]}" 2>&1 | mask_token || true)"

  if echo "$response" | grep -q '"ok":true'; then
    # Extract message_id (best-effort, no jq required)
    message_id="$(echo "$response" | sed -n 's/.*"message_id":\([0-9]*\).*/\1/p' | head -1)"
    echo "OK message_id=${message_id}"
    exit 0
  fi

  echo "WARN attempt ${attempt}/${max_attempts}: ${response}" >&2
  if (( attempt < max_attempts )); then
    sleep $((backoff ** attempt))
  fi
done

echo "ERROR: telegram_send failed after ${max_attempts} attempts" >&2
exit 1

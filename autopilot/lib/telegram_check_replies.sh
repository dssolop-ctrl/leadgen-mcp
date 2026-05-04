#!/usr/bin/env bash
# telegram_check_replies.sh — getUpdates polling с persisted offset.
#
# Usage:
#   telegram_check_replies.sh <city>
#
# Output (stdout, JSONL):
#   {"update_id": 12345, "chat_id": -1001234567890, "message_id": 999, "text": "approve 1", "from_user": "..."}
#   ...
#
# Side effect: обновляет runtime/_global/telegram_offset до max(update_id) + 1.
#
# Caller (skill) разбирает строки и применяет к pending_approvals.yaml.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECRETS="${SCRIPT_DIR}/../config/secrets.env"
RUNTIME_ROOT="${SCRIPT_DIR}/../runtime"

if [[ -f "$SECRETS" ]]; then
  set -a; source "$SECRETS"; set +a
fi

: "${TELEGRAM_BOT_TOKEN:?TELEGRAM_BOT_TOKEN not set}"
: "${TELEGRAM_ALLOWLIST_CHAT_IDS:?TELEGRAM_ALLOWLIST_CHAT_IDS not set}"
TELEGRAM_HTTP_TIMEOUT="${TELEGRAM_HTTP_TIMEOUT:-15}"

mask() { sed "s/${TELEGRAM_BOT_TOKEN}/***MASKED***/g"; }

city="${1:?city required}"

# Persisted offset (global, not per-city — Telegram updates are global per bot)
offset_dir="${RUNTIME_ROOT}/_global"
offset_file="${offset_dir}/telegram_offset"
mkdir -p "$offset_dir"
offset="$(cat "$offset_file" 2>/dev/null || echo 0)"

URL="https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getUpdates"

response="$(curl -sS --max-time "$TELEGRAM_HTTP_TIMEOUT" \
  --get "$URL" \
  --data-urlencode "offset=${offset}" \
  --data-urlencode "timeout=0" \
  --data-urlencode "allowed_updates=[\"message\"]" 2>&1 | mask || true)"

if ! echo "$response" | grep -q '"ok":true'; then
  echo "ERROR: getUpdates failed: ${response}" >&2
  exit 1
fi

# Parse with python (jq might not be available)
python - <<PY
import json, sys, os
data = json.loads('''${response//\'/\\\'}''')
allow = set(s.strip() for s in os.environ.get("TELEGRAM_ALLOWLIST_CHAT_IDS","").split(","))
max_uid = ${offset} - 1 if ${offset} > 0 else -1
for upd in data.get("result", []):
    uid = upd.get("update_id")
    if uid is not None and uid > max_uid:
        max_uid = uid
    msg = upd.get("message")
    if not msg: continue
    chat = msg.get("chat", {}).get("id")
    if str(chat) not in allow: continue
    text = msg.get("text", "")
    line = {
        "update_id": uid,
        "chat_id": chat,
        "message_id": msg.get("message_id"),
        "text": text,
        "from_user": msg.get("from", {}).get("username", ""),
        "date": msg.get("date"),
    }
    print(json.dumps(line, ensure_ascii=False))
# write new offset
new_offset = max_uid + 1
with open("${offset_file}", "w", encoding="utf-8") as f:
    f.write(str(new_offset))
PY

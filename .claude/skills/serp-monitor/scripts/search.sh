#!/usr/bin/env bash
# search.sh — поиск в Яндексе через Yandex Search API v2 (Yandex Cloud)
# Использование:
#   search.sh --query "купить квартиру омск"
#   search.sh --query "купить квартиру" --region 66           # Нижний Новгород
#   search.sh --query "купить квартиру" --domain etagi.com    # свой домен для подсветки
#
# Endpoint: https://searchapi.api.cloud.yandex.net/v2/web/search
# Auth:     Authorization: Api-Key <ключ сервисного аккаунта>
# Setup:    .claude/skills/serp-monitor/config/README.md
#
# Возвращает markdown-таблицу top-20 с подсветкой ★ своего домена.
# На коммерческих запросах позиции 1-4 обычно рекламные (premium).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/../../_shared/common.sh"

load_config

# ─── Параметры ────────────────────────────────────────

QUERY=""
REGION="213"   # Москва по умолчанию
DOMAIN="${SITE_DOMAIN:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --query)  QUERY="$2";  shift 2 ;;
    --region) REGION="$2"; shift 2 ;;
    --domain) DOMAIN="$2"; shift 2 ;;
    *) echo "ERROR: Неизвестный параметр: $1" >&2; exit 1 ;;
  esac
done

require_param "query" "$QUERY"

# ─── Проверка конфига ────────────────────────────────

if [[ -z "${YANDEX_API_KEY:-}" ]]; then
  echo "ERROR: YANDEX_API_KEY не задан в tokens.env" >&2
  echo "  → В Yandex Cloud создайте сервисный аккаунт с ролью ai.searchApi.executor" >&2
  echo "  → Выпустите API-ключ: IAM → сервисный аккаунт → API-ключи → Создать" >&2
  echo "  → Подробная инструкция: .claude/skills/serp-monitor/config/README.md" >&2
  exit 1
fi

if [[ -z "${YANDEX_CLOUD_FOLDER_ID:-}" ]]; then
  echo "ERROR: YANDEX_CLOUD_FOLDER_ID не задан в tokens.env" >&2
  echo "  → Скопируйте ID каталога из Yandex Cloud Console (правая панель карточки каталога)" >&2
  exit 1
fi

# ─── Кеш (TTL 6ч из common.sh) ────────────────────────

CACHE_ID="serp_v2_${QUERY}_${REGION}"

if XML_DATA=$(cache_get "$CACHE_ID" 2>/dev/null); then
  :  # есть свежий ответ в кеше — используем
else
  # ─── Запрос к Search API v2 ─────────────────────────

  ENDPOINT="https://searchapi.api.cloud.yandex.net/v2/web/search"

  JSON_BODY=$(QUERY="$QUERY" REGION="$REGION" FOLDER="$YANDEX_CLOUD_FOLDER_ID" "$PYTHON_BIN" - <<'PYEOF'
import json, os
body = {
    "query": {
        "searchType": "SEARCH_TYPE_RU",
        "queryText": os.environ["QUERY"],
        "familyMode": "FAMILY_MODE_MODERATE",
        "fixTypoMode": "FIX_TYPO_MODE_ON"
    },
    "folderId": os.environ["FOLDER"],
    "responseFormat": "FORMAT_XML",
    "region": os.environ["REGION"],
    "groupSpec": {
        "groupMode": "GROUP_MODE_DEEP",
        "groupsOnPage": 10,
        "docsInGroup": 1
    }
}
print(json.dumps(body))
PYEOF
)

  RESPONSE=$(http_post "$ENDPOINT" "$JSON_BODY" "Authorization: Api-Key ${YANDEX_API_KEY}") || {
    echo "ERROR: Search API v2 запрос не выполнен" >&2
    echo "  → Проверьте YANDEX_API_KEY и YANDEX_CLOUD_FOLDER_ID в tokens.env" >&2
    echo "  → Убедитесь что сервисный аккаунт имеет роль ai.searchApi.executor" >&2
    echo "  → Убедитесь что Search API подключён в каталоге (Cloud Console → AI Studio → Search API)" >&2
    exit 1
  }

  # Распаковать rawData (base64 → XML). Передаём RESPONSE через tmpfile,
  # потому что heredoc забирает stdin у python — pipe бы потерялся.
  RESPONSE_FILE=$(mktemp)
  printf '%s' "$RESPONSE" > "$RESPONSE_FILE"

  XML_DATA=$(RESPONSE_FILE="$RESPONSE_FILE" "$PYTHON_BIN" - <<'PYEOF'
import json, sys, os, base64
with open(os.environ["RESPONSE_FILE"], "r", encoding="utf-8") as f:
    raw = f.read()
try:
    data = json.loads(raw)
except json.JSONDecodeError as e:
    print(f"ERROR: Search API вернул не-JSON ответ: {e}", file=sys.stderr)
    print(f"  → первые 300 байт ответа: {raw[:300]}", file=sys.stderr)
    sys.exit(1)

if "rawData" not in data:
    code = data.get("code", "?")
    msg = data.get("message", data.get("error", str(data)[:400]))
    print(f"ERROR: Search API вернул ошибку (code={code}): {msg}", file=sys.stderr)
    s = str(data).upper()
    if "PERMISSION_DENIED" in s:
        print("  → Сервисному аккаунту не хватает роли ai.searchApi.executor", file=sys.stderr)
    elif "UNAUTHENTICATED" in s:
        print("  → API-ключ невалиден или отозван", file=sys.stderr)
    elif "QUOTA" in s or "LIMIT" in s:
        print("  → Исчерпана квота каталога на Search API", file=sys.stderr)
    sys.exit(1)

try:
    xml = base64.b64decode(data["rawData"]).decode("utf-8")
except Exception as e:
    print(f"ERROR: Не удалось декодировать rawData (base64→utf-8): {e}", file=sys.stderr)
    sys.exit(1)
print(xml)
PYEOF
)
  PY_EXIT=$?
  rm -f "$RESPONSE_FILE"
  [[ $PY_EXIT -ne 0 ]] && exit 1
  [[ -z "$XML_DATA" ]] && { echo "ERROR: Пустой ответ от Search API" >&2; exit 1; }

  cache_set "$CACHE_ID" "$XML_DATA"
fi

# ─── Парсинг XML и вывод markdown-таблицы ─────────────
# XML передаём через tmpfile — heredoc уже занимает stdin python.

XML_FILE=$(mktemp)
printf '%s' "$XML_DATA" > "$XML_FILE"

XML_FILE="$XML_FILE" QUERY="$QUERY" REGION="$REGION" DOMAIN="$DOMAIN" "$PYTHON_BIN" - <<'PYEOF'
import xml.etree.ElementTree as ET
import sys, os

with open(os.environ["XML_FILE"], "r", encoding="utf-8") as f:
    xml_data = f.read()
query = os.environ["QUERY"]
region = os.environ["REGION"]
our_domain = os.environ.get("DOMAIN", "").lower().strip()

region_names = {
    "213": "Москва", "2": "Санкт-Петербург", "54": "Екатеринбург",
    "43": "Казань", "47": "Нижний Новгород", "65": "Новосибирск",
    "35": "Краснодар", "39": "Ростов-на-Дону", "62": "Красноярск",
    "51": "Самара", "66": "Омск", "172": "Уфа", "10000": "Россия",
    "159": "Казахстан", "149": "Беларусь",
}
region_name = region_names.get(region, f"lr={region}")

try:
    root = ET.fromstring(xml_data)
except ET.ParseError as e:
    print(f"ERROR: Не удалось разобрать XML-ответ: {e}", file=sys.stderr)
    sys.exit(1)

# Ошибки внутри XML (Search API возвращает XML в rawData даже при ошибке уровня поисковика)
error = root.find(".//error")
if error is not None:
    code = error.get("code", "?")
    text = error.text or "Неизвестная ошибка"
    print(f"ERROR: Search API XML error {code}: {text}", file=sys.stderr)
    sys.exit(1)

# Извлекаем документы
results = []
for i, group in enumerate(root.findall(".//group"), 1):
    doc = group.find("doc")
    if doc is None:
        continue

    url_el = doc.find("url")
    domain_el = doc.find("domain")
    title_el = doc.find("title")
    passages = doc.find("passages")

    url = url_el.text if url_el is not None else "—"
    domain = domain_el.text if domain_el is not None else "—"

    title = ""
    if title_el is not None:
        title = "".join(title_el.itertext()).strip()
    title = title or "—"

    snippet = ""
    if passages is not None:
        passage = passages.find("passage")
        if passage is not None:
            snippet = "".join(passage.itertext()).strip()

    results.append({
        "position": i,
        "url": url,
        "domain": domain.lower().replace("www.", ""),
        "domain_raw": domain,
        "title": title[:80] + ("..." if len(title) > 80 else ""),
        "snippet": snippet[:120] + ("..." if len(snippet) > 120 else ""),
    })

print(f'## Результаты поиска: "{query}" (регион: {region_name})')
print()

if not results:
    print("Результаты не найдены.")
    sys.exit(0)

print("| # | Домен | Заголовок |")
print("|---|-------|-----------|")

our_position = None
our_clean = our_domain.replace("www.", "") if our_domain else ""

for r in results:
    pos = r["position"]
    domain_display = r["domain_raw"]
    is_ours = our_clean and (our_clean in r["domain"])
    if is_ours:
        domain_display = f"{domain_display} ★"
        our_position = pos
    print(f'| {pos} | {domain_display} | {r["title"]} |')
    if pos >= 20:
        remaining = len(results) - 20
        if remaining > 0:
            print(f"| ... | ещё {remaining} результатов | |")
        break

print()

print("### Ваша позиция")
if not our_clean:
    print("Домен не задан. Укажите --domain или SITE_DOMAIN в tokens.env")
elif our_position:
    zone = ""
    if our_position <= 4:
        zone = " (зона Premium — рекламный блок над результатами)"
    elif our_position <= 8:
        zone = " (верхняя часть выдачи)"
    print(f"★ {our_clean} — #{our_position} в результатах{zone}")
else:
    print(f"{our_clean} — не найден в top-{len(results)} (реклама по запросу может не показываться)")
PYEOF
rm -f "$XML_FILE"

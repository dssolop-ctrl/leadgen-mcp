#!/usr/bin/env bash
# render_html.sh — markdown → html для отчётов автопилота.
#
# Usage:
#   render_html.sh <md_path> <out_html_path> [--title "Title"]
#   echo "# ...md..." | render_html.sh - <out_html_path>
#
# Прерогатива (по убыванию):
#   1. pandoc, если установлен (лучшее качество, GFM tables).
#   2. python -m markdown (если есть pip-пакет 'markdown').
#   3. fallback: trivial wrapping (preformatted block).

set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <md|-> <out_html> [--title \"Title\"]" >&2
  exit 2
fi

src="$1"
out="$2"
title="Autopilot Report"
if [[ "${3:-}" == "--title" ]]; then
  title="${4:-Autopilot Report}"
fi

mkdir -p "$(dirname "$out")"

tmp_md=""
cleanup() { [[ -n "$tmp_md" && -f "$tmp_md" ]] && rm -f "$tmp_md"; }
trap cleanup EXIT

if [[ "$src" == "-" ]]; then
  tmp_md="$(mktemp -t autopilot-md.XXXXXX.md)"
  cat > "$tmp_md"
  src="$tmp_md"
fi

# Try pandoc
if command -v pandoc >/dev/null 2>&1; then
  pandoc -f gfm -t html5 \
    --metadata title="$title" \
    --standalone \
    -o "$out" "$src"
  echo "OK rendered with pandoc: $out"
  exit 0
fi

# Try python markdown
if command -v python >/dev/null 2>&1 && python -c "import markdown" 2>/dev/null; then
  python - "$src" "$out" "$title" <<'PY'
import sys, markdown, html
src, out, title = sys.argv[1], sys.argv[2], sys.argv[3]
with open(src, encoding="utf-8") as f: md = f.read()
body = markdown.markdown(md, extensions=["tables", "fenced_code"])
htmldoc = f"""<!DOCTYPE html>
<html><head>
<meta charset="utf-8">
<title>{html.escape(title)}</title>
<style>
body {{ font-family: -apple-system, Segoe UI, Roboto, sans-serif; max-width: 900px; margin: 2em auto; padding: 0 1em; line-height: 1.5; color: #222; }}
table {{ border-collapse: collapse; }}
th, td {{ border: 1px solid #ddd; padding: 6px 10px; }}
th {{ background: #f5f5f5; }}
code {{ background: #f7f7f9; padding: 2px 5px; border-radius: 3px; }}
pre {{ background: #f7f7f9; padding: 12px; border-radius: 6px; overflow: auto; }}
h1, h2, h3 {{ color: #111; }}
.warn {{ color: #c0392b; }}
.ok {{ color: #2c8a3a; }}
</style>
</head><body>
{body}
</body></html>"""
with open(out, "w", encoding="utf-8") as f: f.write(htmldoc)
print(f"OK rendered with python-markdown: {out}")
PY
  exit 0
fi

# Fallback: trivial wrap
echo "WARN: no pandoc / python-markdown found, using trivial fallback" >&2
{
  echo "<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>${title}</title>"
  echo "<style>body{font-family:monospace;white-space:pre-wrap;max-width:900px;margin:2em auto;padding:0 1em}</style>"
  echo "</head><body><h1>${title}</h1><pre>"
  cat "$src" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g'
  echo "</pre></body></html>"
} > "$out"
echo "OK trivial fallback: $out"

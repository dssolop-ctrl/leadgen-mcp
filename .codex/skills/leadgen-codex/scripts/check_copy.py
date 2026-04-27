#!/usr/bin/env python3
"""
check_copy.py — валидация текстов объявлений против copy_blacklist.md.

Использование:
  # одно объявление с CLI
  python check_copy.py --title "Купить квартиру" --title2 "Официальный сайт" \\
      --text "Без СМС и регистрации. Поможем быстро."

  # пачка из TSV (колонки: title, title2, text, callout, sitelink_title, sitelink_description)
  python check_copy.py --tsv drafts.tsv

Коды возврата:
  0 — всё чисто
  1 — найден хоть один hard-блокер
  2 — только мягкие предупреждения

Формат выхода: markdown-таблица с колонками field / value / verdict / rule.
"""
from __future__ import annotations

import argparse
import csv
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

# --- Blacklist (зеркало references/copy_blacklist.md) ---

HARD_BLACKLIST: dict[str, str] = {
    # слово в нижнем регистре → формулировка нарушения
    "whatsapp": "слово WhatsApp запрещено в Директе (модерация)",
    "ватсап": "слово «ватсап» запрещено в Директе (модерация)",
    "vpn": "слово VPN запрещено",
    "впн": "слово «впн» запрещено",
    "официальный сайт": "«официальный сайт» — риск претензий правообладателей",
    "без смс": "«без СМС» — наследие пиратского контента, банит модерация",
    "без смс и регистрации": "«без СМС и регистрации» — запрещено модерацией",
    "бесплатная консультация": "без указания условий может считаться скрытой платой (ФАС)",
}

SOFT_BLACKLIST: dict[str, str] = {
    "гарантия": "без контекста может пониматься как обещание — уточни условие",
    "гарантированно": "проверь подтверждение на странице",
    "100%": "требует обоснования",
    "лучший": "без подтверждения рейтингом — риск ФАС",
    "№1": "требует источник рейтинга",
    "топ-1": "требует источник рейтинга",
    "дешевле всех": "без проверки считается вводом в заблуждение",
    "минимальная цена": "без обоснования — риск ФАС",
    "абсолютно": "часто триггерит модерацию",
    "максимально": "размытая формулировка",
}


@dataclass
class Finding:
    field: str
    value: str
    rule: str
    severity: str  # 'hard' | 'soft'


def scan_text(field: str, value: str, findings: list[Finding]) -> None:
    if not value:
        return
    low = value.lower()
    for phrase, note in HARD_BLACKLIST.items():
        if phrase in low:
            findings.append(Finding(field, value, note, "hard"))
    for phrase, note in SOFT_BLACKLIST.items():
        # чтобы не ловить «гарантия» внутри «гарантированно» дважды — идёт как отдельные ключи
        if re.search(rf"\b{re.escape(phrase)}\b", low):
            findings.append(Finding(field, value, note, "soft"))


def scan_row(row: dict[str, str], findings: list[Finding]) -> None:
    for field in ("title", "title2", "text", "callout", "sitelink_title", "sitelink_description", "display_url"):
        scan_text(field, row.get(field, ""), findings)


def iter_tsv(path: Path) -> Iterable[dict[str, str]]:
    with path.open(encoding="utf-8") as fh:
        reader = csv.DictReader(fh, delimiter="\t")
        for row in reader:
            yield {k: (v or "").strip() for k, v in row.items()}


def render_report(findings: list[Finding]) -> str:
    if not findings:
        return "✅ Проверено. Нарушений не найдено."
    lines = [
        "| severity | field | value | rule |",
        "|---|---|---|---|",
    ]
    for f in findings:
        v = f.value.replace("|", "\\|")
        lines.append(f"| {f.severity} | {f.field} | {v} | {f.rule} |")
    hard_count = sum(1 for f in findings if f.severity == "hard")
    soft_count = sum(1 for f in findings if f.severity == "soft")
    lines.append("")
    lines.append(f"**Итого:** hard={hard_count}, soft={soft_count}.")
    if hard_count:
        lines.append("**Apply запрещён** до устранения hard-блокеров.")
    return "\n".join(lines)


def main() -> None:
    p = argparse.ArgumentParser(description="Проверить тексты объявлений против copy_blacklist.md")
    p.add_argument("--tsv", type=Path, help="TSV-файл с колонками title/title2/text/callout/sitelink_title/sitelink_description/display_url")
    p.add_argument("--title", default="")
    p.add_argument("--title2", default="")
    p.add_argument("--text", default="")
    p.add_argument("--callout", default="")
    p.add_argument("--sitelink-title", default="")
    p.add_argument("--sitelink-description", default="")
    p.add_argument("--display-url", default="")
    args = p.parse_args()

    findings: list[Finding] = []
    if args.tsv:
        if not args.tsv.exists():
            print(f"ERROR: файл не найден: {args.tsv}", file=sys.stderr)
            sys.exit(3)
        for row in iter_tsv(args.tsv):
            scan_row(row, findings)
    else:
        row = {
            "title": args.title,
            "title2": args.title2,
            "text": args.text,
            "callout": args.callout,
            "sitelink_title": args.sitelink_title,
            "sitelink_description": args.sitelink_description,
            "display_url": args.display_url,
        }
        if not any(row.values()):
            print("ERROR: передай --tsv или хотя бы один из --title/--text/...", file=sys.stderr)
            sys.exit(3)
        scan_row(row, findings)

    print(render_report(findings))

    has_hard = any(f.severity == "hard" for f in findings)
    has_soft = any(f.severity == "soft" for f in findings)
    if has_hard:
        sys.exit(1)
    if has_soft:
        sys.exit(2)
    sys.exit(0)


if __name__ == "__main__":
    main()

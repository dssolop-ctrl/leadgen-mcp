#!/usr/bin/env python3
"""
render_audit_report.py — рендерит аудит кампании в фиксированную HTML-верстку
для специалиста, ведущего город.

Использование:
  python render_audit_report.py --input audit.json --output audit.html

Формат audit.json:
{
  "campaign": {"id": "89167235", "name": "Поиск | Вторичка", "city": "omsk"},
  "scope": {"window": "2026-04-13..2026-04-20", "attribution": "LYDC", "out_of_scope": [...]},
  "kpis": {"cpa": 820, "cpa_threshold": 900, "ctr": 8.2, "conversions_7d": 12, "cost_7d": 9840},
  "findings": [
    {"entity_type": "keyword", "entity_id": "...", "entity_name": "...",
     "verdict": "waste", "confidence": "high", "evidence": "Cost 1200 ₽, 0 конверсий за 14 дней"}
  ],
  "proposed_changes": [
    {"change_id": "c1", "entity_type": "negative", "entity_id": null,
     "current": "нет", "proposed": "добавить 'скачать' в минус-слова",
     "reason": "8 кликов по инфо-запросам", "expected_effect": "1-3d",
     "risk": "low", "confidence": "high-confidence operational",
     "approval_needed": false}
  ],
  "applied_changes": [...],  // опционально, если режим был live-apply
  "next_steps": ["Проверить CPA через 3 дня", ...]
}

Верстка предсказуемая — не меняется между запусками. Шапка с KPI, цветовые
badge'и (ok/attention/critical/opportunity/waste), таблицы findings + proposals.
"""
from __future__ import annotations

import argparse
import datetime as dt
import html
import json
import sys
from pathlib import Path
from typing import Any

VERDICT_COLOR = {
    "ok": "#16a34a",
    "opportunity": "#2563eb",
    "attention": "#d97706",
    "waste": "#dc2626",
    "critical": "#991b1b",
}
CONFIDENCE_COLOR = {
    "high": "#16a34a",
    "medium": "#d97706",
    "hypothesis-only": "#6b7280",
    "high-confidence operational": "#16a34a",
    "medium-confidence optimization": "#d97706",
}


def esc(v: Any) -> str:
    if v is None:
        return "—"
    return html.escape(str(v))


def badge(text: str, color: str) -> str:
    return (
        f'<span style="background:{color};color:white;padding:2px 8px;'
        f'border-radius:10px;font-size:12px;font-weight:600;">'
        f"{esc(text)}</span>"
    )


def kpi_card(label: str, value: Any, hint: str = "", color: str = "#0f172a") -> str:
    hint_html = f'<div class="kpi-hint">{esc(hint)}</div>' if hint else ""
    return (
        f'<div class="kpi"><div class="kpi-label">{esc(label)}</div>'
        f'<div class="kpi-value" style="color:{color}">{esc(value)}</div>'
        f"{hint_html}</div>"
    )


def render_findings(findings: list[dict]) -> str:
    if not findings:
        return "<p class=\"muted\">Нет зафиксированных наблюдений.</p>"
    rows = []
    for f in findings:
        v = f.get("verdict", "attention")
        c = f.get("confidence", "medium")
        rows.append(
            f"<tr>"
            f"<td>{esc(f.get('entity_type'))}</td>"
            f"<td>{esc(f.get('entity_name') or f.get('entity_id'))}</td>"
            f"<td>{badge(v, VERDICT_COLOR.get(v, '#6b7280'))}</td>"
            f"<td>{badge(c, CONFIDENCE_COLOR.get(c, '#6b7280'))}</td>"
            f"<td>{esc(f.get('evidence'))}</td>"
            f"</tr>"
        )
    return (
        "<table><thead><tr>"
        "<th>тип</th><th>сущность</th><th>verdict</th><th>уверенность</th><th>evidence</th>"
        "</tr></thead><tbody>" + "".join(rows) + "</tbody></table>"
    )


def render_proposals(changes: list[dict]) -> str:
    if not changes:
        return "<p class=\"muted\">Предложений нет — кампания в норме, либо режим только research.</p>"
    rows = []
    for c in changes:
        approval = "⚠ требует согласия" if c.get("approval_needed") else "ок без согласия"
        approval_color = "#d97706" if c.get("approval_needed") else "#16a34a"
        rows.append(
            f"<tr>"
            f"<td><code>{esc(c.get('change_id'))}</code></td>"
            f"<td>{esc(c.get('entity_type'))}<br><span class=\"muted\">{esc(c.get('entity_id'))}</span></td>"
            f"<td>{esc(c.get('current'))}</td>"
            f"<td><strong>{esc(c.get('proposed'))}</strong></td>"
            f"<td>{esc(c.get('reason'))}</td>"
            f"<td>{esc(c.get('expected_effect'))}</td>"
            f"<td>{esc(c.get('risk'))}</td>"
            f"<td>{badge(c.get('confidence', 'medium'), CONFIDENCE_COLOR.get(c.get('confidence', 'medium'), '#6b7280'))}</td>"
            f"<td>{badge(approval, approval_color)}</td>"
            f"</tr>"
        )
    return (
        "<table><thead><tr>"
        "<th>id</th><th>сущность</th><th>сейчас</th><th>предложено</th><th>почему</th>"
        "<th>горизонт</th><th>риск</th><th>уверенность</th><th>согласие</th>"
        "</tr></thead><tbody>" + "".join(rows) + "</tbody></table>"
    )


def render_applied(applied: list[dict]) -> str:
    if not applied:
        return ""
    rows = []
    for a in applied:
        rows.append(
            f"<tr>"
            f"<td>{esc(a.get('tool_name'))}</td>"
            f"<td>{esc(a.get('entity_type'))} {esc(a.get('entity_id'))}</td>"
            f"<td>{esc(a.get('before_value'))}</td>"
            f"<td><strong>{esc(a.get('after_value'))}</strong></td>"
            f"<td>{esc(a.get('read_back_ok'))}</td>"
            f"</tr>"
        )
    return (
        "<section><h2>Что сделано (live-apply)</h2>"
        "<table><thead><tr>"
        "<th>tool</th><th>сущность</th><th>было</th><th>стало</th><th>read-back</th>"
        "</tr></thead><tbody>" + "".join(rows) + "</tbody></table></section>"
    )


def render_kpis(kpis: dict) -> str:
    cards = []
    if "cpa" in kpis:
        threshold = kpis.get("cpa_threshold", 0)
        color = "#16a34a" if (not threshold or kpis["cpa"] <= threshold) else "#dc2626"
        hint = f"порог {threshold} ₽" if threshold else ""
        cards.append(kpi_card("CPA, 7д", f"{kpis['cpa']} ₽", hint, color))
    if "ctr" in kpis:
        cards.append(kpi_card("CTR, 7д", f"{kpis['ctr']}%"))
    if "conversions_7d" in kpis:
        cards.append(kpi_card("Конверсий, 7д", kpis["conversions_7d"]))
    if "cost_7d" in kpis:
        cards.append(kpi_card("Расход, 7д", f"{kpis['cost_7d']} ₽"))
    if "clicks_7d" in kpis:
        cards.append(kpi_card("Кликов, 7д", kpis["clicks_7d"]))
    return '<div class="kpis">' + "".join(cards) + "</div>" if cards else ""


TEMPLATE = """<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<title>Аудит кампании — {campaign_name}</title>
<style>
  *{{box-sizing:border-box}}
  body{{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;
       max-width:1100px;margin:24px auto;padding:0 20px;color:#0f172a;background:#f8fafc;}}
  header{{border-bottom:2px solid #e2e8f0;padding-bottom:16px;margin-bottom:20px;}}
  h1{{margin:0 0 8px;font-size:24px}}
  h2{{margin-top:28px;font-size:18px;color:#1e293b;border-left:4px solid #2563eb;padding-left:12px}}
  .meta{{color:#64748b;font-size:14px}}
  .muted{{color:#94a3b8}}
  section{{background:white;padding:16px 20px;border-radius:8px;margin-bottom:16px;
          border:1px solid #e2e8f0}}
  .kpis{{display:grid;grid-template-columns:repeat(auto-fill,minmax(180px,1fr));
        gap:12px;margin-bottom:20px}}
  .kpi{{background:white;padding:14px 18px;border-radius:8px;border:1px solid #e2e8f0}}
  .kpi-label{{font-size:12px;color:#64748b;text-transform:uppercase;letter-spacing:0.5px}}
  .kpi-value{{font-size:24px;font-weight:700;margin-top:4px}}
  .kpi-hint{{font-size:11px;color:#94a3b8;margin-top:2px}}
  table{{width:100%;border-collapse:collapse;margin-top:8px;font-size:14px}}
  th{{background:#f1f5f9;text-align:left;padding:10px;border-bottom:2px solid #cbd5e1;
      font-weight:600;color:#475569}}
  td{{padding:10px;border-bottom:1px solid #e2e8f0;vertical-align:top}}
  tr:last-child td{{border-bottom:none}}
  code{{background:#f1f5f9;padding:2px 6px;border-radius:4px;font-size:13px}}
  ul{{margin:6px 0;padding-left:22px}}
  footer{{color:#94a3b8;font-size:12px;margin-top:20px;text-align:center}}
  .scope-chip{{display:inline-block;background:#e2e8f0;color:#475569;
               padding:2px 8px;border-radius:10px;font-size:12px;margin:2px}}
</style>
</head>
<body>
<header>
  <h1>Аудит: {campaign_name}</h1>
  <div class="meta">
    Кампания <code>{campaign_id}</code> · город <strong>{city}</strong> ·
    окно {window} · атрибуция {attribution}
  </div>
  <div class="meta">Сгенерировано: {generated_at}</div>
</header>

<section>
  <h2>KPI окна</h2>
  {kpis_html}
</section>

<section>
  <h2>Что проверено (scope)</h2>
  <div>{scope_chips}</div>
  {out_of_scope_html}
</section>

<section>
  <h2>Что найдено (findings)</h2>
  {findings_html}
</section>

<section>
  <h2>Что предлагается (proposed changes)</h2>
  {proposals_html}
</section>

{applied_html}

<section>
  <h2>Что дальше</h2>
  {next_steps_html}
</section>

<footer>
  Отчёт leadgen skill · render_audit_report.py · предсказуемая верстка, один шаблон на все аудиты
</footer>
</body>
</html>
"""


def build_scope_chips(scope: dict) -> str:
    chips = []
    for key in ("campaigns", "adgroups", "checks"):
        if key in scope:
            for item in scope[key]:
                chips.append(f'<span class="scope-chip">{esc(item)}</span>')
    return "".join(chips) or '<span class="muted">Scope не детализирован.</span>'


def build_out_of_scope(scope: dict) -> str:
    out = scope.get("out_of_scope") or []
    if not out:
        return ""
    items = "".join(f"<li>{esc(i)}</li>" for i in out)
    return f"<p class=\"muted\" style=\"margin-top:10px\">Out of scope:</p><ul>{items}</ul>"


def build_next_steps(items: list[str]) -> str:
    if not items:
        return '<p class="muted">Явных next steps не задано.</p>'
    li = "".join(f"<li>{esc(x)}</li>" for x in items)
    return f"<ul>{li}</ul>"


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--input", required=True, type=Path)
    p.add_argument("--output", required=True, type=Path)
    args = p.parse_args()

    if not args.input.exists():
        print(f"ERROR: input not found: {args.input}", file=sys.stderr)
        sys.exit(1)

    data = json.loads(args.input.read_text(encoding="utf-8"))

    campaign = data.get("campaign", {})
    scope = data.get("scope", {})
    kpis = data.get("kpis", {})
    applied = data.get("applied_changes", []) or []

    html_out = TEMPLATE.format(
        campaign_name=esc(campaign.get("name", "без имени")),
        campaign_id=esc(campaign.get("id", "—")),
        city=esc(campaign.get("city", "—")),
        window=esc(scope.get("window", "—")),
        attribution=esc(scope.get("attribution", "LYDC")),
        generated_at=dt.datetime.now().strftime("%Y-%m-%d %H:%M"),
        kpis_html=render_kpis(kpis),
        scope_chips=build_scope_chips(scope),
        out_of_scope_html=build_out_of_scope(scope),
        findings_html=render_findings(data.get("findings", [])),
        proposals_html=render_proposals(data.get("proposed_changes", [])),
        applied_html=render_applied(applied),
        next_steps_html=build_next_steps(data.get("next_steps", [])),
    )

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(html_out, encoding="utf-8")
    print(f"HTML report: {args.output}")


if __name__ == "__main__":
    main()

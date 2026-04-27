# Подключение leadgen-mcp к OpenAI Codex

## 1. Запустить локальный MCP-сервер

```bash
cp server/config.yaml.example server/config.yaml
# Заполните токены в server/config.yaml
docker compose up -d
```

Endpoint разработки:

- SSE: `http://localhost:8080/sse`
- Health: `http://localhost:8080/health`

## 2. MCP-конфиг Codex

В проекте создан локальный конфиг `.codex/mcp.json`:

```json
{
  "mcpServers": {
    "leadgen": {
      "type": "sse",
      "url": "http://localhost:8080/sse"
    }
  }
}
```

Если проектный конфиг не подхватился, добавьте сервер в глобальный MCP-конфиг Codex тем же endpoint и именем `leadgen`.

## 3. Скилл Codex

Codex-адаптация лежит в `.codex/skills/leadgen-codex/`.

Ключевые отличия от Claude-версии:

- файл входа называется `SKILL.md`;
- имя скилла: `leadgen-codex`, чтобы не конфликтовать с существующим `leadgen`;
- добавлен `agents/openai.yaml` для UI-метаданных Codex;
- endpoint и MCP namespace описаны в `config/mcp_endpoint.md`;
- ветки `branches/` подгружаются лениво после роутинга.

## 4. Проверка в Codex

Откройте проект в Codex App или запустите:

```bash
codex -C /path/to/leadgen-mcp
```

В новой сессии проверьте:

```text
проверь MCP leadgen и покажи доступные инструменты
```

Ожидаемо: доступен MCP-сервер `leadgen` с инструментами Директа, Метрики, Wordstat, VK Ads, history и filters.

## 5. Инструкции агента

Codex читает `AGENTS.md` из корня проекта. Бизнес-настройки, цели и CPA-пороги лежат в `PROJECTS.md`; правила API Директа и Метрики — в `METRIKA-ADS-RULES.md`; юридическая проверка контента — в `LEGAL.md`.

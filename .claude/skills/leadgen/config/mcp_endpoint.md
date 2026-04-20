# MCP endpoint — настройка подключения

Этот файл задаёт, к какому MCP-серверу подключается клиент (Claude Code / Cursor / etc).

**Важно:** сам скилл НЕ устанавливает соединение — MCP-клиент читает конфиг из `.mcp.json` в корне проекта на старте. Этот файл — единый источник правды о том, какой endpoint сейчас используется. При смене окружения (dev ↔ prod) — обнови и этот файл, и `.mcp.json`.

## Текущий endpoint

**Режим:** `LOCAL_DOCKER` (разработка)

| Параметр | Значение |
|---|---|
| Transport | SSE |
| URL | `http://localhost:8080/sse` |
| Auth | нет (bearer_token пустой в `server/config.yaml`) |
| Health | `http://localhost:8080/health` |

Сервер поднят в Docker-контейнере `leadgen-mcp` (образ `leadgen-mcp:latest`, собирается из `server/Dockerfile`).

## Доступные режимы

### LOCAL_DOCKER (текущий)

Локальная разработка. Контейнер должен быть запущен:

```bash
docker run -d --name leadgen-mcp -p 8080:8080 \
  -v "C:/git/leadgen-mcp/server/config.yaml:/app/config.yaml:ro" \
  leadgen-mcp:latest
```

`.mcp.json` должен содержать:

```json
{
  "mcpServers": {
    "yandex-direct": {
      "type": "sse",
      "url": "http://localhost:8080/sse"
    }
  }
}
```

### PRODUCTION (будет задано позже)

Placeholder. Пользователь заменит URL на реальный адрес production-сервера.

```json
{
  "mcpServers": {
    "yandex-direct": {
      "type": "http",
      "url": "<PRODUCTION_URL>",
      "headers": { "Authorization": "Bearer <TOKEN>" }
    }
  }
}
```

## Как переключить режим

1. Обнови «Текущий endpoint» в этом файле
2. Обнови `.mcp.json` в корне проекта
3. Перезапусти MCP-клиент (Claude Code: `/mcp` → reconnect, либо рестарт)

# MCP endpoint — настройка подключения

Этот файл задаёт, к какому MCP-серверу подключается Codex.

**Важно:** сам скилл НЕ устанавливает соединение. Codex читает MCP-конфиг из проектного `.codex/mcp.json` или из глобального MCP-конфига Codex. В этом репозитории также есть legacy `.mcp.json` для других MCP-клиентов. Этот файл — справка для агента о текущем endpoint; при смене окружения обнови его и активный Codex MCP-конфиг.

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

Для Codex предпочтителен `.codex/mcp.json`:

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

Если используется legacy `.mcp.json`, допустимо старое имя сервера `yandex-direct`, но фактически endpoint обслуживает весь `leadgen-mcp` (Директ, Метрика, Wordstat, VK Ads, history, filters, imagegen).

### PRODUCTION (будет задано позже)

Placeholder. Пользователь заменит URL на реальный адрес production-сервера.

```json
{
  "mcpServers": {
    "leadgen": {
      "type": "http",
      "url": "<PRODUCTION_URL>",
      "headers": { "Authorization": "Bearer <TOKEN>" }
    }
  }
}
```

## Как переключить режим

1. Обнови «Текущий endpoint» в этом файле
2. Обнови `.codex/mcp.json` или глобальный MCP-конфиг Codex
3. Перезапусти Codex-сессию или переподключи MCP-сервер

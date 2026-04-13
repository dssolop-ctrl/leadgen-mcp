# leadgen-mcp

Self-hosted MCP-сервер для управления рекламой в Яндекс Директе и VK Ads через AI-агентов.

## Платформы

| Платформа | Инструментов |
|-----------|-------------|
| Яндекс Директ | ~50 |
| Яндекс Метрика | ~11 |
| Wordstat | 5 |
| VK Ads | ~30 |

## Быстрый старт

```bash
cd server
cp config.yaml.example config.yaml
# Заполните токены в config.yaml
docker build -t leadgen-mcp .
docker run -p 8080:8080 -v ./config.yaml:/app/config.yaml leadgen-mcp
```

## Структура

```
server/           # Go MCP-сервер
  platform/
    direct/       # Яндекс Директ API v5
    metrika/      # Яндекс Метрика API
    wordstat/     # Яндекс Wordstat API v4
    vk/           # VK Ads API v2
campaigns/        # Файлы кампаний
```

# PostgreSQL MCP для Codex

Подключение настроено для этого проекта в `.codex/config.toml`. Сервер
[crystaldba/postgres-mcp](https://github.com/crystaldba/postgres-mcp) запускается
через `bash scripts/postgres-mcp.sh` в отдельном Docker-контейнере по stdio.

## Запуск

1. Запустите Docker и PostgreSQL: `docker compose up -d postgres`.
2. Убедитесь, что в `backend/.env` задан `DATABASE_URL` текущей локальной базы.
3. Откройте проект в новой сессии Codex и проверьте сервер `postgres` через `/mcp`.

При первом запуске Docker скачает образ. Проверить регистрацию можно командой
`codex mcp get postgres` из корня приложения. Настройка MCP на уровне проекта
поддерживается для доверенных проектов: [документация Codex](https://developers.openai.com/codex/mcp).

Скрипт читает существующий `backend/.env`; пароль не дублируется в конфигурации.
Для другого env-файла можно задать `BACKEND_ENV_FILE` (относительный путь
считается от `backend/`). Адреса `127.0.0.1` и `localhost` с явно заданным портом
заменяются на `host.docker.internal`, чтобы подключиться из контейнера к
опубликованному порту PostgreSQL на macOS. Текущий порт — `55432`, база — `uptime`.

Включён режим сервера `--access-mode=restricted`: SQL-запросы выполняются только
для чтения. Скрипт не создаёт отдельного пользователя БД и использует права
подключения из `DATABASE_URL`.

Поле `cwd` в `.codex/config.toml` содержит абсолютный путь текущего checkout;
при переносе проекта обновите его.

# Uptime App

Базовая структура приложения мониторинга: Go API и Next.js (App Router, TypeScript).

## Требования и запуск

Нужны Go 1.25+ (совместимо с установленным Go 1.25.3), Node.js 20.9+ и npm.

```sh
make setup
```

В двух терминалах из корня проекта:

```sh
make backend-dev
```

```sh
make frontend-dev
```

Фронтенд: http://localhost:3000. API: http://localhost:8080/api/v1/health.
На главной странице есть кнопка проверки API.

## Структура

- `backend/cmd/api` — точка входа HTTP-сервера.
- `backend/internal/config` — конфигурация из переменных окружения.
- `backend/internal/httpapi` — маршруты и HTTP-обработчики.
- `backend/migrations` — место для будущих миграций БД.
- `frontend/src/app` — страницы, layout и стили App Router.
- `frontend/src/components` — компоненты интерфейса.
- `frontend/src/lib` — общие функции и клиент API.
- `frontend/public` — статические файлы.
- `scripts` — скрипты подготовки проекта.

По мере добавления функциональности бизнес-логику можно размещать в `backend/internal`, отдельно от HTTP-обработчиков. База данных и авторизация пока не подключены.

## Конфигурация

`make setup` создаёт локальные файлы окружения, сохраняя существующие.
Next.js автоматически читает `frontend/.env.local`. `API_URL` задаёт адрес Go API
для проксирования `/api/*` через Next.js: браузер обращается к тому же origin.
При изменении `API_URL` перезапустите dev-сервер; для production повторите сборку.

Go читает переменные окружения процесса, файл `backend/.env` автоматически не загружается.
По умолчанию сервер слушает `:8080`. Пример другого порта:

```sh
HTTP_ADDR=:8081 make backend-dev
```

В этом случае укажите `API_URL=http://127.0.0.1:8081` во frontend/.env.local.

## Проверки и сборка

```sh
make check
make build
```

`make check` запускает Go-тесты, go vet, ESLint и проверку TypeScript.
`make build` создаёт `backend/bin/api` и production-сборку Next.js.
Запуск после сборки: `./backend/bin/api` и в отдельном терминале
`cd frontend && npm start`.

Зависимости frontend закреплены в package-lock.json; используйте `npm ci`.
Go API использует стандартную библиотеку и корректно завершает активные запросы при SIGINT/SIGTERM.

## Состояние зависимостей

На момент подготовки `npm audit` сообщает о 5 high-предупреждениях
в dev-цепочке `eslint-config-next → fast-glob → micromatch → braces`.
Предлагаемый npm автоматический fix откатывает eslint-config-next на 14.x,
поэтому он не применён. Следует проверить обновления этой цепочки перед
расширением CI; пакеты относятся к инструментам разработки.

Документация Next.js: https://nextjs.org/docs/app/getting-started/installation.

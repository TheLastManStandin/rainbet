# Rainbet Mines

Небольшое тестовое Mines-приложение с provably fair механикой, SQLite-хранилищем и статическим frontend.

## Быстрый запуск

Требования: Docker, Docker Compose и [Task](https://taskfile.dev/).

```bash
task compose
```

Откройте [http://localhost:8080](http://localhost:8080). Тестовый пользователь: `user` / `user`.

Данные SQLite хранятся в Docker volume `rainbet-data` и сохраняются между перезапусками.

## Публичная ссылка

```bash
task tunnel
```

Команда запускает Compose stack и выводит в терминал временную ссылку Cloudflare Quick Tunnel. Оставьте команду запущенной, пока ссылка нужна; `Ctrl+C` остановит только tunnel, а frontend и backend продолжат работать. Эту ссылку можно использовать для Telegram mini app.

Quick Tunnel предназначен только для тестирования. `POST /api/user/balance` намеренно не требует авторизации, поэтому не публикуйте приложение в production.

## Полезные команды

```bash
task check                              # go vet и все Go tests
task e2e                                # HTTP end-to-end tests
task run                                # запустить только Go API локально
docker compose down                     # остановить stack, сохранив БД
docker compose exec backend sqlite3 /data/rainbet.db
```

## Архитектура

```text
Browser
  │
  ▼
Nginx frontend (HTML, CSS, JavaScript)
  │ /api/* proxy
  ▼
Go API
  │ clean architecture
  ▼
SQLite (/data/rainbet.db)
```

- `frontend/` содержит статический интерфейс Mines. Nginx отдаёт файлы и проксирует `/api/` в backend service.
- `backend/internal/domain` содержит сущности и чистые бизнес-правила аккаунта и Mines, включая расчёт коэффициентов и выплат.
- `backend/internal/application` содержит use cases и входные/выходные порты. Этот слой не знает о HTTP, SQLite и bcrypt.
- `backend/internal/delivery/httpapi` адаптирует HTTP/JSON и Basic Auth к application services.
- `backend/internal/infrastructure` реализует порты для SQLite, bcrypt и provably fair генератора.
- `backend/cmd/api/main.go` — composition root: создаёт адаптеры, связывает зависимости и запускает сервер.
- SQLite schema сохраняет денежные значения в целых центах; игровые операции выполняются атомарно через Unit of Work.
- `docker-compose.yml` запускает frontend и backend, а profile `tunnel` содержит Cloudflare Quick Tunnel service.

Зависимости направлены внутрь: `delivery` и `infrastructure` зависят от `application`/`domain`, а domain не импортирует внешние слои.

## Технологии

- Go 1.22 и стандартный `net/http`
- SQLite через `github.com/mattn/go-sqlite3` (CGO)
- bcrypt для паролей
- HTML, CSS и vanilla JavaScript
- Nginx, Docker Compose и Cloudflare Tunnel

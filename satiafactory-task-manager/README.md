# Satisfactory Task Manager

Веб-приложение для задач в контексте Satisfactory: микросервисы на Go, PostgreSQL, Docker, балансировка task-service через NGINX.

## Быстрый старт (Docker)

### 1. Требования

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (Windows)
- Порты **8080**, **8081**, **5432**, **8090** свободны

### 2. Режимы запуска

| Файл | Назначение |
|------|------------|
| `docker-compose.balance.yml` | **Рекомендуется для сдачи**: 3 реплики task-service + NGINX |
| `docker-compose.minimal.yml` | Отладка: один task-service, без NGINX |
| `docker-compose.yml` | Полный стек (+ satisfactory-data-service) |

**Важно:** не смешивайте minimal и balance одновременно — остановите старые контейнеры перед сменой режима.

### 3. Запуск (балансировка, как в ТЗ)

```powershell
cd c:\satiafactory-task-manager

# Остановить всё и убрать «лишний» одиночный task-service, если был
docker compose -f docker-compose.minimal.yml down 2>$null
docker stop task-service 2>$null
docker rm task-service 2>$null

# Собрать и запустить
docker compose -f docker-compose.balance.yml up -d --build
```

Приложение: **http://localhost:8080**

### 4. Если «всё сломалось» после экспериментов

Частая причина — старый том PostgreSQL без таблиц `users` / `tasks`:

```powershell
docker compose -f docker-compose.balance.yml down -v
docker compose -f docker-compose.balance.yml up -d --build
```

Флаг `-v` удаляет volume `postgres_data`. Данные в БД сбросятся, схема создастся заново (init-скрипты + миграции в сервисах).

### 5. Проверка балансировки

После входа откройте DevTools → Network и несколько раз обновите список задач (`GET /tasks`). В ответах через gateway к nginx заголовок **X-Instance-ID** должен чередоваться: `tasks-1`, `tasks-2`, `tasks-3`.

Или из PowerShell (нужен JWT в cookie после логина в браузере — проще смотреть в DevTools).

Логи NGINX при падении реплики:

```powershell
docker logs nginx-lb --tail 20
```

### 6. Полезные команды

```powershell
docker compose -f docker-compose.balance.yml ps
docker compose -f docker-compose.balance.yml logs -f gateway user-service nginx
docker compose -f docker-compose.balance.yml down
```

## Архитектура (balance)

```
Браузер → gateway:8080 → user-service:8081 (auth)
                      → nginx:8090 → task-service-1|2|3:8082 → redis:6379 (кэш GET /tasks)
                      → postgres:5432
```

## Рецепты Satisfactory

Сервис `satisfactory-data-service` при первом запуске импортирует `services/satisfactory-data-service/data/game-data.json` (данные Satisfactory **v1.0**, источник — [SatisfactoryTools](https://github.com/greeny/SatisfactoryTools)) в PostgreSQL.

На странице «Мои задачи»:
1. Введите название рецепта в поле поиска (минимум 2 символа).
2. Выберите рецепт — появится превью ингредиентов и иконки (из [SatisfactoryTools](https://github.com/greeny/SatisfactoryTools)).
3. Укажите количество партий и создайте задачу — в карточке задачи отобразится, что нужно для крафта.

Ручной импорт (перезаписывает рецепты, предметы и постройки в БД):

```powershell
docker compose -f docker-compose.balance.yml run --rm satisfactory-data-service ./data-service -import
```

Старый формат `Docs.json` (Update 8) по-прежнему поддерживается — укажите `DATA_FILE_PATH=./data/Docs.json`.

## Redis-кэш списка задач

`task-service` кэширует ответы `GET /tasks` в **Redis** (общий для всех 3 реплик):

| Scope | Ключ |
|-------|------|
| все активные | `tasks:list:all` |
| свои | `tasks:list:mine:{userID}` |
| выполненные | `tasks:list:completed` |

- TTL по умолчанию: **60 с** (`REDIS_CACHE_TTL`)
- При создании / изменении / удалении задачи кэш сбрасывается
- В ответе заголовок **`X-Cache: HIT`** или **`MISS`** (виден в DevTools → Network)

Проверка:

```powershell
# После логина обновите список задач несколько раз — первый запрос MISS, следующие HIT
docker logs task-service-1 --tail 5
docker exec redis redis-cli KEYS "tasks:list:*"
```

## Prometheus и Grafana

Каждый HTTP-сервис отдаёт метрики на **`GET /metrics`**:

| Метрика | Описание |
|---------|----------|
| `http_requests_total` | Счётчик запросов (service, method, route, status) |
| `http_request_duration_seconds` | Гистограмма латентности |
| `task_cache_hits_total` / `task_cache_misses_total` | Redis-кэш в task-service |
| `task_service_up` | Реплика task-service жива (по `INSTANCE_ID`) |

| URL | Назначение |
|-----|------------|
| http://localhost:9090 | Prometheus UI |
| http://localhost:3001 | Grafana (`admin` / `admin`) |

Дашборд **Satisfactory Task Manager** подключается автоматически (папка *Satisfactory*).

Проверка:

```powershell
curl http://localhost:8080/metrics
curl http://localhost:9090/targets
```

## RabbitMQ

После каждого изменения задачи **task-service** публикует событие в fanout-exchange **`task.events`**:

| Событие | Когда |
|---------|-------|
| `task.created` | `POST /tasks` |
| `task.updated` | `PATCH /tasks/{id}`, взять задачу |
| `task.deleted` | `DELETE /tasks/{id}` |

**task-worker** (отдельный контейнер) читает очередь `task.worker`:

1. Пишет audit-лог (`docker logs task-worker`)
2. Сбрасывает Redis-кэш списков
3. **Прогревает** кэш `all` / `completed` / `mine` из Postgres — следующий `GET /tasks` чаще отдаёт **HIT** без нагрузки на БД

| URL | Назначение |
|-----|------------|
| http://localhost:15672 | RabbitMQ Management (`guest` / `guest`) |

Метрика в Prometheus: `rabbitmq_messages_published_total{event_type=...}` на task-service.

Проверка:

```powershell
docker logs task-worker --tail 20
# Создайте или измените задачу в UI — в логах worker появится [audit] task.created ...
```

## Деплой на VPS

Пошаговая инструкция: [deploy/VPS.md](deploy/VPS.md)

```bash
git clone https://github.com/dratbo/Test-KR2.git /opt/satisfactory-task-manager
cd /opt/satisfactory-task-manager/satiafactory-task-manager
cp deploy/.env.example deploy/.env   # смените пароли
docker compose -f docker-compose.vps.yml --env-file deploy/.env up -d --build
```

Рекомендуется VPS **4 GB RAM**. Снаружи открыт только порт UI (`GATEWAY_PORT`, по умолчанию 8080).

## Дальше для «нагруженной» работы

Уже есть: горизонтальное масштабирование, NGINX round-robin, `X-Instance-ID`, healthcheck Postgres, **Redis-кэш**, **Prometheus/Grafana**, **RabbitMQ + task-worker**.

План расширения:

1. **Нагрузочное тестирование** — `k6` или `hey` на `http://localhost:8080/tasks`

## Локальный запуск без Docker

Запустите PostgreSQL, затем в отдельных терминалах: `user-service`, `task-service` (с `INSTANCE_ID`), `gateway`. URL задаётся через переменные `DATABASE_URL`, `JWT_SECRET`, `USER_SERVICE_URL`, `TASK_SERVICE_URL`.

# Test-KR2 — Satisfactory Task Manager

Учебный проект: веб-приложение для постановки и ведения производственных задач в контексте игры **Satisfactory**.

Рабочая копия приложения: [`satiafactory-task-manager/`](satiafactory-task-manager/).

## Техническое задание (кратко)

1. **Микросервисная архитектура на Go**
   - `gateway` — UI и API-шлюз
   - `user-service` — регистрация, авторизация (JWT), избранные исполнители
   - `task-service` — задачи (несколько реплик)
   - `satisfactory-data-service` — рецепты, предметы, постройки, тиры HUB
   - `PostgreSQL` — хранение данных

2. **Горизонтальное масштабирование**
   - 3 реплики `task-service` за **NGINX** (round-robin)
   - Проверка балансировки по заголовку `X-Instance-ID`

3. **Функциональность для пользователя**
   - Регистрация / вход
   - Создание задач по рецептам Satisfactory (поиск, превью, количество предм./мин)
   - Задачи команды, свои задачи, выполненные
   - Назначение исполнителя, статусы, подзадачи
   - Русский интерфейс и названия рецептов v1.0

4. **Производственный план (расширение)**
   - Расчёт построек, ресурсов на строительство, цепочки ингредиентов
   - Энергомодули, уровни конвейеров и труб
   - Тир HUB и фильтрация доступных рецептов/построек
   - Сохранение настроек производства в задаче

## Быстрый старт

```powershell
cd satiafactory-task-manager
docker compose -f docker-compose.balance.yml up -d --build
```

Приложение: **http://localhost:8080**

Подробности — в [README проекта](satiafactory-task-manager/README.md).

## На чём остановились (состояние на июнь 2026)

Реализовано и проверено:

- План производства с энергомодулями (marginal greedy allocator по цепочке)
- Логистика (конвейеры / трубы), сворачиваемые секции плана
- Поиск и пагинация задач (5 на страницу)
- Цепочка рецептов вперёд / «задом наперёд»
- Русские названия рецептов v1.0, данные из `game-data.json`
- **Redis-кэш** списков задач (`X-Cache: HIT/MISS`)
- **RabbitMQ** — события `task.created/updated/deleted`, **task-worker** (audit + прогрев кэша)
- **Prometheus + Grafana** — метрики HTTP, кэша, RabbitMQ
- NGINX с динамическим DNS Docker (устойчивость после пересоздания контейнеров)

**Деплой на VPS:** [satiafactory-task-manager/deploy/VPS.md](satiafactory-task-manager/deploy/VPS.md) (git clone + `docker-compose.vps.yml`).

**Запланировано дальше:**

- Нагрузочное тестирование (`k6` / `hey`)

## Стек

Go · HTML templates · HTMX · PostgreSQL · Redis · RabbitMQ · Docker Compose · NGINX · Prometheus · Grafana

--- 

Браузер
   ↓
gateway :8080          ← UI, HTML, расчёт плана
   ├→ user-service :8081        (регистрация, логин, JWT)
   ├→ nginx :8090 → task-service-1|2|3 :8082   (задачи, кэш, события)
   ├→ data-service :8083        (рецепты из game-data.json)
   └→ postgres :5432
task-service → redis (кэш списков)
task-service → rabbitmq → task-worker → redis (инвалидация + прогрев)
prometheus ← /metrics всех сервисов → grafana

---

На VPS (по SSH):

cd /opt/satisfactory-task-manager/satiafactory-task-manager
Остановить
docker compose -f docker-compose.vps.yml --env-file deploy/.env down
Если у вас HTTPS через Caddy:

docker compose -f docker-compose.balance.yml -f docker-compose.https.yml --env-file deploy/.env down
Проверка:


docker compose -f docker-compose.vps.yml ps
# или с https:
docker compose -f docker-compose.balance.yml -f docker-compose.https.yml ps

---

Запустить обратно
Без пересборки (быстро):

docker compose -f docker-compose.vps.yml --env-file deploy/.env up -d
С HTTPS (Caddy):

docker compose -f docker-compose.balance.yml -f docker-compose.https.yml --env-file deploy/.env up -d
С пересборкой (если меняли код):

docker compose -f docker-compose.vps.yml --env-file deploy/.env up -d --build
Проверка:

docker compose -f docker-compose.vps.yml ps
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8080/login
В браузере: https://153.56.132.67:8080



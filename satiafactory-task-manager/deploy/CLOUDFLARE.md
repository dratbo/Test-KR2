# HTTPS через Cloudflare Tunnel

На VPS UI слушает **8080**. **Cloudflare Tunnel** даёт HTTPS на домене без открытия портов на origin и без отдельного reverse proxy на 443.

## Что получится

```
Браузер ──HTTPS──► Cloudflare ──туннель──► localhost:8080 (gateway)
```

Адрес: **https://tasks.ваш-домен.ru**

---

## Требования

- Домен добавлен в [Cloudflare](https://dash.cloudflare.com) (NS делегированы на Cloudflare)
- На VPS уже работает приложение (`docker compose …`, gateway на **8080**)

---

## Шаг 1. Создать туннель в Cloudflare

1. Откройте [Cloudflare Zero Trust](https://one.dash.cloudflare.com/) (бесплатный план подходит).
2. **Networks** → **Tunnels** → **Create a tunnel**.
3. Имя, например: `satisfactory-tasks`.
4. Выберите **Docker** (или Linux) — скопируйте **токен** (`eyJh…`). Он понадобится на VPS.

## Шаг 2. Публичный hostname

В мастере настройки туннеля (или **Public Hostname** → **Add**):

| Поле | Значение |
|------|----------|
| Subdomain | `tasks` (или другое) |
| Domain | ваш домен |
| Service type | **HTTP** |
| URL | `localhost:8080` |

Сохраните.

## Шаг 3. Запустить cloudflared на VPS

### Вариант A — Docker (рекомендуется)

На VPS в каталоге проекта:

```bash
cd /opt/satisfactory-task-manager/satiafactory-task-manager
nano deploy/.env
```

Добавьте строку (вставьте свой токен из шага 1):

```env
CLOUDFLARE_TUNNEL_TOKEN=eyJhIjoi...
```

Запуск туннеля вместе со стеком:

```bash
docker compose -f docker-compose.balance.yml -f docker-compose.cloudflare.yml --env-file deploy/.env up -d
```

Если используете `docker-compose.vps.yml`:

```bash
docker compose -f docker-compose.vps.yml -f docker-compose.cloudflare.yml --env-file deploy/.env up -d
```

### Вариант B — без Docker

```bash
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb -o cloudflared.deb
sudo dpkg -i cloudflared.deb
sudo cloudflared service install <ВАШ_ТОКЕН>
sudo systemctl start cloudflared
sudo systemctl enable cloudflared
```

## Шаг 4. Проверка

1. Откройте **https://tasks.ваш-домен.ru**
2. Должна открыться страница входа Task Manager
3. В Cloudflare → Tunnels — статус **Healthy**

Логи туннеля:

```bash
docker logs cloudflared --tail 30
# или
sudo journalctl -u cloudflared -f
```

---

## SSL в Cloudflare

Туннель уже отдаёт HTTPS пользователям. В **SSL/TLS** → Overview можно оставить **Full** — на origin по-прежнему HTTP `localhost:8080`, это нормально.

---

## Файрвол

Порт **8080** можно закрыть снаружи (только localhost), если ходите только через домен:

```bash
# опционально — UI только через Cloudflare
sudo ufw delete allow 8080/tcp
```

---

## Обновление приложения

```bash
cd /opt/satisfactory-task-manager/satiafactory-task-manager
git pull
docker compose -f docker-compose.balance.yml -f docker-compose.cloudflare.yml --env-file deploy/.env up -d --build
```

Туннель перезапускать не нужно, если токен и `localhost:8080` не менялись.

---

## Частые проблемы

| Симптом | Решение |
|---------|---------|
| 502 / Error 1033 | Gateway не запущен: `docker ps`, `curl http://127.0.0.1:8080/login` |
| Tunnel Unhealthy | Проверить токен в `deploy/.env`, `docker logs cloudflared` |
| Логин не сохраняется | Очистить cookies, открыть сайт только по **https://** домену |
| Долго грузится первый раз | Нормально после cold start на 1 ядре |

---

## Альтернатива: DNS + порт 80

Если не хотите Tunnel: поднимите nginx на **80** → `127.0.0.1:8080`, в DNS включите оранжевое облако, SSL mode **Flexible**. Подробнее — в комментариях к `docker-compose.cloudflare.yml`.

# Деплой на VPS (git clone + сборка)

Подходит для VPS **4 GB RAM**, **1 ядро**, рядом с Telegram proxy.

## Что будет запущено

Полный стек `docker-compose.vps.yml` (на базе balance):

- 3× task-service + NGINX
- Redis, RabbitMQ, task-worker
- Prometheus, Grafana (только `127.0.0.1` на VPS)
- **Снаружи открыт только UI** — порт `GATEWAY_PORT` (по умолчанию **8080**)

## Шаг 1. Подключиться к VPS

```bash
ssh user@ВАШ_IP
```

## Шаг 2. Проверить порты (важно для Telegram proxy)

```bash
ss -tlnp
free -h
docker ps    # если Docker уже есть
```

Если **8080** занят — в `deploy/.env` укажите другой порт, например `GATEWAY_PORT=8088`.

## Шаг 3. Установить Docker (если ещё нет)

```bash
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
newgrp docker
docker compose version
```

Перезагрузка VPS **не обязательна**.

## Шаг 4. Клонировать репозиторий

```bash
sudo mkdir -p /opt
sudo git clone https://github.com/dratbo/Test-KR2.git /opt/satisfactory-task-manager
sudo chown -R $USER:$USER /opt/satisfactory-task-manager
cd /opt/satisfactory-task-manager/satiafactory-task-manager
```

## Шаг 5. Настроить пароли

```bash
cp deploy/.env.example deploy/.env
nano deploy/.env
```

Смените минимум:

- `POSTGRES_PASSWORD`
- `JWT_SECRET`
- `GRAFANA_ADMIN_PASSWORD`

## Шаг 6. (Рекомендуется) Swap 2 GB

Помогает при `docker build` на 4 GB RAM:

```bash
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

## Шаг 7. Сборка и запуск

```bash
cd /opt/satisfactory-task-manager/satiafactory-task-manager
docker compose -f docker-compose.vps.yml --env-file deploy/.env up -d --build
```

Первая сборка — **10–20 минут** на 1 ядре. Proxy не трогается.

## Шаг 8. Проверка

```bash
docker compose -f docker-compose.vps.yml ps
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8080/login
```

В браузере: **http://ВАШ_IP:8080**

### HTTPS (Caddy)

**https://privately.proven.hornet:8443** — см. [HTTPS.md](HTTPS.md) (`docker-compose.https.yml`).

## Шаг 9. Файрвол (если ufw включён)

```bash
sudo ufw allow 8080/tcp    # или ваш GATEWAY_PORT
sudo ufw status
```

Порт Telegram proxy **не закрывайте**.

## Обновление после git push

```bash
cd /opt/satisfactory-task-manager
git pull
cd satiafactory-task-manager
docker compose -f docker-compose.vps.yml --env-file deploy/.env up -d --build
```

## Grafana / Prometheus с вашего ПК

На VPS они слушают только localhost. Туннель:

```bash
ssh -L 9090:127.0.0.1:9090 -L 3001:127.0.0.1:3001 user@ВАШ_IP
```

Открыть: http://localhost:3001 (Grafana)

## Остановка (proxy не затрагивается)

```bash
cd /opt/satisfactory-task-manager/satiafactory-task-manager
docker compose -f docker-compose.vps.yml down
```

Данные Postgres сохраняются в volume `postgres_data`.

## Автоскрипт

```bash
cd /opt/satisfactory-task-manager/satiafactory-task-manager
bash deploy/vps-deploy.sh
```

## Частые проблемы

| Симптом | Решение |
|---------|---------|
| `docker build` падает по памяти | swap 2G, `docker system prune -f`, повторить build |
| 502 при создании задачи | `docker compose ... up -d --force-recreate nginx task-service-1 task-service-2 task-service-3` |
| Порт занят | сменить `GATEWAY_PORT` в `deploy/.env` |
| Не открывается снаружи | `ufw allow`, проверить панель хостинга (security groups) |

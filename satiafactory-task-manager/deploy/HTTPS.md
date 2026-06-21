# HTTPS через Caddy

Reverse proxy **Caddy** перед gateway: TLS снаружи, HTTP внутри Docker-сети.

## Схема

```
Браузер ──HTTPS──► Caddy ──HTTP──► gateway:8080 (внутри docker)
```

## 1. DNS

A-запись вашего домена должна указывать на **публичный IP VPS**.

```bash
dig +short your-domain.example
```

## 2. Файрвол

```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp   # или порт из Caddyfile
sudo ufw status
```

## 3. Настройка Caddyfile

Отредактируйте `deploy/Caddyfile` — укажите свой домен и порт.

Для **Let's Encrypt** (обычный домен `.com`, `.ru`, …):

```
your-domain.example {
    reverse_proxy gateway:8080
}
```

Для локального/тестового домена без публичного CA — блок с `tls internal` (самоподписанный сертификат, браузер покажет предупреждение).

## 4. Запуск

```bash
cd /opt/satisfactory-task-manager/satiafactory-task-manager

docker compose -f docker-compose.balance.yml -f docker-compose.https.yml --env-file deploy/.env up -d --build
```

Добавить Caddy к уже работающему стеку:

```bash
docker compose -f docker-compose.balance.yml -f docker-compose.https.yml --env-file deploy/.env up -d
```

## 5. Проверка

```bash
docker logs caddy --tail 30
curl -sI https://your-domain.example/login
```

## 6. Смена домена

Отредактируйте `deploy/Caddyfile` и пересоздайте Caddy:

```bash
docker compose -f docker-compose.balance.yml -f docker-compose.https.yml up -d --force-recreate caddy
```

## Остановка HTTPS (только HTTP)

```bash
docker compose -f docker-compose.balance.yml -f docker-compose.https.yml stop caddy
docker compose -f docker-compose.balance.yml up -d --force-recreate gateway
```

Второй шаг вернёт gateway на `0.0.0.0:8080` (без override из `https.yml`).

# HTTPS через Caddy (без Cloudflare)

Домен: **privately.proven.hornet**  
Telegram proxy остаётся на **:443**, UI доступен по **https://privately.proven.hornet:8443**

## Схема

```
Браузер ──HTTPS :8443──► Caddy ──HTTP──► gateway:8080
Telegram proxy ───────────── :443 (не трогаем)
```

## 1. DNS

Убедитесь, что **privately.proven.hornet** указывает на **публичный IP VPS** (A-запись).

Проверка с вашего ПК:

```bash
ping privately.proven.hornet
# или
dig +short privately.proven.hornet
```

## 2. Файрвол на VPS

```bash
sudo ufw allow 80/tcp
sudo ufw allow 8443/tcp
sudo ufw status
```

Порт **443** не открывайте для Caddy — он у Telegram.

## 3. Запуск

```bash
cd /opt/satisfactory-task-manager/satiafactory-task-manager

# если ещё не запущено — поднимите стек + Caddy
docker compose -f docker-compose.balance.yml -f docker-compose.https.yml --env-file deploy/.env up -d --build
```

Только добавить Caddy к уже работающему стеку:

```bash
docker compose -f docker-compose.balance.yml -f docker-compose.https.yml --env-file deploy/.env up -d
```

## 4. Проверка

```bash
docker logs caddy --tail 30
curl -sI http://127.0.0.1:8080/login
```

В браузере: **https://privately.proven.hornet:8443**

Первый запуск: Caddy запросит сертификат Let's Encrypt (нужны открытые **80** и доступность домена из интернета).

## 5. Если сертификат не выдаётся

Для доменов, недоступных для проверки Let's Encrypt (например, только внутри mesh-сети), отредактируйте `deploy/Caddyfile`:

```
privately.proven.hornet {
    tls internal
    reverse_proxy gateway:8080
}
```

Перезапуск:

```bash
docker compose -f docker-compose.balance.yml -f docker-compose.https.yml up -d --force-recreate caddy
```

Браузер покажет предупреждение — нажмите «Дополнительно» → перейти на сайт.

## 6. Смена домена

Отредактируйте `deploy/Caddyfile` и пересоздайте Caddy:

```bash
docker compose -f docker-compose.balance.yml -f docker-compose.https.yml up -d --force-recreate caddy
```

## Остановка HTTPS (оставить только HTTP на localhost)

```bash
docker compose -f docker-compose.balance.yml -f docker-compose.https.yml stop caddy
docker compose -f docker-compose.balance.yml up -d --force-recreate gateway
```

Второй шаг вернёт gateway на `0.0.0.0:8080` (без override из https.yml).

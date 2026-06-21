# Test-KR2 — Satisfactory Task Manager

Микросервисное веб-приложение на **Go** для координации производственных задач в игре **Satisfactory**: JWT-авторизация, рецепты v1.0, производственный план, NGINX-балансировка, Redis, RabbitMQ, Prometheus/Grafana.

**Код приложения:** [`satiafactory-task-manager/`](satiafactory-task-manager/)

## Быстрый старт

```bash
cd satiafactory-task-manager
docker compose -f docker-compose.balance.yml up -d --build
```

Приложение: **http://localhost:8080**

Полная документация, архитектура, нагрузочные тесты и деплой — в [README проекта](satiafactory-task-manager/README.md).

## Стек

Go · PostgreSQL · Docker Compose · NGINX · Redis · RabbitMQ · Prometheus · Grafana

## Демо

Схема архитектуры: [satiafactory-task-manager/docx/diagram_architecture.png](satiafactory-task-manager/docx/diagram_architecture.png)

## Демонстрация работы

### 1.	Регистрация

![Регистрация](satiafactory-task-manager/docx/registration.png)

### 2. Вход

![Вход](satiafactory-task-manager/docx/log%20in.png)

### 3.	Создание задачи

![Создание задачи](satiafactory-task-manager/docx/Creating%20a%20task.png)

После входа пользователь переходит к списку задач. Доступны вкладки «Задачи команды», «Свои задачи», «Выполненные», поиск и пагинация.
При создании задачи пользователь выбирает рецепт Satisfactory, указывает целевую производительность и параметры производства. При повторных запросах списка задач заголовок X-Cache принимает значение HIT; заголовок X-Instance-ID чередуется между tasks-1, tasks-2, tasks-3

### 4. Просмотр задачи

![Просмотр задачи](satiafactory-task-manager/docx/Viewing%20the%20issue.png)

# Мониторинг Docker-контейнеров

Этот проект представляет собой макет системы мониторинга Docker-контейнеров. Он демонстрирует базовую функциональность, такую как пинг хостов, хранение результатов в базе данных и их визуализацию.

## Сервисы

- **Backend**: RESTful API для запроса и добавления результатов пинга.
- **Frontend**: React-приложение для отображения результатов пинга.
- **Pinger**: Сервис, который пингует хосты и отправляет результаты на backend.
- **Database**: База данных PostgreSQL для хранения результатов пинга.
- **Nginx**: Обратный прокси-сервер обеспечивает внешний доступ к **frontend** и **backend**.

## Запуск проекта

- Клонируйте репозиторий.
- Выполните команду `docker-compose up --build`.
- Откройте в браузере `http://localhost`.
- Запуск в режиме отладки `DEBUG= docker-compose up`

## Публичные API-эндпоинты

- `GET  /api/hosts`: Получить список хостов для пинга.
- `GET  /api/ping-results`: Получить последние результаты пинга.

## Как это работает

### Pinger

При запуске ожидает доступности **backend** и получает список хостов, которые необходимо отслеживать.

`GET /hosts`

```json
{
    "hosts": [
        {
            "host_id": 1,
            "host_name": "host1" // IP или FQDN
        },
        // ...
    ]
}
```

Запускает сканер для каждого хоста и отправляет результаты на **backend**. 
Интервал сканирования задается переменной окружения `PING_INTERVAL` (по умолчанию `10s`).
Чтобы избежать излишней нагрузки на **backend**, результаты собираются в батчи перед отправкой.

`POST /ping-results`

```json
{
    "ping_results": [
        {
            "host_id": 1,
            "rtt": 100500, // round-trip time, duration ns
            "time": "2006-01-02T15:04:05Z07:00", // RFC3339
            "status": true
        },
        // ...
    ]
}
```

### Backend

Предоставляет следующие API-эндпоинты:

- `GET  /pub/hosts`
- `GET  /pub/ping-results`
- `GET  /ping-results`
- `POST /ping-results`

При запуске ожидает доступности базы данных, получает список новых хостов через переменную окружения `PING_HOSTS` и добавляет их в базу.

Получает результаты пингов на `POST /ping-results` и сохраняет их в базе данных.

Предоставляет последние результаты на эндпоинте `GET /ping-results`. Чтобы минимизировать нагрузку на базу данных, результаты кэшируются в памяти. При запуске кэш заполняется данными из базы даных.

### Nginx

Проксирует запросы к **frontend** и **backend**. Ограничивает доступ к **backend**.

```nginx
server {
    listen 80;

    location / {
        proxy_pass http://frontend:4173/;
    }

    location /api/ {
        proxy_pass http://backend:8080/pub/;
    }
}
```

### Frontend

Каждые 5 секунд запрашивает публичный эндпоинт `GET /api/ping-results` и обновляет страницу с результатами.

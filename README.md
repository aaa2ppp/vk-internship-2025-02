# Docker Container Monitoring

This project provides a solution for monitoring Docker containers by pinging their IP addresses and storing the results in a PostgreSQL database. The results are displayed on a web page.

## Services

- **Backend**: RESTful API for querying and adding ping results.
- **Frontend**: React application to display ping results.
- **Pinger**: Service that pings hosts and sends results to the backend.
- **Database**: PostgreSQL database to store ping results.
- **Nginx**: Reverse proxy to serve the frontend and backend.

## How to Run

1. Clone the repository.
2. Run `docker-compose up --build`.
3. Open `http://localhost` in your browser.

## Public API Endpoints

- `GET  /api/hosts`: Get list of hosts to ping.
- `GET  /api/ping-results`: Get last ping results.


## How it works

### Pinger

- При старте ожидает **backend**, получает список хостов которые нужно пинговать

`GET /hosts`

```json
{
    "hosts": [
        {
            "host_id": 1,
            "host_name": "host1" // IP or FQDN
        },
        // ...
    ]
}
```

- Запускает сканер для каждого хоста и сбрасывает результаты на **backend**.
Чтобы не нагружать базу еденичными запросами, перед отправкой результаты собираются в батчи.

`POST /ping-results`

```json
{
    "ping_resuts": [
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


## Backend

Предоставляет ручки:

- `GET  /pub/hosts`
- `GET  /pub/ping-results`
- `GET  /pub/ping`
- `GET  /ping-results`
- `POST /ping-results`
- `GET  /ping`

При старте ждет базу, получает список новых хостов через переменную окружения `PING_HOSTS`
и добавляет их в базу.

Получает на `POST /ping-results` и скидывает результату в базу. Чтобы не дергать базу лишний раз,
кэширует последние результаты и предоставляет их на ручке `GET /ping-results`.

## Nginx

Проксирует **frontend** и **backend**. Ограничивает доступ к **backend**.

```nginx.conf
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

## Frontend

Каждые 5с дергает публичную ручку `GET /api/ping-results` и обновляет страницу.
Я не дока во фронтенд и в частности в *React*. Как получилось, так получилось...

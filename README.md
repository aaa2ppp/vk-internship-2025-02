# Docker Container Monitoring

This project provides a solution for monitoring Docker containers by pinging their IP addresses and storing the results in a PostgreSQL database. The results are displayed on a web page.

## Services

- **Backend**: RESTful API for querying and adding ping results.
- **Frontend**: React application to display ping results.
- **Pinger**: Service that pings Docker containers and sends results to the backend.
- **Database**: PostgreSQL database to store ping results.
- **Nginx**: Reverse proxy to serve the frontend and backend.

## How to Run

1. Clone the repository.
2. Run `docker-compose up --build`.
3. Open `http://localhost` in your browser.

## API Endpoints

- `GET /ping-results`: Get all ping results.
- `POST /add-ping-result`: Add a new ping result.
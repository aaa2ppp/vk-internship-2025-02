CREATE TABLE ping_results (
    id SERIAL PRIMARY KEY,
    ip VARCHAR(15) NOT NULL,
    ping_time TIMESTAMP NOT NULL,
    success BOOLEAN NOT NULL
);
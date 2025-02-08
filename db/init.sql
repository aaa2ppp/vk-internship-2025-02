CREATE TABLE host (
    host_id SERIAL PRIMARY KEY,
    host_name VARCHAR(128) NOT NULL UNIQUE
);

CREATE TABLE ping_result (
    id BIGSERIAL PRIMARY KEY,
    host_id INT NOT NULL REFERENCES host,
    ip INET NOT NULL,
    ping_time TIMESTAMP NOT NULL,
    ping_rtt int NOT NULL, 
    success BOOLEAN NOT NULL
);

BEGIN;
CREATE TABLE IF NOT EXISTS devices(
    id UUID PRIMARY KEY,
    device_name VARCHAR(50) NOT NULL,
    attached_protocol BOOLEAN NOT NULL,
    protocol VARCHAR(30) NOT NULL,
    device_type VARCHAR(30) NOT NULL,
    latitude CHAR(10) NOT NULL,
    longitude CHAR(10) NOT NULL,
    device_address VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP
);
COMMIT;
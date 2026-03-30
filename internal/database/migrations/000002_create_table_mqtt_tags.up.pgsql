BEGIN;
CREATE TABLE IF NOT EXISTS mqtt_tags(
    id UUID PRIMARY KEY,
    id_device UUID NOT NULL,
    raw_topic VARCHAR(255) NOT NULL,
    formatted_topic VARCHAR(255) NOT NULL,
    tag_name VARCHAR(30) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP,
    CONSTRAINT fk_id_device FOREIGN KEY (id_device) REFERENCES devices(id)
);
COMMIT;
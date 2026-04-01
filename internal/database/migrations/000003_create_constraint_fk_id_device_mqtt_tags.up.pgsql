DO $$
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_id_device') THEN
        ALTER TABLE mqtt_tags 
        ADD CONSTRAINT fk_id_device 
        FOREIGN KEY (id_device) REFERENCES devices(id);
    END IF;
END $$;
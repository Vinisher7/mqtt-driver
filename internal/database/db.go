package database

import (
	"context"
	"database/sql"
	"fmt"
	"mqtt-driver/internal/models"

	_ "github.com/microsoft/go-mssqldb"
)

const query = `
SELECT
    CAST(mt.id       AS VARCHAR(36)) AS tag_id,
    mt.name                          AS tag_name,
    mt.raw_topic,
    mt.formatted_topic,
    CAST(d.id        AS VARCHAR(36)) AS device_id
FROM mqtt_tags mt
JOIN devices d ON d.id = mt.id_device
WHERE d.protocol = 'mqtt'
ORDER BY d.id, mt.id
`

func LoadTags(ctx context.Context, db *sql.DB) ([]models.MQTTTag, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	defer rows.Close()

	var tags []models.MQTTTag
	for rows.Next() {
		var t models.MQTTTag
		if err := rows.Scan(
			&t.TagID,
			&t.TagName,
			&t.RawTopic,
			&t.FormattedTopic,
			&t.DeviceID,
		); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, nil
}

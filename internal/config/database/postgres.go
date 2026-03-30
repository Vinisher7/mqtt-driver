package database

import (
	"context"
	"errors"
	"fmt"
	"mqtt-driver/internal/config/logger"
	"mqtt-driver/internal/models"
	"os"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var (
	POSTGRES_USER     = "POSTGRES_USER"
	POSTGRES_PASSWORD = "POSTGRES_PASSWORD"
	POSTGRES_SERVER   = "POSTGRES_SERVER"
	POSTGRES_PORT     = "POSTGRES_PORT"
	POSTGRES_NAME     = "POSTGRES_DB"
)

func NewPostgresClient(ctx context.Context) (db *pgx.Conn, err error) {
	logger.Info("Init NewPostgresClient", zap.String("journey", "database"))

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv(POSTGRES_USER),
		os.Getenv(POSTGRES_PASSWORD),
		os.Getenv(POSTGRES_SERVER),
		os.Getenv(POSTGRES_PORT),
		os.Getenv(POSTGRES_NAME),
	)

	db, err = pgx.Connect(ctx, connString)
	if err != nil {
		logger.Error("Connect func returned an error", err, zap.String("journey", "database"))
		return nil, fmt.Errorf("error connecting with postgres: %w", err)
	}

	logger.Info("NewPostgresClient executed successfully", zap.String("journey", "database"))

	return db, nil
}

func TestPostgresConnection(ctx context.Context, db *pgx.Conn) (err error) {
	logger.Info("Init TestPostgresConnection", zap.String("journey", "database"))

	err = db.Ping(ctx)
	if err != nil {
		logger.Error("Ping func returned an error", err, zap.String("journey", "database"))
		return fmt.Errorf("error testing a connection with postgres: %w", err)
	}

	logger.Info("TestPostgresConnection executed successfully", zap.String("journey", "database"))

	return nil
}

func LoadTags(ctx context.Context, db *pgx.Conn) (tags []models.MQTTTag, err error) {
	logger.Info("Init LoadTags", zap.String("journey", "database"))

	query := `
		SELECT
			CAST(mt.id AS VARCHAR(36)) AS tag_id,
			mt.tag_name,
			mt.raw_topic,
			mt.formatted_topic,
			CAST(d.id AS VARCHAR(36)) AS device_id
		FROM mqtt_tags mt
		JOIN devices d ON d.id = mt.id_device
		WHERE d.protocol = 'mqtt'
		ORDER BY d.id, mt.id
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		logger.Error("Query func returned an error", err, zap.String("journey", "database"))
		return tags, fmt.Errorf("error : %w", err)
	}

	tags, err = pgx.CollectRows(rows, pgx.RowToStructByName[models.MQTTTag])

	if errors.Is(err, pgx.ErrNoRows) {
		logger.Error("Tags not found", err, zap.String("journey", "database"))
		return tags, fmt.Errorf("error : %w", err)
	}

	if err != nil {
		logger.Error("CollectRows func returned an error", err, zap.String("journey", "database"))
		return tags, fmt.Errorf("error : %w", err)

	}

	logger.Info("LoadTags executed successfully", zap.String("journey", "database"))

	return tags, err
}

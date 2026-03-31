package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"mqtt-driver/internal/config/database"
	"mqtt-driver/internal/config/logger"
	"mqtt-driver/internal/mqtt"

	"go.uber.org/zap"
)

func main() {
	logger.Info("Starting application...")

	// // ── Banco de dados ────────────────────────────────────────────────
	ctx := context.Background()

	db, err := database.NewPostgresClient(ctx)
	if err != nil {
		logger.Error("NewPostgresClient func returned an error", err, zap.String("journey", "database"))
		panic(err)
	}
	defer db.Close(ctx)

	err = database.TestPostgresConnection(ctx, db)
	if err != nil {
		logger.Error("TestPostgresConnection func returned an error", err, zap.String("journey", "database"))
		panic(err)
	}

	tags, err := database.LoadTags(ctx, db)
	if err != nil {
		logger.Error("LoadTags func returned an error", err, zap.String("journey", "database"))
		panic(err)
	}

	// // ── MQTT Publisher ────────────────────────────────────────────────
	pub, err := mqtt.NewPublisher()
	if err != nil {
		logger.Error("NewPublisher func returned an error", err, zap.String("journey", "publisher"))
	}

	// // ── MQTT Subscriber ───────────────────────────────────────────────
	sub, err := mqtt.NewSubscriber(pub)
	if err != nil {
		logger.Error("NewSubscriber func returned an error", err, zap.String("journey", "subscriber"))
	}

	sub.SubscribeGroups(tags)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down application...")
}

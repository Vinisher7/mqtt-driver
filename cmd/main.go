package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"mqtt-driver/internal/config"
	"mqtt-driver/internal/database"
	mqttpkg "mqtt-driver/internal/mqtt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// ── Banco de dados ────────────────────────────────────────────────
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()
	log.Println("[DB] connected to SQL Server")

	// ── Carrega tags no startup ───────────────────────────────────────
	ctx := context.Background()
	tags, err := database.LoadTags(ctx, db)
	if err != nil {
		log.Fatalf("load tags: %v", err)
	}
	log.Printf("[DB] %d tag(s) carregada(s)", len(tags))

	if len(tags) == 0 {
		log.Fatal("nenhuma tag encontrada — verifique o banco e o campo protocol")
	}

	// ── MQTT Publisher ────────────────────────────────────────────────
	pub, err := mqttpkg.NewPublisher(cfg)
	if err != nil {
		log.Fatalf("publisher: %v", err)
	}

	// ── MQTT Subscriber ───────────────────────────────────────────────
	sub, err := mqttpkg.NewSubscriber(cfg, pub)
	if err != nil {
		log.Fatalf("subscriber: %v", err)
	}

	// Assina todos os tópicos divididos em grupos paralelos
	sub.SubscribeGroups(tags)

	// ── Graceful shutdown ─────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
}

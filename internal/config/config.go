package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBServer     string
	DBPort       int
	DBUser       string
	DBPassword   string
	DBName       string
	MQTTBroker   string
	MQTTClientID string
	MQTTUsername string
	MQTTPassword string
	TagGroupSize int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "1433"))
	tagGroupSize, _ := strconv.Atoi(getEnv("TAG_GROUP_SIZE", "5"))

	return &Config{
		DBServer:     getEnv("DB_SERVER", "localhost"),
		DBPort:       dbPort,
		DBUser:       getEnv("DB_USER", "sa"),
		DBPassword:   getEnv("DB_PASSWORD", ""),
		DBName:       getEnv("DB_NAME", ""),
		MQTTBroker:   getEnv("MQTT_BROKER", "tcp://localhost:1883"),
		MQTTClientID: getEnv("MQTT_CLIENT_ID", "mqtt-driver"),
		MQTTUsername: getEnv("MQTT_USERNAME", ""),
		MQTTPassword: getEnv("MQTT_PASSWORD", ""),
		TagGroupSize: tagGroupSize,
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

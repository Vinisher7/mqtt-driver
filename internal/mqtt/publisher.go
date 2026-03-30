package mqtt

import (
	"fmt"
	"mqtt-driver/internal/config/logger"
	"os"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

var (
	MQTT_BROKER    = "MQTT_BROKER"
	MQTT_CLIENT_ID = "MQTT_CLIENT_ID"
	MQTT_USERNAME  = "MQTT_USERNAME"
	MQTTPassword   = "MQTTPassword"
	TAG_GROUP_SIZE = "TAG_GROUP_SIZE"
)

type Publisher struct {
	client paho.Client
}

func NewPublisher() (pub *Publisher, err error) {
	logger.Info("Init NewPublisher", zap.String("journey", "publisher"))

	opts := paho.NewClientOptions().
		AddBroker(os.Getenv(MQTT_BROKER)).
		SetClientID(os.Getenv(MQTT_CLIENT_ID) + "-pub").
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectTimeout(10 * time.Second)

	if os.Getenv(MQTT_USERNAME) == "" {
		err = fmt.Errorf("mqtt username is empty")
		logger.Error("Getenv func returned an error", err, zap.String("journey", "publisher"))
		return pub, err
	}

	opts.SetUsername(os.Getenv(MQTT_USERNAME)).SetPassword(os.Getenv(MQTTPassword))

	c := paho.NewClient(opts)
	if tok := c.Connect(); tok.Wait() && tok.Error() != nil {
		logger.Error("Connect func returned an error", err, zap.String("journey", "publisher"))
		return pub, fmt.Errorf("error creating a connection with the message broker: %w", tok.Error())
	}

	logger.Info("NewPublisher executed successfully", zap.String("journey", "publisher"))

	return &Publisher{
		client: c,
	}, nil
}

func (p *Publisher) Publish(deviceID, tagID, payload string) {
	logger.Info("Init Publish", zap.String("journey", "publisher"))

	topic := fmt.Sprintf("/devices/formatted_data/erd/%s/%s", deviceID, tagID)

	tok := p.client.Publish(topic, 0, true, payload)

	tok.Wait()

	if err := tok.Error(); err != nil {
		message := fmt.Sprintf("Publish func returned an error on topic=%s", topic)
		logger.Error(message, err, zap.String("journey", "publisher"))
	}

	logger.Info("Publish executed successfully", zap.String("journey", "publisher"))
}

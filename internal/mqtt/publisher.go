package mqtt

import (
	"fmt"
	"log"
	"time"

	"mqtt-driver/internal/config"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type Publisher struct {
	client paho.Client
}

func NewPublisher(cfg *config.Config) (*Publisher, error) {
	opts := paho.NewClientOptions().
		AddBroker(cfg.MQTTBroker).
		SetClientID(cfg.MQTTClientID + "-pub").
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectTimeout(10 * time.Second)

	if cfg.MQTTUsername != "" {
		opts.SetUsername(cfg.MQTTUsername).SetPassword(cfg.MQTTPassword)
	}

	c := paho.NewClient(opts)
	if tok := c.Connect(); tok.Wait() && tok.Error() != nil {
		return nil, fmt.Errorf("mqtt publisher connect: %w", tok.Error())
	}
	log.Println("[Publisher] connected to", cfg.MQTTBroker)
	return &Publisher{client: c}, nil
}

// Publish envia para:
// /devices/formatted_data/erd/{deviceID}/{tagID}
func (p *Publisher) Publish(deviceID, tagID, payload string) {
	topic := fmt.Sprintf("/devices/formatted_data/erd/%s/%s", deviceID, tagID)
	tok := p.client.Publish(topic, 0, true, payload)
	tok.Wait()
	if err := tok.Error(); err != nil {
		log.Printf("[Publisher] error topic=%s err=%v", topic, err)
	}
}

package mqtt

import (
	"encoding/json"
	"fmt"
	"mqtt-driver/internal/config/logger"
	"mqtt-driver/internal/models"
	"os"
	"strconv"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

var (
	// MQTT_BROKER    = "MQTT_BROKER"
	// MQTT_CLIENT_ID = "MQTT_CLIENT_ID"
	// MQTT_USERNAME  = "MQTT_USERNAME"
	// MQTTPassword   = "MQTTPassword"
	// TAG_GROUP_SIZE = "TAG_GROUP_SIZE"

	BRAZIL_LOCATION = time.FixedZone("UTC-3", -3*60*60)
)

type Subscriber struct {
	client    paho.Client
	publisher *Publisher
}

func NewSubscriber(pub *Publisher) (sub *Subscriber, err error) {
	logger.Info("Init NewSubscriber", zap.String("journey", "subscriber"))

	opts := paho.NewClientOptions().
		AddBroker(os.Getenv(MQTT_BROKER)).
		SetClientID(os.Getenv(MQTT_CLIENT_ID) + "-sub").
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectTimeout(10 * time.Second)

	if os.Getenv(MQTT_USERNAME) == "" {
		err = fmt.Errorf("subscriber username is empty")
		logger.Error("Getenv func returned an error", err, zap.String("journey", "subscriber"))
		return sub, err
	}

	c := paho.NewClient(opts)
	if tok := c.Connect(); tok.Wait() && tok.Error() != nil {
		logger.Error("Connect func returned an error", err, zap.String("journey", "subscriber"))
		return sub, fmt.Errorf("error creating a connection with the message broker: %w", tok.Error())
	}

	logger.Info("NewSubscriber executed successfully", zap.String("journey", "subscriber"))

	return &Subscriber{
		client:    c,
		publisher: pub,
	}, nil
}

func (s *Subscriber) SubscribeGroups(tags []models.MQTTTag) {
	logger.Info("Init SubscribeGroups", zap.String("journey", "subscriber"))

	size, err := strconv.Atoi(os.Getenv(TAG_GROUP_SIZE))
	if err != nil {
		logger.Error("Atoi func returned an error", err, zap.String("journey", "subscriber"))
	}

	groups := chunkTags(tags, size)

	var wg sync.WaitGroup
	for i, group := range groups {
		wg.Add(1)
		go func(groupIdx int, grp []models.MQTTTag) {
			defer wg.Done()
			s.subscribeGroup(groupIdx, grp)
		}(i, group)
	}
	wg.Wait()

	logger.Info("SubscribeGroups executed successfully", zap.String("journey", "subscriber"))
}

func (s *Subscriber) subscribeGroup(groupIdx int, tags []models.MQTTTag) {
	logger.Info("Init subscribeGroup", zap.String("journey", "subscriber"))

	for _, tag := range tags {
		tag := tag

		listenTopic := fmt.Sprintf("/devices/raw_data/erd/%s/%s", tag.DeviceID, tag.RawTopic)

		tok := s.client.Subscribe(listenTopic, 0, func(_ paho.Client, msg paho.Message) {
			s.handleMessage(tag, msg.Payload())
		})

		tok.Wait()

		if err := tok.Error(); err != nil {
			message := fmt.Sprintf("error subscribing group=%d into topic=%s", groupIdx, listenTopic)
			logger.Error(message, err, zap.String("journey", "subscriber"))
			continue
		}

		message := fmt.Sprintf("subscribed group=%d into topic=%s successfully", groupIdx, listenTopic)
		logger.Info(message, zap.String("journey", "subscriber"))
	}

	logger.Info("subscribeGroup executed successfully", zap.String("journey", "subscriber"))
}

func (s *Subscriber) handleMessage(tag models.MQTTTag, rawBytes []byte) {
	logger.Info("Init handleMessage", zap.String("journey", "subscriber"))

	var raw models.RawPayload

	if err := json.Unmarshal(rawBytes, &raw); err != nil {
		message := fmt.Sprintf("invalid JSON received by tag=%s", tag.TagName)
		logger.Error(message, err, zap.String("journey", "subscriber"))
		return
	}

	utcTS, err := convertToUTC(raw.TS)
	if err != nil {
		message := fmt.Sprintf("convertToUTC returned an error to tag=%s", tag.TagName)
		logger.Error(message, err, zap.String("journey", "subscriber"))

		utcTS = time.Now().UTC().Format(time.RFC3339)
		logger.Info("timestamp fallback applied", zap.String("journey", "subscriber"))
	}

	out := models.FormattedPayload{
		TS:  utcTS,
		Val: raw.Val,
	}
	outBytes, err := json.Marshal(out)
	if err != nil {
		logger.Error("error marshaling payload", err, zap.String("journey", "subscriber"))
		return
	}

	s.publisher.Publish(tag.DeviceID, tag.TagID, string(outBytes))

	message := fmt.Sprintf("tag=%s was successfully published with TS=%s and Val=%s", tag.TagName, utcTS, raw.Val)
	logger.Info(message, zap.String("journey", "subscriber"))

	logger.Info("handleMessage executed successfully", zap.String("journey", "subscriber"))
}

func convertToUTC(ts string) (string, error) {
	logger.Info("Init convertToUTC", zap.String("journey", "subscriber"))

	formats := []string{
		"02/01/2006 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, layout := range formats {
		t, err := time.ParseInLocation(layout, ts, BRAZIL_LOCATION)

		if err == nil {
			logger.Info("convertToUTC executed successfully", zap.String("journey", "subscriber"))
			return t.UTC().Format(time.RFC3339), nil
		}
	}

	err := fmt.Errorf("invalid timestamp format: %s", ts)
	logger.Error("convertToUTC returned an error", err, zap.String("journey", "subscriber"))

	return "", err
}

func chunkTags(tags []models.MQTTTag, size int) [][]models.MQTTTag {
	logger.Info("Init chunkTags", zap.String("journey", "subscriber"))

	var chunks [][]models.MQTTTag

	for size < len(tags) {
		tags, chunks = tags[size:], append(chunks, tags[:size])
	}

	logger.Info("chunkTags executed successfully", zap.String("journey", "subscriber"))

	return append(chunks, tags)
}

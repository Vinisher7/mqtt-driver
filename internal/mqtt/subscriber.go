package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"mqtt-driver/internal/config"
	"mqtt-driver/internal/models"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// brazilLocation é UTC-3 fixo (sem DST — horário de Brasília padrão)
var brazilLocation = time.FixedZone("UTC-3", -3*60*60)

type Subscriber struct {
	client    paho.Client
	publisher *Publisher
	cfg       *config.Config
}

func NewSubscriber(cfg *config.Config, pub *Publisher) (*Subscriber, error) {
	opts := paho.NewClientOptions().
		AddBroker(cfg.MQTTBroker).
		SetClientID(cfg.MQTTClientID + "-sub").
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectTimeout(10 * time.Second)

	if cfg.MQTTUsername != "" {
		opts.SetUsername(cfg.MQTTUsername).SetPassword(cfg.MQTTPassword)
	}

	c := paho.NewClient(opts)
	if tok := c.Connect(); tok.Wait() && tok.Error() != nil {
		return nil, fmt.Errorf("mqtt subscriber connect: %w", tok.Error())
	}
	log.Println("[Subscriber] connected to", cfg.MQTTBroker)
	return &Subscriber{client: c, publisher: pub, cfg: cfg}, nil
}

// SubscribeGroups divide as tags em grupos e assina em paralelo
func (s *Subscriber) SubscribeGroups(tags []models.MQTTTag) {
	groups := chunkTags(tags, s.cfg.TagGroupSize)
	log.Printf("[Subscriber] %d tag(s) divididas em %d grupo(s)", len(tags), len(groups))

	var wg sync.WaitGroup
	for i, group := range groups {
		wg.Add(1)
		go func(groupIdx int, grp []models.MQTTTag) {
			defer wg.Done()
			s.subscribeGroup(groupIdx, grp)
		}(i, group)
	}
	wg.Wait()
	log.Println("[Subscriber] todos os grupos inscritos, aguardando mensagens...")
}

func (s *Subscriber) subscribeGroup(groupIdx int, tags []models.MQTTTag) {
	for _, tag := range tags {
		tag := tag // captura para closure

		// tópico de escuta:
		// /devices/raw_data/erd/{deviceID}/{rawTopic}
		listenTopic := fmt.Sprintf("/devices/raw_data/erd/%s/%s", tag.DeviceID, tag.RawTopic)

		tok := s.client.Subscribe(listenTopic, 0, func(_ paho.Client, msg paho.Message) {
			s.handleMessage(tag, msg.Payload())
		})
		tok.Wait()
		if err := tok.Error(); err != nil {
			log.Printf("[Subscriber] group=%d erro ao subscrever topic=%s err=%v", groupIdx, listenTopic, err)
			continue
		}
		log.Printf("[Subscriber] group=%d inscrito em %s", groupIdx, listenTopic)
	}
}

func (s *Subscriber) handleMessage(tag models.MQTTTag, rawBytes []byte) {
	var raw models.RawPayload
	if err := json.Unmarshal(rawBytes, &raw); err != nil {
		log.Printf("[Subscriber] tag=%s json inválido: %v", tag.TagName, err)
		return
	}

	utcTS, err := convertToUTC(raw.TS)
	if err != nil {
		log.Printf("[Subscriber] tag=%s erro no timestamp: %v", tag.TagName, err)
		// usa o horário atual como fallback
		utcTS = time.Now().UTC().Format(time.RFC3339)
	}

	out := models.FormattedPayload{
		TS:  utcTS,
		Val: raw.Val,
	}
	outBytes, _ := json.Marshal(out)

	// publica em:
	// /devices/formatted_data/erd/{deviceID}/{tagID}
	s.publisher.Publish(tag.DeviceID, tag.TagID, string(outBytes))

	log.Printf("[Subscriber] tag=%s → publicado TS=%s Val=%s", tag.TagName, utcTS, raw.Val)
}

// convertToUTC recebe um timestamp em UTC-3 (sem offset) e converte para UTC
// Formatos aceitos: "2006-01-02T15:04:05Z", "2006-01-02T15:04:05", "2006-01-02 15:04:05"
func convertToUTC(ts string) (string, error) {
	formats := []string{
		"02/01/2006 15:04:05", // brasileiro: dd/MM/yyyy HH:mm:ss  ← principal
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, layout := range formats {
		t, err := time.ParseInLocation(layout, ts, brazilLocation)
		if err == nil {
			return t.UTC().Format(time.RFC3339), nil
		}
	}
	return "", fmt.Errorf("formato de timestamp não reconhecido: %s", ts)
}

func chunkTags(tags []models.MQTTTag, size int) [][]models.MQTTTag {
	var chunks [][]models.MQTTTag
	for size < len(tags) {
		tags, chunks = tags[size:], append(chunks, tags[:size])
	}
	return append(chunks, tags)
}

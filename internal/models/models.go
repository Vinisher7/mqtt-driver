package models

// MQTTTag representa uma tag do broker com seus tópicos
type MQTTTag struct {
	TagID          string
	TagName        string
	RawTopic       string
	FormattedTopic string
	DeviceID       string
}

// RawPayload é o formato JSON recebido do broker de campo
// TS vem em horário brasileiro (UTC-3)
type RawPayload struct {
	TS  string `json:"TS"`
	Val string `json:"Val"`
}

// FormattedPayload é o formato JSON publicado no broker de saída
// TS convertido para UTC
type FormattedPayload struct {
	TS  string `json:"TS"`
	Val string `json:"Val"`
}

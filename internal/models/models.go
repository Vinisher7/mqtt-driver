package models

type MQTTTag struct {
	TagID          string
	TagName        string
	RawTopic       string
	FormattedTopic string
	DeviceID       string
}

type RawPayload struct {
	TS  string `json:"TS"`
	Val string `json:"Val"`
}

type FormattedPayload struct {
	TS  string `json:"TS"`
	Val string `json:"Val"`
}

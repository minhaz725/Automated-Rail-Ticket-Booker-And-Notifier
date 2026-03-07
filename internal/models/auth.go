package models

type CapturedAuth struct {
	Authorization string `json:"authorization"`
	DeviceId      string `json:"device_id"`
	DeviceKey     string `json:"device_key"`
	UserAgent     string `json:"user_agent"`
}

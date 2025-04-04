package models

import "time"

type RelayDevice struct {
	DeviceID string    `gorm:"primaryKey" json:"device_id"`
	IP       string    `gorm:"not null" json:"ip"`
	Updated  time.Time `gorm:"not null" json:"updated"`
}

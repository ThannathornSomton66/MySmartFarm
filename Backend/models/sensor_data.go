package models

import "time"

type SensorData struct {
	ID          uint      `gorm:"primaryKey"`       
	DeviceID    string    `gorm:"size:50;not null"` 
	Temperature float64   `gorm:"not null"`
	Humidity    float64   `gorm:"not null"`
	Timestamp   time.Time `gorm:"not null"` 
}

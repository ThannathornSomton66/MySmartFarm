package models

type IntervalSetting struct {
	DeviceID        string `gorm:"primaryKey;size:50"`
	IntervalSeconds int    `gorm:"not null"`
}

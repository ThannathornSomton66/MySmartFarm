package database

import (
	"log"

	"my-smart-farm/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("farm_data.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to SQLite database:", err)
	}

	// AutoMigrate will create/modify the table based on the SensorData struct
	if err := DB.AutoMigrate(&models.SensorData{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}

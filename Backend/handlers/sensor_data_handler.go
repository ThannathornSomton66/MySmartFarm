package handlers

import (
	"time"

	"my-smart-farm/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Handler to create new sensor data then return the server time
func CreateSensorData(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var data models.SensorData

		if err := c.BodyParser(&data); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot parse JSON",
			})
		}

		if data.Timestamp.IsZero() {
			data.Timestamp = time.Now()
		}

		if err := db.Create(&data).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save data",
			})
		}

		// Just return interval (in seconds)
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"intervalSeconds": 30, // or any value you want
		})
	}
}

// Handler to list sensor data
func GetAllSensorData(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var allData []models.SensorData
		if err := db.Find(&allData).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve data",
			})
		}
		return c.JSON(allData)
	}
}

// Handler to fetch data by device ID (optional extra)
func GetSensorDataByDeviceID(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		deviceID := c.Params("deviceID") // e.g., /api/v1/data/device/<deviceID>

		var deviceData []models.SensorData
		if err := db.Where("device_id = ?", deviceID).Find(&deviceData).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve data for device " + deviceID,
			})
		}

		return c.JSON(deviceData)
	}
}

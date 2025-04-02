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

		// Parse JSON from request body into 'data'
		if err := c.BodyParser(&data); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot parse JSON",
			})
		}

		// If the client didn't provide a timestamp, set it to "now"
		if data.Timestamp.IsZero() {
			data.Timestamp = time.Now()
		}

		// Insert data into the database
		if err := db.Create(&data).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save data",
			})
		}

		// ✅ **Return only `ServerTime`**
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"serverTime": time.Now().Format(time.RFC3339), // Return only server time
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

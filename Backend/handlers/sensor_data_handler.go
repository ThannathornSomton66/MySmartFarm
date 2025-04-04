package handlers

import (
	"time"

	"my-smart-farm/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

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

		// Default fallback interval
		interval := 60

		var setting models.IntervalSetting
		err := db.First(&setting, "device_id = ?", data.DeviceID).Error
		if err == nil {
			interval = setting.IntervalSeconds
		} else if err == gorm.ErrRecordNotFound {
			setting = models.IntervalSetting{
				DeviceID:        data.DeviceID,
				IntervalSeconds: interval,
			}
			db.Create(&setting)
		}

		// Align interval: compute time until next aligned slot
		now := time.Now()
		elapsed := now.Unix() % int64(interval)
		wait := interval - int(elapsed) // seconds until next aligned time

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"intervalSeconds": wait,
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

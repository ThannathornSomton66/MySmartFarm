package handlers

import (
	"time"

	"my-smart-farm/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// POST /api/v1/relay/register
func RegisterRelayIP(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var device models.RelayDevice
		if err := c.BodyParser(&device); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid payload",
			})
		}

		device.Updated = time.Now()

		err := db.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).Create(&device).Error

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to store IP",
			})
		}

		return c.JSON(fiber.Map{"message": "Registered!"})
	}
}

// GET /api/v1/relay/:deviceID
func GetRelayIP(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		deviceID := c.Params("deviceID")

		var device models.RelayDevice
		err := db.First(&device, "device_id = ?", deviceID).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "Device not found",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Database error",
			})
		}

		return c.JSON(device)
	}
}

func GetAllRelays(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var relays []models.RelayDevice
		if err := db.Find(&relays).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch relays"})
		}
		return c.JSON(relays)
	}
}

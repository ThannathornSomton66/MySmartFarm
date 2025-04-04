package handlers

import (
	"my-smart-farm/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func SetInterval(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var setting models.IntervalSetting

		if err := c.BodyParser(&setting); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid input",
			})
		}

		if setting.DeviceID == "" || setting.IntervalSeconds <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing or invalid deviceID/intervalSeconds",
			})
		}

		err := db.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).Create(&setting).Error

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save interval setting",
			})
		}

		return c.SendStatus(fiber.StatusOK)
	}
}

func GetAllIntervals(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var intervals []models.IntervalSetting
		if err := db.Find(&intervals).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch interval settings",
			})
		}
		return c.JSON(intervals)
	}
}

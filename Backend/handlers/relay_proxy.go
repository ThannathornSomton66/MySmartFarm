package handlers

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"my-smart-farm/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ProxyRelayCommand sends on/off command to the relay device via IP
func ProxyRelayCommand(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		deviceID := c.Params("deviceID")
		action := c.Params("action") // should be "on" or "off"

		if action != "on" && action != "off" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid action"})
		}

		var relay models.RelayDevice
		if err := db.First(&relay, "device_id = ?", deviceID).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Device not found"})
		}

		url := fmt.Sprintf("http://%s/relay/%s", relay.IP, action)

		// Forward the request to the actual device
		client := http.Client{
			Timeout: 3 * time.Second,
		}
		resp, err := client.Get(url)
		if err != nil {
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "Failed to reach relay device"})
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		return c.Status(resp.StatusCode).Send(body)
	}
}

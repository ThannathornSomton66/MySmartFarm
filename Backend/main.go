package main

import (
	"log"

	"my-smart-farm/database"
	"my-smart-farm/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"gorm.io/gorm"
)

func setupRoutes(app *fiber.App, db *gorm.DB) {
	api := app.Group("/api/v1")

	// POST /api/v1/data -> Create new sensor record
	api.Post("/data", handlers.CreateSensorData(db))

	// GET /api/v1/data -> Retrieve all sensor records
	api.Get("/data", handlers.GetAllSensorData(db))

	// GET /api/v1/data/device/:deviceID -> Retrieve data by device
	api.Get("/data/device/:deviceID", handlers.GetSensorDataByDeviceID(db))

	api.Post("/interval", handlers.SetInterval(db))
	api.Get("/intervals", handlers.GetAllIntervals(db))
	api.Post("/relay/register", handlers.RegisterRelayIP(db))
	api.Get("/relay/:deviceID", handlers.GetRelayIP(db))
	api.Get("/relays", handlers.GetAllRelays(db))
	api.Post("/relay/:deviceID/:action", handlers.ProxyRelayCommand(db))

}

func main() {
	// Initialize the DB
	database.InitDB()
	db := database.DB

	// Initialize Fiber
	app := fiber.New()
	app.Use(cors.New())

	// Set up API routes
	setupRoutes(app, db)

	// Start server on localhost:3000
	log.Fatal(app.Listen(":3000"))
}

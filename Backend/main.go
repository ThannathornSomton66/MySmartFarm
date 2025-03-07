package main

import (
	"log"

	"my-smart-farm/database"
	"my-smart-farm/handlers"

	"github.com/gofiber/fiber/v2"
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
}

func main() {
	// Initialize the DB
	database.InitDB()
	db := database.DB

	// Initialize Fiber
	app := fiber.New()

	// Set up API routes
	setupRoutes(app, db)

	// Start server on localhost:3000
	log.Fatal(app.Listen(":3000"))
}

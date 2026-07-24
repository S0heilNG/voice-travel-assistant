package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func Register(app *fiber.App) {
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173",
		AllowMethods: "GET,POST",
		// ngrok-skip-browser-warning is sent by the frontend; it must be
		// allowed here or the CORS preflight fails in cross-origin dev (it's a
		// no-op on the same-origin staging deploy).
		AllowHeaders: "Content-Type,ngrok-skip-browser-warning",
	}))

	app.Get("/health", healthCheck)
	app.Get("/api/test-automation/flights", testAutomationFlights)
	app.Post("/api/parse", parseText)
}

func healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

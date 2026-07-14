package handlers

import "github.com/gofiber/fiber/v2"

func Register(app *fiber.App) {
	app.Get("/health", healthCheck)
}

func healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

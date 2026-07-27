package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/analytics"
)

// api holds the dependencies handlers need (currently just the analytics
// store). Handlers that log are methods on it; stateless ones stay plain funcs.
type api struct {
	store *analytics.Store
}

func Register(app *fiber.App, store *analytics.Store) {
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173",
		AllowMethods: "GET,POST",
		// ngrok-skip-browser-warning is sent by the frontend; X-Session-Id
		// carries the anonymous analytics session id. Both must be allowed or
		// the CORS preflight fails in cross-origin dev.
		AllowHeaders: "Content-Type,ngrok-skip-browser-warning,X-Session-Id",
	}))

	a := &api{store: store}

	app.Get("/health", healthCheck)
	app.Get("/api/test-automation/flights", testAutomationFlights)
	app.Post("/api/parse", a.parseText)
	app.Post("/api/events", a.postEvent)
}

func healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// sessionID reads the anonymous, client-generated session id, bounded so a
// hostile client can't send a huge value. Empty is fine.
func sessionID(c *fiber.Ctx) string {
	s := c.Get("X-Session-Id")
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

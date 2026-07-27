package main

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/analytics"
	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/config"
	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/handlers"
)

func main() {
	cfg := config.Load()

	// Interaction logging. A failure here must not stop the server — Open
	// returns a disabled (no-op) store on error, and the app runs fully without
	// it.
	store, err := analytics.Open(cfg.AnalyticsDBPath)
	if err != nil {
		log.Printf("analytics: failed to open db (%v) — continuing without logging", err)
	}
	defer store.Close()

	app := fiber.New()

	handlers.Register(app, store)

	log.Printf("server listening on :%s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

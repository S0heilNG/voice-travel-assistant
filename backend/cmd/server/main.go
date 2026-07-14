package main

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/config"
	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/handlers"
)

func main() {
	cfg := config.Load()

	app := fiber.New()

	handlers.Register(app)

	log.Printf("server listening on :%s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

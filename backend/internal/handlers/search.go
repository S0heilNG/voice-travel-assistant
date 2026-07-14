package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/soheilnegahi/voice-travel-assistant/backend/internal/automation"
)

// testAutomationFlights is a manual end-to-end connectivity check between the
// Go backend and automation-service — not the final search feature.
func testAutomationFlights(c *fiber.Ctx) error {
	client := automation.NewAutomationClient()
	result, err := client.SearchFlights("Tehran", "Mashhad", "tomorrow")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"ok":    false,
			"error": err.Error(),
		})
	}
	return c.JSON(result)
}

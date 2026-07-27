package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// eventRequest is a funnel event the frontend reports (clarification_shown,
// search_clicked, voice_error, ...). detail is optional context (missing field,
// error code, ...).
type eventRequest struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

// postEvent records a frontend funnel event. It always returns 204 to the
// client (even when logging is disabled or rate-limited) so analytics can never
// affect the user's experience.
func (a *api) postEvent(c *fiber.Ctx) error {
	var req eventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	t := strings.TrimSpace(req.Type)
	if t == "" || len(t) > 64 {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	a.store.LogEvent(sessionID(c), t, req.Detail) // swallows errors + rate-limits
	return c.SendStatus(fiber.StatusNoContent)
}

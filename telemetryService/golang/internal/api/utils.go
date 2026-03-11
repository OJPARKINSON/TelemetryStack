package api

import (
	"github.com/gofiber/fiber/v3"
)

func respondJSON(c fiber.Ctx, status int, data interface{}) {
	c.Type("application/json")
	c.Status(status)
	c.JSON(data)
}

func respondError(c fiber.Ctx, message string, status int) {
	c.Status(status)
	c.JSON(fiber.Map{"error": message})
}

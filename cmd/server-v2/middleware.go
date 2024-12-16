package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func Middleware(apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("Address", c.IP(), "Calling", c.Method(), "Request", c.OriginalURL())
		requestKey := c.Get("X-API-Key")
		if requestKey != apiKey {
			log.Println("Unauthorized access attempt at", c.IP(), "with API key:", requestKey)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}
		return c.Next()
	}
}

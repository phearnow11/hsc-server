package main

import (
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

var (
	suspiciousIP = map[string]int{}
	blacklistIP  = map[string]bool{}
)

func Middleware(apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("Address", c.IP(), "Calling", c.Method(), "Request", c.OriginalURL())
		if BlockIP(c.IP()) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: Blocked",
			})
		}
		requestKey := c.Get("X-API-Key")
		if requestKey != apiKey {
			log.Println(BlackList(c.IP()))
			log.Println("Unauthorized access attempt at", c.IP(), "with API key:", requestKey)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}
		suspiciousIP[c.IP()] = 0
		return c.Next()
	}
}

func BlackList(ip string) string {
	_, ok := suspiciousIP[ip]
	if !ok {
		suspiciousIP[ip] = 1
		return "Suspicious IP: " + ip
	}
	if suspiciousIP[ip] == 10 {
		blacklistIP[ip] = true
		return "Suspicious IP exceed allowed calling time, blocked: " + ip
	}
	suspiciousIP[ip] = suspiciousIP[ip] + 1
	calling_times := strconv.Itoa(suspiciousIP[ip])
	return "Suspicious IP calling " + calling_times + " times: " + ip
}

func BlockIP(ip string) bool {
	if blacklistIP[ip] {
		log.Println("Blocked IP attempt:", ip)
		return true
	}
	return false
}

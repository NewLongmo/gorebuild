package handlers

import (
	"time"

	"dw0rdwk/backend/internal/config"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

func Health(cfg config.Config) fiber.Handler {
	startedAt := time.Now().UTC()
	return func(c *fiber.Ctx) error {
		return OK(c, fiber.Map{
			"app":       cfg.AppName,
			"env":       cfg.AppEnv,
			"status":    "ok",
			"startedAt": startedAt,
		})
	}
}

func Readiness(cfg config.Config, health *service.HealthService) fiber.Handler {
	startedAt := time.Now().UTC()
	return func(c *fiber.Ctx) error {
		report := health.Readiness(c.UserContext())
		statusCode := fiber.StatusOK
		if report.Status != "ok" {
			statusCode = fiber.StatusServiceUnavailable
		}
		return c.Status(statusCode).JSON(response{
			Code:    statusCode - fiber.StatusOK,
			Message: report.Status,
			Data: fiber.Map{
				"app":       cfg.AppName,
				"env":       cfg.AppEnv,
				"startedAt": startedAt,
				"checks":    report.Checks,
			},
		})
	}
}

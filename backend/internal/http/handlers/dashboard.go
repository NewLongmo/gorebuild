package handlers

import (
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

func Dashboard(svc *service.DashboardService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		stats, err := svc.Stats(c.UserContext())
		if err != nil {
			return err
		}
		return OK(c, stats)
	}
}

func DashboardStatistics(svc *service.DashboardService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		stats, err := svc.Statistics(c.UserContext())
		if err != nil {
			return err
		}
		return OK(c, stats)
	}
}

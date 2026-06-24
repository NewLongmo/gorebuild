package handlers

import (
	"strings"

	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type systemJobPayload struct {
	Enabled *bool `json:"enabled"`
}

func SystemJobs(repo *repository.SystemJobRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		items, err := repo.List(c.UserContext())
		if err != nil {
			return err
		}
		return OK(c, fiber.Map{"items": items})
	}
}

func UpdateSystemJob(repo *repository.SystemJobRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := strings.TrimSpace(c.Params("name"))
		if name == "" {
			return fiber.NewError(fiber.StatusBadRequest, "job name is required")
		}
		var req systemJobPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if req.Enabled == nil {
			return fiber.NewError(fiber.StatusBadRequest, "enabled is required")
		}
		if err := repo.UpdateEnabled(c.UserContext(), name, *req.Enabled); err != nil {
			return err
		}
		auditLog(c, logs, "admin_system_job_update", "job="+name+" enabled="+boolText(*req.Enabled), 0)
		return OK(c, fiber.Map{"name": name, "enabled": *req.Enabled})
	}
}

func RunSystemJob(repo *repository.SystemJobRepository, orders *service.OrderService, syncer *service.ConnectorSyncService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := strings.TrimSpace(c.Params("name"))
		if name == "" {
			return fiber.NewError(fiber.StatusBadRequest, "job name is required")
		}
		if !isSupportedSystemJob(name) {
			return fiber.NewError(fiber.StatusBadRequest, "unsupported job")
		}
		started, err := repo.MarkStarted(c.UserContext(), name)
		if err != nil {
			return err
		}
		var summary any
		var runErr error
		switch name {
		case "order_queue_recover":
			summary, runErr = orders.RecoverQueues(c.UserContext(), recoverBatchSize(c))
		case "29wk_order_sync":
			summary, runErr = syncer.Sync29WKOrders(c.UserContext(), service.WK29OrderSyncInput{MaxPages: queryInt(c, "maxPages", 20)})
		case "29wk_price_sync":
			summary, runErr = syncer.Sync29WKPricesAll(c.UserContext())
		}
		if err := repo.MarkFinished(c.UserContext(), name, started, summary, runErr); err != nil {
			return err
		}
		if runErr != nil {
			return runErr
		}
		auditLog(c, logs, "admin_system_job_run", "job="+name, 0)
		return OK(c, summary)
	}
}

func isSupportedSystemJob(name string) bool {
	switch strings.TrimSpace(name) {
	case "order_queue_recover", "29wk_order_sync", "29wk_price_sync":
		return true
	default:
		return false
	}
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

package handlers

import (
	"strings"

	appmw "dw0rdwk/backend/internal/http/middleware"
	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type publicOrderAccountPayload struct {
	Account string `json:"account"`
}

type publicOrderPasswordPayload struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

func OrderEvents(orders *repository.OrderRepository, events *repository.OrderEventRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if _, err := orders.Find(c.UserContext(), id); err != nil {
			return err
		}
		items, err := events.ListByOrder(c.UserContext(), id, false, queryInt(c, "limit", 100))
		if err != nil {
			return err
		}
		return OK(c, fiber.Map{"items": items})
	}
}

func AgentOrderEvents(orders *repository.OrderRepository, events *repository.OrderEventRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if _, err := orders.FindByUser(c.UserContext(), id, claims.UID); err != nil {
			return err
		}
		items, err := events.ListByOrder(c.UserContext(), id, true, queryInt(c, "limit", 100))
		if err != nil {
			return err
		}
		return OK(c, fiber.Map{"items": items})
	}
}

func PublicOrderEvents(orders *repository.OrderRepository, events *repository.OrderEventRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		account := strings.TrimSpace(c.Query("account"))
		if account == "" {
			return fiber.NewError(fiber.StatusBadRequest, "account is required")
		}
		order, err := orders.Find(c.UserContext(), id)
		if err != nil {
			return err
		}
		if order.Account != account {
			return fiber.NewError(fiber.StatusForbidden, "order does not belong to this account")
		}
		items, err := events.ListByOrder(c.UserContext(), id, true, queryInt(c, "limit", 100))
		if err != nil {
			return err
		}
		return OK(c, fiber.Map{"items": items})
	}
}

func PublicOrderPassword(orders *repository.OrderRepository, orderService *service.OrderService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req publicOrderPasswordPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		account := strings.TrimSpace(req.Account)
		if account == "" {
			return fiber.NewError(fiber.StatusBadRequest, "account is required")
		}
		password, err := validateOrderPasswordPayload(agentOrderPasswordPayload{Password: req.Password})
		if err != nil {
			return err
		}
		order, err := orders.Find(c.UserContext(), id)
		if err != nil {
			return err
		}
		if order.Account != account {
			return fiber.NewError(fiber.StatusForbidden, "order does not belong to this account")
		}
		if err := orderService.UpdatePasswordAndRefresh(c.UserContext(), id, password, "public_support"); err != nil {
			return err
		}
		return OK(c, fiber.Map{"id": id})
	}
}

func PublicOrderResubmit(orders *repository.OrderRepository, orderService *service.OrderService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req publicOrderAccountPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		account := strings.TrimSpace(req.Account)
		if account == "" {
			return fiber.NewError(fiber.StatusBadRequest, "account is required")
		}
		order, err := orders.Find(c.UserContext(), id)
		if err != nil {
			return err
		}
		if order.Account != account {
			return fiber.NewError(fiber.StatusForbidden, "order does not belong to this account")
		}
		if err := orderService.RequeueSubmit(c.UserContext(), id); err != nil {
			return err
		}
		return OK(c, fiber.Map{"id": id})
	}
}

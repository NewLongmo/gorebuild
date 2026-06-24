package handlers

import (
	"strings"

	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type publicOrderSearchPayload struct {
	Account string `json:"account"`
}

type publicOrderRefreshPayload struct {
	Account string `json:"account"`
}

type publicOrderRow struct {
	ID              uint   `json:"id"`
	Platform        string `json:"platform"`
	School          string `json:"school"`
	StudentName     string `json:"studentName"`
	Account         string `json:"account"`
	CourseName      string `json:"courseName"`
	Status          string `json:"status"`
	DockingStatus   string `json:"dockingStatus"`
	Progress        string `json:"progress"`
	Remarks         string `json:"remarks"`
	Score           string `json:"score"`
	DurationMinutes int    `json:"durationMinutes"`
	CreatedAt       string `json:"createdAt"`
}

type publicOrderSearchResult struct {
	Account string           `json:"account"`
	Items   []publicOrderRow `json:"items"`
	Total   int              `json:"total"`
	Notice  string           `json:"notice"`
}

func PublicOrderSearch(repo *repository.OrderRepository, settings *repository.SettingRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req publicOrderSearchPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		account := strings.TrimSpace(req.Account)
		if account == "" {
			return fiber.NewError(fiber.StatusBadRequest, "account is required")
		}
		orders, err := repo.ListPublicByAccount(c.UserContext(), account, 50)
		if err != nil {
			return err
		}
		values, _ := settings.All(c.UserContext())
		return OK(c, publicOrderSearchResult{
			Account: account,
			Items:   publicOrderRowsFromOrders(orders),
			Total:   len(orders),
			Notice:  strings.TrimSpace(values["open_query_notice"]),
		})
	}
}

func PublicOrderRefresh(repo *repository.OrderRepository, orderService *service.OrderService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req publicOrderRefreshPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		account := strings.TrimSpace(req.Account)
		if account == "" {
			return fiber.NewError(fiber.StatusBadRequest, "account is required")
		}
		order, err := repo.Find(c.UserContext(), id)
		if err != nil {
			return err
		}
		if order.Account != account {
			return fiber.NewError(fiber.StatusForbidden, "order does not belong to this account")
		}
		if err := orderService.MarkRefreshRequested(c.UserContext(), id); err != nil {
			return err
		}
		return OK(c, fiber.Map{"id": id})
	}
}

func publicOrderRowsFromOrders(orders []models.Order) []publicOrderRow {
	items := make([]publicOrderRow, 0, len(orders))
	for _, order := range orders {
		items = append(items, publicOrderRow{
			ID:              order.ID,
			Platform:        order.Platform,
			School:          order.School,
			StudentName:     order.StudentName,
			Account:         order.Account,
			CourseName:      order.CourseName,
			Status:          order.Status,
			DockingStatus:   order.DockingStatus,
			Progress:        order.Progress,
			Remarks:         order.Remarks,
			Score:           order.Score,
			DurationMinutes: order.DurationMinutes,
			CreatedAt:       order.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return items
}

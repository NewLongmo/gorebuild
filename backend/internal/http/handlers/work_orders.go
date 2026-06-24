package handlers

import (
	"strings"

	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

const (
	workOrderStatusPending  = "待回复"
	workOrderStatusAnswered = "已回复"
	workOrderStatusClosed   = "已关闭"
	workOrderStatusRejected = "已驳回"
	workOrderStatusIgnored  = "不做处理"
)

type workOrderPayload struct {
	Category      string `json:"category"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	AttachmentURL string `json:"attachmentUrl"`
}

type workOrderActionPayload struct {
	Action        string  `json:"action"`
	Answer        string  `json:"answer"`
	Progress      *int    `json:"progress"`
	AttachmentURL *string `json:"attachmentUrl"`
	UserVisible   *bool   `json:"userVisible"`
}

func AgentWorkOrders(users *repository.UserRepository, workOrders *repository.WorkOrderRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		data, err := workOrders.List(c.UserContext(), c.Query("q"), c.Query("status"), &user.ID, queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func AgentCreateWorkOrder(users *repository.UserRepository, workOrders *repository.WorkOrderRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		var req workOrderPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateWorkOrderPayload(req); err != nil {
			return err
		}
		item := models.WorkOrder{
			UserID:        user.ID,
			Category:      strings.TrimSpace(req.Category),
			Title:         strings.TrimSpace(req.Title),
			Content:       strings.TrimSpace(req.Content),
			AttachmentURL: strings.TrimSpace(req.AttachmentURL),
			Status:        workOrderStatusPending,
			Progress:      0,
			UserVisible:   true,
		}
		if err := workOrders.Create(c.UserContext(), &item); err != nil {
			return err
		}
		auditLog(c, logs, "agent_work_order_create", auditText("work_order", item.ID, item.Title), 0)
		return OK(c, item)
	}
}

func AgentReplyWorkOrder(users *repository.UserRepository, workOrders *repository.WorkOrderRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req struct {
			Content string `json:"content"`
		}
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		content := strings.TrimSpace(req.Content)
		if content == "" {
			return fiber.NewError(fiber.StatusBadRequest, "content is required")
		}
		if len(content) > 5000 {
			return fiber.NewError(fiber.StatusBadRequest, "content is too long")
		}
		values := map[string]any{
			"content": content,
			"answer":  "",
			"status":  workOrderStatusPending,
		}
		if err := workOrders.UpdateByUser(c.UserContext(), id, user.ID, values); err != nil {
			return err
		}
		auditLog(c, logs, "agent_work_order_reply", auditText("work_order", id, "reopened"), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func AgentDeleteWorkOrder(users *repository.UserRepository, workOrders *repository.WorkOrderRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if err := workOrders.DeleteByUser(c.UserContext(), id, user.ID); err != nil {
			return err
		}
		auditLog(c, logs, "agent_work_order_delete", auditText("work_order", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func WorkOrders(repo *repository.WorkOrderRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := repo.List(c.UserContext(), c.Query("q"), c.Query("status"), nil, queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func UpdateWorkOrder(repo *repository.WorkOrderRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req workOrderActionPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		values, err := workOrderActionValues(req)
		if err != nil {
			return err
		}
		if err := repo.Update(c.UserContext(), id, values); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_work_order_update", auditText("work_order", id, auditFields(values)), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func DeleteWorkOrder(repo *repository.WorkOrderRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if err := repo.Delete(c.UserContext(), id); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_work_order_delete", auditText("work_order", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func validateWorkOrderPayload(req workOrderPayload) error {
	category := strings.TrimSpace(req.Category)
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if category == "" || title == "" || content == "" {
		return fiber.NewError(fiber.StatusBadRequest, "category, title, and content are required")
	}
	if len(category) > 80 {
		return fiber.NewError(fiber.StatusBadRequest, "category is too long")
	}
	if len(title) > 160 {
		return fiber.NewError(fiber.StatusBadRequest, "title is too long")
	}
	if len(content) > 5000 {
		return fiber.NewError(fiber.StatusBadRequest, "content is too long")
	}
	attachmentURL := strings.TrimSpace(req.AttachmentURL)
	if len(attachmentURL) > 500 {
		return fiber.NewError(fiber.StatusBadRequest, "attachmentUrl is too long")
	}
	return nil
}

func workOrderActionValues(req workOrderActionPayload) (map[string]any, error) {
	action := strings.TrimSpace(req.Action)
	answer := strings.TrimSpace(req.Answer)
	values := map[string]any{}
	switch action {
	case "answer":
		if answer == "" {
			return nil, fiber.NewError(fiber.StatusBadRequest, "answer is required")
		}
		values["answer"] = answer
		values["status"] = workOrderStatusAnswered
	case "reject":
		if answer == "" {
			return nil, fiber.NewError(fiber.StatusBadRequest, "answer is required")
		}
		values["answer"] = answer
		values["status"] = workOrderStatusRejected
	case "close":
		values["status"] = workOrderStatusClosed
	case "ignore":
		values["status"] = workOrderStatusIgnored
	default:
		return nil, fiber.NewError(fiber.StatusBadRequest, "action must be answer, reject, close, or ignore")
	}
	if answer != "" && len(answer) > 5000 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "answer is too long")
	}
	if req.Progress != nil {
		if *req.Progress < 0 || *req.Progress > 100 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "progress must be between 0 and 100")
		}
		progress := *req.Progress
		if action == "close" && progress == 0 {
			progress = 100
		}
		values["progress"] = progress
	} else if action == "close" {
		values["progress"] = 100
	}
	if req.AttachmentURL != nil {
		attachmentURL := strings.TrimSpace(*req.AttachmentURL)
		if len(attachmentURL) > 500 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "attachmentUrl is too long")
		}
		values["attachment_url"] = attachmentURL
	}
	if req.UserVisible != nil {
		values["user_visible"] = *req.UserVisible
	}
	return values, nil
}

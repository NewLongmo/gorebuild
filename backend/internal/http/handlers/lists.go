package handlers

import (
	"context"
	"errors"
	"strconv"
	"strings"

	appmw "dw0rdwk/backend/internal/http/middleware"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/password"
	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

func Users(repo *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := repo.List(c.UserContext(), c.Query("q"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

type adminAgentTreeNode struct {
	ID             uint                  `json:"id"`
	ParentID       uint                  `json:"parentId"`
	Account        string                `json:"account"`
	Name           string                `json:"name"`
	Balance        float64               `json:"balance"`
	PriceRate      float64               `json:"priceRate"`
	Role           string                `json:"role"`
	Status         string                `json:"status"`
	CreatedAt      string                `json:"createdAt"`
	LastIP         string                `json:"lastIp"`
	ParentAccount  string                `json:"parentAccount"`
	Depth          int                   `json:"depth"`
	DirectChildren int                   `json:"directChildren"`
	Children       []*adminAgentTreeNode `json:"children,omitempty"`
}

type adminAgentTreeResponse struct {
	Items     []*adminAgentTreeNode `json:"items"`
	Total     int                   `json:"total"`
	Matched   int                   `json:"matched"`
	Truncated bool                  `json:"truncated"`
}

func AgentTree(repo *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		users, truncated, err := repo.ListAgentsForTree(c.UserContext(), queryInt(c, "limit", 5000))
		if err != nil {
			return err
		}
		tree := buildAdminAgentTree(users)
		matched := len(users)
		if search := strings.TrimSpace(c.Query("q")); search != "" {
			tree, matched = filterAdminAgentTree(tree, search)
		}
		return OK(c, adminAgentTreeResponse{
			Items:     tree,
			Total:     len(users),
			Matched:   matched,
			Truncated: truncated,
		})
	}
}

func Classes(repo *repository.ClassRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var status *string
		if raw := strings.TrimSpace(c.Query("status")); raw != "" {
			status = &raw
		}

		data, err := repo.List(c.UserContext(), c.Query("q"), status, c.Query("category"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

type categoryPayload struct {
	Sort        int    `json:"sort"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Pinned      bool   `json:"pinned"`
	Description string `json:"description"`
}

func Categories(repo *repository.CategoryRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := repo.List(c.UserContext(), c.Query("q"), c.Query("status"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func CreateCategory(repo *repository.CategoryRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req categoryPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateCategoryPayload(req); err != nil {
			return err
		}
		item := categoryFromPayload(req)
		if err := repo.Create(c.UserContext(), &item); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_category_create", auditText("category", item.ID, "name="+item.Name+" status="+item.Status), 0)
		return OK(c, item)
	}
}

func UpdateCategory(repo *repository.CategoryRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req categoryPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateCategoryPayload(req); err != nil {
			return err
		}
		values := map[string]any{
			"sort":        req.Sort,
			"name":        strings.TrimSpace(req.Name),
			"status":      categoryStatus(req.Status),
			"pinned":      req.Pinned,
			"description": strings.TrimSpace(req.Description),
		}
		if err := repo.Update(c.UserContext(), id, values); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_category_update", auditText("category", id, auditFields(values)), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func DeleteCategory(repo *repository.CategoryRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		result, err := repo.DeleteCascade(c.UserContext(), id)
		if err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_category_delete", auditText("category", id, cascadeDeleteAudit(result)), 0)
		return OK(c, result)
	}
}

func Orders(repo *repository.OrderRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		flashMode, err := queryBool(c, "flashMode")
		if err != nil {
			return err
		}
		data, err := repo.List(c.UserContext(), c.Query("q"), c.Query("status"), flashMode, queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

type userPayload struct {
	ParentID  uint    `json:"parentId"`
	Account   string  `json:"account"`
	Password  string  `json:"password"`
	Name      string  `json:"name"`
	Balance   float64 `json:"balance"`
	PriceRate float64 `json:"priceRate"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
}

type balancePayload struct {
	Amount float64 `json:"amount"`
	Reason string  `json:"reason"`
}

func CreateUser(repo *repository.UserRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req userPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if strings.TrimSpace(req.Account) == "" || strings.TrimSpace(req.Password) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "account and password are required")
		}
		if err := validateUserPayload(req); err != nil {
			return err
		}
		user := userFromPayload(req)
		passwordHash, err := password.Hash(req.Password)
		if err != nil {
			return err
		}
		user.PasswordHash = passwordHash
		if err := repo.Create(c.UserContext(), &user); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_user_create", auditText("user", user.ID, "account="+user.Account+" role="+user.Role+" status="+user.Status), user.Balance)
		return OK(c, user)
	}
}

func UpdateUser(repo *repository.UserRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req userPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateUserPayload(req); err != nil {
			return err
		}
		values := map[string]any{}
		if req.ParentID > 0 {
			values["parent_id"] = req.ParentID
		}
		if strings.TrimSpace(req.Account) != "" {
			values["account"] = strings.TrimSpace(req.Account)
		}
		if strings.TrimSpace(req.Password) != "" {
			passwordHash, err := password.Hash(req.Password)
			if err != nil {
				return err
			}
			values["password_hash"] = passwordHash
		}
		if strings.TrimSpace(req.Name) != "" {
			values["name"] = strings.TrimSpace(req.Name)
		}
		if req.Balance >= 0 {
			values["balance"] = req.Balance
		}
		if req.PriceRate > 0 {
			values["price_rate"] = req.PriceRate
		}
		if strings.TrimSpace(req.Role) != "" {
			values["role"] = strings.TrimSpace(req.Role)
		}
		if strings.TrimSpace(req.Status) != "" {
			values["status"] = strings.TrimSpace(req.Status)
		}
		if err := repo.Update(c.UserContext(), id, values); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_user_update", auditText("user", id, auditFields(values)), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func DeleteUser(repo *repository.UserRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if err := repo.Delete(c.UserContext(), id); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_user_delete", auditText("user", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func ResetUserPassword(repo *repository.UserRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if claims, ok := appmw.Claims(c); ok && claims.UID == id {
			return fiber.NewError(fiber.StatusBadRequest, "cannot reset own password")
		}
		plain, hash, err := resetPasswordHash("")
		if err != nil {
			return err
		}
		if err := repo.Update(c.UserContext(), id, map[string]any{"password_hash": hash}); err != nil {
			return err
		}
		auditLog(c, logs, "admin_user_password_reset", auditText("user", id, "password reset"), 0)
		return OK(c, resetPasswordResponse{ID: id, Password: plain})
	}
}

func AdjustUserBalance(repo *repository.UserRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req balancePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if req.Amount == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "amount must not be zero")
		}
		if err := repo.AdjustBalanceChecked(c.UserContext(), id, req.Amount); err != nil {
			if repository.IsInsufficientBalance(err) {
				return fiber.NewError(fiber.StatusBadRequest, "insufficient balance")
			}
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		detail := "amount=" + strconv.FormatFloat(req.Amount, 'f', 2, 64)
		if reason := strings.TrimSpace(req.Reason); reason != "" {
			detail += " reason=" + reason
		}
		auditLog(c, logs, "admin_user_balance", auditText("user", id, detail), req.Amount)
		return OK(c, fiber.Map{"id": id})
	}
}

type classPayload struct {
	Sort            int     `json:"sort"`
	Name            string  `json:"name"`
	QueryParam      string  `json:"queryParam"`
	DockingCode     string  `json:"dockingCode"`
	Price           float64 `json:"price"`
	QueryPlatform   string  `json:"queryPlatform"`
	DockingPlatform string  `json:"dockingPlatform"`
	PriceOperator   string  `json:"priceOperator"`
	Description     string  `json:"description"`
	Status          string  `json:"status"`
	Category        string  `json:"category"`
	BridgeEnabled   bool    `json:"bridgeEnabled"`
}

func CreateClass(repo *repository.ClassRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req classPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if strings.TrimSpace(req.Name) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "name is required")
		}
		if err := validateClassPayload(req); err != nil {
			return err
		}
		item := classFromPayload(req)
		if err := repo.Create(c.UserContext(), &item); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_class_create", auditText("class", item.ID, "name="+item.Name+" status="+item.Status), item.Price)
		return OK(c, item)
	}
}

func UpdateClass(repo *repository.ClassRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req classPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateClassPayload(req); err != nil {
			return err
		}
		values := map[string]any{
			"sort":           req.Sort,
			"price":          req.Price,
			"bridge_enabled": req.BridgeEnabled,
		}
		putString(values, "name", req.Name)
		putString(values, "query_param", req.QueryParam)
		putString(values, "docking_code", req.DockingCode)
		putString(values, "query_platform", req.QueryPlatform)
		putString(values, "docking_platform", req.DockingPlatform)
		putString(values, "price_operator", req.PriceOperator)
		putString(values, "description", req.Description)
		putString(values, "status", req.Status)
		putString(values, "category", req.Category)
		if err := repo.Update(c.UserContext(), id, values); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_class_update", auditText("class", id, auditFields(values)), req.Price)
		return OK(c, fiber.Map{"id": id})
	}
}

func DeleteClass(repo *repository.ClassRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if err := repo.Delete(c.UserContext(), id); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_class_delete", auditText("class", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

type specialPricePayload struct {
	UserID  uint    `json:"userId"`
	ClassID uint    `json:"classId"`
	Mode    int     `json:"mode"`
	Price   float64 `json:"price"`
}

func SpecialPrices(repo *repository.SpecialPriceRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := repo.List(c.UserContext(), c.Query("q"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func UpsertSpecialPrice(repo *repository.SpecialPriceRepository, users *repository.UserRepository, classes *repository.ClassRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req specialPricePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateSpecialPricePayload(req); err != nil {
			return err
		}
		if err := ensureSpecialPriceRefs(c, users, classes, req.UserID, req.ClassID); err != nil {
			return err
		}
		item := models.SpecialPrice{UserID: req.UserID, ClassID: req.ClassID, Mode: req.Mode, Price: req.Price}
		if err := repo.Upsert(c.UserContext(), &item); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_special_price_upsert", auditText("special_price", item.ID, "user="+strconv.FormatUint(uint64(item.UserID), 10)+" class="+strconv.FormatUint(uint64(item.ClassID), 10)), item.Price)
		return OK(c, item)
	}
}

func UpdateSpecialPrice(repo *repository.SpecialPriceRepository, users *repository.UserRepository, classes *repository.ClassRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req specialPricePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateSpecialPricePayload(req); err != nil {
			return err
		}
		if err := ensureSpecialPriceRefs(c, users, classes, req.UserID, req.ClassID); err != nil {
			return err
		}
		values := map[string]any{
			"user_id":  req.UserID,
			"class_id": req.ClassID,
			"mode":     req.Mode,
			"price":    req.Price,
		}
		if err := repo.Update(c.UserContext(), id, values); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_special_price_update", auditText("special_price", id, auditFields(values)), req.Price)
		return OK(c, fiber.Map{"id": id})
	}
}

func DeleteSpecialPrice(repo *repository.SpecialPriceRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if err := repo.Delete(c.UserContext(), id); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_special_price_delete", auditText("special_price", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

type orderPayload struct {
	UserID          uint    `json:"userId"`
	ClassID         uint    `json:"classId"`
	ConnectorID     uint    `json:"connectorId"`
	ExecutionMode   string  `json:"executionMode"`
	PluginCode      string  `json:"pluginCode"`
	WorkerID        string  `json:"workerId"`
	ProxyID         uint    `json:"proxyId"`
	RemoteOrderID   string  `json:"remoteOrderId"`
	Platform        string  `json:"platform"`
	School          string  `json:"school"`
	StudentName     string  `json:"studentName"`
	Account         string  `json:"account"`
	AccountPassword string  `json:"accountPassword"`
	CourseID        string  `json:"courseId"`
	CourseName      string  `json:"courseName"`
	Fee             float64 `json:"fee"`
	DockingCode     string  `json:"dockingCode"`
	FlashMode       bool    `json:"flashMode"`
	DockingStatus   string  `json:"dockingStatus"`
	Status          string  `json:"status"`
	Progress        string  `json:"progress"`
	RetryCount      int     `json:"retryCount"`
	Remarks         string  `json:"remarks"`
	Score           string  `json:"score"`
	DurationMinutes int     `json:"durationMinutes"`
}

type orderBatchPayload struct {
	IDs []uint `json:"ids"`
}

type orderBatchItemResult struct {
	ID     uint   `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type orderBatchResult struct {
	Requested int                    `json:"requested,omitempty"`
	Requeued  int                    `json:"requeued,omitempty"`
	Refunded  int                    `json:"refunded,omitempty"`
	Deleted   int                    `json:"deleted,omitempty"`
	Skipped   int                    `json:"skipped"`
	Failed    int                    `json:"failed"`
	Items     []orderBatchItemResult `json:"items"`
}

func CreateOrder(orderService *service.OrderService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req orderPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if strings.TrimSpace(req.Account) == "" || strings.TrimSpace(req.CourseName) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "account and courseName are required")
		}
		if err := validateOrderPayload(req); err != nil {
			return err
		}
		order := orderFromPayload(req)
		order.SourceIP = c.IP()
		if err := orderService.Submit(c.UserContext(), &order); err != nil {
			return err
		}
		auditLog(c, logs, "admin_order_create", auditText("order", order.ID, "status="+order.Status+" flash="+strconv.FormatBool(order.FlashMode)), order.Fee)
		return OK(c, order)
	}
}

func RefreshOrder(orderService *service.OrderService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if err := orderService.MarkRefreshRequested(c.UserContext(), id); err != nil {
			return err
		}
		auditLog(c, logs, "admin_order_refresh", auditText("order", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func BatchRefreshOrders(orderService *service.OrderService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := orderBatchIDs(c)
		if err != nil {
			return err
		}
		result := orderBatchResult{Items: make([]orderBatchItemResult, 0, len(ids))}
		for _, id := range ids {
			if err := orderService.MarkRefreshRequested(c.UserContext(), id); err != nil {
				if isFinalizedRefreshError(err) {
					result.Skipped++
					result.Items = append(result.Items, orderBatchItemResult{ID: id, Status: "skipped", Error: err.Error()})
					continue
				}
				result.Failed++
				result.Items = append(result.Items, orderBatchItemResult{ID: id, Status: "failed", Error: err.Error()})
				continue
			}
			result.Requested++
			result.Items = append(result.Items, orderBatchItemResult{ID: id, Status: "requested"})
		}
		auditLog(c, logs, "admin_order_batch_refresh", orderBatchAuditText("refresh", result), float64(result.Requested))
		return OK(c, result)
	}
}

func BatchResubmitOrders(orderService *service.OrderService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := orderBatchIDs(c)
		if err != nil {
			return err
		}
		result := orderBatchResult{Items: make([]orderBatchItemResult, 0, len(ids))}
		for _, id := range ids {
			if err := orderService.RequeueSubmit(c.UserContext(), id); err != nil {
				if isFinalizedResubmitError(err) {
					result.Skipped++
					result.Items = append(result.Items, orderBatchItemResult{ID: id, Status: "skipped", Error: err.Error()})
					continue
				}
				result.Failed++
				result.Items = append(result.Items, orderBatchItemResult{ID: id, Status: "failed", Error: err.Error()})
				continue
			}
			result.Requeued++
			result.Items = append(result.Items, orderBatchItemResult{ID: id, Status: "requeued"})
		}
		auditLog(c, logs, "admin_order_batch_resubmit", orderBatchAuditText("resubmit", result), float64(result.Requeued))
		return OK(c, result)
	}
}

func RefundOrder(repo *repository.OrderRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		order, err := repo.Refund(c.UserContext(), id)
		if err != nil {
			if repository.IsOrderAlreadyRefunded(err) {
				return fiber.NewError(fiber.StatusBadRequest, "order already refunded")
			}
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_order_refund", auditText("order", id, "user="+strconv.FormatUint(uint64(order.UserID), 10)), order.Fee)
		return OK(c, fiber.Map{"id": id})
	}
}

func BatchRefundOrders(repo *repository.OrderRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := orderBatchIDs(c)
		if err != nil {
			return err
		}
		result := orderBatchResult{Items: make([]orderBatchItemResult, 0, len(ids))}
		for _, id := range ids {
			order, err := repo.Refund(c.UserContext(), id)
			if err != nil {
				if repository.IsOrderAlreadyRefunded(err) {
					result.Skipped++
					result.Items = append(result.Items, orderBatchItemResult{ID: id, Status: "skipped", Error: "order already refunded"})
					continue
				}
				result.Failed++
				result.Items = append(result.Items, orderBatchItemResult{ID: id, Status: "failed", Error: err.Error()})
				continue
			}
			result.Refunded++
			result.Items = append(result.Items, orderBatchItemResult{ID: id, Status: "refunded"})
			auditLog(c, logs, "admin_order_refund", auditText("order", id, "user="+strconv.FormatUint(uint64(order.UserID), 10)), order.Fee)
		}
		if result.Refunded > 0 {
			invalidateDashboard(c.UserContext(), dashboard)
		}
		auditLog(c, logs, "admin_order_batch_refund", orderBatchAuditText("refund", result), float64(result.Refunded))
		return OK(c, result)
	}
}

func BatchDeleteOrders(repo *repository.OrderRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := orderBatchIDs(c)
		if err != nil {
			return err
		}
		result := orderBatchResult{Items: make([]orderBatchItemResult, 0, len(ids))}
		for _, id := range ids {
			if err := repo.Delete(c.UserContext(), id); err != nil {
				if repository.IsNotFound(err) {
					result.Skipped++
					result.Items = append(result.Items, orderBatchItemResult{ID: id, Status: "skipped", Error: "order not found"})
					continue
				}
				result.Failed++
				result.Items = append(result.Items, orderBatchItemResult{ID: id, Status: "failed", Error: err.Error()})
				continue
			}
			result.Deleted++
			result.Items = append(result.Items, orderBatchItemResult{ID: id, Status: "deleted"})
		}
		if result.Deleted > 0 {
			invalidateDashboard(c.UserContext(), dashboard)
		}
		auditLog(c, logs, "admin_order_batch_delete", orderBatchAuditText("delete", result), float64(result.Deleted))
		return OK(c, result)
	}
}

func RecoverOrderQueues(orderService *service.OrderService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := orderService.RecoverQueues(c.UserContext(), recoverBatchSize(c))
		if err != nil {
			return err
		}
		auditLog(c, logs, "admin_order_queue_recover", "recovered="+strconv.Itoa(result.Recovered), float64(result.Recovered))
		return OK(c, result)
	}
}

func UpdateOrder(repo *repository.OrderRepository, connectors *repository.ConnectorRepository, plugins *repository.PlatformPluginRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req orderPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateOrderPayload(req); err != nil {
			return err
		}
		if err := ensureAdminOrderRoute(c.UserContext(), connectors, plugins, req); err != nil {
			return err
		}
		values := orderUpdateValues(req)
		if err := repo.Update(c.UserContext(), id, values); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_order_update", auditText("order", id, auditFields(values)), req.Fee)
		return OK(c, fiber.Map{"id": id})
	}
}

func ensureAdminOrderRoute(ctx context.Context, connectors *repository.ConnectorRepository, plugins *repository.PlatformPluginRepository, req orderPayload) error {
	if orderPayloadUsesPlugin(req) {
		_, err := pluginForPurpose(ctx, plugins, req.PluginCode, "order")
		return err
	}
	return ensureActiveConnector(ctx, connectors, req.ConnectorID)
}

func orderPayloadUsesPlugin(req orderPayload) bool {
	return strings.EqualFold(strings.TrimSpace(req.ExecutionMode), "plugin") || strings.TrimSpace(req.PluginCode) != ""
}

func ensureActiveConnector(ctx context.Context, connectors *repository.ConnectorRepository, id uint) error {
	if id == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "connectorId is required")
	}
	connector, err := connectors.Find(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return fiber.NewError(fiber.StatusBadRequest, "connector not found")
		}
		return err
	}
	if connector.Status != "active" {
		return fiber.NewError(fiber.StatusBadRequest, "connector is disabled")
	}
	if !connector.OrderSyncEnabled {
		return fiber.NewError(fiber.StatusBadRequest, "connector order sync is disabled")
	}
	if connector.BaseURL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "connector baseUrl is required")
	}
	return nil
}

func DeleteOrder(repo *repository.OrderRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if err := repo.Delete(c.UserContext(), id); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_order_delete", auditText("order", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func queryInt(c *fiber.Ctx, key string, fallback int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func queryBool(c *fiber.Ctx, key string) (*bool, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, key+" must be true or false")
	}
	return &parsed, nil
}

func buildAdminAgentTree(users []models.User) []*adminAgentTreeNode {
	nodes := make(map[uint]*adminAgentTreeNode, len(users))
	childrenByParent := make(map[uint][]*adminAgentTreeNode, len(users))
	for _, user := range users {
		nodes[user.ID] = adminAgentTreeNodeFromUser(user)
	}
	for _, user := range users {
		node := nodes[user.ID]
		if parent := nodes[user.ParentID]; parent != nil {
			node.ParentAccount = parent.Account
		}
		if user.ParentID != 0 && user.ParentID != user.ID {
			childrenByParent[user.ParentID] = append(childrenByParent[user.ParentID], node)
		}
	}

	roots := make([]*adminAgentTreeNode, 0)
	for _, user := range users {
		if user.ParentID == 0 || user.ParentID == user.ID || nodes[user.ParentID] == nil {
			roots = append(roots, nodes[user.ID])
		}
	}

	visited := make(map[uint]bool, len(users))
	var attach func(node *adminAgentTreeNode, depth int, path map[uint]bool)
	attach = func(node *adminAgentTreeNode, depth int, path map[uint]bool) {
		if node == nil || path[node.ID] {
			return
		}
		path[node.ID] = true
		visited[node.ID] = true
		node.Depth = depth
		for _, child := range childrenByParent[node.ID] {
			if path[child.ID] {
				continue
			}
			attach(child, depth+1, cloneVisitPath(path))
			node.Children = append(node.Children, child)
		}
		node.DirectChildren = len(node.Children)
	}
	for _, root := range roots {
		attach(root, 0, map[uint]bool{})
	}
	for _, user := range users {
		if visited[user.ID] {
			continue
		}
		root := nodes[user.ID]
		roots = append(roots, root)
		attach(root, 0, map[uint]bool{})
	}
	return roots
}

func adminAgentTreeNodeFromUser(user models.User) *adminAgentTreeNode {
	return &adminAgentTreeNode{
		ID:        user.ID,
		ParentID:  user.ParentID,
		Account:   user.Account,
		Name:      user.Name,
		Balance:   user.Balance,
		PriceRate: user.PriceRate,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		LastIP:    user.LastIP,
	}
}

func cloneVisitPath(path map[uint]bool) map[uint]bool {
	next := make(map[uint]bool, len(path))
	for key, value := range path {
		next[key] = value
	}
	return next
}

func filterAdminAgentTree(nodes []*adminAgentTreeNode, search string) ([]*adminAgentTreeNode, int) {
	search = strings.ToLower(strings.TrimSpace(search))
	if search == "" {
		return nodes, countAdminAgentTree(nodes)
	}
	filtered := make([]*adminAgentTreeNode, 0, len(nodes))
	matched := 0
	for _, node := range nodes {
		next, ok, count := filterAdminAgentNode(node, search)
		if ok {
			filtered = append(filtered, next)
			matched += count
		}
	}
	return filtered, matched
}

func filterAdminAgentNode(node *adminAgentTreeNode, search string) (*adminAgentTreeNode, bool, int) {
	if node == nil {
		return nil, false, 0
	}
	copy := *node
	copy.Children = nil
	matched := adminAgentNodeMatches(node, search)
	count := 0
	for _, child := range node.Children {
		next, ok, childCount := filterAdminAgentNode(child, search)
		if ok {
			copy.Children = append(copy.Children, next)
			count += childCount
		}
	}
	if matched {
		count++
	}
	return &copy, matched || len(copy.Children) > 0, count
}

func adminAgentNodeMatches(node *adminAgentTreeNode, search string) bool {
	return strings.Contains(strings.ToLower(node.Account), search) ||
		strings.Contains(strings.ToLower(node.Name), search) ||
		strings.Contains(strconv.FormatUint(uint64(node.ID), 10), search)
}

func countAdminAgentTree(nodes []*adminAgentTreeNode) int {
	total := 0
	for _, node := range nodes {
		total++
		total += countAdminAgentTree(node.Children)
	}
	return total
}

func recoverBatchSize(c *fiber.Ctx) int {
	size := queryInt(c, "batchSize", 500)
	if size < 1 {
		return 500
	}
	if size > 5000 {
		return 5000
	}
	return size
}

func orderBatchIDs(c *fiber.Ctx) ([]uint, error) {
	var req orderBatchPayload
	if err := c.BodyParser(&req); err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	return normalizeOrderBatchIDs(req.IDs)
}

func normalizeOrderBatchIDs(ids []uint) ([]uint, error) {
	if len(ids) == 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "ids are required")
	}
	if len(ids) > 500 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "ids must contain 500 or fewer orders")
	}
	seen := make(map[uint]bool, len(ids))
	normalized := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "ids must be positive")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		normalized = append(normalized, id)
	}
	return normalized, nil
}

func isFinalizedRefreshError(err error) bool {
	var validationErr service.ValidationError
	return errors.As(err, &validationErr) && validationErr.Message == "finalized orders cannot be refreshed"
}

func isFinalizedResubmitError(err error) bool {
	var validationErr service.ValidationError
	return errors.As(err, &validationErr) && validationErr.Message == "finalized orders cannot be resubmitted"
}

func orderBatchAuditText(action string, result orderBatchResult) string {
	count := result.Requested
	if action == "resubmit" {
		count = result.Requeued
	}
	if action == "refund" {
		count = result.Refunded
	}
	if action == "delete" {
		count = result.Deleted
	}
	return "action=" + action +
		" count=" + strconv.Itoa(count) +
		" skipped=" + strconv.Itoa(result.Skipped) +
		" failed=" + strconv.Itoa(result.Failed)
}

func pathID(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	return uint(id), nil
}

func userFromPayload(req userPayload) models.User {
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}
	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "agent"
	}
	priceRate := req.PriceRate
	if priceRate == 0 {
		priceRate = 1
	}
	return models.User{
		ParentID:  req.ParentID,
		Account:   strings.TrimSpace(req.Account),
		Name:      strings.TrimSpace(req.Name),
		Balance:   req.Balance,
		PriceRate: priceRate,
		Role:      role,
		Status:    status,
	}
}

func classFromPayload(req classPayload) models.CourseClass {
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "online"
	}
	operator := strings.TrimSpace(req.PriceOperator)
	if operator == "" {
		operator = "*"
	}
	return models.CourseClass{
		Sort:            req.Sort,
		Name:            strings.TrimSpace(req.Name),
		QueryParam:      strings.TrimSpace(req.QueryParam),
		DockingCode:     strings.TrimSpace(req.DockingCode),
		Price:           req.Price,
		QueryPlatform:   strings.TrimSpace(req.QueryPlatform),
		DockingPlatform: strings.TrimSpace(req.DockingPlatform),
		PriceOperator:   operator,
		Description:     strings.TrimSpace(req.Description),
		Status:          status,
		Category:        strings.TrimSpace(req.Category),
		BridgeEnabled:   req.BridgeEnabled,
	}
}

func categoryFromPayload(req categoryPayload) models.CourseCategory {
	return models.CourseCategory{
		Sort:        req.Sort,
		Name:        strings.TrimSpace(req.Name),
		Status:      categoryStatus(req.Status),
		Pinned:      req.Pinned,
		Description: strings.TrimSpace(req.Description),
	}
}

func categoryStatus(value string) string {
	status := strings.TrimSpace(value)
	if status == "" {
		return "active"
	}
	return status
}

func orderFromPayload(req orderPayload) models.Order {
	return models.Order{
		UserID:          req.UserID,
		ClassID:         req.ClassID,
		ConnectorID:     req.ConnectorID,
		ExecutionMode:   strings.TrimSpace(req.ExecutionMode),
		PluginCode:      strings.TrimSpace(req.PluginCode),
		WorkerID:        strings.TrimSpace(req.WorkerID),
		ProxyID:         req.ProxyID,
		RemoteOrderID:   strings.TrimSpace(req.RemoteOrderID),
		Platform:        strings.TrimSpace(req.Platform),
		School:          strings.TrimSpace(req.School),
		StudentName:     strings.TrimSpace(req.StudentName),
		Account:         strings.TrimSpace(req.Account),
		AccountPassword: strings.TrimSpace(req.AccountPassword),
		CourseID:        strings.TrimSpace(req.CourseID),
		CourseName:      strings.TrimSpace(req.CourseName),
		Fee:             req.Fee,
		DockingCode:     strings.TrimSpace(req.DockingCode),
		FlashMode:       req.FlashMode,
		DockingStatus:   defaultOrderDockingStatus(req.DockingStatus),
		Status:          defaultOrderStatus(req.Status),
		Progress:        strings.TrimSpace(req.Progress),
		RetryCount:      req.RetryCount,
		Remarks:         strings.TrimSpace(req.Remarks),
		Score:           strings.TrimSpace(req.Score),
		DurationMinutes: req.DurationMinutes,
	}
}

func orderUpdateValues(req orderPayload) map[string]any {
	return map[string]any{
		"user_id":          req.UserID,
		"class_id":         req.ClassID,
		"connector_id":     req.ConnectorID,
		"execution_mode":   defaultOrderExecutionMode(req.ExecutionMode, req.PluginCode),
		"plugin_code":      strings.TrimSpace(req.PluginCode),
		"worker_id":        strings.TrimSpace(req.WorkerID),
		"proxy_id":         req.ProxyID,
		"remote_order_id":  strings.TrimSpace(req.RemoteOrderID),
		"platform":         strings.TrimSpace(req.Platform),
		"school":           strings.TrimSpace(req.School),
		"student_name":     strings.TrimSpace(req.StudentName),
		"account":          strings.TrimSpace(req.Account),
		"account_password": strings.TrimSpace(req.AccountPassword),
		"course_id":        strings.TrimSpace(req.CourseID),
		"course_name":      strings.TrimSpace(req.CourseName),
		"fee":              req.Fee,
		"docking_code":     strings.TrimSpace(req.DockingCode),
		"flash_mode":       req.FlashMode,
		"docking_status":   defaultOrderDockingStatus(req.DockingStatus),
		"status":           defaultOrderStatus(req.Status),
		"progress":         strings.TrimSpace(req.Progress),
		"retry_count":      req.RetryCount,
		"remarks":          strings.TrimSpace(req.Remarks),
		"score":            strings.TrimSpace(req.Score),
		"duration_minutes": req.DurationMinutes,
	}
}

func defaultOrderExecutionMode(value string, pluginCode string) string {
	mode := strings.TrimSpace(value)
	if strings.TrimSpace(pluginCode) != "" {
		return "plugin"
	}
	if mode == "" {
		return "connector"
	}
	return mode
}

func defaultOrderStatus(value string) string {
	status := strings.TrimSpace(value)
	if status == "" {
		return "pending"
	}
	return status
}

func defaultOrderDockingStatus(value string) string {
	status := strings.TrimSpace(value)
	if status == "" {
		return "pending"
	}
	return status
}

func validateUserPayload(req userPayload) error {
	if req.Balance < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "balance must be zero or greater")
	}
	if req.PriceRate < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "priceRate must be zero or greater")
	}
	if status := strings.TrimSpace(req.Status); status != "" && !oneOf(status, "active", "disabled") {
		return fiber.NewError(fiber.StatusBadRequest, "status must be active or disabled")
	}
	if role := strings.TrimSpace(req.Role); role != "" && !oneOf(role, "admin", "agent") {
		return fiber.NewError(fiber.StatusBadRequest, "role must be admin or agent")
	}
	return nil
}

func validateClassPayload(req classPayload) error {
	if req.Price < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "price must be zero or greater")
	}
	if status := strings.TrimSpace(req.Status); status != "" && !oneOf(status, "online", "offline") {
		return fiber.NewError(fiber.StatusBadRequest, "status must be online or offline")
	}
	if operator := strings.TrimSpace(req.PriceOperator); operator != "" && !oneOf(operator, "*", "+") {
		return fiber.NewError(fiber.StatusBadRequest, "priceOperator must be * or +")
	}
	return nil
}

func validateCategoryPayload(req categoryPayload) error {
	if strings.TrimSpace(req.Name) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if len(strings.TrimSpace(req.Name)) > 120 {
		return fiber.NewError(fiber.StatusBadRequest, "name is too long")
	}
	if len(strings.TrimSpace(req.Description)) > 2000 {
		return fiber.NewError(fiber.StatusBadRequest, "description is too long")
	}
	if status := strings.TrimSpace(req.Status); status != "" && !oneOf(status, "active", "disabled") {
		return fiber.NewError(fiber.StatusBadRequest, "status must be active or disabled")
	}
	return nil
}

func validateSpecialPricePayload(req specialPricePayload) error {
	if req.UserID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "userId is required")
	}
	if req.ClassID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "classId is required")
	}
	if req.Price < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "price must be zero or greater")
	}
	if req.Mode < 0 || req.Mode > 2 {
		return fiber.NewError(fiber.StatusBadRequest, "mode must be 0, 1, or 2")
	}
	return nil
}

func validateOrderPayload(req orderPayload) error {
	if req.Fee < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "fee must be zero or greater")
	}
	if req.RetryCount < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "retryCount must be zero or greater")
	}
	if req.DurationMinutes < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "durationMinutes must be zero or greater")
	}
	if status := strings.TrimSpace(req.Status); status != "" && !oneOf(status, "pending", "queued", "processing", "done", "failed", "cancelled", "refunded") {
		return fiber.NewError(fiber.StatusBadRequest, "status is invalid")
	}
	if dockingStatus := strings.TrimSpace(req.DockingStatus); dockingStatus != "" && !oneOf(dockingStatus, "pending", "sent", "plugin_sent", "refresh_requested", "failed", "queue_failed", "cancelled", "refunded") {
		return fiber.NewError(fiber.StatusBadRequest, "dockingStatus is invalid")
	}
	if mode := strings.TrimSpace(req.ExecutionMode); mode != "" && !oneOf(mode, "connector", "plugin") {
		return fiber.NewError(fiber.StatusBadRequest, "executionMode is invalid")
	}
	return nil
}

func ensureSpecialPriceRefs(c *fiber.Ctx, users *repository.UserRepository, classes *repository.ClassRepository, userID uint, classID uint) error {
	if _, err := users.Find(c.UserContext(), userID); err != nil {
		if repository.IsNotFound(err) {
			return fiber.NewError(fiber.StatusBadRequest, "user not found")
		}
		return err
	}
	if _, err := classes.Find(c.UserContext(), classID); err != nil {
		if repository.IsNotFound(err) {
			return fiber.NewError(fiber.StatusBadRequest, "class not found")
		}
		return err
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func putString(values map[string]any, key string, value string) {
	if strings.TrimSpace(value) != "" {
		values[key] = strings.TrimSpace(value)
	}
}

func invalidateDashboard(ctx context.Context, dashboard *service.DashboardService) {
	if dashboard != nil {
		dashboard.Invalidate(ctx)
	}
}

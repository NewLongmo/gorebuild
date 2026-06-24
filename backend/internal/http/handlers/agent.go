package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	appmw "dw0rdwk/backend/internal/http/middleware"
	"dw0rdwk/backend/internal/models"
	passwordhash "dw0rdwk/backend/internal/password"
	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type agentProfile struct {
	ID              uint    `json:"id"`
	Account         string  `json:"account"`
	Name            string  `json:"name"`
	Balance         float64 `json:"balance"`
	PriceRate       float64 `json:"priceRate"`
	Role            string  `json:"role"`
	APIKey          string  `json:"apiKey"`
	InviteCode      string  `json:"inviteCode"`
	InvitePriceRate float64 `json:"invitePriceRate"`
	Notice          string  `json:"notice"`
}

type agentDashboard struct {
	Balance      float64                      `json:"balance"`
	PriceRate    float64                      `json:"priceRate"`
	Orders       int64                        `json:"orders"`
	Pending      int64                        `json:"pending"`
	Done         int64                        `json:"done"`
	Failed       int64                        `json:"failed"`
	TodayOrders  int64                        `json:"todayOrders"`
	Unfinished   int64                        `json:"unfinished"`
	Refreshing   int64                        `json:"refreshing"`
	SubAgents    int64                        `json:"subAgents"`
	TotalSpend   float64                      `json:"totalSpend"`
	SiteNotice   string                       `json:"siteNotice"`
	PopupNotice  string                       `json:"popupNotice"`
	NoticeURL    string                       `json:"noticeUrl"`
	ParentNotice string                       `json:"parentNotice"`
	OwnNotice    string                       `json:"ownNotice"`
	Trend7       []repository.DailyOrderPoint `json:"trend7"`
}

type agentClassRow struct {
	models.CourseClass
	UserPrice float64 `json:"userPrice"`
}

type agentOrderPayload struct {
	ClassID         uint   `json:"classId"`
	School          string `json:"school"`
	StudentName     string `json:"studentName"`
	Account         string `json:"account"`
	AccountPassword string `json:"accountPassword"`
	CourseID        string `json:"courseId"`
	CourseName      string `json:"courseName"`
	FlashMode       bool   `json:"flashMode"`
	DurationMinutes int    `json:"durationMinutes"`
}

type agentOrderPasswordPayload struct {
	Password string `json:"password"`
}

type childAgentPayload struct {
	Account   string  `json:"account"`
	Password  string  `json:"password"`
	Name      string  `json:"name"`
	Balance   float64 `json:"balance"`
	PriceRate float64 `json:"priceRate"`
	Status    string  `json:"status"`
}

type childBalancePayload struct {
	Amount float64 `json:"amount"`
}

type passwordChangePayload struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type agentNoticePayload struct {
	Notice string `json:"notice"`
}

type agentInvitePayload struct {
	Code      string  `json:"code"`
	PriceRate float64 `json:"priceRate"`
}

func AgentProfile(users *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		return OK(c, agentProfile{
			ID:              user.ID,
			Account:         user.Account,
			Name:            user.Name,
			Balance:         user.Balance,
			PriceRate:       user.PriceRate,
			Role:            user.Role,
			APIKey:          user.APIKey,
			InviteCode:      user.InviteCode,
			InvitePriceRate: user.InvitePriceRate,
			Notice:          user.Notice,
		})
	}
}

func AgentRegenerateAPIKey(users *repository.UserRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		key, err := generateAPIKey()
		if err != nil {
			return err
		}
		if err := users.Update(c.UserContext(), user.ID, map[string]any{"api_key": key}); err != nil {
			return err
		}
		auditLog(c, logs, "agent_api_key_regenerate", auditText("user", user.ID, "api key regenerated"), 0)
		return OK(c, fiber.Map{"apiKey": key})
	}
}

func AgentDisableAPIKey(users *repository.UserRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		if err := users.Update(c.UserContext(), user.ID, map[string]any{"api_key": ""}); err != nil {
			return err
		}
		auditLog(c, logs, "agent_api_key_disable", auditText("user", user.ID, "api key disabled"), 0)
		return OK(c, fiber.Map{"ok": true})
	}
}

func AgentDashboard(users *repository.UserRepository, orders *repository.OrderRepository, settings *repository.SettingRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		configs, err := settings.All(c.UserContext())
		if err != nil {
			return err
		}
		parentNotice := ""
		if user.ParentID != 0 {
			parent, err := users.Find(c.UserContext(), user.ParentID)
			if err != nil && !repository.IsNotFound(err) {
				return err
			}
			parentNotice = parent.Notice
		}
		totalOrders, err := orders.CountByUser(c.UserContext(), user.ID, nil)
		if err != nil {
			return err
		}
		pending, err := orders.CountByUser(c.UserContext(), user.ID, []string{"pending", "queued", "processing"})
		if err != nil {
			return err
		}
		done, err := orders.CountByUser(c.UserContext(), user.ID, []string{"done"})
		if err != nil {
			return err
		}
		failed, err := orders.CountByUser(c.UserContext(), user.ID, []string{"failed"})
		if err != nil {
			return err
		}
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		todayOrders, err := orders.CountByUserSince(c.UserContext(), user.ID, today, nil)
		if err != nil {
			return err
		}
		unfinished, err := orders.CountByUserNotStatuses(c.UserContext(), user.ID, []string{"done", "cancelled", "refunded"})
		if err != nil {
			return err
		}
		refreshing, err := orders.CountByUserDockingStatuses(c.UserContext(), user.ID, []string{"refresh_requested"})
		if err != nil {
			return err
		}
		subAgents, err := users.CountChildren(c.UserContext(), user.ID)
		if err != nil {
			return err
		}
		totalSpend, err := orders.SumFeesByUser(c.UserContext(), user.ID)
		if err != nil {
			return err
		}
		trend, err := orders.DailyTrendByUser(c.UserContext(), user.ID, 7)
		if err != nil {
			return err
		}
		return OK(c, agentDashboard{
			Balance:      user.Balance,
			PriceRate:    user.PriceRate,
			Orders:       totalOrders,
			Pending:      pending,
			Done:         done,
			Failed:       failed,
			TodayOrders:  todayOrders,
			Unfinished:   unfinished,
			Refreshing:   refreshing,
			SubAgents:    subAgents,
			TotalSpend:   totalSpend,
			SiteNotice:   configs["site_notice"],
			PopupNotice:  configs["popup_notice"],
			NoticeURL:    configs["notice_url"],
			ParentNotice: parentNotice,
			OwnNotice:    user.Notice,
			Trend7:       trend,
		})
	}
}

func AgentLogs(logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		data, err := logs.ListByUser(c.UserContext(), claims.UID, c.Query("q"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func AgentChangePassword(users *repository.UserRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		var req passwordChangePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if !passwordhash.Verify(user.PasswordHash, req.CurrentPassword) {
			return fiber.NewError(fiber.StatusBadRequest, "current password is incorrect")
		}
		if len(strings.TrimSpace(req.NewPassword)) < 8 {
			return fiber.NewError(fiber.StatusBadRequest, "new password must be at least 8 characters")
		}
		if req.CurrentPassword == req.NewPassword {
			return fiber.NewError(fiber.StatusBadRequest, "new password must be different")
		}
		hash, err := passwordhash.Hash(req.NewPassword)
		if err != nil {
			return err
		}
		if err := users.Update(c.UserContext(), user.ID, map[string]any{"password_hash": hash}); err != nil {
			return err
		}
		auditLog(c, logs, "agent_password_update", auditText("user", user.ID, "password changed"), 0)
		return OK(c, fiber.Map{"ok": true})
	}
}

func AgentUpdateNotice(users *repository.UserRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		var req agentNoticePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		notice := strings.TrimSpace(req.Notice)
		if len(notice) > 5000 {
			return fiber.NewError(fiber.StatusBadRequest, "notice is too long")
		}
		if err := users.Update(c.UserContext(), user.ID, map[string]any{"notice": notice}); err != nil {
			return err
		}
		auditLog(c, logs, "agent_notice_update", auditText("user", user.ID, "notice updated"), 0)
		return OK(c, fiber.Map{"notice": notice})
	}
}

func AgentUpdateInvite(users *repository.UserRepository, invites *repository.InviteCodeRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		var req agentInvitePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if strings.TrimSpace(req.Code) == "" {
			code, err := generateInviteCode(8)
			if err != nil {
				return err
			}
			req.Code = code
		}
		if err := validateInvitePayload(inviteCodePayload{Code: req.Code, PriceRate: req.PriceRate}); err != nil {
			return err
		}
		code := repository.NormalizeInviteCode(req.Code)
		priceRate := normalizeInvitePriceRate(req.PriceRate)
		if priceRate < user.PriceRate {
			return fiber.NewError(fiber.StatusBadRequest, "invite priceRate must be greater than or equal to your priceRate")
		}
		exists, err := invites.CodeExists(c.UserContext(), code)
		if err != nil {
			return err
		}
		if exists {
			return fiber.NewError(fiber.StatusBadRequest, "invite code already exists")
		}
		exists, err = users.InviteCodeExists(c.UserContext(), code, user.ID)
		if err != nil {
			return err
		}
		if exists {
			return fiber.NewError(fiber.StatusBadRequest, "invite code already exists")
		}
		values := map[string]any{"invite_code": code, "invite_price_rate": priceRate}
		if err := users.Update(c.UserContext(), user.ID, values); err != nil {
			return err
		}
		auditLog(c, logs, "agent_invite_update", auditText("user", user.ID, "code="+code+" priceRate="+strconv.FormatFloat(priceRate, 'f', 4, 64)), 0)
		return OK(c, fiber.Map{"inviteCode": code, "invitePriceRate": priceRate})
	}
}

func AgentChildren(users *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		data, err := users.ListChildren(c.UserContext(), claims.UID, c.Query("q"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func AgentCreateChild(users *repository.UserRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		var req childAgentPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if strings.TrimSpace(req.Account) == "" || strings.TrimSpace(req.Password) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "account and password are required")
		}
		if err := validateChildAgentPayload(req, true); err != nil {
			return err
		}
		hash, err := passwordhash.Hash(req.Password)
		if err != nil {
			return err
		}
		child := models.User{
			Account:      strings.TrimSpace(req.Account),
			PasswordHash: hash,
			Name:         strings.TrimSpace(req.Name),
			PriceRate:    childPriceRate(req.PriceRate),
			Status:       childStatus(req.Status),
		}
		if err := users.CreateChildWithBalance(c.UserContext(), claims.UID, &child, req.Balance); err != nil {
			if repository.IsInsufficientBalance(err) {
				return fiber.NewError(fiber.StatusBadRequest, "insufficient balance")
			}
			return err
		}
		auditLog(c, logs, "agent_child_create", auditText("user", child.ID, "account="+child.Account+" status="+child.Status), -req.Balance)
		return OK(c, child)
	}
}

func AgentUpdateChild(users *repository.UserRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req childAgentPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateChildAgentPayload(req, false); err != nil {
			return err
		}
		values := map[string]any{}
		if strings.TrimSpace(req.Password) != "" {
			hash, err := passwordhash.Hash(req.Password)
			if err != nil {
				return err
			}
			values["password_hash"] = hash
		}
		putString(values, "name", req.Name)
		if req.PriceRate > 0 {
			values["price_rate"] = req.PriceRate
		}
		putString(values, "status", req.Status)
		if err := users.UpdateChild(c.UserContext(), claims.UID, id, values); err != nil {
			return err
		}
		auditLog(c, logs, "agent_child_update", auditText("user", id, auditFields(values)), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func AgentResetChildPassword(users *repository.UserRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if id == claims.UID {
			return fiber.NewError(fiber.StatusBadRequest, "cannot reset own password")
		}
		plain, hash, err := resetPasswordHash("")
		if err != nil {
			return err
		}
		if err := users.UpdateChild(c.UserContext(), claims.UID, id, map[string]any{"password_hash": hash}); err != nil {
			return err
		}
		auditLog(c, logs, "agent_child_password_reset", auditText("user", id, "password reset"), 0)
		return OK(c, resetPasswordResponse{ID: id, Password: plain})
	}
}

func AgentTransferChildBalance(users *repository.UserRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req childBalancePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if req.Amount == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "amount must not be zero")
		}
		result, err := users.TransferBalanceToChild(c.UserContext(), claims.UID, id, req.Amount)
		if err != nil {
			if repository.IsInsufficientBalance(err) {
				if req.Amount < 0 {
					return fiber.NewError(fiber.StatusBadRequest, "child balance is insufficient")
				}
				return fiber.NewError(fiber.StatusBadRequest, "insufficient balance")
			}
			return err
		}
		auditLog(c, logs, "agent_child_balance", auditText("user", id, "amount="+strconv.FormatFloat(req.Amount, 'f', 2, 64)+" charged="+strconv.FormatFloat(result.ChargedAmount, 'f', 2, 64)), -result.ChargedAmount)
		return OK(c, result)
	}
}

func AgentCategories(categories *repository.CategoryRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		items, err := categories.ListActive(c.UserContext(), queryInt(c, "limit", 200))
		if err != nil {
			return err
		}
		return OK(c, fiber.Map{"items": items})
	}
}

func AgentClasses(users *repository.UserRepository, classes *repository.ClassRepository, specialPrices *repository.SpecialPriceRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		status := "online"
		page, err := classes.List(c.UserContext(), c.Query("q"), &status, c.Query("category"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		items := make([]agentClassRow, 0, len(page.Items))
		for _, item := range page.Items {
			special, hasSpecial, err := specialPrices.FindForUserClass(c.UserContext(), user.ID, item.ID)
			if err != nil {
				return err
			}
			items = append(items, agentClassRow{
				CourseClass: item,
				UserPrice:   agentPrice(user, item, specialPricePtr(special, hasSpecial)),
			})
		}
		return OK(c, repository.Page[agentClassRow]{
			Items:   items,
			Total:   page.Total,
			Page:    page.Page,
			PerPage: page.PerPage,
		})
	}
}

func AgentOrders(orders *repository.OrderRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		data, err := orders.ListByUser(c.UserContext(), claims.UID, c.Query("q"), c.Query("status"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func AgentCreateOrder(
	users *repository.UserRepository,
	classes *repository.ClassRepository,
	specialPrices *repository.SpecialPriceRepository,
	connectors *repository.ConnectorRepository,
	plugins *repository.PlatformPluginRepository,
	settings *repository.SettingRepository,
	orderService *service.OrderService,
	logs *repository.LogRepository,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		var req agentOrderPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		user, err := users.Find(c.UserContext(), claims.UID)
		if err != nil {
			return err
		}
		order, err := createAgentOrderFromPayload(c.UserContext(), c.IP(), user, req, users, classes, specialPrices, connectors, plugins, settings, orderService)
		if err != nil {
			return err
		}
		auditLog(c, logs, "agent_order_create", auditText("order", order.ID, "class="+order.Platform+" flash="+strconv.FormatBool(order.FlashMode)), -order.Fee)
		return OK(c, order)
	}
}

func AgentRefreshOrder(orders *repository.OrderRepository, orderService *service.OrderService, logs *repository.LogRepository) fiber.Handler {
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
		if err := orderService.MarkRefreshRequested(c.UserContext(), id); err != nil {
			return err
		}
		auditLog(c, logs, "agent_order_refresh", auditText("order", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func AgentCancelOrder(orders *repository.OrderRepository, orderService *service.OrderService, logs *repository.LogRepository) fiber.Handler {
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
		if err := orderService.Cancel(c.UserContext(), id); err != nil {
			return err
		}
		auditLog(c, logs, "agent_order_cancel", auditText("order", id, "cancelled by agent"), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func AgentUpdateOrderPassword(orders *repository.OrderRepository, orderService *service.OrderService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req agentOrderPasswordPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		password, err := validateOrderPasswordPayload(req)
		if err != nil {
			return err
		}
		if _, err := orders.FindByUser(c.UserContext(), id, claims.UID); err != nil {
			return err
		}
		if err := orderService.UpdatePasswordAndRefresh(c.UserContext(), id, password, "agent"); err != nil {
			return err
		}
		auditLog(c, logs, "agent_order_password_update", auditText("order", id, "password updated and refresh queued"), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func currentUser(c *fiber.Ctx, users *repository.UserRepository) (models.User, error) {
	claims, ok := appmw.Claims(c)
	if !ok {
		return models.User{}, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	return users.Find(c.UserContext(), claims.UID)
}

func defaultConnector(c *fiber.Ctx, connectors *repository.ConnectorRepository, settings *repository.SettingRepository) (models.Connector, error) {
	values, err := settings.All(c.UserContext())
	if err == nil {
		if raw := strings.TrimSpace(values["default_connector_id"]); raw != "" {
			id, parseErr := strconv.ParseUint(raw, 10, 64)
			if parseErr != nil || id == 0 {
				return models.Connector{}, fiber.NewError(fiber.StatusBadRequest, "default_connector_id is invalid")
			}
			connector, findErr := connectors.Find(c.UserContext(), uint(id))
			if findErr != nil {
				return models.Connector{}, findErr
			}
			if connector.Status != "active" || !connector.OrderSyncEnabled || connector.BaseURL == "" {
				return models.Connector{}, fiber.NewError(fiber.StatusBadRequest, "default connector is disabled")
			}
			return connector, nil
		}
	}
	connector, err := connectors.FirstActive(c.UserContext())
	if err != nil {
		if repository.IsNotFound(err) {
			return models.Connector{}, fiber.NewError(fiber.StatusBadRequest, "default connector is not configured")
		}
		return models.Connector{}, err
	}
	return connector, nil
}

func agentPrice(user models.User, class models.CourseClass, special *models.SpecialPrice) float64 {
	rate := user.PriceRate
	if rate == 0 {
		rate = 1
	}
	basePrice := class.Price
	if special != nil {
		switch special.Mode {
		case 0:
			return nonNegativePrice(defaultAgentPrice(basePrice, rate, class.PriceOperator) - special.Price)
		case 1:
			basePrice = nonNegativePrice(basePrice - special.Price)
		case 2:
			return nonNegativePrice(special.Price)
		}
	}
	return nonNegativePrice(defaultAgentPrice(basePrice, rate, class.PriceOperator))
}

func defaultAgentPrice(basePrice, rate float64, operator string) float64 {
	if operator == "+" {
		return basePrice + rate
	}
	return basePrice * rate
}

func specialPricePtr(item models.SpecialPrice, ok bool) *models.SpecialPrice {
	if !ok {
		return nil
	}
	return &item
}

func nonNegativePrice(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}

func validateOrderPasswordPayload(req agentOrderPasswordPayload) (string, error) {
	password := strings.TrimSpace(req.Password)
	if password == "" {
		return "", fiber.NewError(fiber.StatusBadRequest, "password is required")
	}
	if len(password) > 160 {
		return "", fiber.NewError(fiber.StatusBadRequest, "password is too long")
	}
	return password, nil
}

func validateChildAgentPayload(req childAgentPayload, creating bool) error {
	if creating && req.Balance < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "balance must be zero or greater")
	}
	if req.PriceRate < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "priceRate must be zero or greater")
	}
	if status := strings.TrimSpace(req.Status); status != "" && !oneOf(status, "active", "disabled") {
		return fiber.NewError(fiber.StatusBadRequest, "status must be active or disabled")
	}
	return nil
}

func childPriceRate(value float64) float64 {
	if value == 0 {
		return 1
	}
	return value
}

func childStatus(value string) string {
	status := strings.TrimSpace(value)
	if status == "" {
		return "active"
	}
	return status
}

func generateAPIKey() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

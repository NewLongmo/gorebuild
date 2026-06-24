package handlers

import (
	"strconv"
	"strings"

	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type classIDsPayload struct {
	IDs []uint `json:"ids"`
}

type classBatchStatusPayload struct {
	IDs    []uint `json:"ids"`
	Status string `json:"status"`
}

type classBatchMovePayload struct {
	IDs      []uint `json:"ids"`
	Category string `json:"category"`
}

type classBatchPatchPayload struct {
	Updates []classBatchPatchItem `json:"updates"`
}

type classBatchPatchItem struct {
	ID              uint     `json:"id"`
	Name            *string  `json:"name"`
	QueryParam      *string  `json:"queryParam"`
	DockingCode     *string  `json:"dockingCode"`
	Price           *float64 `json:"price"`
	QueryPlatform   *string  `json:"queryPlatform"`
	DockingPlatform *string  `json:"dockingPlatform"`
	PriceOperator   *string  `json:"priceOperator"`
	Description     *string  `json:"description"`
	Status          *string  `json:"status"`
	Category        *string  `json:"category"`
	Sort            *int     `json:"sort"`
	BridgeEnabled   *bool    `json:"bridgeEnabled"`
}

type classKeywordPayload struct {
	Scope      string `json:"scope"`
	ScopeID    string `json:"scopeId"`
	OldKeyword string `json:"oldKeyword"`
	NewKeyword string `json:"newKeyword"`
}

type classPrefixPayload struct {
	Scope   string `json:"scope"`
	ScopeID string `json:"scopeId"`
	Prefix  string `json:"prefix"`
}

type classDeduplicatePayload struct {
	Scope    string `json:"scope"`
	ScopeID  string `json:"scopeId"`
	Strategy string `json:"strategy"`
	Limit    int    `json:"limit"`
}

func BatchUpdateClassStatus(repo *repository.ClassRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req classBatchStatusPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateClassIDs(req.IDs); err != nil {
			return err
		}
		status := strings.TrimSpace(req.Status)
		if !oneOf(status, "online", "offline") {
			return fiber.NewError(fiber.StatusBadRequest, "status must be online or offline")
		}
		affected, err := repo.BulkUpdateStatus(c.UserContext(), req.IDs, status)
		if err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_class_batch_status", "count="+strconv.FormatInt(affected, 10)+" status="+status, 0)
		return OK(c, fiber.Map{"affected": affected})
	}
}

func BatchMoveClasses(repo *repository.ClassRepository, categories *repository.CategoryRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req classBatchMovePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateClassIDs(req.IDs); err != nil {
			return err
		}
		category := strings.TrimSpace(req.Category)
		if category != "" {
			if _, err := categories.FindByName(c.UserContext(), category); err != nil {
				if repository.IsNotFound(err) {
					return fiber.NewError(fiber.StatusBadRequest, "category not found")
				}
				return err
			}
		}
		affected, err := repo.BulkMove(c.UserContext(), req.IDs, category)
		if err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_class_batch_move", "count="+strconv.FormatInt(affected, 10)+" category="+category, 0)
		return OK(c, fiber.Map{"affected": affected})
	}
}

func BatchDeleteClasses(repo *repository.ClassRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req classIDsPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateClassIDs(req.IDs); err != nil {
			return err
		}
		affected, err := repo.BulkDelete(c.UserContext(), req.IDs)
		if err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_class_batch_delete", "count="+strconv.FormatInt(affected, 10), 0)
		return OK(c, fiber.Map{"affected": affected})
	}
}

func BatchPatchClasses(repo *repository.ClassRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req classBatchPatchPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if len(req.Updates) == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "updates are required")
		}
		updated := 0
		for _, item := range req.Updates {
			values, err := classBatchPatchValues(item)
			if err != nil {
				return err
			}
			if len(values) == 0 {
				continue
			}
			if err := repo.Update(c.UserContext(), item.ID, values); err != nil {
				return err
			}
			updated++
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_class_batch_patch", "count="+strconv.Itoa(updated), 0)
		return OK(c, fiber.Map{"updated": updated})
	}
}

func ReplaceClassKeywords(repo *repository.ClassRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req classKeywordPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		oldKeyword := strings.TrimSpace(req.OldKeyword)
		if oldKeyword == "" {
			return fiber.NewError(fiber.StatusBadRequest, "oldKeyword is required")
		}
		if err := validateClassScope(req.Scope, req.ScopeID); err != nil {
			return err
		}
		affected, err := repo.ReplaceKeyword(c.UserContext(), normalizeClassScope(req.Scope), req.ScopeID, oldKeyword, strings.TrimSpace(req.NewKeyword))
		if err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_class_keyword_replace", "count="+strconv.FormatInt(affected, 10)+" scope="+normalizeClassScope(req.Scope), 0)
		return OK(c, fiber.Map{"affected": affected})
	}
}

func AddClassPrefix(repo *repository.ClassRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req classPrefixPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		prefix := strings.TrimSpace(req.Prefix)
		if prefix == "" {
			return fiber.NewError(fiber.StatusBadRequest, "prefix is required")
		}
		if err := validateClassScope(req.Scope, req.ScopeID); err != nil {
			return err
		}
		affected, err := repo.AddPrefix(c.UserContext(), normalizeClassScope(req.Scope), req.ScopeID, prefix)
		if err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_class_prefix", "count="+strconv.FormatInt(affected, 10)+" scope="+normalizeClassScope(req.Scope), 0)
		return OK(c, fiber.Map{"affected": affected})
	}
}

func PreviewClassDeduplicate(repo *repository.ClassRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req classDeduplicatePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateClassScope(req.Scope, req.ScopeID); err != nil {
			return err
		}
		groups, err := repo.DeduplicatePreview(c.UserContext(), normalizeClassScope(req.Scope), req.ScopeID, normalizeDeduplicateStrategy(req.Strategy), req.Limit)
		if err != nil {
			return err
		}
		total := 0
		for _, group := range groups {
			total += len(group.DeleteIDs)
		}
		return OK(c, fiber.Map{"groups": groups, "deleteCount": total})
	}
}

func ApplyClassDeduplicate(repo *repository.ClassRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req classDeduplicatePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateClassScope(req.Scope, req.ScopeID); err != nil {
			return err
		}
		affected, err := repo.DeduplicateApply(c.UserContext(), normalizeClassScope(req.Scope), req.ScopeID, normalizeDeduplicateStrategy(req.Strategy), req.Limit)
		if err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_class_deduplicate", "count="+strconv.FormatInt(affected, 10)+" scope="+normalizeClassScope(req.Scope), 0)
		return OK(c, fiber.Map{"affected": affected})
	}
}

func classBatchPatchValues(item classBatchPatchItem) (map[string]any, error) {
	if item.ID == 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "id is required")
	}
	values := map[string]any{}
	if item.Name != nil {
		name := strings.TrimSpace(*item.Name)
		if name == "" {
			return nil, fiber.NewError(fiber.StatusBadRequest, "name is required")
		}
		values["name"] = name
	}
	putOptionalString(values, "query_param", item.QueryParam)
	putOptionalString(values, "docking_code", item.DockingCode)
	putOptionalString(values, "query_platform", item.QueryPlatform)
	putOptionalString(values, "docking_platform", item.DockingPlatform)
	putOptionalString(values, "description", item.Description)
	putOptionalString(values, "category", item.Category)
	if item.Price != nil {
		if *item.Price < 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "price must be zero or greater")
		}
		values["price"] = *item.Price
	}
	if item.Sort != nil {
		values["sort"] = *item.Sort
	}
	if item.Status != nil {
		status := strings.TrimSpace(*item.Status)
		if !oneOf(status, "online", "offline") {
			return nil, fiber.NewError(fiber.StatusBadRequest, "status must be online or offline")
		}
		values["status"] = status
	}
	if item.PriceOperator != nil {
		operator := strings.TrimSpace(*item.PriceOperator)
		if !oneOf(operator, "*", "+") {
			return nil, fiber.NewError(fiber.StatusBadRequest, "priceOperator must be * or +")
		}
		values["price_operator"] = operator
	}
	if item.BridgeEnabled != nil {
		values["bridge_enabled"] = *item.BridgeEnabled
	}
	return values, nil
}

func putOptionalString(values map[string]any, key string, value *string) {
	if value != nil {
		values[key] = strings.TrimSpace(*value)
	}
}

func validateClassIDs(ids []uint) error {
	if len(ids) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "ids are required")
	}
	for _, id := range ids {
		if id == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "ids are invalid")
		}
	}
	if len(ids) > 500 {
		return fiber.NewError(fiber.StatusBadRequest, "too many ids")
	}
	return nil
}

func validateClassScope(scope string, scopeID string) error {
	scope = normalizeClassScope(scope)
	if (scope == "category" || scope == "docking") && strings.TrimSpace(scopeID) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "scopeId is required")
	}
	if !oneOf(scope, "all", "category", "docking") {
		return fiber.NewError(fiber.StatusBadRequest, "scope is invalid")
	}
	return nil
}

func normalizeClassScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "all"
	}
	return scope
}

func normalizeDeduplicateStrategy(strategy string) string {
	strategy = strings.TrimSpace(strategy)
	if strategy == "keep_newer" {
		return "keep_newer"
	}
	return "keep_older"
}

package handlers

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	connectoradapter "dw0rdwk/backend/internal/connectors"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type connectorPayload struct {
	Name      string `json:"name"`
	BaseURL   string `json:"baseUrl"`
	AppKey    string `json:"appKey"`
	AppSecret string `json:"appSecret"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	TimeoutMS *int   `json:"timeoutMs"`
	SortOrder *int   `json:"sortOrder"`

	OrderSyncEnabled       *bool    `json:"orderSyncEnabled"`
	SourceSyncEnabled      *bool    `json:"sourceSyncEnabled"`
	PriceMode              string   `json:"priceMode"`
	PriceValue             *float64 `json:"priceValue"`
	PriceRounding          string   `json:"priceRounding"`
	ReplaceRulesJSON       string   `json:"replaceRulesJson"`
	CategoryPriceRulesJSON string   `json:"categoryPriceRulesJson"`
}

func Connectors(repo *repository.ConnectorRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := repo.List(c.UserContext(), c.Query("q"), c.Query("status"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func CreateConnector(repo *repository.ConnectorRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req connectorPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateConnectorPayload(req); err != nil {
			return err
		}
		connector := connectorFromPayload(req)
		if err := repo.Create(c.UserContext(), &connector); err != nil {
			return err
		}
		auditLog(c, logs, "admin_connector_create", auditText("connector", connector.ID, "name="+connector.Name+" kind="+connector.Kind+" status="+connector.Status), 0)
		return OK(c, connector)
	}
}

func UpdateConnector(repo *repository.ConnectorRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req connectorPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateConnectorPayload(req); err != nil {
			return err
		}
		values := connectorUpdateValues(req)
		if err := repo.Update(c.UserContext(), id, values); err != nil {
			return err
		}
		auditLog(c, logs, "admin_connector_update", auditText("connector", id, auditFields(values)), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func DeleteConnector(repo *repository.ConnectorRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
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
		auditLog(c, logs, "admin_connector_delete", auditText("connector", id, cascadeDeleteAudit(result)), 0)
		return OK(c, result)
	}
}

func Logs(repo *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := repo.List(c.UserContext(), c.Query("q"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func Settings(settings *service.SettingService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := settings.All(c.UserContext())
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func SaveSettings(settings *service.SettingService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		values := map[string]string{}
		if err := c.BodyParser(&values); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := settings.Upsert(c.UserContext(), values); err != nil {
			return err
		}
		auditLog(c, logs, "admin_settings_update", "settings count="+strconv.Itoa(len(values)), 0)
		return OK(c, fiber.Map{"ok": true})
	}
}

func connectorFromPayload(req connectorPayload) models.Connector {
	replaceRules, _ := normalizeConnectorReplaceRulesJSON(req.ReplaceRulesJSON)
	categoryRules, _ := normalizeConnectorCategoryPriceRulesJSON(req.CategoryPriceRulesJSON)
	return models.Connector{
		Name:                   strings.TrimSpace(req.Name),
		BaseURL:                strings.TrimSpace(req.BaseURL),
		AppKey:                 strings.TrimSpace(req.AppKey),
		AppSecret:              strings.TrimSpace(req.AppSecret),
		Kind:                   normalizeConnectorKind(req.Kind),
		Status:                 normalizeConnectorStatus(req.Status),
		TimeoutMS:              normalizeConnectorTimeout(req.TimeoutMS),
		SortOrder:              intValue(req.SortOrder, 10),
		OrderSyncEnabled:       boolValue(req.OrderSyncEnabled, true),
		SourceSyncEnabled:      boolValue(req.SourceSyncEnabled, true),
		PriceMode:              normalizeConnectorPriceMode(req.PriceMode),
		PriceValue:             normalizeConnectorPriceValue(req.PriceValue),
		PriceRounding:          normalizeConnectorPriceRounding(req.PriceRounding),
		ReplaceRulesJSON:       replaceRules,
		CategoryPriceRulesJSON: categoryRules,
	}
}

func connectorUpdateValues(req connectorPayload) map[string]any {
	replaceRules, _ := normalizeConnectorReplaceRulesJSON(req.ReplaceRulesJSON)
	categoryRules, _ := normalizeConnectorCategoryPriceRulesJSON(req.CategoryPriceRulesJSON)
	values := map[string]any{
		"name":                      strings.TrimSpace(req.Name),
		"base_url":                  strings.TrimSpace(req.BaseURL),
		"app_key":                   strings.TrimSpace(req.AppKey),
		"kind":                      normalizeConnectorKind(req.Kind),
		"status":                    normalizeConnectorStatus(req.Status),
		"timeout_ms":                normalizeConnectorTimeout(req.TimeoutMS),
		"sort_order":                intValue(req.SortOrder, 10),
		"order_sync_enabled":        boolValue(req.OrderSyncEnabled, true),
		"source_sync_enabled":       boolValue(req.SourceSyncEnabled, true),
		"price_mode":                normalizeConnectorPriceMode(req.PriceMode),
		"price_value":               normalizeConnectorPriceValue(req.PriceValue),
		"price_rounding":            normalizeConnectorPriceRounding(req.PriceRounding),
		"replace_rules_json":        replaceRules,
		"category_price_rules_json": categoryRules,
	}
	putString(values, "app_secret", req.AppSecret)
	return values
}

func normalizeConnectorKind(kind string) string {
	return connectoradapter.NormalizeKind(kind)
}

func normalizeConnectorStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "active"
	}
	return status
}

func validateConnectorPayload(req connectorPayload) error {
	if strings.TrimSpace(req.Name) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if status := strings.TrimSpace(req.Status); status != "" && !oneOf(status, "active", "disabled") {
		return fiber.NewError(fiber.StatusBadRequest, "status must be active or disabled")
	}
	baseURL := strings.TrimSpace(req.BaseURL)
	if connectorNeedsBaseURL(req) && baseURL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "baseUrl is required for active connectors")
	}
	if baseURL != "" && !isHTTPURL(baseURL) {
		return fiber.NewError(fiber.StatusBadRequest, "baseUrl must be a valid http or https URL")
	}
	if req.TimeoutMS != nil && *req.TimeoutMS < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "timeoutMs must not be negative")
	}
	if req.TimeoutMS != nil && *req.TimeoutMS > 60000 {
		return fiber.NewError(fiber.StatusBadRequest, "timeoutMs must be less than or equal to 60000")
	}
	if req.SortOrder != nil && *req.SortOrder < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "sortOrder must not be negative")
	}
	if !oneOf(normalizeConnectorPriceMode(req.PriceMode), "multiplier", "fixed_add") {
		return fiber.NewError(fiber.StatusBadRequest, "priceMode must be multiplier or fixed_add")
	}
	priceValue := normalizeConnectorPriceValue(req.PriceValue)
	if priceValue < 0 || priceValue > 100000 {
		return fiber.NewError(fiber.StatusBadRequest, "priceValue must be between 0 and 100000")
	}
	if !oneOf(normalizeConnectorPriceRounding(req.PriceRounding), "none", "floor", "ceil", "round") {
		return fiber.NewError(fiber.StatusBadRequest, "priceRounding must be none, floor, ceil, or round")
	}
	if _, err := normalizeConnectorReplaceRulesJSON(req.ReplaceRulesJSON); err != nil {
		return err
	}
	if _, err := normalizeConnectorCategoryPriceRulesJSON(req.CategoryPriceRulesJSON); err != nil {
		return err
	}
	return nil
}

func normalizeConnectorTimeout(timeoutMS *int) int {
	if timeoutMS == nil || *timeoutMS == 0 {
		return 8000
	}
	return *timeoutMS
}

func isHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func connectorNeedsBaseURL(req connectorPayload) bool {
	status := strings.TrimSpace(req.Status)
	return status == "" || status == "active"
}

func normalizeConnectorPriceMode(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "multiplier"
	}
	return value
}

func normalizeConnectorPriceValue(value *float64) float64 {
	if value == nil {
		return 1
	}
	return *value
}

func normalizeConnectorPriceRounding(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "none"
	}
	return value
}

type connectorReplaceRule struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type connectorCategoryPriceRule struct {
	PriceMode     string   `json:"priceMode"`
	PriceValue    *float64 `json:"priceValue"`
	PriceRounding string   `json:"priceRounding"`
}

func normalizeConnectorReplaceRulesJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "[]", nil
	}
	var rules []connectorReplaceRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return "", fiber.NewError(fiber.StatusBadRequest, "replaceRulesJson must be a valid array")
	}
	normalized := make([]connectorReplaceRule, 0, len(rules))
	for _, rule := range rules {
		from := strings.TrimSpace(rule.From)
		if from == "" {
			continue
		}
		normalized = append(normalized, connectorReplaceRule{From: from, To: rule.To})
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func normalizeConnectorCategoryPriceRulesJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}", nil
	}
	var rules map[string]connectorCategoryPriceRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return "", fiber.NewError(fiber.StatusBadRequest, "categoryPriceRulesJson must be a valid object")
	}
	normalized := map[string]connectorCategoryPriceRule{}
	for key, rule := range rules {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		priceMode := normalizeConnectorPriceMode(rule.PriceMode)
		if !oneOf(priceMode, "multiplier", "fixed_add") {
			return "", fiber.NewError(fiber.StatusBadRequest, "category priceMode must be multiplier or fixed_add")
		}
		priceRounding := normalizeConnectorPriceRounding(rule.PriceRounding)
		if !oneOf(priceRounding, "none", "floor", "ceil", "round") {
			return "", fiber.NewError(fiber.StatusBadRequest, "category priceRounding must be none, floor, ceil, or round")
		}
		priceValue := normalizeConnectorPriceValue(rule.PriceValue)
		if priceValue < 0 || priceValue > 100000 {
			return "", fiber.NewError(fiber.StatusBadRequest, "category priceValue must be between 0 and 100000")
		}
		rule.PriceMode = priceMode
		rule.PriceValue = &priceValue
		rule.PriceRounding = priceRounding
		normalized[key] = rule
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

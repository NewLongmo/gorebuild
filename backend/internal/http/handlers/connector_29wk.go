package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	connectoradapter "dw0rdwk/backend/internal/connectors"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type wk29PullPayload struct {
	ConnectorID  uint    `json:"connectorId"`
	PriceRate    float64 `json:"priceRate"`
	SkipExisting bool    `json:"skipExisting"`
}

type wk29SyncPayload struct {
	ConnectorID       uint                    `json:"connectorId"`
	PriceRate         float64                 `json:"priceRate"`
	SyncCategories    *bool                   `json:"syncCategories"`
	UpdateName        bool                    `json:"updateName"`
	UpdateDescription bool                    `json:"updateDescription"`
	SkipExisting      bool                    `json:"skipExisting"`
	Categories        []wk29SyncCategoryInput `json:"categories"`
}

type wk29SyncCategoryInput struct {
	UpstreamID    string                 `json:"upstreamId"`
	Enabled       bool                   `json:"enabled"`
	Strategy      string                 `json:"strategy"`
	LocalCategory string                 `json:"localCategory"`
	Products      []wk29SyncProductInput `json:"products"`
}

type wk29SyncProductInput struct {
	UpstreamID string `json:"upstreamId"`
	Sync       bool   `json:"sync"`
}

type wk29PullResponse struct {
	ConnectorID     uint                  `json:"connectorId"`
	PriceRate       float64               `json:"priceRate"`
	TotalCategories int                   `json:"totalCategories"`
	TotalProducts   int                   `json:"totalProducts"`
	LocalCategories []string              `json:"localCategories"`
	Categories      []wk29CategoryPreview `json:"categories"`
}

type wk29CategoryPreview struct {
	UpstreamID    string               `json:"upstreamId"`
	UpstreamName  string               `json:"upstreamName"`
	Enabled       bool                 `json:"enabled"`
	Strategy      string               `json:"strategy"`
	LocalCategory string               `json:"localCategory"`
	ProductCount  int                  `json:"productCount"`
	Products      []wk29ProductPreview `json:"products"`
}

type wk29ProductPreview struct {
	UpstreamID      string  `json:"upstreamId"`
	KcID            string  `json:"kcId"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Status          string  `json:"status"`
	SourcePrice     float64 `json:"sourcePrice"`
	GeneratedPrice  float64 `json:"generatedPrice"`
	Existing        bool    `json:"existing"`
	ExistingClassID uint    `json:"existingClassId"`
	Sync            bool    `json:"sync"`
}

type wk29SyncResponse struct {
	TotalCategories    int `json:"totalCategories"`
	TotalProducts      int `json:"totalProducts"`
	Inserted           int `json:"inserted"`
	Updated            int `json:"updated"`
	Skipped            int `json:"skipped"`
	CategoriesInserted int `json:"categoriesInserted"`
	CategoriesUpdated  int `json:"categoriesUpdated"`
}

type connectorBalanceResponse struct {
	ConnectorID uint           `json:"connectorId"`
	Balance     float64        `json:"balance"`
	Raw         map[string]any `json:"raw"`
}

type wk29PriceSyncPayload struct {
	ConnectorID        uint     `json:"connectorId"`
	UpstreamCategoryID string   `json:"upstreamCategoryId"`
	PriceMode          string   `json:"priceMode"`
	PriceValue         *float64 `json:"priceValue"`
	PriceRounding      string   `json:"priceRounding"`
	OfflineMissing     bool     `json:"offlineMissing"`
}

type wk29PriceSyncResponse struct {
	ConnectorID uint  `json:"connectorId"`
	Total       int   `json:"total"`
	Updated     int   `json:"updated"`
	Missing     int   `json:"missing"`
	Offlined    int64 `json:"offlined"`
}

type wk29OrderSyncPayload struct {
	ConnectorID uint `json:"connectorId"`
	MaxPages    int  `json:"maxPages"`
}

func ConnectorBalance(connectors *repository.ConnectorRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		connector, err := active29WKConnector(c.UserContext(), connectors, id)
		if err != nil {
			return err
		}
		balance, raw, err := connectoradapter.Query29WKBalance(c.UserContext(), connector)
		if err != nil {
			return err
		}
		return OK(c, connectorBalanceResponse{ConnectorID: connector.ID, Balance: balance, Raw: raw})
	}
}

func Sync29WKOrders(syncer *service.ConnectorSyncService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req wk29OrderSyncPayload
		if len(c.Body()) > 0 {
			if err := c.BodyParser(&req); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
			}
		}
		result, err := syncer.Sync29WKOrders(c.UserContext(), service.WK29OrderSyncInput{
			ConnectorID: req.ConnectorID,
			MaxPages:    req.MaxPages,
		})
		if err != nil {
			return err
		}
		auditLog(c, logs, "admin_29wk_order_sync", service.WK29OrderSyncSummary(result), 0)
		return OK(c, result)
	}
}

func Sync29WKPrices(syncer *service.ConnectorSyncService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req wk29PriceSyncPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		result, err := syncer.Sync29WKPrices(c.UserContext(), service.WK29PriceSyncInput{
			ConnectorID:        req.ConnectorID,
			UpstreamCategoryID: req.UpstreamCategoryID,
			PriceMode:          req.PriceMode,
			PriceValue:         req.PriceValue,
			PriceRounding:      req.PriceRounding,
			OfflineMissing:     req.OfflineMissing,
		})
		if err != nil {
			return err
		}
		auditLog(c, logs, "admin_29wk_price_sync", fmt.Sprintf("connector=%d total=%d updated=%d missing=%d offlined=%d", result.ConnectorID, result.Total, result.Updated, result.Missing, result.Offlined), 0)
		return OK(c, result)
	}
}

func Pull29WKClasses(connectors *repository.ConnectorRepository, categories *repository.CategoryRepository, classes *repository.ClassRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req wk29PullPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateWK29PriceRate(req.PriceRate); err != nil {
			return err
		}
		connector, err := active29WKConnector(c.UserContext(), connectors, req.ConnectorID)
		if err != nil {
			return err
		}
		pricing := wk29PricingFromConnector(connector, req.PriceRate)
		upstream, err := connectoradapter.Fetch29WKClasses(c.UserContext(), connector)
		if err != nil {
			return err
		}
		localCategories, localCategorySet, err := localCategoryNames(c.UserContext(), categories)
		if err != nil {
			return err
		}
		existing, err := existingWK29Classes(c.UserContext(), classes, connector.ID, upstream)
		if err != nil {
			return err
		}
		return OK(c, buildWK29Preview(connector.ID, pricing, req.SkipExisting, upstream, existing, localCategories, localCategorySet))
	}
}

func Sync29WKClasses(
	connectors *repository.ConnectorRepository,
	categories *repository.CategoryRepository,
	classes *repository.ClassRepository,
	dashboard *service.DashboardService,
	logs *repository.LogRepository,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req wk29SyncPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if len(req.Categories) == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "categories are required")
		}
		if err := validateWK29PriceRate(req.PriceRate); err != nil {
			return err
		}
		connector, err := active29WKConnector(c.UserContext(), connectors, req.ConnectorID)
		if err != nil {
			return err
		}
		if !connector.SourceSyncEnabled {
			return fiber.NewError(fiber.StatusBadRequest, "connector source sync is disabled")
		}
		pricing := wk29PricingFromConnector(connector, req.PriceRate)
		upstream, err := connectoradapter.Fetch29WKClasses(c.UserContext(), connector)
		if err != nil {
			return err
		}
		existing, err := existingWK29Classes(c.UserContext(), classes, connector.ID, upstream)
		if err != nil {
			return err
		}
		result, err := syncWK29Classes(c.UserContext(), req, pricing, connector, upstream, existing, categories, classes)
		if err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_29wk_sync", fmt.Sprintf("connector=%d inserted=%d updated=%d skipped=%d", connector.ID, result.Inserted, result.Updated, result.Skipped), 0)
		return OK(c, result)
	}
}

func active29WKConnector(ctx context.Context, repo *repository.ConnectorRepository, id uint) (models.Connector, error) {
	if id == 0 {
		return models.Connector{}, fiber.NewError(fiber.StatusBadRequest, "connectorId is required")
	}
	connector, err := repo.Find(ctx, id)
	if err != nil {
		return models.Connector{}, err
	}
	if !connectoradapter.Is29WKKind(connector.Kind) {
		return models.Connector{}, fiber.NewError(fiber.StatusBadRequest, "connector kind must be 29wk")
	}
	if connector.Status != "active" {
		return models.Connector{}, fiber.NewError(fiber.StatusBadRequest, "connector is disabled")
	}
	if strings.TrimSpace(connector.BaseURL) == "" {
		return models.Connector{}, fiber.NewError(fiber.StatusBadRequest, "connector baseUrl is required")
	}
	return connector, nil
}

func validateWK29PriceRate(value float64) error {
	if value == 0 {
		return nil
	}
	if value < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "priceRate must be zero or greater")
	}
	if value > 1000 {
		return fiber.NewError(fiber.StatusBadRequest, "priceRate must be less than or equal to 1000")
	}
	return nil
}

func localCategoryNames(ctx context.Context, repo *repository.CategoryRepository) ([]string, map[string]struct{}, error) {
	items, err := repo.ListActive(ctx, 500)
	if err != nil {
		return nil, nil, err
	}
	names := make([]string, 0, len(items))
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
		set[name] = struct{}{}
	}
	return names, set, nil
}

func existingWK29Classes(ctx context.Context, repo *repository.ClassRepository, connectorID uint, upstream []connectoradapter.WK29Class) (map[string]models.CourseClass, error) {
	codes := make([]string, 0, len(upstream))
	for _, item := range upstream {
		codes = append(codes, item.UpstreamID)
	}
	return repo.ListByDockingCodes(ctx, strconv.FormatUint(uint64(connectorID), 10), codes)
}

func buildWK29Preview(
	connectorID uint,
	pricing wk29PricingConfig,
	skipExisting bool,
	upstream []connectoradapter.WK29Class,
	existing map[string]models.CourseClass,
	localCategories []string,
	localCategorySet map[string]struct{},
) wk29PullResponse {
	response := wk29PullResponse{
		ConnectorID:     connectorID,
		PriceRate:       pricing.PriceValue,
		LocalCategories: localCategories,
	}
	categoryIndex := map[string]int{}
	for _, item := range upstream {
		categoryID := defaultString(item.CategoryID, "0")
		categoryName := applyWK29Replacements(defaultString(item.CategoryName, "未分类"), pricing.ReplaceRules)
		index, ok := categoryIndex[categoryID]
		if !ok {
			_, hasLocal := localCategorySet[categoryName]
			strategy := "create_new"
			localCategory := ""
			if hasLocal {
				strategy = "bind_existing"
				localCategory = categoryName
			}
			response.Categories = append(response.Categories, wk29CategoryPreview{
				UpstreamID:    categoryID,
				UpstreamName:  categoryName,
				Enabled:       hasLocal,
				Strategy:      strategy,
				LocalCategory: localCategory,
			})
			index = len(response.Categories) - 1
			categoryIndex[categoryID] = index
		}
		localClass, exists := existing[item.UpstreamID]
		product := wk29ProductPreview{
			UpstreamID:      item.UpstreamID,
			KcID:            item.KcID,
			Name:            applyWK29Replacements(item.Name, pricing.ReplaceRules),
			Description:     applyWK29Replacements(item.Description, pricing.ReplaceRules),
			Status:          item.Status,
			SourcePrice:     item.Price,
			GeneratedPrice:  wk29GeneratedPrice(item.Price, categoryID, pricing),
			Existing:        exists,
			ExistingClassID: localClass.ID,
			Sync:            response.Categories[index].Enabled && !(skipExisting && exists),
		}
		response.Categories[index].Products = append(response.Categories[index].Products, product)
		response.Categories[index].ProductCount++
		response.TotalProducts++
	}
	response.TotalCategories = len(response.Categories)
	return response
}

func syncWK29Classes(
	ctx context.Context,
	req wk29SyncPayload,
	pricing wk29PricingConfig,
	connector models.Connector,
	upstream []connectoradapter.WK29Class,
	existing map[string]models.CourseClass,
	categories *repository.CategoryRepository,
	classes *repository.ClassRepository,
) (wk29SyncResponse, error) {
	result := wk29SyncResponse{TotalProducts: len(upstream)}
	categorySelections := map[string]wk29SyncCategoryInput{}
	for _, category := range req.Categories {
		category.UpstreamID = strings.TrimSpace(category.UpstreamID)
		if category.UpstreamID == "" {
			continue
		}
		categorySelections[category.UpstreamID] = category
	}
	productSelections := map[string]bool{}
	for _, category := range req.Categories {
		for _, product := range category.Products {
			productSelections[strings.TrimSpace(product.UpstreamID)] = product.Sync
		}
	}
	ensuredCategories := map[string]struct{}{}
	connectorID := strconv.FormatUint(uint64(connector.ID), 10)
	seenCategories := map[string]struct{}{}
	for _, item := range upstream {
		categoryID := defaultString(item.CategoryID, "0")
		categoryName := applyWK29Replacements(defaultString(item.CategoryName, "未分类"), pricing.ReplaceRules)
		selection, ok := categorySelections[categoryID]
		if !ok || !selection.Enabled || !productSelections[item.UpstreamID] {
			result.Skipped++
			continue
		}
		if _, ok := seenCategories[categoryID]; !ok {
			result.TotalCategories++
			seenCategories[categoryID] = struct{}{}
		}
		localCategory, err := resolveWK29CategoryName(selection, categoryID, categoryName)
		if err != nil {
			return result, err
		}
		if syncCategoriesEnabled(req.SyncCategories) {
			created, err := ensureWK29Category(ctx, categories, localCategory, ensuredCategories)
			if err != nil {
				return result, err
			}
			if created {
				result.CategoriesInserted++
			}
		}
		price := wk29GeneratedPrice(item.Price, categoryID, pricing)
		name := truncateRunes(applyWK29Replacements(item.Name, pricing.ReplaceRules), 160)
		description := truncateRunes(applyWK29Replacements(item.Description, pricing.ReplaceRules), 500)
		if localClass, ok := existing[item.UpstreamID]; ok {
			if req.SkipExisting {
				result.Skipped++
				continue
			}
			values := map[string]any{
				"sort":             item.Sort,
				"price":            price,
				"query_param":      item.UpstreamID,
				"docking_code":     item.UpstreamID,
				"query_platform":   connectorID,
				"docking_platform": connectorID,
				"price_operator":   "*",
				"status":           item.Status,
				"category":         localCategory,
				"bridge_enabled":   item.Status == "online",
			}
			if req.UpdateName {
				values["name"] = name
			}
			if req.UpdateDescription {
				values["description"] = description
			}
			if err := classes.Update(ctx, localClass.ID, values); err != nil {
				return result, err
			}
			result.Updated++
			continue
		}
		class := models.CourseClass{
			Sort:            item.Sort,
			Name:            name,
			QueryParam:      item.UpstreamID,
			DockingCode:     item.UpstreamID,
			Price:           price,
			QueryPlatform:   connectorID,
			DockingPlatform: connectorID,
			PriceOperator:   "*",
			Description:     description,
			Status:          item.Status,
			Category:        truncateRunes(localCategory, 64),
			BridgeEnabled:   item.Status == "online",
		}
		if err := classes.Create(ctx, &class); err != nil {
			return result, err
		}
		existing[item.UpstreamID] = class
		result.Inserted++
	}
	return result, nil
}

func resolveWK29CategoryName(selection wk29SyncCategoryInput, upstreamID string, upstreamName string) (string, error) {
	strategy := strings.TrimSpace(selection.Strategy)
	switch strategy {
	case "", "bind_existing":
		name := strings.TrimSpace(selection.LocalCategory)
		if name == "" {
			return "", fiber.NewError(fiber.StatusBadRequest, "localCategory is required for bind_existing")
		}
		return truncateRunes(name, 64), nil
	case "create_new":
		return truncateRunes(defaultString(upstreamName, "未分类"), 64), nil
	case "independent":
		return truncateRunes(fmt.Sprintf("%s (ID:%s)", defaultString(upstreamName, "未分类"), upstreamID), 64), nil
	default:
		return "", fiber.NewError(fiber.StatusBadRequest, "invalid category strategy")
	}
}

func ensureWK29Category(ctx context.Context, repo *repository.CategoryRepository, name string, ensured map[string]struct{}) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	if _, ok := ensured[name]; ok {
		return false, nil
	}
	ensured[name] = struct{}{}
	_, err := repo.FindByName(ctx, name)
	if err == nil {
		return false, nil
	}
	if !repository.IsNotFound(err) {
		return false, err
	}
	item := models.CourseCategory{Name: name, Status: "active"}
	if err := repo.Create(ctx, &item); err != nil {
		return false, err
	}
	return true, nil
}

func syncCategoriesEnabled(value *bool) bool {
	return value == nil || *value
}

type wk29PricingConfig struct {
	PriceMode     string
	PriceValue    float64
	PriceRounding string
	ReplaceRules  []connectorReplaceRule
	CategoryRules map[string]connectorCategoryPriceRule
}

func wk29PricingFromConnector(connector models.Connector, requestPriceRate float64) wk29PricingConfig {
	priceMode := normalizeConnectorPriceMode(connector.PriceMode)
	priceValue := connector.PriceValue
	if priceValue == 0 {
		priceValue = 1
	}
	priceRounding := normalizeConnectorPriceRounding(connector.PriceRounding)
	if requestPriceRate > 0 {
		priceMode = "multiplier"
		priceValue = requestPriceRate
		priceRounding = "none"
	}
	return wk29PricingConfig{
		PriceMode:     priceMode,
		PriceValue:    priceValue,
		PriceRounding: priceRounding,
		ReplaceRules:  parseWK29ReplaceRules(connector.ReplaceRulesJSON),
		CategoryRules: parseWK29CategoryPriceRules(connector.CategoryPriceRulesJSON),
	}
}

func wk29PricingFromPriceSyncPayload(connector models.Connector, req wk29PriceSyncPayload) (wk29PricingConfig, error) {
	pricing := wk29PricingFromConnector(connector, 0)
	if strings.TrimSpace(req.PriceMode) != "" {
		priceMode := normalizeConnectorPriceMode(req.PriceMode)
		if !oneOf(priceMode, "multiplier", "fixed_add") {
			return pricing, fiber.NewError(fiber.StatusBadRequest, "priceMode is invalid")
		}
		pricing.PriceMode = priceMode
	}
	if req.PriceValue != nil {
		priceValue := normalizeConnectorPriceValue(req.PriceValue)
		if priceValue < 0 {
			return pricing, fiber.NewError(fiber.StatusBadRequest, "priceValue must be zero or greater")
		}
		pricing.PriceValue = priceValue
	}
	if strings.TrimSpace(req.PriceRounding) != "" {
		priceRounding := normalizeConnectorPriceRounding(req.PriceRounding)
		if !oneOf(priceRounding, "none", "floor", "ceil", "round") {
			return pricing, fiber.NewError(fiber.StatusBadRequest, "priceRounding is invalid")
		}
		pricing.PriceRounding = priceRounding
	}
	return pricing, nil
}

func parseWK29ReplaceRules(raw string) []connectorReplaceRule {
	normalized, err := normalizeConnectorReplaceRulesJSON(raw)
	if err != nil {
		return nil
	}
	var rules []connectorReplaceRule
	if err := json.Unmarshal([]byte(normalized), &rules); err != nil {
		return nil
	}
	return rules
}

func parseWK29CategoryPriceRules(raw string) map[string]connectorCategoryPriceRule {
	normalized, err := normalizeConnectorCategoryPriceRulesJSON(raw)
	if err != nil {
		return map[string]connectorCategoryPriceRule{}
	}
	var rules map[string]connectorCategoryPriceRule
	if err := json.Unmarshal([]byte(normalized), &rules); err != nil {
		return map[string]connectorCategoryPriceRule{}
	}
	return rules
}

func wk29GeneratedPrice(sourcePrice float64, categoryID string, pricing wk29PricingConfig) float64 {
	priceMode := pricing.PriceMode
	priceValue := pricing.PriceValue
	priceRounding := pricing.PriceRounding
	if rule, ok := pricing.CategoryRules[strings.TrimSpace(categoryID)]; ok {
		priceMode = normalizeConnectorPriceMode(rule.PriceMode)
		priceValue = normalizeConnectorPriceValue(rule.PriceValue)
		priceRounding = normalizeConnectorPriceRounding(rule.PriceRounding)
	}

	var price float64
	switch priceMode {
	case "fixed_add":
		price = sourcePrice + priceValue
	default:
		price = sourcePrice * priceValue
	}
	switch priceRounding {
	case "floor":
		price = math.Floor(price)
	case "ceil":
		price = math.Ceil(price)
	case "round":
		price = math.Round(price)
	}
	if price < 0 {
		price = 0
	}
	return connectoradapter.RoundPrice(price)
}

func applyWK29Replacements(value string, rules []connectorReplaceRule) string {
	result := value
	for _, rule := range rules {
		if strings.TrimSpace(rule.From) == "" {
			continue
		}
		result = strings.ReplaceAll(result, rule.From, rule.To)
	}
	return result
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func truncateRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	connectoradapter "dw0rdwk/backend/internal/connectors"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/repository"
)

const defaultWK29CronMaxPages = 20
const wk29BatchPageSize = 500

type ConnectorSyncService struct {
	connectors *repository.ConnectorRepository
	classes    *repository.ClassRepository
	orders     *repository.OrderRepository
	events     *repository.OrderEventRepository
	dashboard  *DashboardService
	logs       *repository.LogRepository
	jobs       *repository.SystemJobRepository
}

type WK29OrderSyncInput struct {
	ConnectorID uint `json:"connectorId"`
	MaxPages    int  `json:"maxPages"`
}

type WK29OrderSyncResult struct {
	Connectors int                         `json:"connectors"`
	Fetched    int                         `json:"fetched"`
	Matched    int                         `json:"matched"`
	Updated    int                         `json:"updated"`
	Skipped    int                         `json:"skipped"`
	Failed     int                         `json:"failed"`
	Items      []WK29OrderSyncConnectorRow `json:"items"`
}

type WK29OrderSyncConnectorRow struct {
	ConnectorID uint   `json:"connectorId"`
	Name        string `json:"name"`
	Fetched     int    `json:"fetched"`
	Matched     int    `json:"matched"`
	Updated     int    `json:"updated"`
	Skipped     int    `json:"skipped"`
	Failed      int    `json:"failed"`
	Error       string `json:"error,omitempty"`
}

type WK29PriceSyncInput struct {
	ConnectorID        uint     `json:"connectorId"`
	UpstreamCategoryID string   `json:"upstreamCategoryId"`
	PriceMode          string   `json:"priceMode"`
	PriceValue         *float64 `json:"priceValue"`
	PriceRounding      string   `json:"priceRounding"`
	OfflineMissing     bool     `json:"offlineMissing"`
}

type WK29PriceSyncResult struct {
	ConnectorID uint  `json:"connectorId"`
	Total       int   `json:"total"`
	Updated     int   `json:"updated"`
	Missing     int   `json:"missing"`
	Offlined    int64 `json:"offlined"`
}

type WK29PriceSyncAllResult struct {
	Connectors int                         `json:"connectors"`
	Total      int                         `json:"total"`
	Updated    int                         `json:"updated"`
	Missing    int                         `json:"missing"`
	Offlined   int64                       `json:"offlined"`
	Failed     int                         `json:"failed"`
	Items      []WK29PriceSyncConnectorRow `json:"items"`
}

type WK29PriceSyncConnectorRow struct {
	ConnectorID uint   `json:"connectorId"`
	Name        string `json:"name"`
	Total       int    `json:"total"`
	Updated     int    `json:"updated"`
	Missing     int    `json:"missing"`
	Offlined    int64  `json:"offlined"`
	Error       string `json:"error,omitempty"`
}

func (s *ConnectorSyncService) Sync29WKOrders(ctx context.Context, input WK29OrderSyncInput) (WK29OrderSyncResult, error) {
	maxPages := normalizeWK29MaxPages(input.MaxPages)
	if input.ConnectorID != 0 {
		connector, err := s.require29WKConnector(ctx, input.ConnectorID, "order")
		if err != nil {
			return WK29OrderSyncResult{}, err
		}
		row, err := s.sync29WKOrdersForConnector(ctx, connector, maxPages)
		result := aggregateWK29OrderSyncRows([]WK29OrderSyncConnectorRow{row})
		if err != nil {
			return result, err
		}
		return result, nil
	}

	connectors, err := s.active29WKConnectors(ctx, "order")
	if err != nil {
		return WK29OrderSyncResult{}, err
	}
	rows := make([]WK29OrderSyncConnectorRow, 0, len(connectors))
	for _, connector := range connectors {
		row, err := s.sync29WKOrdersForConnector(ctx, connector, maxPages)
		if err != nil {
			row.Error = err.Error()
		}
		rows = append(rows, row)
	}
	return aggregateWK29OrderSyncRows(rows), nil
}

func (s *ConnectorSyncService) Sync29WKPrices(ctx context.Context, input WK29PriceSyncInput) (WK29PriceSyncResult, error) {
	connector, err := s.require29WKConnector(ctx, input.ConnectorID, "source")
	if err != nil {
		return WK29PriceSyncResult{}, err
	}
	pricing, err := wk29PricingFromPriceSyncInput(connector, input)
	if err != nil {
		return WK29PriceSyncResult{}, err
	}
	upstream, err := connectoradapter.Fetch29WKClasses(ctx, connector)
	if err != nil {
		return WK29PriceSyncResult{}, err
	}
	connectorID := strconv.FormatUint(uint64(connector.ID), 10)
	filterCategory := strings.TrimSpace(input.UpstreamCategoryID)
	result := WK29PriceSyncResult{ConnectorID: connector.ID}
	keepCodes := make([]string, 0, len(upstream))
	for _, item := range upstream {
		if filterCategory != "" && filterCategory != "all" && strings.TrimSpace(item.CategoryID) != filterCategory {
			continue
		}
		result.Total++
		keepCodes = append(keepCodes, item.UpstreamID)
		updated, err := s.classes.UpdateByDockingCode(ctx, connectorID, item.UpstreamID, map[string]any{
			"price":          wk29GeneratedPrice(item.Price, item.CategoryID, pricing),
			"status":         item.Status,
			"bridge_enabled": item.Status == "online",
			"description":    truncateRunes(applyWK29Replacements(item.Description, pricing.ReplaceRules), 500),
		})
		if err != nil {
			return result, err
		}
		if updated {
			result.Updated++
		} else {
			result.Missing++
		}
	}
	if input.OfflineMissing && (filterCategory == "" || filterCategory == "all") {
		offlined, err := s.classes.MarkMissingDockingOffline(ctx, connectorID, keepCodes)
		if err != nil {
			return result, err
		}
		result.Offlined = offlined
	}
	if result.Updated > 0 || result.Offlined > 0 {
		s.invalidateDashboard(ctx)
	}
	return result, nil
}

func (s *ConnectorSyncService) Sync29WKPricesAll(ctx context.Context) (WK29PriceSyncAllResult, error) {
	connectors, err := s.active29WKConnectors(ctx, "source")
	if err != nil {
		return WK29PriceSyncAllResult{}, err
	}
	rows := make([]WK29PriceSyncConnectorRow, 0, len(connectors))
	for _, connector := range connectors {
		result, err := s.Sync29WKPrices(ctx, WK29PriceSyncInput{ConnectorID: connector.ID})
		row := WK29PriceSyncConnectorRow{
			ConnectorID: connector.ID,
			Name:        connector.Name,
			Total:       result.Total,
			Updated:     result.Updated,
			Missing:     result.Missing,
			Offlined:    result.Offlined,
		}
		if err != nil {
			row.Error = err.Error()
		}
		rows = append(rows, row)
	}
	return aggregateWK29PriceSyncRows(rows), nil
}

func (s *ConnectorSyncService) LogSystem(ctx context.Context, typ string, text string) {
	if s == nil || s.logs == nil {
		return
	}
	_ = s.logs.Create(ctx, &models.OperationLog{
		Type:      strings.TrimSpace(typ),
		Text:      strings.TrimSpace(text),
		CreatedAt: time.Now(),
	})
}

func (s *ConnectorSyncService) sync29WKOrdersForConnector(ctx context.Context, connector models.Connector, maxPages int) (WK29OrderSyncConnectorRow, error) {
	row := WK29OrderSyncConnectorRow{ConnectorID: connector.ID, Name: connector.Name}
	seen := map[string]struct{}{}
	for page := 0; page < maxPages; page++ {
		statuses, err := connectoradapter.Fetch29WKOrderStatuses(ctx, connector)
		if err != nil {
			return row, err
		}
		row.Fetched += len(statuses)
		newRows := 0
		for _, status := range statuses {
			signature := wk29OrderStatusSignature(status)
			if _, ok := seen[signature]; ok {
				row.Skipped++
				continue
			}
			seen[signature] = struct{}{}
			newRows++
			if err := s.apply29WKOrderStatus(ctx, connector.ID, status, &row); err != nil {
				row.Failed++
			}
		}
		if len(statuses) < wk29BatchPageSize || newRows == 0 {
			break
		}
	}
	if row.Updated > 0 {
		s.invalidateDashboard(ctx)
	}
	return row, nil
}

func (s *ConnectorSyncService) apply29WKOrderStatus(ctx context.Context, connectorID uint, status connectoradapter.WK29OrderStatusRow, result *WK29OrderSyncConnectorRow) error {
	order, ok, err := s.orders.FindSyncCandidate(ctx, connectorID, status.RemoteOrderID, status.Account, status.CourseName, status.DockingCode)
	if err != nil {
		return err
	}
	if !ok {
		result.Skipped++
		return nil
	}
	result.Matched++
	normalizedStatus := strings.TrimSpace(status.Normalized)
	if normalizedStatus == "" {
		normalizedStatus = connectoradapter.Map29WKStatus(status.Status, status.Progress)
	}
	values := map[string]any{
		"status":         normalizedStatus,
		"docking_status": dockingStatusFromOrderStatus(normalizedStatus),
	}
	if remoteID := strings.TrimSpace(status.RemoteOrderID); remoteID != "" {
		values["remote_order_id"] = remoteID
	}
	if progress := strings.TrimSpace(status.Progress); progress != "" {
		values["progress"] = truncateRunes(progress, 160)
	}
	if remarks := strings.TrimSpace(status.Remarks); remarks != "" {
		values["remarks"] = truncateRunes(remarks, 500)
	}
	updated, err := s.orders.UpdateSyncCandidate(ctx, order.ID, values)
	if err != nil {
		return err
	}
	if updated {
		result.Updated++
		s.logOrderSyncEvent(ctx, order, normalizedStatus, values)
	} else {
		result.Skipped++
	}
	return nil
}

func (s *ConnectorSyncService) logOrderSyncEvent(ctx context.Context, order models.Order, status string, values map[string]any) {
	if s == nil || s.events == nil || order.ID == 0 {
		return
	}
	parts := []string{"上游进度已同步"}
	if status != "" {
		parts = append(parts, "状态="+status)
	}
	if remoteID, ok := values["remote_order_id"].(string); ok && strings.TrimSpace(remoteID) != "" {
		parts = append(parts, "远程订单="+strings.TrimSpace(remoteID))
	}
	progress, _ := values["progress"].(string)
	if strings.TrimSpace(progress) != "" {
		parts = append(parts, "进度="+strings.TrimSpace(progress))
	}
	remarks, _ := values["remarks"].(string)
	if strings.TrimSpace(remarks) != "" {
		parts = append(parts, "备注="+strings.TrimSpace(remarks))
	}
	_ = s.events.Create(ctx, &models.OrderEvent{
		OrderID:       order.ID,
		UserID:        order.UserID,
		Level:         "info",
		Source:        "29wk_sync",
		EventType:     "order_status_synced",
		Content:       truncateRunes(strings.Join(parts, "；"), 1000),
		Progress:      truncateRunes(progress, 160),
		VisibleToUser: true,
		CreatedAt:     time.Now(),
	})
}

func (s *ConnectorSyncService) JobEnabled(ctx context.Context, name string) bool {
	if s == nil || s.jobs == nil {
		return true
	}
	return s.jobs.IsEnabled(ctx, name)
}

func (s *ConnectorSyncService) MarkJobStarted(ctx context.Context, name string) time.Time {
	started := time.Now()
	if s == nil || s.jobs == nil {
		return started
	}
	if value, err := s.jobs.MarkStarted(ctx, name); err == nil {
		return value
	}
	return started
}

func (s *ConnectorSyncService) MarkJobFinished(ctx context.Context, name string, started time.Time, summary any, runErr error) {
	if s == nil || s.jobs == nil {
		return
	}
	_ = s.jobs.MarkFinished(ctx, name, started, summary, runErr)
}

func (s *ConnectorSyncService) require29WKConnector(ctx context.Context, id uint, purpose string) (models.Connector, error) {
	if id == 0 {
		return models.Connector{}, ValidationError{Message: "connectorId is required"}
	}
	connector, err := s.connectors.Find(ctx, id)
	if err != nil {
		return models.Connector{}, err
	}
	if !connectoradapter.Is29WKKind(connector.Kind) {
		return models.Connector{}, ValidationError{Message: "connector kind must be 29wk"}
	}
	if connector.Status != "active" {
		return models.Connector{}, ValidationError{Message: "connector is disabled"}
	}
	if strings.TrimSpace(connector.BaseURL) == "" {
		return models.Connector{}, ValidationError{Message: "connector baseUrl is required"}
	}
	if purpose == "order" && !connector.OrderSyncEnabled {
		return models.Connector{}, ValidationError{Message: "connector order sync is disabled"}
	}
	if purpose == "source" && !connector.SourceSyncEnabled {
		return models.Connector{}, ValidationError{Message: "connector source sync is disabled"}
	}
	return connector, nil
}

func (s *ConnectorSyncService) active29WKConnectors(ctx context.Context, purpose string) ([]models.Connector, error) {
	items, err := s.connectors.ListActiveWithBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]models.Connector, 0, len(items))
	for _, connector := range items {
		if !connectoradapter.Is29WKKind(connector.Kind) {
			continue
		}
		if purpose == "order" && !connector.OrderSyncEnabled {
			continue
		}
		if purpose == "source" && !connector.SourceSyncEnabled {
			continue
		}
		result = append(result, connector)
	}
	return result, nil
}

func (s *ConnectorSyncService) invalidateDashboard(ctx context.Context) {
	if s != nil && s.dashboard != nil {
		s.dashboard.Invalidate(ctx)
	}
}

func normalizeWK29MaxPages(value int) int {
	if value <= 0 {
		return defaultWK29CronMaxPages
	}
	if value > 100 {
		return 100
	}
	return value
}

func aggregateWK29OrderSyncRows(rows []WK29OrderSyncConnectorRow) WK29OrderSyncResult {
	result := WK29OrderSyncResult{Items: rows}
	for _, row := range rows {
		result.Connectors++
		result.Fetched += row.Fetched
		result.Matched += row.Matched
		result.Updated += row.Updated
		result.Skipped += row.Skipped
		result.Failed += row.Failed
		if row.Error != "" {
			result.Failed++
		}
	}
	return result
}

func aggregateWK29PriceSyncRows(rows []WK29PriceSyncConnectorRow) WK29PriceSyncAllResult {
	result := WK29PriceSyncAllResult{Items: rows}
	for _, row := range rows {
		result.Connectors++
		result.Total += row.Total
		result.Updated += row.Updated
		result.Missing += row.Missing
		result.Offlined += row.Offlined
		if row.Error != "" {
			result.Failed++
		}
	}
	return result
}

func wk29OrderStatusSignature(row connectoradapter.WK29OrderStatusRow) string {
	parts := []string{
		strings.TrimSpace(row.RemoteOrderID),
		strings.TrimSpace(row.Account),
		strings.TrimSpace(row.CourseName),
		strings.TrimSpace(row.DockingCode),
		strings.TrimSpace(row.Status),
		strings.TrimSpace(row.Progress),
	}
	return strings.Join(parts, "\x00")
}

func dockingStatusFromOrderStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "failed":
		return "failed"
	case "cancelled":
		return "cancelled"
	case "refunded":
		return "refunded"
	default:
		return "sent"
	}
}

type wk29PricingConfig struct {
	PriceMode     string
	PriceValue    float64
	PriceRounding string
	ReplaceRules  []connectorReplaceRule
	CategoryRules map[string]connectorCategoryPriceRule
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

func wk29PricingFromConnector(connector models.Connector) wk29PricingConfig {
	priceValue := connector.PriceValue
	if priceValue == 0 {
		priceValue = 1
	}
	return wk29PricingConfig{
		PriceMode:     normalizeConnectorPriceMode(connector.PriceMode),
		PriceValue:    priceValue,
		PriceRounding: normalizeConnectorPriceRounding(connector.PriceRounding),
		ReplaceRules:  parseWK29ReplaceRules(connector.ReplaceRulesJSON),
		CategoryRules: parseWK29CategoryPriceRules(connector.CategoryPriceRulesJSON),
	}
}

func wk29PricingFromPriceSyncInput(connector models.Connector, input WK29PriceSyncInput) (wk29PricingConfig, error) {
	pricing := wk29PricingFromConnector(connector)
	if strings.TrimSpace(input.PriceMode) != "" {
		priceMode := normalizeConnectorPriceMode(input.PriceMode)
		if !oneOf(priceMode, "multiplier", "fixed_add") {
			return pricing, ValidationError{Message: "priceMode is invalid"}
		}
		pricing.PriceMode = priceMode
	}
	if input.PriceValue != nil {
		priceValue := normalizeConnectorPriceValue(input.PriceValue)
		if priceValue < 0 {
			return pricing, ValidationError{Message: "priceValue must be zero or greater"}
		}
		pricing.PriceValue = priceValue
	}
	if strings.TrimSpace(input.PriceRounding) != "" {
		priceRounding := normalizeConnectorPriceRounding(input.PriceRounding)
		if !oneOf(priceRounding, "none", "floor", "ceil", "round") {
			return pricing, ValidationError{Message: "priceRounding is invalid"}
		}
		pricing.PriceRounding = priceRounding
	}
	return pricing, nil
}

func parseWK29ReplaceRules(raw string) []connectorReplaceRule {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var rules []connectorReplaceRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return nil
	}
	normalized := make([]connectorReplaceRule, 0, len(rules))
	for _, rule := range rules {
		if strings.TrimSpace(rule.From) == "" {
			continue
		}
		normalized = append(normalized, rule)
	}
	return normalized
}

func parseWK29CategoryPriceRules(raw string) map[string]connectorCategoryPriceRule {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]connectorCategoryPriceRule{}
	}
	var rules map[string]connectorCategoryPriceRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return map[string]connectorCategoryPriceRule{}
	}
	result := map[string]connectorCategoryPriceRule{}
	for key, rule := range rules {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		result[key] = rule
	}
	return result
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

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func truncateRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func WK29OrderSyncSummary(result WK29OrderSyncResult) string {
	return fmt.Sprintf("connectors=%d fetched=%d matched=%d updated=%d skipped=%d failed=%d", result.Connectors, result.Fetched, result.Matched, result.Updated, result.Skipped, result.Failed)
}

func WK29PriceSyncSummary(result WK29PriceSyncAllResult) string {
	return fmt.Sprintf("connectors=%d total=%d updated=%d missing=%d offlined=%d failed=%d", result.Connectors, result.Total, result.Updated, result.Missing, result.Offlined, result.Failed)
}

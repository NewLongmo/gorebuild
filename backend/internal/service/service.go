package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"dw0rdwk/backend/internal/auth"
	"dw0rdwk/backend/internal/cache"
	"dw0rdwk/backend/internal/config"
	"dw0rdwk/backend/internal/models"
	passwordhash "dw0rdwk/backend/internal/password"
	"dw0rdwk/backend/internal/platforms"
	"dw0rdwk/backend/internal/queue"
	"dw0rdwk/backend/internal/repository"
)

type Registry struct {
	Auth          *AuthService
	Dashboard     *DashboardService
	Health        *HealthService
	Orders        *OrderService
	Settings      *SettingService
	ConnectorSync *ConnectorSyncService
}

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

type DependencyError struct {
	Message string
	Err     error
}

func (e DependencyError) Error() string {
	return e.Message
}

func (e DependencyError) Unwrap() error {
	return e.Err
}

func NewRegistry(repos *repository.Registry, redisClient *cache.Client, authCfg config.AuthConfig, redisCfg config.RedisConfig) *Registry {
	tokenManager := auth.NewManager(authCfg, redisClient)
	dashboard := &DashboardService{
		repo:     repos.Dashboard,
		settings: repos.Settings,
		cache:    redisClient,
	}
	return &Registry{
		Auth: &AuthService{
			users:   repos.Users,
			invites: repos.InviteCodes,
			tokens:  tokenManager,
		},
		Dashboard: dashboard,
		Health: &HealthService{
			db:            repos.Health,
			cache:         redisClient,
			redisRequired: redisCfg.Enabled,
		},
		Orders: &OrderService{
			repo:       repos.Orders,
			connectors: repos.Connectors,
			plugins:    repos.PlatformPlugins,
			cache:      redisClient,
			events:     repos.OrderEvents,
		},
		Settings: &SettingService{
			repo:       repos.Settings,
			connectors: repos.Connectors,
			cache:      redisClient,
		},
		ConnectorSync: &ConnectorSyncService{
			connectors: repos.Connectors,
			classes:    repos.Classes,
			orders:     repos.Orders,
			events:     repos.OrderEvents,
			dashboard:  dashboard,
			logs:       repos.Logs,
			jobs:       repos.SystemJobs,
		},
	}
}

type HealthReport struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

type HealthService struct {
	db            *repository.HealthRepository
	cache         *cache.Client
	redisRequired bool
}

func (s *HealthService) Readiness(ctx context.Context) HealthReport {
	report := HealthReport{
		Status: "ok",
		Checks: map[string]string{
			"mysql": "ok",
			"redis": "ok",
		},
	}
	if err := s.db.Ping(ctx); err != nil {
		report.Status = "degraded"
		report.Checks["mysql"] = err.Error()
	}
	if err := s.cache.Ping(ctx); err != nil {
		report.Status = "degraded"
		report.Checks["redis"] = err.Error()
	}
	if !s.cache.Enabled() {
		report.Checks["redis"] = "disabled"
		if s.redisRequired {
			report.Status = "degraded"
		}
	}
	return report
}

type OrderService struct {
	repo       *repository.OrderRepository
	connectors *repository.ConnectorRepository
	plugins    *repository.PlatformPluginRepository
	cache      *cache.Client
	events     *repository.OrderEventRepository
}

type QueueRecoveryResult struct {
	Recovered int `json:"recovered"`
}

func (s *OrderService) Submit(ctx context.Context, order *models.Order) error {
	if err := s.validateExecutionRoute(ctx, order, "order"); err != nil {
		return err
	}
	prepareSubmittedOrder(order)
	if err := s.repo.Create(ctx, order); err != nil {
		return err
	}
	s.recordOrderEvent(ctx, *order, "info", "order_service", "order_submitted", "订单已提交，等待队列处理", "", true)
	if err := s.cache.PushJSON(ctx, queue.SubmitKey(order.FlashMode), map[string]any{
		"id":            order.ID,
		"connectorId":   order.ConnectorID,
		"executionMode": order.ExecutionMode,
		"pluginCode":    order.PluginCode,
		"flashMode":     order.FlashMode,
		"submittedAt":   time.Now().UTC(),
	}); err != nil {
		_ = s.repo.Update(ctx, order.ID, map[string]any{
			"status":         "failed",
			"docking_status": "queue_failed",
			"remarks":        "order queue unavailable",
		})
		s.recordOrderEvent(ctx, *order, "error", "order_service", "order_queue_failed", "订单队列不可用，提交任务未入队", "", true)
		return DependencyError{Message: "order queue unavailable", Err: err}
	}
	_ = s.cache.Delete(ctx, "dashboard:stats")
	return nil
}

func prepareSubmittedOrder(order *models.Order) {
	prepareExecutionRoute(order)
	order.Status = "queued"
	order.DockingStatus = "pending"
}

func prepareResubmittedOrder(order *models.Order) {
	prepareExecutionRoute(order)
	order.Status = "queued"
	order.DockingStatus = "pending"
	order.Progress = ""
	order.Remarks = ""
	order.RetryCount = 0
}

func (s *OrderService) RecoverQueues(ctx context.Context, batchSize int) (QueueRecoveryResult, error) {
	return s.recoverQueues(ctx, batchSize, false)
}

func (s *OrderService) RecoverStartupQueues(ctx context.Context, batchSize int) (QueueRecoveryResult, error) {
	return s.recoverQueues(ctx, batchSize, true)
}

func (s *OrderService) recoverQueues(ctx context.Context, batchSize int, includeInFlight bool) (QueueRecoveryResult, error) {
	if s.cache == nil || !s.cache.Enabled() {
		return QueueRecoveryResult{}, DependencyError{Message: "order queue unavailable", Err: cache.ErrDisabled}
	}
	if batchSize < 1 {
		batchSize = 100
	}
	if err := s.cache.Delete(ctx, queue.AllKeys()...); err != nil {
		return QueueRecoveryResult{}, DependencyError{Message: "order queue recovery failed", Err: err}
	}

	result := QueueRecoveryResult{}
	var afterID uint
	for {
		orders, err := s.recoverableOrders(ctx, afterID, batchSize, includeInFlight)
		if err != nil {
			return result, err
		}
		if len(orders) == 0 {
			break
		}
		for _, order := range orders {
			if err := s.cache.PushJSON(ctx, recoveryQueueKey(order), recoveryQueueTask(order)); err != nil {
				return result, DependencyError{Message: "order queue recovery failed", Err: err}
			}
			result.Recovered++
			afterID = order.ID
		}
		if len(orders) < batchSize {
			break
		}
	}
	_ = s.cache.Delete(ctx, "dashboard:stats")
	return result, nil
}

func (s *OrderService) recoverableOrders(ctx context.Context, afterID uint, batchSize int, includeInFlight bool) ([]models.Order, error) {
	if includeInFlight {
		return s.repo.ListRecoverableQueueAfter(ctx, afterID, batchSize)
	}
	return s.repo.ListQueuedAfter(ctx, afterID, batchSize)
}

func (s *OrderService) MarkRefreshRequested(ctx context.Context, id uint) error {
	return s.markRefreshRequested(ctx, id, "", "order_service")
}

func (s *OrderService) MarkMakeupRequested(ctx context.Context, id uint) error {
	return s.markRefreshRequested(ctx, id, "budan", "order_service")
}

func (s *OrderService) markRefreshRequested(ctx context.Context, id uint, action string, source string) error {
	order, err := s.repo.Find(ctx, id)
	if err != nil {
		return err
	}
	if order.Status == "cancelled" || order.Status == "refunded" {
		return ValidationError{Message: "finalized orders cannot be refreshed"}
	}
	if err := s.validateExecutionRoute(ctx, &order, "refresh"); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, id, map[string]any{
		"status":         "queued",
		"docking_status": "refresh_requested",
	}); err != nil {
		return err
	}
	task := map[string]any{
		"id":          id,
		"flashMode":   order.FlashMode,
		"requestedAt": time.Now().UTC(),
	}
	if strings.TrimSpace(action) != "" {
		task["action"] = strings.TrimSpace(action)
	}
	if err := s.cache.PushJSON(ctx, queue.RefreshKey(order.FlashMode), task); err != nil {
		_ = s.repo.Update(ctx, id, map[string]any{
			"status":         order.Status,
			"docking_status": order.DockingStatus,
			"remarks":        order.Remarks,
		})
		s.recordOrderEvent(ctx, order, "error", source, "order_refresh_queue_failed", "刷新队列不可用，任务未入队", order.Progress, true)
		return DependencyError{Message: "order refresh queue unavailable", Err: err}
	}
	eventType := "order_refresh_requested"
	content := "刷新任务已入队"
	if strings.EqualFold(action, "budan") {
		eventType = "order_makeup_requested"
		content = "补单任务已入队"
	}
	s.recordOrderEvent(ctx, order, "info", source, eventType, content, order.Progress, true)
	_ = s.cache.Delete(ctx, "dashboard:stats")
	return nil
}

func (s *OrderService) RequeueSubmit(ctx context.Context, id uint) error {
	order, err := s.repo.Find(ctx, id)
	if err != nil {
		return err
	}
	if order.Status == "cancelled" || order.Status == "refunded" {
		return ValidationError{Message: "finalized orders cannot be resubmitted"}
	}
	if err := s.validateExecutionRoute(ctx, &order, "order"); err != nil {
		return err
	}
	previous := map[string]any{
		"status":         order.Status,
		"docking_status": order.DockingStatus,
		"progress":       order.Progress,
		"retry_count":    order.RetryCount,
		"remarks":        order.Remarks,
	}
	prepareResubmittedOrder(&order)
	if err := s.repo.Update(ctx, id, map[string]any{
		"status":         order.Status,
		"docking_status": order.DockingStatus,
		"progress":       order.Progress,
		"retry_count":    order.RetryCount,
		"remarks":        order.Remarks,
	}); err != nil {
		return err
	}
	if err := s.cache.PushJSON(ctx, queue.SubmitKey(order.FlashMode), map[string]any{
		"id":            id,
		"connectorId":   order.ConnectorID,
		"executionMode": order.ExecutionMode,
		"pluginCode":    order.PluginCode,
		"flashMode":     order.FlashMode,
		"resubmittedAt": time.Now().UTC(),
	}); err != nil {
		_ = s.repo.Update(ctx, id, previous)
		s.recordOrderEvent(ctx, order, "error", "order_service", "order_resubmit_queue_failed", "提交队列不可用，重新上号任务未入队", "", true)
		return DependencyError{Message: "order submit queue unavailable", Err: err}
	}
	s.recordOrderEvent(ctx, order, "info", "order_service", "order_resubmitted", "订单已重新上号并进入队列", "", true)
	_ = s.cache.Delete(ctx, "dashboard:stats")
	return nil
}

func (s *OrderService) Cancel(ctx context.Context, id uint) error {
	order, err := s.repo.Find(ctx, id)
	if err != nil {
		return err
	}
	if !isOrderCancelable(order.Status) {
		return ValidationError{Message: "order cannot be cancelled"}
	}
	if err := s.repo.Update(ctx, id, map[string]any{
		"status":         "cancelled",
		"docking_status": "cancelled",
		"remarks":        "cancelled by agent",
	}); err != nil {
		return err
	}
	s.recordOrderEvent(ctx, order, "warning", "order_service", "order_cancelled", "订单已取消", order.Progress, true)
	_ = s.cache.Delete(ctx, "dashboard:stats")
	return nil
}

func (s *OrderService) UpdatePasswordAndRefresh(ctx context.Context, id uint, password string, source string) error {
	order, err := s.repo.Find(ctx, id)
	if err != nil {
		return err
	}
	if order.Status == "cancelled" || order.Status == "refunded" {
		return ValidationError{Message: "finalized orders cannot be refreshed"}
	}
	if err := s.repo.Update(ctx, id, map[string]any{"account_password": strings.TrimSpace(password)}); err != nil {
		return err
	}
	s.recordOrderEvent(ctx, order, "info", source, "order_password_updated", "学习密码已更新", order.Progress, true)
	return s.markRefreshRequested(ctx, id, "", source)
}

func isOrderCancelable(status string) bool {
	switch strings.TrimSpace(status) {
	case "", "pending", "queued", "processing":
		return true
	default:
		return false
	}
}

func prepareExecutionRoute(order *models.Order) {
	order.PluginCode = strings.TrimSpace(order.PluginCode)
	order.ExecutionMode = strings.TrimSpace(order.ExecutionMode)
	if order.PluginCode != "" {
		order.ExecutionMode = "plugin"
		order.ConnectorID = 0
		return
	}
	if order.ExecutionMode == "" {
		order.ExecutionMode = "connector"
	}
}

func (s *OrderService) validateExecutionRoute(ctx context.Context, order *models.Order, purpose string) error {
	prepareExecutionRoute(order)
	if order.ExecutionMode == "plugin" {
		return s.validatePlugin(ctx, order.PluginCode, purpose)
	}
	return s.validateConnector(ctx, order.ConnectorID)
}

func (s *OrderService) validateConnector(ctx context.Context, id uint) error {
	if id == 0 {
		return ValidationError{Message: "connectorId is required"}
	}
	connector, err := s.connectors.Find(ctx, id)
	if err != nil {
		if repository.IsNotFound(err) {
			return ValidationError{Message: "connector not found"}
		}
		return err
	}
	if connector.Status != "active" {
		return ValidationError{Message: "connector is disabled"}
	}
	if !connector.OrderSyncEnabled {
		return ValidationError{Message: "connector order sync is disabled"}
	}
	if connector.BaseURL == "" {
		return ValidationError{Message: "connector baseUrl is required"}
	}
	return nil
}

func (s *OrderService) validatePlugin(ctx context.Context, code string, purpose string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return ValidationError{Message: "pluginCode is required"}
	}
	if _, ok := platforms.PluginCodeFromPlatform(platforms.PlatformRef(code)); !ok {
		return ValidationError{Message: "pluginCode is invalid"}
	}
	if s.plugins == nil {
		return ValidationError{Message: "platform plugin repository is unavailable"}
	}
	plugin, err := s.plugins.Find(ctx, code)
	if err != nil {
		if repository.IsNotFound(err) {
			return ValidationError{Message: "platform plugin not found"}
		}
		return err
	}
	if plugin.Status != "active" {
		return ValidationError{Message: "platform plugin is disabled"}
	}
	if purpose == "order" && !plugin.SupportsSubmit {
		return ValidationError{Message: "platform plugin does not support submit"}
	}
	if purpose == "refresh" && !plugin.SupportsRefresh {
		return ValidationError{Message: "platform plugin does not support refresh"}
	}
	return nil
}

func (s *OrderService) recordOrderEvent(ctx context.Context, order models.Order, level string, source string, eventType string, content string, progress string, visible bool) {
	if s == nil || s.events == nil || order.ID == 0 {
		return
	}
	_ = s.events.Create(ctx, &models.OrderEvent{
		OrderID:       order.ID,
		UserID:        order.UserID,
		Level:         strings.TrimSpace(level),
		Source:        strings.TrimSpace(source),
		EventType:     strings.TrimSpace(eventType),
		Content:       truncateServiceText(content, 1000),
		Progress:      truncateServiceText(progress, 160),
		VisibleToUser: visible,
		CreatedAt:     time.Now(),
	})
}

func truncateServiceText(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func recoveryQueueTask(order models.Order) map[string]any {
	return map[string]any{
		"id":            order.ID,
		"connectorId":   order.ConnectorID,
		"executionMode": order.ExecutionMode,
		"pluginCode":    order.PluginCode,
		"flashMode":     order.FlashMode,
		"retryCount":    order.RetryCount,
		"recoveredAt":   time.Now().UTC(),
		"dockingStatus": order.DockingStatus,
	}
}

func recoveryQueueKey(order models.Order) string {
	if order.DockingStatus == "refresh_requested" {
		return queue.RefreshKey(order.FlashMode)
	}
	return queue.SubmitKey(order.FlashMode)
}

type AuthService struct {
	users   *repository.UserRepository
	invites *repository.InviteCodeRepository
	tokens  *auth.Manager
}

const dummyPasswordHash = "v1$120000$Z29yZWJ1aWxkLWR1bW15IQ$jahWCKRrR5tdJ6Svwjs7H5jmCQyZe71Dm7hj9MGRwP8"

type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt int64     `json:"expiresAt"`
	User      LoginUser `json:"user"`
}

type LoginUser struct {
	UID     uint   `json:"uid"`
	Account string `json:"account"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	Status  string `json:"status"`
}

type RegisterInput struct {
	Account    string
	Password   string
	Name       string
	InviteCode string
	SourceIP   string
}

func (s *AuthService) Login(ctx context.Context, account, password string) (LoginResult, error) {
	user, err := s.users.FindByAccount(ctx, account)
	if err != nil {
		if !repository.IsNotFound(err) {
			return LoginResult{}, err
		}
		user.PasswordHash = dummyPasswordHash
	}
	passwordOK := passwordhash.Verify(user.PasswordHash, password)
	if err != nil || !passwordOK {
		return LoginResult{}, errors.New("invalid credentials")
	}
	if user.Status != "" && user.Status != "active" {
		return LoginResult{}, errors.New("account disabled")
	}

	token, claims, err := s.tokens.Issue(ctx, user.ID, user.Account, user.Role)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		Token:     token,
		ExpiresAt: claims.Exp,
		User: LoginUser{
			UID:     user.ID,
			Account: user.Account,
			Name:    user.Name,
			Role:    user.Role,
			Status:  user.Status,
		},
	}, nil
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (LoginResult, error) {
	account := strings.TrimSpace(input.Account)
	if account == "" || strings.TrimSpace(input.Password) == "" {
		return LoginResult{}, ValidationError{Message: "account and password are required"}
	}
	if len(account) > 64 {
		return LoginResult{}, ValidationError{Message: "account must be 64 characters or fewer"}
	}
	if len(input.Password) < 8 {
		return LoginResult{}, ValidationError{Message: "password must be at least 8 characters"}
	}
	passwordHash, err := passwordhash.Hash(input.Password)
	if err != nil {
		return LoginResult{}, err
	}
	user := models.User{
		Account:      account,
		PasswordHash: passwordHash,
		Name:         strings.TrimSpace(input.Name),
		LastIP:       strings.TrimSpace(input.SourceIP),
	}
	if len(user.Name) > 120 {
		return LoginResult{}, ValidationError{Message: "name must be 120 characters or fewer"}
	}
	if err := s.invites.RegisterUser(ctx, input.InviteCode, &user); err != nil {
		switch {
		case repository.IsInviteCodeExpired(err):
			return LoginResult{}, ValidationError{Message: "invite code is expired"}
		case repository.IsInviteCodeExhausted(err):
			return LoginResult{}, ValidationError{Message: "invite code has no remaining uses"}
		case repository.IsInviteCodeInvalid(err):
			return LoginResult{}, ValidationError{Message: "invite code is invalid"}
		default:
			return LoginResult{}, err
		}
	}
	token, claims, err := s.tokens.Issue(ctx, user.ID, user.Account, user.Role)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		Token:     token,
		ExpiresAt: claims.Exp,
		User: LoginUser{
			UID:     user.ID,
			Account: user.Account,
			Name:    user.Name,
			Role:    user.Role,
			Status:  user.Status,
		},
	}, nil
}

func (s *AuthService) Validate(ctx context.Context, token string) (auth.Claims, error) {
	claims, err := s.tokens.Validate(ctx, token)
	if err != nil {
		return auth.Claims{}, err
	}
	user, err := s.users.Find(ctx, claims.UID)
	if err != nil {
		if repository.IsNotFound(err) {
			return auth.Claims{}, auth.ErrInvalidToken
		}
		return auth.Claims{}, err
	}
	if user.Status != "" && user.Status != "active" {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	claims.Account = user.Account
	claims.Role = user.Role
	return claims, nil
}

func (s *AuthService) Logout(ctx context.Context, claims auth.Claims) error {
	return s.tokens.Revoke(ctx, claims)
}

type DashboardService struct {
	repo     *repository.DashboardRepository
	settings *repository.SettingRepository
	cache    *cache.Client
}

func (s *DashboardService) Stats(ctx context.Context) (repository.DashboardStats, error) {
	var stats repository.DashboardStats
	if ok, err := s.cache.GetJSON(ctx, "dashboard:stats", &stats); err == nil && ok {
		s.populateQueueLengths(ctx, &stats)
		return stats, nil
	}

	stats, err := s.repo.Stats(ctx)
	if err != nil {
		return stats, err
	}
	_ = s.cache.SetJSON(ctx, "dashboard:stats", stats, s.cacheTTL(ctx))
	s.populateQueueLengths(ctx, &stats)
	return stats, nil
}

func (s *DashboardService) Statistics(ctx context.Context) (repository.DashboardStatistics, error) {
	return s.repo.Statistics(ctx)
}

func (s *DashboardService) Invalidate(ctx context.Context) {
	_ = s.cache.Delete(ctx, "dashboard:stats")
}

func (s *DashboardService) queueLength(ctx context.Context, key string) int64 {
	length, err := s.cache.ListLength(ctx, key)
	if err != nil {
		return 0
	}
	return length
}

func (s *DashboardService) populateQueueLengths(ctx context.Context, stats *repository.DashboardStats) {
	stats.QueueSubmit = s.queueLength(ctx, queue.OrderSubmit)
	stats.QueueSubmitFlash = s.queueLength(ctx, queue.OrderSubmitFlash)
	stats.QueueRefresh = s.queueLength(ctx, queue.OrderRefresh)
	stats.QueueRefreshFlash = s.queueLength(ctx, queue.OrderRefreshFlash)
	stats.QueueOrders = stats.QueueSubmit + stats.QueueSubmitFlash
	stats.QueueRefreshes = stats.QueueRefresh + stats.QueueRefreshFlash
}

func (s *DashboardService) cacheTTL(ctx context.Context) time.Duration {
	settings, err := s.settings.All(ctx)
	if err != nil {
		return dashboardCacheTTL(nil)
	}
	return dashboardCacheTTL(settings)
}

func dashboardCacheTTL(settings map[string]string) time.Duration {
	const defaultTTL = 30 * time.Second
	seconds, err := strconv.Atoi(settings["dashboard_cache_seconds"])
	if err != nil {
		return defaultTTL
	}
	if seconds < 5 {
		seconds = 5
	}
	if seconds > 3600 {
		seconds = 3600
	}
	return time.Duration(seconds) * time.Second
}

type SettingService struct {
	repo       *repository.SettingRepository
	connectors *repository.ConnectorRepository
	cache      *cache.Client
}

func (s *SettingService) All(ctx context.Context) (map[string]string, error) {
	settings := map[string]string{}
	if ok, err := s.cache.GetJSON(ctx, "settings:all", &settings); err == nil && ok {
		return settings, nil
	}
	settings, err := s.repo.All(ctx)
	if err != nil {
		return nil, err
	}
	_ = s.cache.SetJSON(ctx, "settings:all", settings, 60*time.Second)
	return settings, nil
}

func (s *SettingService) Upsert(ctx context.Context, values map[string]string) error {
	normalized, err := validateSettings(values)
	if err != nil {
		return err
	}
	if err := s.validateConnectorSettings(ctx, normalized); err != nil {
		return err
	}
	if err := s.repo.Upsert(ctx, normalized); err != nil {
		return err
	}
	return s.cache.Delete(ctx, "settings:all", "dashboard:stats")
}

func (s *SettingService) validateConnectorSettings(ctx context.Context, values map[string]string) error {
	raw := strings.TrimSpace(values["default_connector_id"])
	if raw == "" {
		return nil
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return ValidationError{Message: "default_connector_id must be a valid connector id"}
	}
	if s.connectors == nil {
		return nil
	}
	connector, err := s.connectors.Find(ctx, uint(id))
	if err != nil {
		if repository.IsNotFound(err) {
			return ValidationError{Message: "default connector not found"}
		}
		return err
	}
	if connector.Status != "active" || !connector.OrderSyncEnabled || strings.TrimSpace(connector.BaseURL) == "" {
		return ValidationError{Message: "default connector must be active and have a baseUrl"}
	}
	return nil
}

func validateSettings(values map[string]string) (map[string]string, error) {
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if len(value) > 2000 {
			return nil, ValidationError{Message: key + " is too long"}
		}
		switch key {
		case "site_name":
			if value == "" {
				return nil, ValidationError{Message: "site_name is required"}
			}
			if len(value) > 120 {
				return nil, ValidationError{Message: "site_name must be 120 characters or fewer"}
			}
		case "dashboard_cache_seconds":
			seconds, err := strconv.Atoi(value)
			if err != nil {
				return nil, ValidationError{Message: "dashboard_cache_seconds must be an integer"}
			}
			if seconds < 5 || seconds > 3600 {
				return nil, ValidationError{Message: "dashboard_cache_seconds must be between 5 and 3600"}
			}
		case "order_auto_refresh":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return nil, ValidationError{Message: "order_auto_refresh must be true or false"}
			}
			value = strconv.FormatBool(parsed)
		case "default_connector_id":
			if value != "" {
				id, err := strconv.ParseUint(value, 10, 64)
				if err != nil || id == 0 {
					return nil, ValidationError{Message: "default_connector_id must be a valid connector id"}
				}
				value = strconv.FormatUint(id, 10)
			}
		}
		normalized[key] = value
	}
	return normalized, nil
}

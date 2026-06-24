package http

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"dw0rdwk/backend/internal/config"
	"dw0rdwk/backend/internal/http/handlers"
	appmw "dw0rdwk/backend/internal/http/middleware"
	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

const (
	loginAccountRateLimitMax = 5
	loginIPRateLimitMax      = 30
	loginRateLimitWindow     = time.Minute
)

var loginAttempts = struct {
	sync.Mutex
	buckets map[string]loginAttemptBucket
}{
	buckets: map[string]loginAttemptBucket{},
}

type loginAttemptBucket struct {
	count int
	reset time.Time
}

func NewServer(cfg config.Config, repos *repository.Registry, services *service.Registry) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		BodyLimit:    cfg.HTTPBodyLimit,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
		Prefork:      cfg.HTTPPrefork,
		ErrorHandler: handlers.ErrorHandler,
	})

	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(rejectTraceMethods)
	app.Use(securityHeaders)
	app.Use(logger.New())
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: strings.Join(cfg.CORSAllowOrigins, ","),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))

	app.Get("/healthz", handlers.Health(cfg))
	app.Get("/readyz", handlers.Readiness(cfg, services.Health))
	app.Post("/api.php", handlers.LegacyAPI(repos.Users, repos.Classes, repos.SpecialPrices, repos.Orders, repos.Connectors, repos.Settings, services.Orders, repos.Logs))
	app.Get("/api.php", handlers.LegacyAPI(repos.Users, repos.Classes, repos.SpecialPrices, repos.Orders, repos.Connectors, repos.Settings, services.Orders, repos.Logs))
	MountAPI(app, repos, services, cfg)

	return app
}

func MountAPI(app *fiber.App, repos *repository.Registry, services *service.Registry, cfg config.Config) {
	api := app.Group("/api/v1")
	api.Get("/health", handlers.Health(cfg))
	api.Get("/ready", handlers.Readiness(cfg, services.Health))
	api.Post("/auth/login", loginRateLimiter(), handlers.Login(services.Auth))
	api.Post("/auth/register", loginRateLimiter(), handlers.Register(services.Auth))
	api.Post("/public/orders/search", handlers.PublicOrderSearch(repos.Orders, repos.Settings))
	api.Post("/public/orders/:id/refresh", handlers.PublicOrderRefresh(repos.Orders, services.Orders))
	api.Get("/public/orders/:id/events", handlers.PublicOrderEvents(repos.Orders, repos.OrderEvents))
	api.Patch("/public/orders/:id/password", handlers.PublicOrderPassword(repos.Orders, services.Orders))
	api.Post("/public/orders/:id/resubmit", handlers.PublicOrderResubmit(repos.Orders, services.Orders))

	protected := api.Group("", appmw.RequireAuth(services.Auth))
	protected.Get("/auth/me", handlers.Me())
	protected.Post("/auth/logout", handlers.Logout(services.Auth))
	protected.Get("/me/dashboard", handlers.AgentDashboard(repos.Users, repos.Orders, repos.Settings))
	protected.Get("/me/profile", handlers.AgentProfile(repos.Users))
	protected.Get("/me/logs", handlers.AgentLogs(repos.Logs))
	protected.Post("/me/password", handlers.AgentChangePassword(repos.Users, repos.Logs))
	protected.Put("/me/notice", handlers.AgentUpdateNotice(repos.Users, repos.Logs))
	protected.Put("/me/invite", handlers.AgentUpdateInvite(repos.Users, repos.InviteCodes, repos.Logs))
	protected.Post("/me/api-key", handlers.AgentRegenerateAPIKey(repos.Users, repos.Logs))
	protected.Delete("/me/api-key", handlers.AgentDisableAPIKey(repos.Users, repos.Logs))
	protected.Get("/me/agents", handlers.AgentChildren(repos.Users))
	protected.Post("/me/agents", handlers.AgentCreateChild(repos.Users, repos.Logs))
	protected.Patch("/me/agents/:id", handlers.AgentUpdateChild(repos.Users, repos.Logs))
	protected.Post("/me/agents/:id/password/reset", handlers.AgentResetChildPassword(repos.Users, repos.Logs))
	protected.Post("/me/agents/:id/balance", handlers.AgentTransferChildBalance(repos.Users, repos.Logs))
	protected.Get("/me/categories", handlers.AgentCategories(repos.Categories))
	protected.Get("/me/classes", handlers.AgentClasses(repos.Users, repos.Classes, repos.SpecialPrices))
	protected.Get("/me/order-submit/bootstrap", handlers.AgentOrderSubmitBootstrap(repos.Users, repos.Categories, repos.Favorites, repos.Classes, repos.SpecialPrices))
	protected.Get("/me/order-submit/recommendations", handlers.AgentOrderSubmitRecommendations(repos.Users, repos.Recommendations, repos.SpecialPrices, repos.Orders, repos.Settings))
	protected.Post("/me/course-query", handlers.AgentCourseQuery(repos.Users, repos.Classes, repos.Connectors, repos.PlatformPlugins, repos.Settings, repos.Logs))
	protected.Get("/me/class-favorites", handlers.AgentClassFavorites(repos.Favorites))
	protected.Put("/me/class-favorites/:id", handlers.AddAgentClassFavorite(repos.Favorites, repos.Classes, repos.Logs))
	protected.Delete("/me/class-favorites/:id", handlers.RemoveAgentClassFavorite(repos.Favorites, repos.Logs))
	protected.Get("/me/orders", handlers.AgentOrders(repos.Orders))
	protected.Post("/me/orders", handlers.AgentCreateOrder(repos.Users, repos.Classes, repos.SpecialPrices, repos.Connectors, repos.PlatformPlugins, repos.Settings, services.Orders, repos.Logs))
	protected.Post("/me/orders/batch", handlers.AgentBatchCreateOrders(repos.Users, repos.Classes, repos.SpecialPrices, repos.Connectors, repos.PlatformPlugins, repos.Settings, services.Orders, repos.Logs))
	protected.Get("/me/orders/:id/events", handlers.AgentOrderEvents(repos.Orders, repos.OrderEvents))
	protected.Post("/me/orders/:id/refresh", handlers.AgentRefreshOrder(repos.Orders, services.Orders, repos.Logs))
	protected.Post("/me/orders/:id/cancel", handlers.AgentCancelOrder(repos.Orders, services.Orders, repos.Logs))
	protected.Patch("/me/orders/:id/password", handlers.AgentUpdateOrderPassword(repos.Orders, services.Orders, repos.Logs))
	protected.Get("/me/work-orders", handlers.AgentWorkOrders(repos.Users, repos.WorkOrders))
	protected.Post("/me/work-orders", handlers.AgentCreateWorkOrder(repos.Users, repos.WorkOrders, repos.Logs))
	protected.Patch("/me/work-orders/:id/reply", handlers.AgentReplyWorkOrder(repos.Users, repos.WorkOrders, repos.Logs))
	protected.Delete("/me/work-orders/:id", handlers.AgentDeleteWorkOrder(repos.Users, repos.WorkOrders, repos.Logs))
	protected.Post("/me/recharge-cards/query", handlers.AgentQueryRechargeCard(repos.RechargeCards))
	protected.Post("/me/recharge-cards/redeem", handlers.AgentRedeemRechargeCard(repos.Users, repos.RechargeCards, repos.Logs))
	admin := protected.Group("", appmw.RequireRole("admin"))
	admin.Get("/dashboard", handlers.Dashboard(services.Dashboard))
	admin.Get("/dashboard/statistics", handlers.DashboardStatistics(services.Dashboard))
	admin.Get("/users", handlers.Users(repos.Users))
	admin.Get("/agents/tree", handlers.AgentTree(repos.Users))
	admin.Get("/invite-codes", handlers.InviteCodes(repos.InviteCodes))
	admin.Post("/invite-codes", handlers.CreateInviteCode(repos.InviteCodes, repos.Users, repos.Logs))
	admin.Patch("/invite-codes/:id", handlers.UpdateInviteCode(repos.InviteCodes, repos.Logs))
	admin.Delete("/invite-codes/:id", handlers.DeleteInviteCode(repos.InviteCodes, repos.Logs))
	admin.Post("/users", handlers.CreateUser(repos.Users, services.Dashboard, repos.Logs))
	admin.Patch("/users/:id", handlers.UpdateUser(repos.Users, services.Dashboard, repos.Logs))
	admin.Post("/users/:id/password/reset", handlers.ResetUserPassword(repos.Users, repos.Logs))
	admin.Post("/users/:id/balance", handlers.AdjustUserBalance(repos.Users, services.Dashboard, repos.Logs))
	admin.Delete("/users/:id", handlers.DeleteUser(repos.Users, services.Dashboard, repos.Logs))
	admin.Get("/classes", handlers.Classes(repos.Classes))
	admin.Post("/classes", handlers.CreateClass(repos.Classes, services.Dashboard, repos.Logs))
	admin.Post("/classes/batch/status", handlers.BatchUpdateClassStatus(repos.Classes, services.Dashboard, repos.Logs))
	admin.Post("/classes/batch/move", handlers.BatchMoveClasses(repos.Classes, repos.Categories, services.Dashboard, repos.Logs))
	admin.Post("/classes/batch/delete", handlers.BatchDeleteClasses(repos.Classes, services.Dashboard, repos.Logs))
	admin.Patch("/classes/batch", handlers.BatchPatchClasses(repos.Classes, services.Dashboard, repos.Logs))
	admin.Post("/classes/keywords/replace", handlers.ReplaceClassKeywords(repos.Classes, services.Dashboard, repos.Logs))
	admin.Post("/classes/prefix", handlers.AddClassPrefix(repos.Classes, services.Dashboard, repos.Logs))
	admin.Post("/classes/deduplicate/preview", handlers.PreviewClassDeduplicate(repos.Classes))
	admin.Post("/classes/deduplicate/apply", handlers.ApplyClassDeduplicate(repos.Classes, services.Dashboard, repos.Logs))
	admin.Patch("/classes/:id", handlers.UpdateClass(repos.Classes, services.Dashboard, repos.Logs))
	admin.Delete("/classes/:id", handlers.DeleteClass(repos.Classes, services.Dashboard, repos.Logs))
	admin.Get("/categories", handlers.Categories(repos.Categories))
	admin.Post("/categories", handlers.CreateCategory(repos.Categories, services.Dashboard, repos.Logs))
	admin.Patch("/categories/:id", handlers.UpdateCategory(repos.Categories, services.Dashboard, repos.Logs))
	admin.Delete("/categories/:id", handlers.DeleteCategory(repos.Categories, services.Dashboard, repos.Logs))
	admin.Get("/special-prices", handlers.SpecialPrices(repos.SpecialPrices))
	admin.Post("/special-prices", handlers.UpsertSpecialPrice(repos.SpecialPrices, repos.Users, repos.Classes, services.Dashboard, repos.Logs))
	admin.Patch("/special-prices/:id", handlers.UpdateSpecialPrice(repos.SpecialPrices, repos.Users, repos.Classes, services.Dashboard, repos.Logs))
	admin.Delete("/special-prices/:id", handlers.DeleteSpecialPrice(repos.SpecialPrices, services.Dashboard, repos.Logs))
	admin.Get("/orders", handlers.Orders(repos.Orders))
	admin.Post("/orders", handlers.CreateOrder(services.Orders, repos.Logs))
	admin.Post("/orders/queues/recover", handlers.RecoverOrderQueues(services.Orders, repos.Logs))
	admin.Post("/orders/batch/refresh", handlers.BatchRefreshOrders(services.Orders, repos.Logs))
	admin.Post("/orders/batch/resubmit", handlers.BatchResubmitOrders(services.Orders, repos.Logs))
	admin.Post("/orders/batch/refund", handlers.BatchRefundOrders(repos.Orders, services.Dashboard, repos.Logs))
	admin.Post("/orders/batch/delete", handlers.BatchDeleteOrders(repos.Orders, services.Dashboard, repos.Logs))
	admin.Get("/orders/:id/events", handlers.OrderEvents(repos.Orders, repos.OrderEvents))
	admin.Patch("/orders/:id", handlers.UpdateOrder(repos.Orders, repos.Connectors, repos.PlatformPlugins, services.Dashboard, repos.Logs))
	admin.Delete("/orders/:id", handlers.DeleteOrder(repos.Orders, services.Dashboard, repos.Logs))
	admin.Post("/orders/:id/refresh", handlers.RefreshOrder(services.Orders, repos.Logs))
	admin.Post("/orders/:id/refund", handlers.RefundOrder(repos.Orders, services.Dashboard, repos.Logs))
	admin.Get("/work-orders", handlers.WorkOrders(repos.WorkOrders))
	admin.Patch("/work-orders/:id", handlers.UpdateWorkOrder(repos.WorkOrders, services.Dashboard, repos.Logs))
	admin.Delete("/work-orders/:id", handlers.DeleteWorkOrder(repos.WorkOrders, services.Dashboard, repos.Logs))
	admin.Get("/recharge-cards", handlers.RechargeCards(repos.RechargeCards))
	admin.Post("/recharge-cards", handlers.CreateRechargeCards(repos.RechargeCards, services.Dashboard, repos.Logs))
	admin.Delete("/recharge-cards/:id", handlers.DeleteRechargeCard(repos.RechargeCards, services.Dashboard, repos.Logs))
	admin.Get("/recommendations", handlers.Recommendations(repos.Recommendations))
	admin.Post("/recommendations", handlers.CreateRecommendation(repos.Recommendations, repos.Classes, repos.Logs))
	admin.Patch("/recommendations/:id", handlers.UpdateRecommendation(repos.Recommendations, repos.Classes, repos.Logs))
	admin.Delete("/recommendations/:id", handlers.DeleteRecommendation(repos.Recommendations, repos.Logs))
	admin.Get("/menus", handlers.Menus(repos.Menus))
	admin.Post("/menus", handlers.CreateMenu(repos.Menus, repos.Logs))
	admin.Patch("/menus/sort", handlers.SortMenus(repos.Menus, repos.Logs))
	admin.Patch("/menus/:id", handlers.UpdateMenu(repos.Menus, repos.Logs))
	admin.Delete("/menus/:id", handlers.DeleteMenu(repos.Menus, repos.Logs))
	admin.Get("/connectors", handlers.Connectors(repos.Connectors))
	admin.Post("/connectors", handlers.CreateConnector(repos.Connectors, repos.Logs))
	admin.Post("/connectors/:id/balance", handlers.ConnectorBalance(repos.Connectors))
	admin.Post("/connectors/29wk/pull", handlers.Pull29WKClasses(repos.Connectors, repos.Categories, repos.Classes))
	admin.Post("/connectors/29wk/sync", handlers.Sync29WKClasses(repos.Connectors, repos.Categories, repos.Classes, services.Dashboard, repos.Logs))
	admin.Post("/connectors/29wk/orders/sync", handlers.Sync29WKOrders(services.ConnectorSync, repos.Logs))
	admin.Post("/connectors/29wk/prices/sync", handlers.Sync29WKPrices(services.ConnectorSync, repos.Logs))
	admin.Patch("/connectors/:id", handlers.UpdateConnector(repos.Connectors, repos.Logs))
	admin.Delete("/connectors/:id", handlers.DeleteConnector(repos.Connectors, services.Dashboard, repos.Logs))
	admin.Get("/platform-plugins", handlers.PlatformPlugins(repos.PlatformPlugins))
	admin.Post("/platform-plugins", handlers.CreatePlatformPlugin(repos.PlatformPlugins, repos.Logs))
	admin.Patch("/platform-plugins/:code", handlers.UpdatePlatformPlugin(repos.PlatformPlugins, repos.Logs))
	admin.Delete("/platform-plugins/:code", handlers.DeletePlatformPlugin(repos.PlatformPlugins, repos.Logs))
	admin.Get("/worker-nodes", handlers.WorkerNodes(repos.WorkerNodes))
	admin.Get("/worker-commands", handlers.WorkerCommands(repos.WorkerCommands))
	admin.Post("/worker-nodes/:workerId/commands", handlers.CreateWorkerCommand(repos.WorkerCommands, repos.Logs))
	admin.Get("/worker-proxies", handlers.WorkerProxies(repos.WorkerProxies))
	admin.Post("/worker-proxies", handlers.CreateWorkerProxy(repos.WorkerProxies, repos.Logs))
	admin.Patch("/worker-proxies/:id", handlers.UpdateWorkerProxy(repos.WorkerProxies, repos.Logs))
	admin.Delete("/worker-proxies/:id", handlers.DeleteWorkerProxy(repos.WorkerProxies, repos.Logs))
	admin.Get("/settings", handlers.Settings(services.Settings))
	admin.Put("/settings", handlers.SaveSettings(services.Settings, repos.Logs))
	admin.Get("/system/jobs", handlers.SystemJobs(repos.SystemJobs))
	admin.Patch("/system/jobs/:name", handlers.UpdateSystemJob(repos.SystemJobs, repos.Logs))
	admin.Post("/system/jobs/:name/run", handlers.RunSystemJob(repos.SystemJobs, services.Orders, services.ConnectorSync, repos.Logs))
	admin.Get("/logs", handlers.Logs(repos.Logs))
}

func rejectTraceMethods(c *fiber.Ctx) error {
	method := c.Method()
	if method == fiber.MethodTrace || method == "TRACK" {
		return fiber.NewError(fiber.StatusMethodNotAllowed, "method not allowed")
	}
	return c.Next()
}

func securityHeaders(c *fiber.Ctx) error {
	c.Set("X-Frame-Options", "DENY")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Referrer-Policy", "no-referrer")
	c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	c.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self'; object-src 'none'")
	c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	return c.Next()
}

func loginRateLimiter() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if loginRateLimited(c.IP(), loginAccountFromBody(c.Body()), time.Now()) {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many login attempts")
		}
		return c.Next()
	}
}

func loginRateLimited(ip, account string, now time.Time) bool {
	if rateLimitExceeded("login:ip:"+ip, loginIPRateLimitMax, loginRateLimitWindow, now) {
		return true
	}
	if account == "" {
		return false
	}
	return rateLimitExceeded("login:account:"+account, loginAccountRateLimitMax, loginRateLimitWindow, now)
}

func rateLimitExceeded(key string, max int, window time.Duration, now time.Time) bool {
	loginAttempts.Lock()
	defer loginAttempts.Unlock()

	bucket := loginAttempts.buckets[key]
	if bucket.reset.IsZero() || !now.Before(bucket.reset) {
		bucket = loginAttemptBucket{reset: now.Add(window)}
	}
	if bucket.count >= max {
		loginAttempts.buckets[key] = bucket
		cleanupLoginAttempts(now)
		return true
	}
	bucket.count++
	loginAttempts.buckets[key] = bucket
	cleanupLoginAttempts(now)
	return false
}

func cleanupLoginAttempts(now time.Time) {
	if len(loginAttempts.buckets) < 1000 {
		return
	}
	for key, bucket := range loginAttempts.buckets {
		if !now.Before(bucket.reset) {
			delete(loginAttempts.buckets, key)
		}
	}
}

func loginAccountFromBody(body []byte) string {
	var req struct {
		Account string `json:"account"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(req.Account))
}

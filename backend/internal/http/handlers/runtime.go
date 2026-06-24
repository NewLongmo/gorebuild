package handlers

import (
	"encoding/json"
	"strings"

	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type platformPluginPayload struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	SortOrder       int    `json:"sortOrder"`
	SupportsQuery   *bool  `json:"supportsQuery"`
	SupportsSubmit  *bool  `json:"supportsSubmit"`
	SupportsRefresh *bool  `json:"supportsRefresh"`
	MaxConcurrency  int    `json:"maxConcurrency"`
	AccountSerial   *bool  `json:"accountSerial"`
	ConfigJSON      string `json:"configJson"`
}

type workerCommandPayload struct {
	Command string `json:"command"`
}

type workerProxyPayload struct {
	Name           string `json:"name"`
	ProxyURL       string `json:"proxyUrl"`
	Kind           string `json:"kind"`
	Status         string `json:"status"`
	MaxConcurrency int    `json:"maxConcurrency"`
}

func PlatformPlugins(repo *repository.PlatformPluginRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := repo.List(c.UserContext(), c.Query("q"), c.Query("status"), queryInt(c, "page", 1), queryInt(c, "perPage", 50))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func CreatePlatformPlugin(repo *repository.PlatformPluginRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req platformPluginPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		item, err := platformPluginFromPayload(req)
		if err != nil {
			return err
		}
		if err := repo.Create(c.UserContext(), &item); err != nil {
			return err
		}
		auditLog(c, logs, "admin_platform_plugin_create", "plugin="+item.Code, 0)
		return OK(c, item)
	}
}

func UpdatePlatformPlugin(repo *repository.PlatformPluginRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := strings.TrimSpace(c.Params("code"))
		if code == "" {
			return fiber.NewError(fiber.StatusBadRequest, "plugin code is required")
		}
		var req platformPluginPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		values, err := platformPluginValues(req, false)
		if err != nil {
			return err
		}
		if err := repo.Update(c.UserContext(), code, values); err != nil {
			return err
		}
		auditLog(c, logs, "admin_platform_plugin_update", "plugin="+code+" "+auditFields(values), 0)
		return OK(c, fiber.Map{"code": code})
	}
}

func DeletePlatformPlugin(repo *repository.PlatformPluginRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := strings.TrimSpace(c.Params("code"))
		if code == "" {
			return fiber.NewError(fiber.StatusBadRequest, "plugin code is required")
		}
		if err := repo.Delete(c.UserContext(), code); err != nil {
			return err
		}
		auditLog(c, logs, "admin_platform_plugin_delete", "plugin="+code, 0)
		return OK(c, fiber.Map{"code": code})
	}
}

func WorkerNodes(repo *repository.WorkerNodeRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		items, err := repo.List(c.UserContext())
		if err != nil {
			return err
		}
		return OK(c, fiber.Map{"items": items})
	}
}

func WorkerCommands(repo *repository.WorkerCommandRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := repo.List(c.UserContext(), c.Query("workerId"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func CreateWorkerCommand(repo *repository.WorkerCommandRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		workerID := strings.TrimSpace(c.Params("workerId"))
		if workerID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "workerId is required")
		}
		var req workerCommandPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		command := strings.TrimSpace(req.Command)
		if !validWorkerCommand(command) {
			return fiber.NewError(fiber.StatusBadRequest, "command must be pause_accept, resume_accept, or stop")
		}
		item := models.WorkerCommand{WorkerID: workerID, Command: command}
		if err := repo.Create(c.UserContext(), &item); err != nil {
			return err
		}
		auditLog(c, logs, "admin_worker_command_create", "worker="+workerID+" command="+command, 0)
		return OK(c, item)
	}
}

func WorkerProxies(repo *repository.WorkerProxyRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := repo.List(c.UserContext(), c.Query("q"), c.Query("status"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func CreateWorkerProxy(repo *repository.WorkerProxyRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req workerProxyPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		item, err := workerProxyFromPayload(req)
		if err != nil {
			return err
		}
		if err := repo.Create(c.UserContext(), &item); err != nil {
			return err
		}
		auditLog(c, logs, "admin_worker_proxy_create", auditText("proxy", item.ID, item.Name), 0)
		return OK(c, repository.WorkerProxyRow{WorkerProxy: item, MaskedURL: maskProxyURLForResponse(item.ProxyURL)})
	}
}

func UpdateWorkerProxy(repo *repository.WorkerProxyRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req workerProxyPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		values, err := workerProxyValues(req, false)
		if err != nil {
			return err
		}
		if err := repo.Update(c.UserContext(), id, values); err != nil {
			return err
		}
		auditLog(c, logs, "admin_worker_proxy_update", auditText("proxy", id, auditFields(values)), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func DeleteWorkerProxy(repo *repository.WorkerProxyRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if err := repo.Delete(c.UserContext(), id); err != nil {
			return err
		}
		auditLog(c, logs, "admin_worker_proxy_delete", auditText("proxy", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func platformPluginFromPayload(req platformPluginPayload) (models.PlatformPlugin, error) {
	code := normalizePluginCode(req.Code)
	if code == "" {
		return models.PlatformPlugin{}, fiber.NewError(fiber.StatusBadRequest, "code is required")
	}
	values, err := platformPluginValues(req, true)
	if err != nil {
		return models.PlatformPlugin{}, err
	}
	item := models.PlatformPlugin{
		Code:            code,
		Name:            runtimeStringValue(values, "name"),
		Description:     runtimeStringValue(values, "description"),
		Status:          runtimeStringValue(values, "status"),
		SortOrder:       runtimeIntValue(values, "sort_order"),
		SupportsQuery:   runtimeBoolValue(values, "supports_query", false),
		SupportsSubmit:  runtimeBoolValue(values, "supports_submit", true),
		SupportsRefresh: runtimeBoolValue(values, "supports_refresh", true),
		MaxConcurrency:  runtimeIntValue(values, "max_concurrency"),
		AccountSerial:   runtimeBoolValue(values, "account_serial", true),
		ConfigJSON:      runtimeStringValue(values, "config_json"),
	}
	if item.Name == "" {
		return models.PlatformPlugin{}, fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	return item, nil
}

func platformPluginValues(req platformPluginPayload, fillDefaults bool) (map[string]any, error) {
	values := map[string]any{}
	if name := strings.TrimSpace(req.Name); name != "" {
		if len(name) > 120 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "name is too long")
		}
		values["name"] = name
	}
	if description := strings.TrimSpace(req.Description); description != "" {
		if len(description) > 500 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "description is too long")
		}
		values["description"] = description
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		if status != "active" && status != "disabled" {
			return nil, fiber.NewError(fiber.StatusBadRequest, "status must be active or disabled")
		}
		values["status"] = status
	}
	if req.SortOrder != 0 {
		values["sort_order"] = req.SortOrder
	}
	if req.SupportsQuery != nil {
		values["supports_query"] = *req.SupportsQuery
	}
	if req.SupportsSubmit != nil {
		values["supports_submit"] = *req.SupportsSubmit
	}
	if req.SupportsRefresh != nil {
		values["supports_refresh"] = *req.SupportsRefresh
	}
	if req.MaxConcurrency != 0 {
		if req.MaxConcurrency < 1 || req.MaxConcurrency > 100 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "maxConcurrency must be between 1 and 100")
		}
		values["max_concurrency"] = req.MaxConcurrency
	}
	if req.AccountSerial != nil {
		values["account_serial"] = *req.AccountSerial
	}
	configJSON := strings.TrimSpace(req.ConfigJSON)
	if configJSON != "" {
		if len(configJSON) > 5000 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "configJson is too long")
		}
		if !json.Valid([]byte(configJSON)) {
			return nil, fiber.NewError(fiber.StatusBadRequest, "configJson must be valid JSON")
		}
		values["config_json"] = configJSON
	}
	if fillDefaults {
		if req.SupportsSubmit == nil {
			values["supports_submit"] = true
		}
		if req.SupportsRefresh == nil {
			values["supports_refresh"] = true
		}
		if req.AccountSerial == nil {
			values["account_serial"] = true
		}
		if _, ok := values["status"]; !ok {
			values["status"] = "disabled"
		}
		if _, ok := values["config_json"]; !ok {
			values["config_json"] = "{}"
		}
		if _, ok := values["max_concurrency"]; !ok {
			values["max_concurrency"] = 1
		}
	}
	return values, nil
}

func workerProxyFromPayload(req workerProxyPayload) (models.WorkerProxy, error) {
	values, err := workerProxyValues(req, true)
	if err != nil {
		return models.WorkerProxy{}, err
	}
	item := models.WorkerProxy{
		Name:           runtimeStringValue(values, "name"),
		ProxyURL:       runtimeStringValue(values, "proxy_url"),
		Kind:           runtimeStringValue(values, "kind"),
		Status:         runtimeStringValue(values, "status"),
		MaxConcurrency: runtimeIntValue(values, "max_concurrency"),
	}
	if item.ProxyURL == "" {
		return models.WorkerProxy{}, fiber.NewError(fiber.StatusBadRequest, "proxyUrl is required")
	}
	return item, nil
}

func workerProxyValues(req workerProxyPayload, fillDefaults bool) (map[string]any, error) {
	values := map[string]any{}
	if name := strings.TrimSpace(req.Name); name != "" {
		if len(name) > 120 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "name is too long")
		}
		values["name"] = name
	}
	if proxyURL := strings.TrimSpace(req.ProxyURL); proxyURL != "" {
		if len(proxyURL) > 500 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "proxyUrl is too long")
		}
		values["proxy_url"] = proxyURL
	}
	if kind := strings.TrimSpace(req.Kind); kind != "" {
		if kind != "http" && kind != "socks5" {
			return nil, fiber.NewError(fiber.StatusBadRequest, "kind must be http or socks5")
		}
		values["kind"] = kind
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		if status != "active" && status != "disabled" {
			return nil, fiber.NewError(fiber.StatusBadRequest, "status must be active or disabled")
		}
		values["status"] = status
	}
	if req.MaxConcurrency != 0 {
		if req.MaxConcurrency < 1 || req.MaxConcurrency > 100 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "maxConcurrency must be between 1 and 100")
		}
		values["max_concurrency"] = req.MaxConcurrency
	}
	if fillDefaults {
		if _, ok := values["kind"]; !ok {
			values["kind"] = "http"
		}
		if _, ok := values["status"]; !ok {
			values["status"] = "active"
		}
		if _, ok := values["max_concurrency"]; !ok {
			values["max_concurrency"] = 1
		}
	}
	return values, nil
}

func validWorkerCommand(command string) bool {
	switch strings.TrimSpace(command) {
	case "pause_accept", "resume_accept", "stop":
		return true
	default:
		return false
	}
}

func normalizePluginCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "plugin:")
	return value
}

func runtimeStringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func runtimeIntValue(values map[string]any, key string) int {
	value, _ := values[key].(int)
	return value
}

func runtimeBoolValue(values map[string]any, key string, fallback bool) bool {
	value, ok := values[key].(bool)
	if !ok {
		return fallback
	}
	return value
}

func maskProxyURLForResponse(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	at := strings.LastIndex(value, "@")
	if at >= 0 {
		schemeSep := strings.Index(value, "://")
		if schemeSep >= 0 && schemeSep+3 < at {
			return value[:schemeSep+3] + "***:***" + value[at:]
		}
		return "***" + value[at:]
	}
	if len(value) <= 16 {
		return value
	}
	return value[:8] + "..." + value[len(value)-6:]
}

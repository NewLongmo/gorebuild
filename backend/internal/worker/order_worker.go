package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"dw0rdwk/backend/internal/cache"
	"dw0rdwk/backend/internal/config"
	connectoradapter "dw0rdwk/backend/internal/connectors"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/platforms"
	"dw0rdwk/backend/internal/queue"
	"dw0rdwk/backend/internal/repository"
)

type OrderWorker struct {
	cfg       config.WorkerConfig
	cache     workerCache
	repos     *repository.Registry
	platforms *platforms.Registry
	workerID  string
	hostname  string
	acceptNew atomic.Bool
	running   atomic.Int64
	currentID atomic.Uint64
}

type workerCache interface {
	Enabled() bool
	PushJSON(ctx context.Context, key string, value any) error
	PopJSON(ctx context.Context, key string, timeout time.Duration, dest any) (bool, error)
	PopJSONFrom(ctx context.Context, keys []string, timeout time.Duration, dest any) (bool, error)
	Delete(ctx context.Context, keys ...string) error
}

type queueTask struct {
	ID          uint      `json:"id"`
	ConnectorID uint      `json:"connectorId"`
	FlashMode   bool      `json:"flashMode"`
	Action      string    `json:"action"`
	SubmittedAt time.Time `json:"submittedAt"`
	RequestedAt time.Time `json:"requestedAt"`
	RecoveredAt time.Time `json:"recoveredAt"`
}

type orderRoute struct {
	Order     models.Order
	Connector models.Connector
	Plugin    models.PlatformPlugin
}

type connectorResponse struct {
	RemoteOrderID string `json:"remoteOrderId"`
	Status        string `json:"status"`
	Progress      string `json:"progress"`
	Remarks       string `json:"remarks"`
}

func NewOrderWorker(cfg config.WorkerConfig, cacheClient *cache.Client, repos *repository.Registry) *OrderWorker {
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}
	if cfg.RetryDelayMS < 0 {
		cfg.RetryDelayMS = 0
	}
	hostname, _ := os.Hostname()
	worker := &OrderWorker{
		cfg:       cfg,
		cache:     cacheClient,
		repos:     repos,
		platforms: platforms.DefaultRegistry(),
		workerID:  fmt.Sprintf("%s-%d", strings.TrimSpace(hostname), os.Getpid()),
		hostname:  hostname,
	}
	if strings.TrimSpace(worker.workerID) == fmt.Sprintf("-%d", os.Getpid()) {
		worker.workerID = fmt.Sprintf("worker-%d", os.Getpid())
	}
	worker.acceptNew.Store(true)
	return worker
}

func (w *OrderWorker) Start(ctx context.Context) {
	if !w.cfg.Enabled {
		log.Printf("order worker disabled")
		return
	}
	if !w.cache.Enabled() {
		log.Printf("order worker disabled: redis unavailable")
		return
	}
	go w.statusLoop(ctx)
	for i := 0; i < w.cfg.Concurrency; i++ {
		go w.loop(ctx, i+1, queue.SubmitPriorityKeys(), w.processSubmit)
		go w.loop(ctx, i+1, queue.RefreshPriorityKeys(), w.processRefresh)
	}
}

func (w *OrderWorker) statusLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	w.publishWorkerStatus(ctx, "running", "运行中")
	defer w.markWorkerStopped(context.Background())
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.handleWorkerCommands(ctx)
			message := "运行中"
			if !w.acceptNew.Load() {
				message = "暂停接单"
			}
			w.publishWorkerStatus(ctx, "running", message)
		}
	}
}

func (w *OrderWorker) publishWorkerStatus(ctx context.Context, status string, message string) {
	if w == nil || w.repos == nil || w.repos.WorkerNodes == nil {
		return
	}
	now := time.Now()
	currentID := uint(w.currentID.Load())
	currentPlugin := ""
	if currentID != 0 && w.repos.Orders != nil {
		if order, err := w.repos.Orders.Find(ctx, currentID); err == nil {
			currentPlugin = order.PluginCode
		}
	}
	_ = w.repos.WorkerNodes.UpsertHeartbeat(ctx, models.WorkerNode{
		WorkerID:          w.workerID,
		Hostname:          w.hostname,
		Status:            status,
		AcceptNew:         w.acceptNew.Load(),
		MaxConcurrency:    w.cfg.Concurrency * 2,
		RunningCount:      int(w.running.Load()),
		CurrentOrderID:    currentID,
		CurrentPluginCode: currentPlugin,
		Message:           message,
		StartedAt:         &now,
		HeartbeatAt:       &now,
	})
}

func (w *OrderWorker) markWorkerStopped(ctx context.Context) {
	if w == nil || w.repos == nil || w.repos.WorkerNodes == nil {
		return
	}
	_ = w.repos.WorkerNodes.MarkStopped(ctx, w.workerID, "worker stopped")
}

func (w *OrderWorker) handleWorkerCommands(ctx context.Context) {
	if w == nil || w.repos == nil || w.repos.WorkerCommands == nil {
		return
	}
	commands, err := w.repos.WorkerCommands.PendingForWorker(ctx, w.workerID, 20)
	if err != nil {
		log.Printf("order worker commands: %v", err)
		return
	}
	for _, command := range commands {
		status := "done"
		result := "ok"
		switch strings.TrimSpace(command.Command) {
		case "pause_accept":
			w.acceptNew.Store(false)
			result = "paused"
		case "resume_accept":
			w.acceptNew.Store(true)
			result = "resumed"
		case "stop":
			w.acceptNew.Store(false)
			result = "stop requested; accepting new tasks is disabled"
		default:
			status = "failed"
			result = "unsupported command"
		}
		if err := w.repos.WorkerCommands.Finish(ctx, command.ID, status, result); err != nil {
			log.Printf("order worker finish command %d: %v", command.ID, err)
		}
	}
}

func (w *OrderWorker) loop(ctx context.Context, id int, queues []string, handler func(context.Context, queueTask) error) {
	log.Printf("order worker %d consuming %s", id, strings.Join(queues, ","))
	if w.workerID == "" {
		w.acceptNew.Store(true)
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var task queueTask
		ok, err := w.cache.PopJSONFrom(ctx, queues, 5*time.Second, &task)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("order worker pop %s: %v", strings.Join(queues, ","), err)
			continue
		}
		if !ok {
			continue
		}
		if !w.acceptNew.Load() {
			if err := w.cache.PushJSON(ctx, pausedRequeueKey(queues, task), task); err != nil {
				log.Printf("order worker pause requeue task %+v: %v", task, err)
			}
			time.Sleep(2 * time.Second)
			continue
		}
		w.running.Add(1)
		w.currentID.Store(uint64(task.ID))
		if err := handler(ctx, task); err != nil {
			log.Printf("order worker handle %s task %+v: %v", strings.Join(queues, ","), task, err)
		}
		w.currentID.Store(0)
		w.running.Add(-1)
	}
}

func pausedRequeueKey(queues []string, task queueTask) string {
	if len(queues) > 0 && (queues[0] == queue.OrderRefreshFlash || queues[0] == queue.OrderRefresh) {
		return queue.RefreshKey(task.FlashMode)
	}
	return queue.SubmitKey(task.FlashMode)
}

func (w *OrderWorker) processSubmit(ctx context.Context, task queueTask) error {
	route, err := w.loadOrderRoute(ctx, task.ID)
	if err != nil {
		if route.Order.ID != 0 {
			return w.failOrder(ctx, route.Order, err.Error())
		}
		return err
	}
	order := route.Order
	if !w.shouldProcessTask(order, task) {
		return nil
	}
	if isPluginOrder(order) {
		return w.processPluginSubmit(ctx, order, route.Plugin)
	}
	connector := route.Connector
	if connector.Status != "active" {
		return w.failOrder(ctx, order, "connector disabled")
	}
	if !connector.OrderSyncEnabled {
		return w.failOrder(ctx, order, "connector order sync disabled")
	}
	if err := w.markProcessing(ctx, &order); err != nil {
		return err
	}

	response, err := callConnector(ctx, connector, "/orders", order)
	if err != nil {
		return w.retryOrFail(ctx, order, queue.SubmitKey(order.FlashMode), err.Error())
	}
	return w.applyConnectorResponse(ctx, order, response, "submitted")
}

func (w *OrderWorker) processRefresh(ctx context.Context, task queueTask) error {
	route, err := w.loadOrderRoute(ctx, task.ID)
	if err != nil {
		if route.Order.ID != 0 {
			return w.failOrder(ctx, route.Order, err.Error())
		}
		return err
	}
	order := route.Order
	if !w.shouldProcessTask(order, task) {
		return nil
	}
	if isPluginOrder(order) {
		return w.processPluginRefresh(ctx, order, route.Plugin, task.Action)
	}
	connector := route.Connector
	if connector.Status != "active" {
		return w.failOrder(ctx, order, "connector disabled")
	}
	if !connector.OrderSyncEnabled {
		return w.failOrder(ctx, order, "connector order sync disabled")
	}
	if err := w.markProcessing(ctx, &order); err != nil {
		return err
	}

	path := "/orders/refresh"
	if connectoradapter.Is29WKKind(connector.Kind) && strings.EqualFold(task.Action, "budan") {
		path = "/orders/budan"
	}
	response, err := callConnector(ctx, connector, path, order)
	if err != nil {
		return w.retryOrFail(ctx, order, queue.RefreshKey(order.FlashMode), err.Error(), task.Action)
	}
	return w.applyConnectorResponse(ctx, order, response, "refresh")
}

func (w *OrderWorker) processPluginSubmit(ctx context.Context, order models.Order, plugin models.PlatformPlugin) error {
	if plugin.Status != "active" {
		return w.failOrder(ctx, order, "platform plugin disabled")
	}
	if !plugin.SupportsSubmit {
		return w.failOrder(ctx, order, "platform plugin submit disabled")
	}
	if !w.platforms.Has(plugin.Code) {
		return w.failOrder(ctx, order, "platform plugin adapter is not installed")
	}
	if blocked, reason, err := w.pluginCapacityBlocked(ctx, order, plugin); err != nil {
		return w.retryOrFail(ctx, order, queue.SubmitKey(order.FlashMode), err.Error())
	} else if blocked {
		return w.deferPluginOrder(ctx, order, queue.SubmitKey(order.FlashMode), reason)
	}
	if err := w.markProcessing(ctx, &order); err != nil {
		return err
	}
	proxy, hasProxy, err := w.leaseProxy(ctx)
	if err != nil {
		return w.retryOrFail(ctx, order, queue.SubmitKey(order.FlashMode), "lease proxy failed: "+err.Error())
	}
	if hasProxy {
		order.ProxyID = proxy.ID
		_ = w.repos.Orders.Update(ctx, order.ID, map[string]any{"proxy_id": proxy.ID})
	}
	_ = w.repos.Orders.Update(ctx, order.ID, map[string]any{"worker_id": w.workerID})
	w.logPluginProgress(ctx, order, "0%", "直跑插件开始执行: "+plugin.Code)

	response, runErr := w.platforms.Submit(ctx, plugin.Code, platforms.OrderInput{
		Order:    order,
		Plugin:   plugin,
		ProxyURL: proxy.ProxyURL,
		Action:   "submit",
		Log: func(progress string, message string) {
			w.logPluginProgress(ctx, order, progress, message)
		},
	})
	w.releaseProxy(ctx, proxy.ID, runErr == nil, runErr)
	if runErr != nil {
		return w.retryOrFail(ctx, order, queue.SubmitKey(order.FlashMode), runErr.Error())
	}
	return w.applyPluginResponse(ctx, order, response, "submitted")
}

func (w *OrderWorker) processPluginRefresh(ctx context.Context, order models.Order, plugin models.PlatformPlugin, action string) error {
	if plugin.Status != "active" {
		return w.failOrder(ctx, order, "platform plugin disabled")
	}
	if !plugin.SupportsRefresh {
		return w.failOrder(ctx, order, "platform plugin refresh disabled")
	}
	if !w.platforms.Has(plugin.Code) {
		return w.failOrder(ctx, order, "platform plugin adapter is not installed")
	}
	if blocked, reason, err := w.pluginCapacityBlocked(ctx, order, plugin); err != nil {
		return w.retryOrFail(ctx, order, queue.RefreshKey(order.FlashMode), err.Error(), action)
	} else if blocked {
		return w.deferPluginOrder(ctx, order, queue.RefreshKey(order.FlashMode), reason, action)
	}
	if err := w.markProcessing(ctx, &order); err != nil {
		return err
	}
	proxy, hasProxy, err := w.leaseProxy(ctx)
	if err != nil {
		return w.retryOrFail(ctx, order, queue.RefreshKey(order.FlashMode), "lease proxy failed: "+err.Error(), action)
	}
	if hasProxy {
		order.ProxyID = proxy.ID
		_ = w.repos.Orders.Update(ctx, order.ID, map[string]any{"proxy_id": proxy.ID})
	}
	_ = w.repos.Orders.Update(ctx, order.ID, map[string]any{"worker_id": w.workerID})
	w.logPluginProgress(ctx, order, order.Progress, "直跑插件开始刷新: "+plugin.Code)

	response, runErr := w.platforms.Refresh(ctx, plugin.Code, platforms.OrderInput{
		Order:    order,
		Plugin:   plugin,
		ProxyURL: proxy.ProxyURL,
		Action:   strings.TrimSpace(action),
		Log: func(progress string, message string) {
			w.logPluginProgress(ctx, order, progress, message)
		},
	})
	w.releaseProxy(ctx, proxy.ID, runErr == nil, runErr)
	if runErr != nil {
		return w.retryOrFail(ctx, order, queue.RefreshKey(order.FlashMode), runErr.Error(), action)
	}
	return w.applyPluginResponse(ctx, order, response, "refresh")
}

func (w *OrderWorker) leaseProxy(ctx context.Context) (models.WorkerProxy, bool, error) {
	if w == nil || w.repos == nil || w.repos.WorkerProxies == nil {
		return models.WorkerProxy{}, false, nil
	}
	return w.repos.WorkerProxies.Lease(ctx)
}

func (w *OrderWorker) pluginCapacityBlocked(ctx context.Context, order models.Order, plugin models.PlatformPlugin) (bool, string, error) {
	if w == nil || w.repos == nil || w.repos.Orders == nil {
		return false, "", nil
	}
	limit := plugin.MaxConcurrency
	if limit < 1 {
		limit = 1
	}
	running, err := w.repos.Orders.CountProcessingPlugin(ctx, plugin.Code, "", order.ID)
	if err != nil {
		return false, "", fmt.Errorf("check plugin concurrency: %w", err)
	}
	if running >= int64(limit) {
		return true, fmt.Sprintf("插件 %s 并发已满，等待执行", plugin.Code), nil
	}
	if plugin.AccountSerial && strings.TrimSpace(order.Account) != "" {
		accountRunning, err := w.repos.Orders.CountProcessingPlugin(ctx, plugin.Code, order.Account, order.ID)
		if err != nil {
			return false, "", fmt.Errorf("check plugin account concurrency: %w", err)
		}
		if accountRunning > 0 {
			return true, fmt.Sprintf("账号 %s 已有同插件任务执行中，等待串行执行", order.Account), nil
		}
	}
	return false, "", nil
}

func (w *OrderWorker) deferPluginOrder(ctx context.Context, order models.Order, queueKey string, reason string, action ...string) error {
	if w == nil || w.cache == nil {
		return errors.New("order queue unavailable")
	}
	if err := w.repos.Orders.Update(ctx, order.ID, map[string]any{
		"status":         "queued",
		"docking_status": requeuedDockingStatus(queueKey),
		"remarks":        truncate(reason, 480),
	}); err != nil {
		return err
	}
	w.invalidateDashboard(ctx)
	timer := time.NewTimer(2 * time.Second)
	select {
	case <-ctx.Done():
		timer.Stop()
		return ctx.Err()
	case <-timer.C:
	}
	return w.cache.PushJSON(ctx, queueKey, buildRetryTask(order, action...))
}

func (w *OrderWorker) releaseProxy(ctx context.Context, proxyID uint, success bool, runErr error) {
	if proxyID == 0 || w == nil || w.repos == nil || w.repos.WorkerProxies == nil {
		return
	}
	resultText := ""
	if runErr != nil {
		resultText = runErr.Error()
	}
	if err := w.repos.WorkerProxies.Release(ctx, proxyID, success, resultText); err != nil {
		log.Printf("order worker release proxy %d: %v", proxyID, err)
	}
}

func (w *OrderWorker) applyPluginResponse(ctx context.Context, order models.Order, response platforms.OrderResponse, action string) error {
	status := normalizeConnectorStatus(response.Status)
	values := map[string]any{
		"status":         status,
		"docking_status": "plugin_sent",
		"worker_id":      w.workerID,
	}
	if response.RemoteOrderID != "" {
		values["remote_order_id"] = response.RemoteOrderID
	}
	if response.Progress != "" {
		values["progress"] = response.Progress
	}
	if response.Remarks != "" {
		values["remarks"] = response.Remarks
	}
	if err := w.repos.Orders.Update(ctx, order.ID, values); err != nil {
		return err
	}
	w.invalidateDashboard(ctx)
	return w.logOrder(ctx, order, "plugin_"+action, "platform plugin accepted order")
}

func (w *OrderWorker) logPluginProgress(ctx context.Context, order models.Order, progress string, message string) {
	message = strings.TrimSpace(message)
	if message == "" || w == nil || w.repos == nil || w.repos.OrderEvents == nil {
		return
	}
	_ = w.repos.OrderEvents.Create(ctx, &models.OrderEvent{
		OrderID:       order.ID,
		UserID:        order.UserID,
		Level:         "info",
		Source:        "platform_plugin",
		EventType:     "plugin_progress",
		Content:       truncate(message, 1000),
		Progress:      truncate(progress, 160),
		VisibleToUser: true,
		CreatedAt:     time.Now(),
	})
}

func (w *OrderWorker) loadOrderRoute(ctx context.Context, orderID uint) (orderRoute, error) {
	order, err := w.repos.Orders.Find(ctx, orderID)
	if err != nil {
		return orderRoute{}, err
	}
	if isPluginOrder(order) {
		if strings.TrimSpace(order.PluginCode) == "" {
			return orderRoute{Order: order}, fmt.Errorf("order %d has no plugin code", order.ID)
		}
		if w.repos.PlatformPlugins == nil {
			return orderRoute{Order: order}, fmt.Errorf("platform plugin repository unavailable")
		}
		plugin, err := w.repos.PlatformPlugins.Find(ctx, order.PluginCode)
		if err != nil {
			return orderRoute{Order: order}, err
		}
		return orderRoute{Order: order, Plugin: plugin}, nil
	}
	if order.ConnectorID == 0 {
		return orderRoute{Order: order}, fmt.Errorf("order %d has no connector", order.ID)
	}
	connector, err := w.repos.Connectors.Find(ctx, order.ConnectorID)
	if err != nil {
		return orderRoute{Order: order}, err
	}
	return orderRoute{Order: order, Connector: connector}, nil
}

func isPluginOrder(order models.Order) bool {
	return strings.EqualFold(strings.TrimSpace(order.ExecutionMode), "plugin") || strings.TrimSpace(order.PluginCode) != ""
}

func (w *OrderWorker) shouldProcessTask(order models.Order, task queueTask) bool {
	if order.Status == "queued" {
		return true
	}
	return order.Status == "processing" && !task.RecoveredAt.IsZero()
}

func (w *OrderWorker) markProcessing(ctx context.Context, order *models.Order) error {
	if order.Status == "processing" {
		return nil
	}
	if err := w.repos.Orders.Update(ctx, order.ID, map[string]any{
		"status": "processing",
	}); err != nil {
		return err
	}
	order.Status = "processing"
	w.invalidateDashboard(ctx)
	return nil
}

func (w *OrderWorker) applyConnectorResponse(ctx context.Context, order models.Order, response connectorResponse, action string) error {
	status := normalizeConnectorStatus(response.Status)
	values := map[string]any{
		"status":         status,
		"docking_status": "sent",
	}
	if response.RemoteOrderID != "" {
		values["remote_order_id"] = response.RemoteOrderID
	}
	if response.Progress != "" {
		values["progress"] = response.Progress
	}
	if response.Remarks != "" {
		values["remarks"] = response.Remarks
	}
	if err := w.repos.Orders.Update(ctx, order.ID, values); err != nil {
		return err
	}
	w.invalidateDashboard(ctx)
	return w.logOrder(ctx, order, "order_"+action, "connector accepted order")
}

func (w *OrderWorker) failOrder(ctx context.Context, order models.Order, reason string) error {
	if err := w.repos.Orders.Update(ctx, order.ID, map[string]any{
		"status":         "failed",
		"docking_status": "failed",
		"retry_count":    order.RetryCount,
		"remarks":        truncate(reason, 480),
	}); err != nil {
		return err
	}
	w.invalidateDashboard(ctx)
	return w.logOrder(ctx, order, "order_failed", reason)
}

func (w *OrderWorker) retryOrFail(ctx context.Context, order models.Order, queueKey string, reason string, action ...string) error {
	nextRetryCount := order.RetryCount + 1
	order.RetryCount = nextRetryCount
	if nextRetryCount >= w.cfg.MaxAttempts {
		return w.failOrder(ctx, order, reason)
	}

	if err := w.requeueOrder(ctx, order, queueKey, reason, action...); err != nil {
		if retryableContextError(err) {
			return err
		}
		order.Remarks = reason
		return w.failOrder(ctx, order, fmt.Sprintf("retry queue unavailable: %v", err))
	}
	return w.logOrder(ctx, order, "order_retry", reason)
}

func (w *OrderWorker) requeueOrder(ctx context.Context, order models.Order, queueKey string, reason string, action ...string) error {
	if err := w.repos.Orders.Update(ctx, order.ID, map[string]any{
		"status":         "queued",
		"docking_status": requeuedDockingStatus(queueKey),
		"retry_count":    order.RetryCount,
		"remarks":        truncate(reason, 480),
	}); err != nil {
		return err
	}
	w.invalidateDashboard(ctx)

	if delay := time.Duration(w.cfg.RetryDelayMS) * time.Millisecond; delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return w.cache.PushJSON(ctx, queueKey, buildRetryTask(order, action...))
}

func requeuedDockingStatus(queueKey string) string {
	if queueKey == queue.OrderRefresh || queueKey == queue.OrderRefreshFlash {
		return "refresh_requested"
	}
	return "pending"
}

func buildRetryTask(order models.Order, action ...string) map[string]any {
	task := map[string]any{
		"id":          order.ID,
		"connectorId": order.ConnectorID,
		"flashMode":   order.FlashMode,
		"retryCount":  order.RetryCount,
		"requestedAt": time.Now().UTC(),
	}
	if len(action) > 0 && strings.TrimSpace(action[0]) != "" {
		task["action"] = strings.TrimSpace(action[0])
	}
	return task
}

func retryableContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func (w *OrderWorker) invalidateDashboard(ctx context.Context) {
	if w.cache == nil {
		return
	}
	_ = w.cache.Delete(ctx, "dashboard:stats")
}

func (w *OrderWorker) logOrder(ctx context.Context, order models.Order, logType string, text string) error {
	if w.repos.OrderEvents != nil {
		_ = w.repos.OrderEvents.Create(ctx, &models.OrderEvent{
			OrderID:       order.ID,
			UserID:        order.UserID,
			Level:         orderEventLevel(logType),
			Source:        "order_worker",
			EventType:     logType,
			Content:       truncate(text, 1000),
			Progress:      order.Progress,
			VisibleToUser: true,
			CreatedAt:     time.Now(),
		})
	}
	return w.repos.Logs.Create(ctx, &models.OperationLog{
		UserID: order.UserID,
		Type:   logType,
		Text:   fmt.Sprintf("order=%d %s", order.ID, truncate(text, 460)),
		Amount: order.Fee,
	})
}

func orderEventLevel(logType string) string {
	if strings.Contains(logType, "failed") {
		return "error"
	}
	if strings.Contains(logType, "retry") {
		return "warning"
	}
	return "info"
}

func callConnector(ctx context.Context, connector models.Connector, path string, order models.Order) (connectorResponse, error) {
	if strings.TrimSpace(connector.BaseURL) == "" {
		return connectorResponse{}, fmt.Errorf("connector base URL is empty")
	}
	if connectoradapter.Is29WKKind(connector.Kind) {
		response, err := call29WKConnector(ctx, connector, path, order)
		if err != nil {
			return connectorResponse{}, err
		}
		return connectorResponse{
			RemoteOrderID: response.RemoteOrderID,
			Status:        response.Status,
			Progress:      response.Progress,
			Remarks:       response.Remarks,
		}, nil
	}
	timeout := time.Duration(connector.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 8 * time.Second
	}

	payload, err := json.Marshal(map[string]any{
		"order": order,
		"auth": map[string]string{
			"appKey":    connector.AppKey,
			"appSecret": connector.AppSecret,
		},
		"credentials": map[string]string{
			"account":  order.Account,
			"password": order.AccountPassword,
		},
		"options": map[string]any{
			"flashMode": order.FlashMode,
			"priority":  orderPriority(order),
		},
	})
	if err != nil {
		return connectorResponse{}, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	url := strings.TrimRight(connector.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return connectorResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return connectorResponse{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return connectorResponse{}, fmt.Errorf("connector returned %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	if len(body) == 0 {
		return connectorResponse{Status: "processing"}, nil
	}
	var parsed connectorResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return connectorResponse{}, fmt.Errorf("decode connector response: %w", err)
	}
	return parsed, nil
}

func call29WKConnector(ctx context.Context, connector models.Connector, path string, order models.Order) (connectoradapter.OrderResponse, error) {
	switch path {
	case "/orders":
		return connectoradapter.Submit29WKOrder(ctx, connector, order)
	case "/orders/refresh":
		return connectoradapter.Refresh29WKOrder(ctx, connector, order)
	case "/orders/budan":
		return connectoradapter.Budan29WKOrder(ctx, connector, order)
	default:
		return connectoradapter.OrderResponse{}, fmt.Errorf("unsupported 29wk connector path %s", path)
	}
}

func orderPriority(order models.Order) string {
	if order.FlashMode {
		return "flash"
	}
	return "normal"
}

func normalizeConnectorStatus(status string) string {
	normalized := strings.TrimSpace(status)
	switch normalized {
	case "pending", "queued", "processing", "done", "failed", "cancelled", "refunded":
		return normalized
	default:
		return "processing"
	}
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"dw0rdwk/backend/internal/cache"
	"dw0rdwk/backend/internal/config"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/queue"
	"dw0rdwk/backend/internal/repository"
)

func TestDashboardCacheTTL(t *testing.T) {
	tests := []struct {
		name     string
		settings map[string]string
		want     time.Duration
	}{
		{name: "missing", settings: nil, want: 30 * time.Second},
		{name: "invalid", settings: map[string]string{"dashboard_cache_seconds": "abc"}, want: 30 * time.Second},
		{name: "below minimum", settings: map[string]string{"dashboard_cache_seconds": "1"}, want: 5 * time.Second},
		{name: "valid", settings: map[string]string{"dashboard_cache_seconds": "120"}, want: 120 * time.Second},
		{name: "above maximum", settings: map[string]string{"dashboard_cache_seconds": "9999"}, want: 3600 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dashboardCacheTTL(tt.settings); got != tt.want {
				t.Fatalf("dashboardCacheTTL() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPrepareSubmittedOrderForcesQueueState(t *testing.T) {
	order := models.Order{
		Status:        "pending",
		DockingStatus: "manual",
		FlashMode:     true,
	}

	prepareSubmittedOrder(&order)

	if order.Status != "queued" {
		t.Fatalf("Status = %q, want queued", order.Status)
	}
	if order.DockingStatus != "pending" {
		t.Fatalf("DockingStatus = %q, want pending", order.DockingStatus)
	}
	if !order.FlashMode {
		t.Fatal("FlashMode should be preserved")
	}
}

func TestPrepareResubmittedOrderResetsProgress(t *testing.T) {
	order := models.Order{
		Status:        "failed",
		DockingStatus: "queue_failed",
		Progress:      "80%",
		RetryCount:    3,
		Remarks:       "old error",
		FlashMode:     true,
	}

	prepareResubmittedOrder(&order)

	if order.Status != "queued" {
		t.Fatalf("Status = %q, want queued", order.Status)
	}
	if order.DockingStatus != "pending" {
		t.Fatalf("DockingStatus = %q, want pending", order.DockingStatus)
	}
	if order.Progress != "" {
		t.Fatalf("Progress = %q, want empty", order.Progress)
	}
	if order.Remarks != "" {
		t.Fatalf("Remarks = %q, want empty", order.Remarks)
	}
	if order.RetryCount != 0 {
		t.Fatalf("RetryCount = %d, want 0", order.RetryCount)
	}
	if !order.FlashMode {
		t.Fatal("FlashMode should be preserved")
	}
}

func TestPopulateQueueLengthsDoesNotTrustCachedQueueValues(t *testing.T) {
	service := DashboardService{
		cache: cache.Open(context.Background(), config.RedisConfig{Enabled: false}),
	}
	stats := repository.DashboardStats{
		QueueOrders:       99,
		QueueRefreshes:    88,
		QueueSubmit:       77,
		QueueSubmitFlash:  66,
		QueueRefresh:      55,
		QueueRefreshFlash: 44,
	}

	service.populateQueueLengths(context.Background(), &stats)

	if stats.QueueOrders != 0 {
		t.Fatalf("QueueOrders = %d, want 0", stats.QueueOrders)
	}
	if stats.QueueRefreshes != 0 {
		t.Fatalf("QueueRefreshes = %d, want 0", stats.QueueRefreshes)
	}
	if stats.QueueSubmit != 0 {
		t.Fatalf("QueueSubmit = %d, want 0", stats.QueueSubmit)
	}
	if stats.QueueSubmitFlash != 0 {
		t.Fatalf("QueueSubmitFlash = %d, want 0", stats.QueueSubmitFlash)
	}
	if stats.QueueRefresh != 0 {
		t.Fatalf("QueueRefresh = %d, want 0", stats.QueueRefresh)
	}
	if stats.QueueRefreshFlash != 0 {
		t.Fatalf("QueueRefreshFlash = %d, want 0", stats.QueueRefreshFlash)
	}
}

func TestRecoverQueuesReturnsUnavailableWhenRedisDisabled(t *testing.T) {
	service := OrderService{
		cache: cache.Open(context.Background(), config.RedisConfig{Enabled: false}),
	}

	_, err := service.RecoverQueues(context.Background(), 500)
	var dependencyErr DependencyError
	if !errors.As(err, &dependencyErr) {
		t.Fatalf("RecoverQueues() error = %v, want DependencyError", err)
	}
	if dependencyErr.Message != "order queue unavailable" {
		t.Fatalf("DependencyError.Message = %q", dependencyErr.Message)
	}
}

func TestRecoveryQueueTaskPreservesQueueFields(t *testing.T) {
	order := models.Order{ID: 10, ConnectorID: 20, FlashMode: true, RetryCount: 2, DockingStatus: "refresh_requested"}

	task := recoveryQueueTask(order)

	if task["id"] != order.ID {
		t.Fatalf("id = %#v, want %d", task["id"], order.ID)
	}
	if task["connectorId"] != order.ConnectorID {
		t.Fatalf("connectorId = %#v, want %d", task["connectorId"], order.ConnectorID)
	}
	if task["flashMode"] != true {
		t.Fatalf("flashMode = %#v, want true", task["flashMode"])
	}
	if task["retryCount"] != order.RetryCount {
		t.Fatalf("retryCount = %#v, want %d", task["retryCount"], order.RetryCount)
	}
	if task["dockingStatus"] != "refresh_requested" {
		t.Fatalf("dockingStatus = %#v, want refresh_requested", task["dockingStatus"])
	}
	if _, ok := task["recoveredAt"].(time.Time); !ok {
		t.Fatalf("recoveredAt = %#v, want time.Time", task["recoveredAt"])
	}
}

func TestRecoveryQueueKey(t *testing.T) {
	tests := []struct {
		name  string
		order models.Order
		want  string
	}{
		{name: "normal submit", order: models.Order{FlashMode: false, DockingStatus: "pending"}, want: queue.OrderSubmit},
		{name: "flash submit", order: models.Order{FlashMode: true, DockingStatus: "pending"}, want: queue.OrderSubmitFlash},
		{name: "normal refresh", order: models.Order{FlashMode: false, DockingStatus: "refresh_requested"}, want: queue.OrderRefresh},
		{name: "flash refresh", order: models.Order{FlashMode: true, DockingStatus: "refresh_requested"}, want: queue.OrderRefreshFlash},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := recoveryQueueKey(tt.order); got != tt.want {
				t.Fatalf("recoveryQueueKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsOrderCancelable(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{status: "", want: true},
		{status: "pending", want: true},
		{status: "queued", want: true},
		{status: "processing", want: true},
		{status: "done", want: false},
		{status: "failed", want: false},
		{status: "cancelled", want: false},
		{status: "refunded", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := isOrderCancelable(tt.status); got != tt.want {
				t.Fatalf("isOrderCancelable(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestValidateSettings(t *testing.T) {
	values, err := validateSettings(map[string]string{
		"site_name":               "  DW0RDWK  ",
		"dashboard_cache_seconds": "60",
		"order_auto_refresh":      "TRUE",
		"custom_key":              " custom value ",
	})
	if err != nil {
		t.Fatalf("validateSettings() error = %v", err)
	}
	if values["site_name"] != "DW0RDWK" {
		t.Fatalf("site_name = %q", values["site_name"])
	}
	if values["order_auto_refresh"] != "true" {
		t.Fatalf("order_auto_refresh = %q", values["order_auto_refresh"])
	}
	if values["custom_key"] != "custom value" {
		t.Fatalf("custom_key = %q", values["custom_key"])
	}
}

func TestValidateSettingsRejectsInvalidKnownValues(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
	}{
		{name: "empty site name", values: map[string]string{"site_name": " "}},
		{name: "cache seconds not integer", values: map[string]string{"dashboard_cache_seconds": "abc"}},
		{name: "cache seconds below minimum", values: map[string]string{"dashboard_cache_seconds": "4"}},
		{name: "cache seconds above maximum", values: map[string]string{"dashboard_cache_seconds": "3601"}},
		{name: "auto refresh invalid", values: map[string]string{"order_auto_refresh": "yes"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := validateSettings(tt.values); err == nil {
				t.Fatal("validateSettings() accepted invalid setting")
			}
		})
	}
}

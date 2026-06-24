package worker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"dw0rdwk/backend/internal/config"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/queue"
)

func TestCallConnectorSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/orders" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload["order"] == nil || payload["auth"] == nil {
			t.Fatalf("missing order/auth payload: %#v", payload)
		}
		options, ok := payload["options"].(map[string]any)
		if !ok {
			t.Fatalf("missing options payload: %#v", payload)
		}
		if options["flashMode"] != true || options["priority"] != "flash" {
			t.Fatalf("unexpected options payload: %#v", options)
		}
		_ = json.NewEncoder(w).Encode(connectorResponse{
			RemoteOrderID: "remote-1",
			Status:        "processing",
			Progress:      "25%",
			Remarks:       "accepted",
		})
	}))
	defer server.Close()

	response, err := callConnector(context.Background(), models.Connector{
		BaseURL:   server.URL,
		AppKey:    "key",
		AppSecret: "secret",
		TimeoutMS: 1000,
	}, "/orders", models.Order{ID: 10, Account: "student", FlashMode: true})
	if err != nil {
		t.Fatalf("call connector: %v", err)
	}
	if response.RemoteOrderID != "remote-1" || response.Status != "processing" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestCallConnectorRejectsErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad upstream", http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := callConnector(context.Background(), models.Connector{
		BaseURL:   server.URL,
		TimeoutMS: int(time.Second / time.Millisecond),
	}, "/orders", models.Order{})
	if err == nil {
		t.Fatal("expected connector error")
	}
}

func TestInvalidateDashboardDeletesStatsCache(t *testing.T) {
	cache := &fakeWorkerCache{}
	worker := &OrderWorker{cache: cache}

	worker.invalidateDashboard(context.Background())

	if !reflect.DeepEqual(cache.deleted, []string{"dashboard:stats"}) {
		t.Fatalf("deleted keys = %#v", cache.deleted)
	}
}

func TestLoopConsumesFlashQueueBeforeNormalQueue(t *testing.T) {
	cache := &fakeWorkerCache{
		tasks: map[string][]queueTask{
			queue.OrderSubmit:      {{ID: 1, FlashMode: false}},
			queue.OrderSubmitFlash: {{ID: 2, FlashMode: true}},
		},
	}
	worker := &OrderWorker{cache: cache}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var handled []uint
	worker.loop(ctx, 1, queue.SubmitPriorityKeys(), func(_ context.Context, task queueTask) error {
		handled = append(handled, task.ID)
		if len(handled) == 2 {
			cancel()
		}
		return nil
	})

	if !reflect.DeepEqual(handled, []uint{2, 1}) {
		t.Fatalf("handled order = %#v, want flash task before normal task", handled)
	}
	if !reflect.DeepEqual(cache.popKeys, [][]string{queue.SubmitPriorityKeys(), queue.SubmitPriorityKeys()}) {
		t.Fatalf("pop keys = %#v, want submit priority keys", cache.popKeys)
	}
}

func TestNormalizeConnectorStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{name: "empty defaults to processing", status: "", want: "processing"},
		{name: "processing preserved", status: "processing", want: "processing"},
		{name: "done preserved", status: "done", want: "done"},
		{name: "trimmed queued preserved", status: " queued ", want: "queued"},
		{name: "refunded preserved", status: "refunded", want: "refunded"},
		{name: "unknown defaults to processing", status: "completed", want: "processing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeConnectorStatus(tt.status); got != tt.want {
				t.Fatalf("normalizeConnectorStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewOrderWorkerNormalizesRetryConfig(t *testing.T) {
	worker := NewOrderWorker(
		config.WorkerConfig{Enabled: true, Concurrency: 0, MaxAttempts: 0, RetryDelayMS: -1},
		nil,
		nil,
	)

	if worker.cfg.Concurrency != 1 {
		t.Fatalf("Concurrency = %d, want 1", worker.cfg.Concurrency)
	}
	if worker.cfg.MaxAttempts != 1 {
		t.Fatalf("MaxAttempts = %d, want 1", worker.cfg.MaxAttempts)
	}
	if worker.cfg.RetryDelayMS != 0 {
		t.Fatalf("RetryDelayMS = %d, want 0", worker.cfg.RetryDelayMS)
	}
}

func TestBuildRetryTaskPreservesPriorityFields(t *testing.T) {
	order := models.Order{ID: 10, ConnectorID: 20, FlashMode: true, RetryCount: 2}

	task := buildRetryTask(order)

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
	if _, ok := task["requestedAt"].(time.Time); !ok {
		t.Fatalf("requestedAt = %#v, want time.Time", task["requestedAt"])
	}
}

func TestRequeuedDockingStatus(t *testing.T) {
	tests := []struct {
		name     string
		queueKey string
		want     string
	}{
		{name: "normal submit", queueKey: queue.OrderSubmit, want: "pending"},
		{name: "flash submit", queueKey: queue.OrderSubmitFlash, want: "pending"},
		{name: "normal refresh", queueKey: queue.OrderRefresh, want: "refresh_requested"},
		{name: "flash refresh", queueKey: queue.OrderRefreshFlash, want: "refresh_requested"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := requeuedDockingStatus(tt.queueKey); got != tt.want {
				t.Fatalf("requeuedDockingStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShouldProcessTask(t *testing.T) {
	worker := &OrderWorker{}
	tests := []struct {
		name  string
		order models.Order
		task  queueTask
		want  bool
	}{
		{name: "queued normal task", order: models.Order{Status: "queued"}, want: true},
		{name: "processing normal task skipped", order: models.Order{Status: "processing"}, want: false},
		{
			name:  "processing recovered task",
			order: models.Order{Status: "processing"},
			task:  queueTask{RecoveredAt: time.Now()},
			want:  true,
		},
		{name: "done skipped", order: models.Order{Status: "done"}, want: false},
		{name: "failed skipped", order: models.Order{Status: "failed"}, want: false},
		{name: "cancelled skipped", order: models.Order{Status: "cancelled"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := worker.shouldProcessTask(tt.order, tt.task); got != tt.want {
				t.Fatalf("shouldProcessTask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryableContextError(t *testing.T) {
	if !retryableContextError(context.Canceled) {
		t.Fatal("context.Canceled should be detected")
	}
	if !retryableContextError(context.DeadlineExceeded) {
		t.Fatal("context.DeadlineExceeded should be detected")
	}
	if retryableContextError(errors.New("redis down")) {
		t.Fatal("non-context error should not be detected")
	}
}

type fakeWorkerCache struct {
	mu      sync.Mutex
	deleted []string
	tasks   map[string][]queueTask
	popKeys [][]string
}

func (f *fakeWorkerCache) Enabled() bool {
	return true
}

func (f *fakeWorkerCache) PopJSON(context.Context, string, time.Duration, any) (bool, error) {
	return false, nil
}

func (f *fakeWorkerCache) PushJSON(context.Context, string, any) error {
	return nil
}

func (f *fakeWorkerCache) PopJSONFrom(_ context.Context, keys []string, _ time.Duration, dest any) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.popKeys = append(f.popKeys, append([]string(nil), keys...))
	taskDest, ok := dest.(*queueTask)
	if !ok {
		return false, nil
	}
	for _, key := range keys {
		tasks := f.tasks[key]
		if len(tasks) == 0 {
			continue
		}
		*taskDest = tasks[0]
		f.tasks[key] = tasks[1:]
		return true, nil
	}
	return false, nil
}

func (f *fakeWorkerCache) Delete(_ context.Context, keys ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.deleted = append(f.deleted, keys...)
	return nil
}

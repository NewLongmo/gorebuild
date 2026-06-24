package platforms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"dw0rdwk/backend/internal/models"
)

var ErrAdapterNotFound = errors.New("platform adapter not found")
var ErrUnsupportedAction = errors.New("platform action is not supported")

type CourseQueryInput struct {
	Class    models.CourseClass
	School   string
	Account  string
	Password string
	Type     string
}

type CourseCandidate struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Raw  map[string]any `json:"raw"`
}

type CourseQueryResult struct {
	Raw        map[string]any    `json:"raw"`
	Candidates []CourseCandidate `json:"candidates"`
	UserInfo   string            `json:"userinfo"`
}

type OrderInput struct {
	Order    models.Order
	Plugin   models.PlatformPlugin
	ProxyURL string
	Action   string
	Log      func(progress string, message string)
}

type OrderResponse struct {
	RemoteOrderID string `json:"remoteOrderId"`
	Status        string `json:"status"`
	Progress      string `json:"progress"`
	Remarks       string `json:"remarks"`
}

type Adapter interface {
	Code() string
	Query(ctx context.Context, input CourseQueryInput) (CourseQueryResult, error)
	Submit(ctx context.Context, input OrderInput) (OrderResponse, error)
	Refresh(ctx context.Context, input OrderInput) (OrderResponse, error)
}

type Registry struct {
	adapters map[string]Adapter
}

func NewRegistry(adapters ...Adapter) *Registry {
	registry := &Registry{adapters: map[string]Adapter{}}
	for _, adapter := range adapters {
		registry.Register(adapter)
	}
	return registry
}

func (r *Registry) Register(adapter Adapter) {
	if r == nil || adapter == nil {
		return
	}
	code := normalizeCode(adapter.Code())
	if code == "" {
		return
	}
	r.adapters[code] = adapter
}

func (r *Registry) Query(ctx context.Context, code string, input CourseQueryInput) (CourseQueryResult, error) {
	adapter, err := r.adapter(code)
	if err != nil {
		return CourseQueryResult{}, err
	}
	return adapter.Query(ctx, input)
}

func (r *Registry) Submit(ctx context.Context, code string, input OrderInput) (OrderResponse, error) {
	adapter, err := r.adapter(code)
	if err != nil {
		return OrderResponse{}, err
	}
	return adapter.Submit(ctx, input)
}

func (r *Registry) Refresh(ctx context.Context, code string, input OrderInput) (OrderResponse, error) {
	adapter, err := r.adapter(code)
	if err != nil {
		return OrderResponse{}, err
	}
	return adapter.Refresh(ctx, input)
}

func (r *Registry) Has(code string) bool {
	if r == nil {
		return false
	}
	_, ok := r.adapters[normalizeCode(code)]
	return ok
}

func (r *Registry) adapter(code string) (Adapter, error) {
	if r == nil {
		return nil, ErrAdapterNotFound
	}
	adapter, ok := r.adapters[normalizeCode(code)]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrAdapterNotFound, strings.TrimSpace(code))
	}
	return adapter, nil
}

func DefaultRegistry() *Registry {
	return NewRegistry(manualAdapter{})
}

func DefaultPluginSeeds() []models.PlatformPlugin {
	return []models.PlatformPlugin{
		{
			Code:            "manual",
			Name:            "手动直跑",
			Description:     "内置占位插件：接收订单并记录执行日志，用于验证直跑插件链路和人工处理。",
			Status:          "active",
			SortOrder:       10,
			SupportsQuery:   true,
			SupportsSubmit:  true,
			SupportsRefresh: true,
			MaxConcurrency:  2,
			AccountSerial:   true,
			ConfigJSON:      `{"autoComplete":false}`,
		},
		{Code: "icourse163", Name: "中国大学MOOC", Description: "预留平台适配器，待迁移 Python 逻辑后启用。", Status: "disabled", SortOrder: 100, SupportsQuery: true, SupportsSubmit: true, SupportsRefresh: true, MaxConcurrency: 1, AccountSerial: true, ConfigJSON: "{}"},
		{Code: "xueqiplus", Name: "学起Plus", Description: "预留平台适配器，待迁移 Python 逻辑后启用。", Status: "disabled", SortOrder: 110, SupportsQuery: true, SupportsSubmit: true, SupportsRefresh: true, MaxConcurrency: 1, AccountSerial: false, ConfigJSON: "{}"},
		{Code: "gxela", Name: "高校继续教育平台", Description: "预留平台适配器，待迁移 Python 逻辑后启用。", Status: "disabled", SortOrder: 120, SupportsQuery: true, SupportsSubmit: true, SupportsRefresh: true, MaxConcurrency: 1, AccountSerial: true, ConfigJSON: "{}"},
		{Code: "qingshuxuetang", Name: "青书学堂", Description: "预留平台适配器，待迁移 Python 逻辑后启用。", Status: "disabled", SortOrder: 130, SupportsQuery: true, SupportsSubmit: true, SupportsRefresh: true, MaxConcurrency: 1, AccountSerial: true, ConfigJSON: "{}"},
		{Code: "jxjypt", Name: "继续教育平台", Description: "预留平台适配器，待迁移 Python 逻辑后启用。", Status: "disabled", SortOrder: 140, SupportsQuery: true, SupportsSubmit: true, SupportsRefresh: true, MaxConcurrency: 1, AccountSerial: true, ConfigJSON: "{}"},
		{Code: "ttcdw", Name: "天天成长", Description: "预留平台适配器，待迁移 Python 逻辑后启用。", Status: "disabled", SortOrder: 150, SupportsQuery: true, SupportsSubmit: true, SupportsRefresh: true, MaxConcurrency: 1, AccountSerial: true, ConfigJSON: "{}"},
		{Code: "welearn", Name: "WeLearn", Description: "预留平台适配器，待迁移 Python 逻辑后启用。", Status: "disabled", SortOrder: 160, SupportsQuery: true, SupportsSubmit: true, SupportsRefresh: true, MaxConcurrency: 1, AccountSerial: true, ConfigJSON: "{}"},
	}
}

func PluginCodeFromPlatform(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(value), "plugin:") {
		code := normalizeCode(value[len("plugin:"):])
		return code, code != ""
	}
	return "", false
}

func PlatformRef(code string) string {
	code = normalizeCode(code)
	if code == "" {
		return ""
	}
	return "plugin:" + code
}

func normalizeCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

type manualAdapter struct{}

func (manualAdapter) Code() string {
	return "manual"
}

func (manualAdapter) Query(ctx context.Context, input CourseQueryInput) (CourseQueryResult, error) {
	if err := ctx.Err(); err != nil {
		return CourseQueryResult{}, err
	}
	courseID := strings.TrimSpace(input.Class.DockingCode)
	if courseID == "" {
		courseID = fmt.Sprintf("class-%d", input.Class.ID)
	}
	name := strings.TrimSpace(input.Class.Name)
	if name == "" {
		name = "手动直跑课程"
	}
	raw := map[string]any{
		"plugin":  "manual",
		"classId": input.Class.ID,
		"account": strings.TrimSpace(input.Account),
	}
	return CourseQueryResult{
		Raw: raw,
		Candidates: []CourseCandidate{{
			ID:   courseID,
			Name: name,
			Raw:  raw,
		}},
		UserInfo: strings.TrimSpace(strings.Join([]string{input.School, input.Account}, " ")),
	}, nil
}

func (manualAdapter) Submit(ctx context.Context, input OrderInput) (OrderResponse, error) {
	if err := ctx.Err(); err != nil {
		return OrderResponse{}, err
	}
	if input.Log != nil {
		input.Log("0%", "手动直跑插件已接收订单")
	}
	if manualAutoComplete(input.Plugin.ConfigJSON) {
		if input.Log != nil {
			input.Log("100%", "手动直跑插件已自动完成")
		}
		return OrderResponse{
			RemoteOrderID: fmt.Sprintf("manual-%d", input.Order.ID),
			Status:        "done",
			Progress:      "100%",
			Remarks:       "手动直跑插件已自动完成",
		}, nil
	}
	return OrderResponse{
		RemoteOrderID: fmt.Sprintf("manual-%d", input.Order.ID),
		Status:        "processing",
		Progress:      "0%",
		Remarks:       "手动直跑插件已接收，等待人工或外部执行器完成",
	}, nil
}

func (manualAdapter) Refresh(ctx context.Context, input OrderInput) (OrderResponse, error) {
	if err := ctx.Err(); err != nil {
		return OrderResponse{}, err
	}
	if input.Log != nil {
		input.Log(input.Order.Progress, "手动直跑插件已记录刷新/补刷请求")
	}
	return OrderResponse{
		RemoteOrderID: input.Order.RemoteOrderID,
		Status:        "processing",
		Progress:      input.Order.Progress,
		Remarks:       "手动直跑插件已记录刷新/补刷请求",
	}, nil
}

func manualAutoComplete(raw string) bool {
	var cfg struct {
		AutoComplete bool `json:"autoComplete"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &cfg); err != nil {
		return false
	}
	return cfg.AutoComplete
}

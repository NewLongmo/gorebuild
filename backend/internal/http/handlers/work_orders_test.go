package handlers

import "testing"

func TestValidateWorkOrderPayload(t *testing.T) {
	tests := []struct {
		name    string
		req     workOrderPayload
		wantErr bool
	}{
		{name: "valid", req: workOrderPayload{Category: "订单问题", Title: "订单异常", Content: "订单 1001 没有进度"}},
		{name: "valid attachment", req: workOrderPayload{Category: "订单问题", Title: "订单异常", Content: "订单 1001 没有进度", AttachmentURL: "https://example.test/a.png"}},
		{name: "missing category", req: workOrderPayload{Title: "订单异常", Content: "订单 1001 没有进度"}, wantErr: true},
		{name: "missing title", req: workOrderPayload{Category: "订单问题", Content: "订单 1001 没有进度"}, wantErr: true},
		{name: "missing content", req: workOrderPayload{Category: "订单问题", Title: "订单异常"}, wantErr: true},
		{name: "attachment too long", req: workOrderPayload{Category: "订单问题", Title: "订单异常", Content: "订单 1001 没有进度", AttachmentURL: longString(501)}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkOrderPayload(tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateWorkOrderPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkOrderActionValues(t *testing.T) {
	tests := []struct {
		name       string
		req        workOrderActionPayload
		wantStatus string
		wantFields map[string]any
		wantErr    bool
	}{
		{name: "answer", req: workOrderActionPayload{Action: "answer", Answer: "已处理"}, wantStatus: workOrderStatusAnswered},
		{name: "reject", req: workOrderActionPayload{Action: "reject", Answer: "信息不足"}, wantStatus: workOrderStatusRejected},
		{name: "close", req: workOrderActionPayload{Action: "close"}, wantStatus: workOrderStatusClosed, wantFields: map[string]any{"progress": 100}},
		{name: "ignore", req: workOrderActionPayload{Action: "ignore"}, wantStatus: workOrderStatusIgnored},
		{name: "explicit zero progress", req: workOrderActionPayload{Action: "answer", Answer: "重置进度", Progress: workOrderIntPtr(0)}, wantStatus: workOrderStatusAnswered, wantFields: map[string]any{"progress": 0}},
		{name: "attachment clear", req: workOrderActionPayload{Action: "ignore", AttachmentURL: workOrderStringPtr("")}, wantStatus: workOrderStatusIgnored, wantFields: map[string]any{"attachment_url": ""}},
		{name: "user visible", req: workOrderActionPayload{Action: "ignore", UserVisible: workOrderBoolPtr(false)}, wantStatus: workOrderStatusIgnored, wantFields: map[string]any{"user_visible": false}},
		{name: "answer needs text", req: workOrderActionPayload{Action: "answer"}, wantErr: true},
		{name: "unknown action", req: workOrderActionPayload{Action: "bad"}, wantErr: true},
		{name: "progress invalid", req: workOrderActionPayload{Action: "ignore", Progress: workOrderIntPtr(101)}, wantErr: true},
		{name: "action attachment too long", req: workOrderActionPayload{Action: "ignore", AttachmentURL: workOrderStringPtr(longString(501))}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := workOrderActionValues(tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("workOrderActionValues() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got := values["status"]; got != tt.wantStatus {
				t.Fatalf("status = %v, want %v", got, tt.wantStatus)
			}
			for key, want := range tt.wantFields {
				if got := values[key]; got != want {
					t.Fatalf("%s = %v, want %v", key, got, want)
				}
			}
		})
	}
}

func workOrderIntPtr(value int) *int {
	return &value
}

func workOrderStringPtr(value string) *string {
	return &value
}

func workOrderBoolPtr(value bool) *bool {
	return &value
}

func longString(size int) string {
	data := make([]byte, size)
	for i := range data {
		data[i] = 'a'
	}
	return string(data)
}

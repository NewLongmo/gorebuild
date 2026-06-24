package handlers

import "testing"

func TestValidateConnectorPayload(t *testing.T) {
	validTimeout := 8000

	tests := []struct {
		name    string
		payload connectorPayload
		wantErr bool
	}{
		{
			name: "active connector accepts http URL",
			payload: connectorPayload{
				Name:      "primary",
				BaseURL:   "http://example.com/api",
				Status:    "active",
				TimeoutMS: &validTimeout,
			},
		},
		{
			name: "active connector accepts https URL",
			payload: connectorPayload{
				Name:    "primary",
				BaseURL: "https://api.example.com",
				Status:  "active",
			},
		},
		{
			name: "disabled connector can omit base URL",
			payload: connectorPayload{
				Name:   "manual",
				Status: "disabled",
			},
		},
		{
			name: "invalid status is rejected",
			payload: connectorPayload{
				Name:    "primary",
				BaseURL: "https://api.example.com",
				Status:  "paused",
			},
			wantErr: true,
		},
		{
			name: "name is required",
			payload: connectorPayload{
				BaseURL: "https://api.example.com",
				Status:  "active",
			},
			wantErr: true,
		},
		{
			name: "active connector requires base URL",
			payload: connectorPayload{
				Name:   "primary",
				Status: "active",
			},
			wantErr: true,
		},
		{
			name: "empty status defaults to active and requires base URL",
			payload: connectorPayload{
				Name: "primary",
			},
			wantErr: true,
		},
		{
			name: "ftp URL is rejected",
			payload: connectorPayload{
				Name:    "primary",
				BaseURL: "ftp://example.com",
				Status:  "active",
			},
			wantErr: true,
		},
		{
			name: "relative URL is rejected",
			payload: connectorPayload{
				Name:    "primary",
				BaseURL: "/orders",
				Status:  "active",
			},
			wantErr: true,
		},
		{
			name: "hostless URL is rejected",
			payload: connectorPayload{
				Name:    "primary",
				BaseURL: "https://",
				Status:  "active",
			},
			wantErr: true,
		},
		{
			name: "negative timeout is rejected",
			payload: connectorPayload{
				Name:      "primary",
				BaseURL:   "https://api.example.com",
				Status:    "active",
				TimeoutMS: intPtr(-1),
			},
			wantErr: true,
		},
		{
			name: "timeout over max is rejected",
			payload: connectorPayload{
				Name:      "primary",
				BaseURL:   "https://api.example.com",
				Status:    "active",
				TimeoutMS: intPtr(60001),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConnectorPayload(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateConnectorPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeConnectorTimeout(t *testing.T) {
	tests := []struct {
		name  string
		input *int
		want  int
	}{
		{name: "nil defaults", input: nil, want: 8000},
		{name: "zero defaults", input: intPtr(0), want: 8000},
		{name: "positive preserved", input: intPtr(12000), want: 12000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeConnectorTimeout(tt.input); got != tt.want {
				t.Fatalf("normalizeConnectorTimeout() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestConnectorUpdateValuesAllowClearingNonSecretFields(t *testing.T) {
	values := connectorUpdateValues(connectorPayload{
		Name:      "  backup  ",
		BaseURL:   " ",
		AppKey:    " ",
		AppSecret: " ",
		Kind:      " ",
		Status:    "disabled",
		TimeoutMS: intPtr(0),
	})

	if values["name"] != "backup" {
		t.Fatalf("name = %#v", values["name"])
	}
	if values["base_url"] != "" {
		t.Fatalf("base_url = %#v, want empty string", values["base_url"])
	}
	if values["app_key"] != "" {
		t.Fatalf("app_key = %#v, want empty string", values["app_key"])
	}
	if _, ok := values["app_secret"]; ok {
		t.Fatal("blank app_secret should not overwrite the stored secret")
	}
	if values["kind"] != "generic" {
		t.Fatalf("kind = %#v", values["kind"])
	}
	if values["timeout_ms"] != 8000 {
		t.Fatalf("timeout_ms = %#v", values["timeout_ms"])
	}
}

func intPtr(value int) *int {
	return &value
}

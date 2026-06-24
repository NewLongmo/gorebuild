package handlers

import (
	"io"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"dw0rdwk/backend/internal/models"
	passwordhash "dw0rdwk/backend/internal/password"

	"github.com/gofiber/fiber/v2"
)

func TestAuditFields(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]any
		want   string
	}{
		{name: "empty", values: map[string]any{}, want: "fields=none"},
		{name: "sorted visible fields", values: map[string]any{"status": "active", "account": "demo"}, want: "fields=account,status"},
		{name: "secret only", values: map[string]any{"password_hash": "x", "app_secret": "y"}, want: "fields=secret"},
		{name: "mixed excludes secret", values: map[string]any{"password_hash": "x", "status": "active"}, want: "fields=status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := auditFields(tt.values); got != tt.want {
				t.Fatalf("auditFields() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResetPasswordHash(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPlain string
		wantErr   bool
	}{
		{name: "defaults to legacy reset password", wantPlain: defaultResetPassword},
		{name: "trims custom password", input: " custom-pass ", wantPlain: "custom-pass"},
		{name: "rejects too long password", input: strings.Repeat("a", 161), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plain, hash, err := resetPasswordHash(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resetPasswordHash() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if plain != tt.wantPlain {
				t.Fatalf("plain = %q, want %q", plain, tt.wantPlain)
			}
			if !passwordhash.Verify(hash, plain) {
				t.Fatalf("hash does not verify plain password")
			}
		})
	}
}

func TestQueryBool(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		value, err := queryBool(c, "flashMode")
		if err != nil {
			return err
		}
		if value == nil {
			return c.SendString("nil")
		}
		if *value {
			return c.SendString("true")
		}
		return c.SendString("false")
	})

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantBody   string
	}{
		{name: "missing", wantStatus: 200, wantBody: "nil"},
		{name: "true", query: "?flashMode=true", wantStatus: 200, wantBody: "true"},
		{name: "false", query: "?flashMode=false", wantStatus: 200, wantBody: "false"},
		{name: "invalid", query: "?flashMode=flash", wantStatus: 400, wantBody: "flashMode must be true or false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest("GET", "/"+tt.query, nil))
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if string(body) != tt.wantBody {
				t.Fatalf("Body = %q, want %q", string(body), tt.wantBody)
			}
		})
	}
}

func TestRecoverBatchSize(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(strconv.Itoa(recoverBatchSize(c)))
	})

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{name: "missing uses default", want: "500"},
		{name: "valid", query: "?batchSize=200", want: "200"},
		{name: "invalid uses default", query: "?batchSize=abc", want: "500"},
		{name: "below minimum uses default", query: "?batchSize=0", want: "500"},
		{name: "above maximum clamps", query: "?batchSize=9999", want: "5000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest("GET", "/"+tt.query, nil))
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if string(body) != tt.want {
				t.Fatalf("Body = %q, want %q", string(body), tt.want)
			}
		})
	}
}

func TestNormalizeOrderBatchIDs(t *testing.T) {
	tests := []struct {
		name    string
		ids     []uint
		want    []uint
		wantErr bool
	}{
		{name: "deduplicates preserving order", ids: []uint{3, 1, 3, 2}, want: []uint{3, 1, 2}},
		{name: "empty rejected", ids: nil, wantErr: true},
		{name: "zero rejected", ids: []uint{1, 0}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeOrderBatchIDs(tt.ids)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeOrderBatchIDs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("normalizeOrderBatchIDs() = %#v, want %#v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("normalizeOrderBatchIDs()[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidateUserPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload userPayload
		wantErr bool
	}{
		{name: "defaults accepted", payload: userPayload{}},
		{name: "admin active accepted", payload: userPayload{Role: "admin", Status: "active", Balance: 10, PriceRate: 1}},
		{name: "disabled agent accepted", payload: userPayload{Role: "agent", Status: "disabled"}},
		{name: "negative balance rejected", payload: userPayload{Balance: -0.01}, wantErr: true},
		{name: "negative price rate rejected", payload: userPayload{PriceRate: -1}, wantErr: true},
		{name: "invalid role rejected", payload: userPayload{Role: "owner"}, wantErr: true},
		{name: "invalid status rejected", payload: userPayload{Status: "locked"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUserPayload(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateUserPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateClassPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload classPayload
		wantErr bool
	}{
		{name: "defaults accepted", payload: classPayload{}},
		{name: "online multiply accepted", payload: classPayload{Status: "online", PriceOperator: "*", Price: 1}},
		{name: "offline plus accepted", payload: classPayload{Status: "offline", PriceOperator: "+"}},
		{name: "negative price rejected", payload: classPayload{Price: -1}, wantErr: true},
		{name: "invalid status rejected", payload: classPayload{Status: "hidden"}, wantErr: true},
		{name: "invalid operator rejected", payload: classPayload{PriceOperator: "-"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateClassPayload(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateClassPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateOrderPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload orderPayload
		wantErr bool
	}{
		{name: "defaults accepted", payload: orderPayload{}},
		{name: "queued pending accepted", payload: orderPayload{Status: "queued", DockingStatus: "pending", Fee: 1}},
		{name: "sent processing accepted", payload: orderPayload{Status: "processing", DockingStatus: "sent"}},
		{name: "cancelled accepted", payload: orderPayload{Status: "cancelled", DockingStatus: "cancelled"}},
		{name: "refunded accepted", payload: orderPayload{Status: "refunded", DockingStatus: "refunded"}},
		{name: "negative fee rejected", payload: orderPayload{Fee: -1}, wantErr: true},
		{name: "negative retry rejected", payload: orderPayload{RetryCount: -1}, wantErr: true},
		{name: "negative duration rejected", payload: orderPayload{DurationMinutes: -1}, wantErr: true},
		{name: "invalid status rejected", payload: orderPayload{Status: "unknown"}, wantErr: true},
		{name: "invalid docking status rejected", payload: orderPayload{DockingStatus: "unknown"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOrderPayload(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateOrderPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentPriceWithSpecialPrice(t *testing.T) {
	user := models.User{PriceRate: 2}
	class := models.CourseClass{Price: 10, PriceOperator: "*"}

	tests := []struct {
		name    string
		special *models.SpecialPrice
		want    float64
	}{
		{name: "default multiplier", want: 20},
		{name: "mode 0 subtracts final user price", special: &models.SpecialPrice{Mode: 0, Price: 3}, want: 17},
		{name: "mode 1 subtracts base before multiplier", special: &models.SpecialPrice{Mode: 1, Price: 3}, want: 14},
		{name: "mode 2 fixed price", special: &models.SpecialPrice{Mode: 2, Price: 6.5}, want: 6.5},
		{name: "does not go negative", special: &models.SpecialPrice{Mode: 0, Price: 99}, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := agentPrice(user, class, tt.special); got != tt.want {
				t.Fatalf("agentPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateOrderPasswordPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload agentOrderPasswordPayload
		want    string
		wantErr bool
	}{
		{name: "trims password", payload: agentOrderPasswordPayload{Password: " new-pass "}, want: "new-pass"},
		{name: "empty rejected", payload: agentOrderPasswordPayload{Password: "  "}, wantErr: true},
		{name: "too long rejected", payload: agentOrderPasswordPayload{Password: strings.Repeat("a", 161)}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateOrderPasswordPayload(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateOrderPasswordPayload() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("validateOrderPasswordPayload() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOrderUpdateValues(t *testing.T) {
	values := orderUpdateValues(orderPayload{
		UserID:          10,
		ClassID:         20,
		ConnectorID:     30,
		RemoteOrderID:   "  ",
		Platform:        " web ",
		School:          " ",
		StudentName:     " Alice ",
		Account:         " account-1 ",
		CourseID:        "",
		CourseName:      " Course ",
		Fee:             12.5,
		DockingCode:     " ",
		FlashMode:       true,
		DockingStatus:   "",
		Status:          "",
		Progress:        " 75% ",
		RetryCount:      2,
		Remarks:         "",
		Score:           " 98 ",
		DurationMinutes: 45,
	})

	want := map[string]any{
		"user_id":          uint(10),
		"class_id":         uint(20),
		"connector_id":     uint(30),
		"execution_mode":   "connector",
		"plugin_code":      "",
		"worker_id":        "",
		"proxy_id":         uint(0),
		"remote_order_id":  "",
		"platform":         "web",
		"school":           "",
		"student_name":     "Alice",
		"account":          "account-1",
		"account_password": "",
		"course_id":        "",
		"course_name":      "Course",
		"fee":              12.5,
		"docking_code":     "",
		"flash_mode":       true,
		"docking_status":   "pending",
		"status":           "pending",
		"progress":         "75%",
		"retry_count":      2,
		"remarks":          "",
		"score":            "98",
		"duration_minutes": 45,
	}

	for key, wantValue := range want {
		if gotValue, ok := values[key]; !ok {
			t.Fatalf("orderUpdateValues() missing key %q", key)
		} else if gotValue != wantValue {
			t.Fatalf("orderUpdateValues()[%q] = %#v, want %#v", key, gotValue, wantValue)
		}
	}
	if len(values) != len(want) {
		t.Fatalf("orderUpdateValues() returned %d fields, want %d: %#v", len(values), len(want), values)
	}
}

func TestLegacyOrderRows(t *testing.T) {
	rows := legacyOrderRows([]models.Order{{
		ID:          12,
		Platform:    "平台",
		School:      "学校",
		StudentName: "张三",
		Account:     "student",
		CourseName:  "课程",
		Status:      "processing",
		Progress:    "50%",
		Remarks:     "ok",
	}})
	if len(rows) != 1 {
		t.Fatalf("legacyOrderRows() returned %d rows, want 1", len(rows))
	}
	row := rows[0]
	want := map[string]any{
		"id":      uint(12),
		"ptname":  "平台",
		"school":  "学校",
		"name":    "张三",
		"user":    "student",
		"kcname":  "课程",
		"status":  "processing",
		"process": "50%",
		"remarks": "ok",
	}
	for key, wantValue := range want {
		if row[key] != wantValue {
			t.Fatalf("row[%q] = %#v, want %#v", key, row[key], wantValue)
		}
	}
}

func TestLegacyNormalizeCourseQueryResponse(t *testing.T) {
	body := []byte(`{"code":1,"msg":"ok","data":[{"id":"101","name":"大学语文"},{"courseId":"202","courseName":"高等数学"}]}`)
	result, candidates, err := legacyNormalizeCourseQueryResponse(body)
	if err != nil {
		t.Fatalf("legacyNormalizeCourseQueryResponse() error = %v", err)
	}
	if result["code"] == nil {
		t.Fatal("result code missing")
	}
	if len(candidates) != 2 {
		t.Fatalf("candidates length = %d, want 2", len(candidates))
	}
	if candidates[0].ID != "101" || candidates[0].Name != "大学语文" {
		t.Fatalf("first candidate = %#v", candidates[0])
	}
	if candidates[1].ID != "202" || candidates[1].Name != "高等数学" {
		t.Fatalf("second candidate = %#v", candidates[1])
	}
}

func TestLegacyBestCourseMatch(t *testing.T) {
	candidates := []legacyCourseCandidate{
		{ID: "1", Name: "大学英语一"},
		{ID: "2", Name: "高等数学"},
	}
	match, ok := legacyBestCourseMatch(candidates, "大学英语")
	if !ok {
		t.Fatal("legacyBestCourseMatch() did not match close course name")
	}
	if match.ID != "1" {
		t.Fatalf("matched ID = %q, want 1", match.ID)
	}
	_, ok = legacyBestCourseMatch(candidates, "计算机网络")
	if ok {
		t.Fatal("legacyBestCourseMatch() matched unrelated course")
	}
}

func TestLegacyCourseQueryURL(t *testing.T) {
	url := legacyCourseQueryURL(models.Connector{BaseURL: "https://example.com/api/"})
	if url != "https://example.com/api/courses/query" {
		t.Fatalf("legacyCourseQueryURL() = %q", url)
	}
}

package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dw0rdwk/backend/internal/models"
)

func TestWK29APIURL(t *testing.T) {
	tests := []struct {
		name string
		base string
		want string
	}{
		{name: "root", base: "https://example.com", want: "https://example.com/api.php?act=get"},
		{name: "trailing slash", base: "https://example.com/base/", want: "https://example.com/base/api.php?act=get"},
		{name: "api file", base: "https://example.com/base/api.php", want: "https://example.com/base/api.php?act=get"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WK29APIURL(tt.base, "get")
			if err != nil {
				t.Fatalf("WK29APIURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("WK29APIURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubmit29WKOrderPostsLegacyFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api.php" || r.URL.Query().Get("act") != "add" {
			t.Fatalf("unexpected URL %s", r.URL.String())
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		want := map[string]string{
			"uid":      "1001",
			"key":      "secret",
			"platform": "29",
			"school":   "school",
			"user":     "student",
			"pass":     "pass",
			"kcid":     "course-1",
			"kcname":   "大学语文",
			"score":    "90",
			"shichang": "30",
		}
		for key, value := range want {
			if got := r.Form.Get(key); got != value {
				t.Fatalf("form[%s] = %q, want %q", key, got, value)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "提交成功", "id": "remote-1"})
	}))
	defer server.Close()

	response, err := Submit29WKOrder(context.Background(), models.Connector{
		BaseURL:   server.URL,
		AppKey:    "1001",
		AppSecret: "secret",
		Kind:      Kind29WK,
	}, models.Order{
		ClassID:         20,
		DockingCode:     "29",
		School:          "school",
		Account:         "student",
		AccountPassword: "pass",
		CourseID:        "course-1",
		CourseName:      "大学语文",
		Score:           "90",
		DurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("Submit29WKOrder() error = %v", err)
	}
	if response.RemoteOrderID != "remote-1" || response.Status != "processing" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestRefresh29WKOrderMatchesCourse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("act") != "chadan" {
			t.Fatalf("act = %q", r.URL.Query().Get("act"))
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got := r.Form.Get("username"); got != "student" {
			t.Fatalf("username = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 1,
			"data": []map[string]any{
				{"id": "1", "user": "student", "kcname": "英语", "status": "进行中", "process": "20%"},
				{"id": "2", "user": "student", "kcname": "大学语文", "status": "已完成", "process": "100%", "remarks": "ok"},
			},
		})
	}))
	defer server.Close()

	response, err := Refresh29WKOrder(context.Background(), models.Connector{
		BaseURL:   server.URL + "/api.php",
		AppKey:    "1001",
		AppSecret: "secret",
	}, models.Order{Account: "student", CourseName: "大学语文"})
	if err != nil {
		t.Fatalf("Refresh29WKOrder() error = %v", err)
	}
	if response.RemoteOrderID != "2" || response.Status != "done" || response.Progress != "100%" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestQuery29WKCoursesNormalizesCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("act") != "get" {
			t.Fatalf("act = %q", r.URL.Query().Get("act"))
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got := r.Form.Get("platform"); got != "query-code" {
			t.Fatalf("platform = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 1,
			"data": []map[string]any{
				{"kcid": "101", "kcname": "大学语文"},
				{"courseId": "202", "courseName": "高等数学"},
			},
		})
	}))
	defer server.Close()

	_, candidates, err := Query29WKCourses(context.Background(), models.Connector{
		BaseURL:   server.URL,
		AppKey:    "1001",
		AppSecret: "secret",
	}, CourseQueryInput{
		Class:    models.CourseClass{QueryParam: "query-code", DockingCode: "dock-code"},
		School:   "school",
		Account:  "student",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("Query29WKCourses() error = %v", err)
	}
	if len(candidates) != 2 || candidates[0].ID != "101" || candidates[1].Name != "高等数学" {
		t.Fatalf("unexpected candidates: %#v", candidates)
	}
}

func TestFetch29WKClasses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("act") != "getclass" {
			t.Fatalf("act = %q", r.URL.Query().Get("act"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 1,
			"data": []map[string]any{
				{"sort": 10, "cid": 11, "name": "skyriver自营", "price": 0.6, "fenlei": 1, "fenleiname": "skyriver项目", "content": "说明", "status": 1, "miaoshua": 1},
			},
		})
	}))
	defer server.Close()

	classes, err := Fetch29WKClasses(context.Background(), models.Connector{
		BaseURL:   server.URL,
		AppKey:    "1001",
		AppSecret: "secret",
	})
	if err != nil {
		t.Fatalf("Fetch29WKClasses() error = %v", err)
	}
	if len(classes) != 1 {
		t.Fatalf("len(classes) = %d, want 1", len(classes))
	}
	if classes[0].UpstreamID != "11" || classes[0].Status != "online" || !classes[0].FlashEnabled {
		t.Fatalf("unexpected class: %+v", classes[0])
	}
}

func TestFetch29WKOrderStatusesParsesBatchRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("act") != "plchadan" {
			t.Fatalf("act = %q", r.URL.Query().Get("act"))
		}
		if r.Header.Get("User-Agent") == "" {
			t.Fatal("missing browser user agent")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if r.Form.Get("uid") != "1001" || r.Form.Get("key") != "secret" {
			t.Fatalf("unexpected auth form: %#v", r.Form)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 1,
			"data": []map[string]any{
				{"id": "remote-1", "status": "已完成", "remarks": "ok", "process": "100%", "user": "student", "pass": "pass", "kcname": "大学语文", "cid": "29"},
			},
		})
	}))
	defer server.Close()

	rows, err := Fetch29WKOrderStatuses(context.Background(), models.Connector{
		BaseURL:   server.URL,
		AppKey:    "1001",
		AppSecret: "secret",
	})
	if err != nil {
		t.Fatalf("Fetch29WKOrderStatuses() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.RemoteOrderID != "remote-1" || row.Normalized != "done" || row.Progress != "100%" || row.DockingCode != "29" {
		t.Fatalf("unexpected row: %+v", row)
	}
}

func TestFetch29WKOrderStatusesRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", responseLimitReadBuffer)))
	}))
	defer server.Close()

	_, err := Fetch29WKOrderStatuses(context.Background(), models.Connector{BaseURL: server.URL})
	if err == nil {
		t.Fatal("expected oversized response error")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error = %v, want response size error", err)
	}
}

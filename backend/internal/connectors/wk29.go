package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"dw0rdwk/backend/internal/models"
)

type WK29Class struct {
	Sort         int
	UpstreamID   string
	KcID         string
	Name         string
	Noun         string
	Price        float64
	CategoryID   string
	CategoryName string
	Description  string
	Status       string
	FlashEnabled bool
}

type WK29OrderStatusRow struct {
	RemoteOrderID string
	Status        string
	Normalized    string
	Remarks       string
	Progress      string
	Account       string
	Password      string
	CourseName    string
	DockingCode   string
	Raw           map[string]any
}

const (
	wk29UserAgent           = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36"
	max29WKResponseBytes    = 32 << 20
	responseLimitReadBuffer = max29WKResponseBytes + 1
)

func WK29APIURL(baseURL string, act string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", fmt.Errorf("connector baseUrl is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("connector baseUrl must be a valid http or https URL")
	}
	trimmedPath := strings.TrimRight(parsed.Path, "/")
	if !strings.EqualFold(path.Base(trimmedPath), "api.php") {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api.php"
	}
	query := parsed.Query()
	query.Set("act", strings.TrimSpace(act))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func Query29WKCourses(ctx context.Context, connector models.Connector, input CourseQueryInput) (map[string]any, []CourseCandidate, error) {
	body, err := post29WKForm(ctx, connector, "get", url.Values{
		"platform": []string{queryPlatform(input.Class)},
		"school":   []string{strings.TrimSpace(input.School)},
		"user":     []string{strings.TrimSpace(input.Account)},
		"pass":     []string{strings.TrimSpace(input.Password)},
		"type":     []string{strings.TrimSpace(input.Type)},
	})
	if err != nil {
		return nil, nil, err
	}
	return NormalizeCourseQueryResponse(body)
}

func Submit29WKOrder(ctx context.Context, connector models.Connector, order models.Order) (OrderResponse, error) {
	body, err := post29WKForm(ctx, connector, "add", url.Values{
		"platform": []string{orderPlatform(order)},
		"school":   []string{strings.TrimSpace(order.School)},
		"user":     []string{strings.TrimSpace(order.Account)},
		"pass":     []string{strings.TrimSpace(order.AccountPassword)},
		"kcid":     []string{strings.TrimSpace(order.CourseID)},
		"kcname":   []string{strings.TrimSpace(order.CourseName)},
		"score":    []string{strings.TrimSpace(order.Score)},
		"shichang": []string{durationString(order.DurationMinutes)},
	})
	if err != nil {
		return OrderResponse{}, err
	}
	parsed, err := decodeObject(body)
	if err != nil {
		return OrderResponse{}, err
	}
	if !is29WKAddSuccess(parsed) {
		return OrderResponse{}, fmt.Errorf("%s", firstStringDefault(parsed, "提交失败", "msg", "message", "error"))
	}
	return OrderResponse{
		RemoteOrderID: firstString(parsed, "id", "oid", "yid", "remoteOrderId"),
		Status:        "processing",
		Remarks:       firstString(parsed, "msg", "message"),
	}, nil
}

func Refresh29WKOrder(ctx context.Context, connector models.Connector, order models.Order) (OrderResponse, error) {
	values := url.Values{}
	if remoteID := strings.TrimSpace(order.RemoteOrderID); remoteID != "" {
		values.Set("yid", remoteID)
	} else {
		values.Set("username", strings.TrimSpace(order.Account))
	}
	body, err := post29WKForm(ctx, connector, "chadan", values)
	if err != nil {
		return OrderResponse{}, err
	}
	parsed, err := decodeObject(body)
	if err != nil {
		return OrderResponse{}, err
	}
	if !isCode(parsed, "1") {
		return OrderResponse{}, fmt.Errorf("%s", firstStringDefault(parsed, "查询失败", "msg", "message"))
	}
	rows := rowsFromAny(parsed["data"])
	row, ok := bestOrderRow(rows, order)
	if !ok {
		return OrderResponse{}, fmt.Errorf("上游未返回匹配订单")
	}
	return responseFromOrderRow(row), nil
}

func Budan29WKOrder(ctx context.Context, connector models.Connector, order models.Order) (OrderResponse, error) {
	remoteID := strings.TrimSpace(order.RemoteOrderID)
	if remoteID == "" {
		refreshed, err := Refresh29WKOrder(ctx, connector, order)
		if err != nil {
			return OrderResponse{}, err
		}
		remoteID = refreshed.RemoteOrderID
	}
	if remoteID == "" {
		return OrderResponse{}, fmt.Errorf("上游订单号为空，无法补刷")
	}
	body, err := post29WKForm(ctx, connector, "budan", url.Values{"id": []string{remoteID}})
	if err != nil {
		return OrderResponse{}, err
	}
	parsed, err := decodeObject(body)
	if err != nil {
		return OrderResponse{}, err
	}
	if !isCode(parsed, "1") && !isCode(parsed, "0") {
		return OrderResponse{}, fmt.Errorf("%s", firstStringDefault(parsed, "补刷失败", "msg", "message"))
	}
	return OrderResponse{
		RemoteOrderID: remoteID,
		Status:        "processing",
		Remarks:       firstString(parsed, "msg", "message"),
	}, nil
}

func Fetch29WKOrderStatuses(ctx context.Context, connector models.Connector) ([]WK29OrderStatusRow, error) {
	body, err := post29WKForm(ctx, connector, "plchadan", nil)
	if err != nil {
		return nil, err
	}
	parsed, err := decodeObject(body)
	if err != nil {
		return nil, err
	}
	if _, ok := parsed["data"]; !ok {
		if !isCode(parsed, "1") && !isCode(parsed, "0") {
			return nil, fmt.Errorf("%s", firstStringDefault(parsed, "批量查单失败", "msg", "message", "error"))
		}
		return []WK29OrderStatusRow{}, nil
	}
	rows := rowsFromAny(parsed["data"])
	result := make([]WK29OrderStatusRow, 0, len(rows))
	for _, row := range rows {
		status := firstString(row, "status", "status_text")
		progress := firstString(row, "process", "progress")
		item := WK29OrderStatusRow{
			RemoteOrderID: firstString(row, "id", "oid", "yid"),
			Status:        status,
			Normalized:    map29WKStatus(status, progress),
			Remarks:       firstString(row, "remarks", "remark", "msg", "message"),
			Progress:      progress,
			Account:       firstString(row, "user", "username", "account"),
			Password:      firstString(row, "pass", "password"),
			CourseName:    firstString(row, "kcname", "courseName", "course_name", "name"),
			DockingCode:   firstString(row, "cid", "noun", "platform", "dockingCode", "docking_code"),
			Raw:           copyMap(row),
		}
		if item.RemoteOrderID == "" && item.Account == "" && item.CourseName == "" && item.DockingCode == "" {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func Query29WKBalance(ctx context.Context, connector models.Connector) (float64, map[string]any, error) {
	body, err := post29WKForm(ctx, connector, "getmoney", nil)
	if err != nil {
		return 0, nil, err
	}
	parsed, err := decodeObject(body)
	if err != nil {
		return 0, nil, err
	}
	if value := firstString(parsed, "money", "balance"); value != "" {
		amount, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, parsed, fmt.Errorf("connector balance is invalid")
		}
		return amount, parsed, nil
	}
	if value, ok := parsed["money"]; ok {
		return floatFromAny(value), parsed, nil
	}
	if value, ok := parsed["balance"]; ok {
		return floatFromAny(value), parsed, nil
	}
	return 0, parsed, fmt.Errorf("%s", firstStringDefault(parsed, "查询余额失败", "msg", "message", "error"))
}

func Fetch29WKClasses(ctx context.Context, connector models.Connector) ([]WK29Class, error) {
	body, err := post29WKForm(ctx, connector, "getclass", nil)
	if err != nil {
		return nil, err
	}
	parsed, err := decodeObject(body)
	if err != nil {
		return nil, err
	}
	if !isCode(parsed, "1") {
		return nil, fmt.Errorf("%s", firstStringDefault(parsed, "获取商品库失败", "msg", "message"))
	}
	items := rowsFromAny(parsed["data"])
	classes := make([]WK29Class, 0, len(items))
	for _, item := range items {
		class := WK29Class{
			Sort:         intFromAny(item["sort"]),
			UpstreamID:   firstString(item, "cid", "id"),
			KcID:         firstString(item, "kcid"),
			Name:         firstString(item, "name", "title"),
			Noun:         firstString(item, "noun"),
			Price:        floatFromAny(item["price"]),
			CategoryID:   firstString(item, "fenlei", "categoryId", "category_id"),
			CategoryName: firstString(item, "fenleiname", "categoryName", "category_name", "fenleiName"),
			Description:  firstString(item, "content", "description", "desc"),
			Status:       classStatus(firstString(item, "status")),
			FlashEnabled: boolFromAny(item["miaoshua"]),
		}
		if class.UpstreamID == "" {
			class.UpstreamID = class.Noun
		}
		if class.UpstreamID == "" {
			class.UpstreamID = class.KcID
		}
		if class.UpstreamID == "" {
			class.UpstreamID = class.Name
		}
		if class.Noun == "" {
			class.Noun = class.UpstreamID
		}
		if class.CategoryName == "" {
			class.CategoryName = class.CategoryID
		}
		if class.UpstreamID == "" || class.Name == "" {
			continue
		}
		classes = append(classes, class)
	}
	return classes, nil
}

func NormalizeCourseQueryResponse(body []byte) (map[string]any, []CourseCandidate, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		result := map[string]any{"code": 1, "msg": "查询成功", "data": []map[string]any{}}
		return result, nil, nil
	}
	var parsed any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&parsed); err != nil {
		return nil, nil, fmt.Errorf("decode connector response: %w", err)
	}
	switch value := parsed.(type) {
	case map[string]any:
		result := copyMap(value)
		if _, ok := result["code"]; !ok {
			result["code"] = 1
		}
		if _, ok := result["msg"]; !ok {
			result["msg"] = "查询成功"
		}
		candidates := CourseCandidatesFromAny(result["data"])
		if _, ok := result["data"]; !ok {
			result["data"] = CourseCandidateRows(candidates)
		}
		return result, candidates, nil
	case []any:
		candidates := CourseCandidatesFromAny(value)
		return map[string]any{"code": 1, "msg": "查询成功", "data": CourseCandidateRows(candidates)}, candidates, nil
	default:
		return nil, nil, fmt.Errorf("connector response must be object or array")
	}
}

func CourseCandidatesFromAny(value any) []CourseCandidate {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	candidates := make([]CourseCandidate, 0, len(items))
	for _, item := range items {
		switch row := item.(type) {
		case map[string]any:
			raw := copyMap(row)
			id := firstString(row, "id", "kcid", "courseId", "course_id")
			name := firstString(row, "name", "kcname", "courseName", "course_name", "title")
			if id == "" && name == "" {
				continue
			}
			candidates = append(candidates, CourseCandidate{ID: id, Name: name, Raw: raw})
		case string:
			name := strings.TrimSpace(row)
			if name != "" {
				candidates = append(candidates, CourseCandidate{Name: name, Raw: map[string]any{"name": name}})
			}
		}
	}
	return candidates
}

func CourseCandidateRows(candidates []CourseCandidate) []map[string]any {
	rows := make([]map[string]any, 0, len(candidates))
	for _, candidate := range candidates {
		row := copyMap(candidate.Raw)
		if _, ok := row["id"]; !ok && candidate.ID != "" {
			row["id"] = candidate.ID
		}
		if _, ok := row["name"]; !ok && candidate.Name != "" {
			row["name"] = candidate.Name
		}
		rows = append(rows, row)
	}
	return rows
}

func post29WKForm(ctx context.Context, connector models.Connector, act string, values url.Values) ([]byte, error) {
	endpoint, err := WK29APIURL(connector.BaseURL, act)
	if err != nil {
		return nil, err
	}
	if values == nil {
		values = url.Values{}
	}
	values.Set("uid", strings.TrimSpace(connector.AppKey))
	values.Set("key", strings.TrimSpace(connector.AppSecret))

	reqCtx, cancel := context.WithTimeout(ctx, connectorTimeout(connector))
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json,text/plain,*/*")
	req.Header.Set("User-Agent", wk29UserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, responseLimitReadBuffer))
	if len(body) > max29WKResponseBytes {
		return nil, fmt.Errorf("connector response exceeds %d bytes", max29WKResponseBytes)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("connector returned %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	return body, nil
}

func connectorTimeout(connector models.Connector) time.Duration {
	timeout := time.Duration(connector.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		return 8 * time.Second
	}
	return timeout
}

func queryPlatform(class models.CourseClass) string {
	if value := strings.TrimSpace(class.QueryParam); value != "" {
		return value
	}
	if value := strings.TrimSpace(class.DockingCode); value != "" {
		return value
	}
	if class.ID != 0 {
		return strconv.FormatUint(uint64(class.ID), 10)
	}
	return ""
}

func orderPlatform(order models.Order) string {
	if value := strings.TrimSpace(order.DockingCode); value != "" {
		return value
	}
	if order.ClassID != 0 {
		return strconv.FormatUint(uint64(order.ClassID), 10)
	}
	return ""
}

func durationString(duration int) string {
	if duration <= 0 {
		return ""
	}
	return strconv.Itoa(duration)
}

func decodeObject(body []byte) (map[string]any, error) {
	var parsed map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode connector response: %w", err)
	}
	return parsed, nil
}

func is29WKAddSuccess(row map[string]any) bool {
	return isCode(row, "0") || firstString(row, "status") == "0"
}

func isCode(row map[string]any, want string) bool {
	return firstString(row, "code") == want
}

func rowsFromAny(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, copyMap(row))
		}
	}
	return rows
}

func bestOrderRow(rows []map[string]any, order models.Order) (map[string]any, bool) {
	remoteID := strings.TrimSpace(order.RemoteOrderID)
	account := strings.TrimSpace(order.Account)
	courseName := strings.TrimSpace(order.CourseName)
	if remoteID != "" {
		for _, row := range rows {
			if firstString(row, "id", "oid", "yid") == remoteID {
				return row, true
			}
		}
	}
	for _, row := range rows {
		if account != "" && firstString(row, "user", "username", "account") != account {
			continue
		}
		if courseName == "" || firstString(row, "kcname", "courseName", "name") == courseName {
			return row, true
		}
	}
	if len(rows) > 0 {
		return rows[0], true
	}
	return nil, false
}

func responseFromOrderRow(row map[string]any) OrderResponse {
	statusText := firstString(row, "status", "status_text")
	progress := firstString(row, "process", "progress")
	return OrderResponse{
		RemoteOrderID: firstString(row, "id", "oid", "yid"),
		Status:        map29WKStatus(statusText, progress),
		Progress:      progress,
		Remarks:       firstString(row, "remarks", "remark", "msg", "message"),
	}
}

func map29WKStatus(status string, progress string) string {
	text := strings.ToLower(strings.TrimSpace(status + " " + progress))
	switch {
	case strings.Contains(text, "refunded") || strings.Contains(text, "退款"):
		return "refunded"
	case strings.Contains(text, "cancel") || strings.Contains(text, "stop") || strings.Contains(text, "取消") || strings.Contains(text, "停止") || strings.Contains(text, "暂停"):
		return "cancelled"
	case strings.Contains(text, "failed") || strings.Contains(text, "error") || strings.Contains(text, "fail") || strings.Contains(text, "失败") || strings.Contains(text, "异常") || strings.Contains(text, "错误"):
		return "failed"
	case strings.Contains(text, "done") || strings.Contains(text, "complete") || strings.Contains(text, "success") || strings.Contains(text, "完成") || strings.Contains(text, "已刷完"):
		return "done"
	case strings.Contains(text, "100"):
		return "done"
	default:
		return "processing"
	}
}

func Map29WKStatus(status string, progress string) string {
	return map29WKStatus(status, progress)
}

func classStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "1", "online", "active", "true":
		return "online"
	default:
		return "offline"
	}
}

func firstString(row map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := row[key]
		if !ok {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func firstStringDefault(row map[string]any, fallback string, keys ...string) string {
	if value := firstString(row, keys...); value != "" {
		return value
	}
	return fallback
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	case float64:
		return int(typed)
	case int:
		return typed
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
		return parsed
	default:
		return 0
	}
}

func floatFromAny(value any) float64 {
	switch typed := value.(type) {
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	case float64:
		return typed
	case int:
		return float64(typed)
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed
	default:
		return 0
	}
}

func boolFromAny(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case json.Number:
		parsed, _ := typed.Int64()
		return parsed != 0
	case float64:
		return typed != 0
	case string:
		normalized := strings.ToLower(strings.TrimSpace(typed))
		return normalized == "1" || normalized == "true" || normalized == "yes"
	default:
		return false
	}
}

func copyMap(source map[string]any) map[string]any {
	copied := make(map[string]any, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

func RoundPrice(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

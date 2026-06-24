package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	connectoradapter "dw0rdwk/backend/internal/connectors"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

func LegacyAPI(
	users *repository.UserRepository,
	classes *repository.ClassRepository,
	specialPrices *repository.SpecialPriceRepository,
	orders *repository.OrderRepository,
	connectors *repository.ConnectorRepository,
	settings *repository.SettingRepository,
	orderService *service.OrderService,
	logs *repository.LogRepository,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch strings.ToLower(strings.TrimSpace(c.Query("act"))) {
		case "getmoney":
			return legacyGetMoney(c, users)
		case "get":
			return legacyQueryCourse(c, users, classes, connectors, settings, logs)
		case "getclass":
			return legacyGetClass(c, users, classes, specialPrices)
		case "add":
			return legacyAddOrder(c, users, classes, specialPrices, connectors, settings, orderService, logs)
		case "getadd":
			return legacyQueryAndAddOrder(c, users, classes, specialPrices, connectors, settings, orderService, logs)
		case "uporder":
			return legacyRefreshOrder(c, users, orders, orderService)
		case "chadan":
			return legacySearchOrders(c, users, orders)
		case "budan":
			return legacyResubmitOrder(c, users, orders, orderService)
		default:
			return legacyFail(c, -1, "未知接口")
		}
	}
}

type legacyCourseQueryInput struct {
	School   string
	Account  string
	Password string
	Type     string
}

type legacyCourseCandidate struct {
	ID   string
	Name string
	Raw  fiber.Map
}

func legacyGetMoney(c *fiber.Ctx, users *repository.UserRepository) error {
	user, ok, err := legacyAPIUser(c, users)
	if err != nil || !ok {
		return err
	}
	return c.JSON(fiber.Map{"code": 1, "msg": "查询成功", "money": user.Balance})
}

func legacyGetClass(c *fiber.Ctx, users *repository.UserRepository, classes *repository.ClassRepository, specialPrices *repository.SpecialPriceRepository) error {
	user, ok, err := legacyAPIUser(c, users)
	if err != nil || !ok {
		return err
	}
	classID := legacyUint(c, "cid")
	items, err := classes.ListOnlineForAPI(c.UserContext(), classID, 1000)
	if err != nil {
		return err
	}
	data := make([]fiber.Map, 0, len(items))
	for _, item := range items {
		special, hasSpecial, err := specialPrices.FindForUserClass(c.UserContext(), user.ID, item.ID)
		if err != nil {
			return err
		}
		name := item.Name
		if hasSpecial {
			name = "【密价】" + name
		}
		data = append(data, fiber.Map{
			"sort":     item.Sort,
			"cid":      item.ID,
			"kcid":     item.QueryParam,
			"name":     name,
			"noun":     item.DockingCode,
			"price":    agentPrice(user, item, specialPricePtr(special, hasSpecial)),
			"fenlei":   item.Category,
			"content":  item.Description,
			"status":   legacyClassStatus(item.Status),
			"miaoshua": legacyBoolInt(item.BridgeEnabled),
		})
	}
	return c.JSON(fiber.Map{"code": 1, "data": data})
}

func legacyAddOrder(
	c *fiber.Ctx,
	users *repository.UserRepository,
	classes *repository.ClassRepository,
	specialPrices *repository.SpecialPriceRepository,
	connectors *repository.ConnectorRepository,
	settings *repository.SettingRepository,
	orderService *service.OrderService,
	logs *repository.LogRepository,
) error {
	user, ok, err := legacyAPIUser(c, users)
	if err != nil || !ok {
		return err
	}
	classID := legacyUint(c, "platform")
	if classID == 0 {
		return legacyFail(c, 0, "所有项目不能为空")
	}
	class, err := classes.Find(c.UserContext(), classID)
	if err != nil {
		if repository.IsNotFound(err) {
			return legacyFail(c, -2, "网课不存在")
		}
		return err
	}
	if class.Status != "online" || !class.BridgeEnabled {
		return legacyFail(c, -2, "小老弟，商品都下架了你还下什么单呢！")
	}
	account := strings.TrimSpace(legacyValue(c, "user"))
	password := strings.TrimSpace(legacyValue(c, "pass"))
	courseName := strings.TrimSpace(legacyValue(c, "kcname"))
	if account == "" || password == "" || courseName == "" {
		return legacyFail(c, 0, "所有项目不能为空")
	}
	connector, err := defaultConnector(c, connectors, settings)
	if err != nil {
		return legacyFail(c, -1, err.Error())
	}
	special, hasSpecial, err := specialPrices.FindForUserClass(c.UserContext(), user.ID, class.ID)
	if err != nil {
		return err
	}
	fee := agentPrice(user, class, specialPricePtr(special, hasSpecial))
	if fee <= 0 || user.PriceRate < 0.1 {
		return legacyFail(c, -1, "大佬，我得罪不起您，我小本生意，有哪里得罪之处，还望多多包涵")
	}
	if err := users.DeductBalanceIfEnough(c.UserContext(), user.ID, fee); err != nil {
		return legacyFail(c, -1, "余额不足以本次提交")
	}
	order := models.Order{
		UserID:          user.ID,
		ClassID:         class.ID,
		ConnectorID:     connector.ID,
		Platform:        class.Name,
		School:          strings.TrimSpace(legacyValue(c, "school")),
		Account:         account,
		AccountPassword: password,
		CourseID:        strings.TrimSpace(legacyValue(c, "kcid")),
		CourseName:      courseName,
		Fee:             fee,
		DockingCode:     class.DockingCode,
		FlashMode:       legacyBool(c, "miaoshua"),
		SourceIP:        c.IP(),
		Score:           strings.TrimSpace(legacyValue(c, "score")),
		DurationMinutes: legacyInt(c, "shichang"),
	}
	if err := orderService.Submit(c.UserContext(), &order); err != nil {
		_ = users.AdjustBalance(c.UserContext(), user.ID, fee)
		return legacyFail(c, -1, err.Error())
	}
	auditLog(c, logs, "api_order_create", auditText("order", order.ID, "class="+class.Name), -fee)
	return c.JSON(fiber.Map{"code": 0, "msg": "提交成功", "status": 0, "message": "提交成功", "id": strconv.FormatUint(uint64(order.ID), 10)})
}

func legacyQueryCourse(
	c *fiber.Ctx,
	users *repository.UserRepository,
	classes *repository.ClassRepository,
	connectors *repository.ConnectorRepository,
	settings *repository.SettingRepository,
	logs *repository.LogRepository,
) error {
	user, class, connector, input, err := legacyPrepareCourseQuery(c, users, classes, connectors, settings)
	if err != nil {
		return err
	}
	result, candidates, err := legacyCallCourseQuery(c.UserContext(), connector, class, input)
	if err != nil {
		return legacyFail(c, -1, err.Error())
	}
	result["userinfo"] = input.School + " " + input.Account + " " + input.Password
	auditLog(c, logs, "api_course_query", auditText("class", class.ID, input.School+" "+input.Account), 0)
	if input.Type == "xiaochu" {
		return c.JSON(legacyXiaochuCourseResult(result, class, input, candidates))
	}
	_ = user
	return c.JSON(result)
}

func legacyQueryAndAddOrder(
	c *fiber.Ctx,
	users *repository.UserRepository,
	classes *repository.ClassRepository,
	specialPrices *repository.SpecialPriceRepository,
	connectors *repository.ConnectorRepository,
	settings *repository.SettingRepository,
	orderService *service.OrderService,
	logs *repository.LogRepository,
) error {
	user, class, connector, input, err := legacyPrepareCourseQuery(c, users, classes, connectors, settings)
	if err != nil {
		return err
	}
	target := strings.TrimSpace(legacyValue(c, "kcname"))
	if target == "" {
		return legacyFail(c, 0, "所有项目不能为空")
	}
	if class.Status != "online" || !class.BridgeEnabled {
		return legacyFail(c, -2, "小老弟，商品都下架了你还下什么单呢！")
	}
	special, hasSpecial, err := specialPrices.FindForUserClass(c.UserContext(), user.ID, class.ID)
	if err != nil {
		return err
	}
	fee := agentPrice(user, class, specialPricePtr(special, hasSpecial))
	if fee <= 0 || user.PriceRate < 0.1 {
		return legacyFail(c, -1, "大佬，我得罪不起您，我小本生意，有哪里得罪之处，还望多多包涵")
	}
	result, candidates, err := legacyCallCourseQuery(c.UserContext(), connector, class, input)
	if err != nil {
		return legacyFail(c, -1, err.Error())
	}
	match, ok := legacyBestCourseMatch(candidates, target)
	if !ok {
		return legacyFail(c, -1, "请完整输入课程名字")
	}
	if err := users.DeductBalanceIfEnough(c.UserContext(), user.ID, fee); err != nil {
		return legacyFail(c, -1, "余额不足")
	}
	order := models.Order{
		UserID:          user.ID,
		ClassID:         class.ID,
		ConnectorID:     connector.ID,
		Platform:        class.Name,
		School:          input.School,
		Account:         input.Account,
		AccountPassword: input.Password,
		CourseID:        match.ID,
		CourseName:      match.Name,
		Fee:             fee,
		DockingCode:     class.DockingCode,
		FlashMode:       legacyBool(c, "miaoshua"),
		SourceIP:        c.IP(),
	}
	if err := orderService.Submit(c.UserContext(), &order); err != nil {
		_ = users.AdjustBalance(c.UserContext(), user.ID, fee)
		return legacyFail(c, -1, err.Error())
	}
	auditLog(c, logs, "api_order_query_create", auditText("order", order.ID, "class="+class.Name+" queryCode="+fmt.Sprint(result["code"])), -fee)
	return c.JSON(fiber.Map{"code": 0, "msg": "提交成功", "status": 0, "message": "提交成功", "id": strconv.FormatUint(uint64(order.ID), 10)})
}

func legacyRefreshOrder(c *fiber.Ctx, users *repository.UserRepository, orders *repository.OrderRepository, orderService *service.OrderService) error {
	user, ok, err := legacyAPIUser(c, users)
	if err != nil || !ok {
		return err
	}
	id := legacyUint(c, "oid")
	if id == 0 {
		id = legacyUint(c, "id")
	}
	if id == 0 {
		return legacyFail(c, 0, "订单ID不能为空")
	}
	order, err := orders.Find(c.UserContext(), id)
	if err != nil {
		return legacyFail(c, -1, "订单不存在")
	}
	if order.UserID != user.ID {
		return legacyFail(c, -1, "该订单不是你的，无法操作！")
	}
	if err := orderService.MarkRefreshRequested(c.UserContext(), id); err != nil {
		return legacyFail(c, -1, err.Error())
	}
	return c.JSON(fiber.Map{"code": 1, "msg": "同步成功"})
}

func legacySearchOrders(c *fiber.Ctx, users *repository.UserRepository, orders *repository.OrderRepository) error {
	user, ok, err := legacyAPIUser(c, users)
	if err != nil || !ok {
		return err
	}
	account := strings.TrimSpace(legacyValue(c, "username"))
	id := legacyUint(c, "oid")
	if id == 0 {
		id = legacyUint(c, "id")
	}

	var items []models.Order
	if account != "" {
		items, err = orders.ListByUserAccount(c.UserContext(), user.ID, account, 100)
		if err != nil {
			return err
		}
	} else if id != 0 {
		order, findErr := orders.FindByUser(c.UserContext(), id, user.ID)
		if findErr != nil {
			return legacyFail(c, -1, "未查到该账号的下单信息")
		}
		items = []models.Order{order}
	} else {
		return legacyFail(c, -1, "账号不能为空")
	}
	if len(items) == 0 {
		return legacyFail(c, -1, "未查到该账号的下单信息")
	}
	return c.JSON(fiber.Map{"code": 1, "data": legacyOrderRows(items)})
}

func legacyResubmitOrder(c *fiber.Ctx, users *repository.UserRepository, orders *repository.OrderRepository, orderService *service.OrderService) error {
	user, ok, err := legacyAPIUser(c, users)
	if err != nil || !ok {
		return err
	}
	id := legacyUint(c, "id")
	if id == 0 {
		id = legacyUint(c, "oid")
	}
	if id == 0 {
		return legacyFail(c, 0, "订单ID不能为空")
	}
	order, err := orders.FindByUser(c.UserContext(), id, user.ID)
	if err != nil {
		return legacyFail(c, -1, "订单不存在")
	}
	if order.Status == "cancelled" {
		return legacyFail(c, -1, "该订单无法补刷")
	}
	if err := orderService.MarkMakeupRequested(c.UserContext(), id); err != nil {
		return legacyFail(c, -1, err.Error())
	}
	return c.JSON(fiber.Map{"code": 1, "msg": "补单刷新已提交"})
}

func legacyPrepareCourseQuery(
	c *fiber.Ctx,
	users *repository.UserRepository,
	classes *repository.ClassRepository,
	connectors *repository.ConnectorRepository,
	settings *repository.SettingRepository,
) (models.User, models.CourseClass, models.Connector, legacyCourseQueryInput, error) {
	user, ok, err := legacyAPIUser(c, users)
	if err != nil || !ok {
		return models.User{}, models.CourseClass{}, models.Connector{}, legacyCourseQueryInput{}, err
	}
	classID := legacyUint(c, "platform")
	input := legacyCourseQueryInput{
		School:   strings.TrimSpace(legacyValue(c, "school")),
		Account:  strings.TrimSpace(legacyValue(c, "user")),
		Password: strings.TrimSpace(legacyValue(c, "pass")),
		Type:     strings.TrimSpace(legacyValue(c, "type")),
	}
	if classID == 0 || input.School == "" || input.Account == "" || input.Password == "" {
		return models.User{}, models.CourseClass{}, models.Connector{}, legacyCourseQueryInput{}, legacyFail(c, 0, "所有项目不能为空")
	}
	class, err := classes.Find(c.UserContext(), classID)
	if err != nil {
		if repository.IsNotFound(err) {
			return models.User{}, models.CourseClass{}, models.Connector{}, legacyCourseQueryInput{}, legacyFail(c, -2, "网课不存在")
		}
		return models.User{}, models.CourseClass{}, models.Connector{}, legacyCourseQueryInput{}, err
	}
	if class.Status != "online" {
		return models.User{}, models.CourseClass{}, models.Connector{}, legacyCourseQueryInput{}, legacyFail(c, -2, "网课已下架禁止查课！")
	}
	connector, err := defaultConnector(c, connectors, settings)
	if err != nil {
		return models.User{}, models.CourseClass{}, models.Connector{}, legacyCourseQueryInput{}, legacyFail(c, -1, err.Error())
	}
	return user, class, connector, input, nil
}

func legacyCallCourseQuery(ctx context.Context, connector models.Connector, class models.CourseClass, input legacyCourseQueryInput) (fiber.Map, []legacyCourseCandidate, error) {
	if strings.TrimSpace(connector.BaseURL) == "" {
		return nil, nil, fmt.Errorf("connector baseUrl is required")
	}
	if connectoradapter.Is29WKKind(connector.Kind) {
		result, candidates, err := connectoradapter.Query29WKCourses(ctx, connector, connectoradapter.CourseQueryInput{
			Class:    class,
			School:   input.School,
			Account:  input.Account,
			Password: input.Password,
			Type:     input.Type,
		})
		if err != nil {
			return nil, nil, err
		}
		return legacyCourseQueryFromConnectorResult(result, candidates), legacyCandidatesFromConnector(candidates), nil
	}
	timeout := time.Duration(connector.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	payload, err := json.Marshal(fiber.Map{
		"auth": fiber.Map{
			"appKey":    connector.AppKey,
			"appSecret": connector.AppSecret,
		},
		"class": fiber.Map{
			"id":              class.ID,
			"name":            class.Name,
			"queryParam":      class.QueryParam,
			"queryPlatform":   class.QueryPlatform,
			"dockingCode":     class.DockingCode,
			"dockingPlatform": class.DockingPlatform,
		},
		"school":   input.School,
		"account":  input.Account,
		"password": input.Password,
		"type":     input.Type,
	})
	if err != nil {
		return nil, nil, err
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, legacyCourseQueryURL(connector), bytes.NewReader(payload))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("connector returned %d: %s", resp.StatusCode, truncateLegacy(body, 300))
	}
	return legacyNormalizeCourseQueryResponse(body)
}

func legacyCourseQueryFromConnectorResult(result map[string]any, candidates []connectoradapter.CourseCandidate) fiber.Map {
	output := fiber.Map{}
	for key, value := range result {
		output[key] = value
	}
	if _, ok := output["data"]; !ok {
		output["data"] = connectoradapter.CourseCandidateRows(candidates)
	}
	return output
}

func legacyCandidatesFromConnector(candidates []connectoradapter.CourseCandidate) []legacyCourseCandidate {
	result := make([]legacyCourseCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		raw := fiber.Map{}
		for key, value := range candidate.Raw {
			raw[key] = value
		}
		result = append(result, legacyCourseCandidate{ID: candidate.ID, Name: candidate.Name, Raw: raw})
	}
	return result
}

func legacyCourseQueryURL(connector models.Connector) string {
	return strings.TrimRight(connector.BaseURL, "/") + "/courses/query"
}

func legacyNormalizeCourseQueryResponse(body []byte) (fiber.Map, []legacyCourseCandidate, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return fiber.Map{"code": 1, "msg": "查询成功", "data": []fiber.Map{}}, nil, nil
	}
	var parsed any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&parsed); err != nil {
		return nil, nil, fmt.Errorf("decode connector response: %w", err)
	}
	switch value := parsed.(type) {
	case map[string]any:
		result := fiber.Map{}
		for key, item := range value {
			result[key] = item
		}
		if _, ok := result["code"]; !ok {
			result["code"] = 1
		}
		if _, ok := result["msg"]; !ok {
			result["msg"] = "查询成功"
		}
		candidates := legacyCourseCandidatesFromAny(result["data"])
		if _, ok := result["data"]; !ok {
			result["data"] = legacyCourseCandidateRows(candidates)
		}
		return result, candidates, nil
	case []any:
		candidates := legacyCourseCandidatesFromAny(value)
		return fiber.Map{"code": 1, "msg": "查询成功", "data": legacyCourseCandidateRows(candidates)}, candidates, nil
	default:
		return nil, nil, fmt.Errorf("connector response must be object or array")
	}
}

func legacyCourseCandidatesFromAny(value any) []legacyCourseCandidate {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	candidates := make([]legacyCourseCandidate, 0, len(items))
	for _, item := range items {
		switch row := item.(type) {
		case map[string]any:
			raw := fiber.Map{}
			for key, value := range row {
				raw[key] = value
			}
			id := firstLegacyString(row, "id", "kcid", "courseId", "course_id")
			name := firstLegacyString(row, "name", "kcname", "courseName", "course_name", "title")
			if id == "" && name == "" {
				continue
			}
			candidates = append(candidates, legacyCourseCandidate{ID: id, Name: name, Raw: raw})
		case string:
			name := strings.TrimSpace(row)
			if name != "" {
				candidates = append(candidates, legacyCourseCandidate{Name: name, Raw: fiber.Map{"name": name}})
			}
		}
	}
	return candidates
}

func firstLegacyString(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func legacyCourseCandidateRows(candidates []legacyCourseCandidate) []fiber.Map {
	rows := make([]fiber.Map, 0, len(candidates))
	for _, candidate := range candidates {
		row := fiber.Map{}
		for key, value := range candidate.Raw {
			row[key] = value
		}
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

func legacyXiaochuCourseResult(result fiber.Map, class models.CourseClass, input legacyCourseQueryInput, candidates []legacyCourseCandidate) fiber.Map {
	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Name != "" {
			names = append(names, candidate.Name)
		}
	}
	return fiber.Map{
		"code": result["code"],
		"msg":  result["msg"],
		"data": []string{class.Name, input.Account, input.Password, input.School, strings.Join(names, ",")},
		"js":   "",
		"info": "昔日之苦，安知异日不在尝之? 共勉",
	}
}

func legacyBestCourseMatch(candidates []legacyCourseCandidate, target string) (legacyCourseCandidate, bool) {
	target = strings.TrimSpace(target)
	if target == "" {
		return legacyCourseCandidate{}, false
	}
	var best legacyCourseCandidate
	bestScore := 0.0
	for _, candidate := range candidates {
		score := legacyCourseNameSimilarity(candidate.Name, target)
		if score > bestScore {
			best = candidate
			bestScore = score
		}
	}
	return best, bestScore >= 0.9
}

func legacyCourseNameSimilarity(left, right string) float64 {
	left = strings.ToLower(strings.TrimSpace(left))
	right = strings.ToLower(strings.TrimSpace(right))
	if left == "" || right == "" {
		return 0
	}
	if left == right {
		return 1
	}
	if strings.Contains(left, right) || strings.Contains(right, left) {
		return 0.95
	}
	leftRunes := []rune(left)
	rightRunes := []rune(right)
	distance := legacyEditDistance(leftRunes, rightRunes)
	longest := len(leftRunes)
	if len(rightRunes) > longest {
		longest = len(rightRunes)
	}
	if longest == 0 {
		return 0
	}
	return 1 - float64(distance)/float64(longest)
}

func legacyEditDistance(left, right []rune) int {
	if len(left) == 0 {
		return len(right)
	}
	if len(right) == 0 {
		return len(left)
	}
	prev := make([]int, len(right)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(left); i++ {
		current := make([]int, len(right)+1)
		current[0] = i
		for j := 1; j <= len(right); j++ {
			cost := 0
			if left[i-1] != right[j-1] {
				cost = 1
			}
			current[j] = minInt(current[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = current
	}
	return prev[len(right)]
}

func minInt(values ...int) int {
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
}

func truncateLegacy(value []byte, limit int) string {
	text := strings.TrimSpace(string(value))
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}

func legacyOrderRows(orders []models.Order) []fiber.Map {
	rows := make([]fiber.Map, 0, len(orders))
	for _, order := range orders {
		rows = append(rows, fiber.Map{
			"id":              order.ID,
			"ptname":          order.Platform,
			"school":          order.School,
			"name":            order.StudentName,
			"user":            order.Account,
			"kcname":          order.CourseName,
			"addtime":         order.CreatedAt.Format("2006-01-02 15:04:05"),
			"courseStartTime": "",
			"courseEndTime":   "",
			"examStartTime":   "",
			"examEndTime":     "",
			"status":          order.Status,
			"process":         order.Progress,
			"remarks":         order.Remarks,
		})
	}
	return rows
}

func legacyAPIUser(c *fiber.Ctx, users *repository.UserRepository) (models.User, bool, error) {
	uid := legacyUint(c, "uid")
	key := strings.TrimSpace(legacyValue(c, "key"))
	if uid == 0 || key == "" {
		return models.User{}, false, legacyFail(c, 0, "所有项目不能为空")
	}
	user, err := users.Find(c.UserContext(), uid)
	if err != nil {
		if repository.IsNotFound(err) {
			return models.User{}, false, legacyFail(c, -1, "用户不存在")
		}
		return models.User{}, false, err
	}
	if user.APIKey == "" || user.APIKey == "0" {
		return models.User{}, false, legacyFail(c, -1, "你还没有开通接口哦")
	}
	if user.APIKey != key {
		return models.User{}, false, legacyFail(c, -2, "密匙错误")
	}
	if user.Status != "" && user.Status != "active" {
		return models.User{}, false, legacyFail(c, -1, "账号已停用")
	}
	return user, true, nil
}

func legacyFail(c *fiber.Ctx, code int, message string) error {
	return c.JSON(fiber.Map{"code": code, "msg": message})
}

func legacyValue(c *fiber.Ctx, key string) string {
	if value := strings.TrimSpace(c.FormValue(key)); value != "" {
		return value
	}
	return strings.TrimSpace(c.Query(key))
}

func legacyUint(c *fiber.Ctx, key string) uint {
	value, err := strconv.ParseUint(legacyValue(c, key), 10, 64)
	if err != nil {
		return 0
	}
	return uint(value)
}

func legacyInt(c *fiber.Ctx, key string) int {
	value, err := strconv.Atoi(legacyValue(c, key))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func legacyBool(c *fiber.Ctx, key string) bool {
	value := strings.ToLower(legacyValue(c, key))
	return value == "1" || value == "true" || value == "yes"
}

func legacyBoolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func legacyClassStatus(status string) int {
	if status == "online" {
		return 1
	}
	return 0
}

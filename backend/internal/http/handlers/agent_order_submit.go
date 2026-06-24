package handlers

import (
	"context"
	"strconv"
	"strings"

	connectoradapter "dw0rdwk/backend/internal/connectors"
	appmw "dw0rdwk/backend/internal/http/middleware"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/platforms"
	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type agentOrderSubmitBootstrap struct {
	Profile     agentProfile            `json:"profile"`
	Categories  []models.CourseCategory `json:"categories"`
	FavoriteIDs []uint                  `json:"favoriteIds"`
	Favorites   []agentClassRow         `json:"favorites"`
}

type agentCourseQueryPayload struct {
	ClassID         uint   `json:"classId"`
	School          string `json:"school"`
	Account         string `json:"account"`
	AccountPassword string `json:"accountPassword"`
	Type            string `json:"type"`
}

type agentCourseCandidateRow struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Raw  map[string]any `json:"raw"`
}

type agentCourseQueryResponse struct {
	ClassID    uint                      `json:"classId"`
	Raw        map[string]any            `json:"raw"`
	Candidates []agentCourseCandidateRow `json:"candidates"`
	UserInfo   string                    `json:"userinfo"`
}

type agentBatchOrderPayload struct {
	ClassID uint                `json:"classId"`
	Entries []agentOrderPayload `json:"entries"`
}

type agentBatchOrderItem struct {
	Index   int    `json:"index"`
	OrderID uint   `json:"orderId"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

type agentBatchOrderResponse struct {
	Requested int                   `json:"requested"`
	Succeeded int                   `json:"succeeded"`
	Failed    int                   `json:"failed"`
	Items     []agentBatchOrderItem `json:"items"`
}

func AgentOrderSubmitBootstrap(
	users *repository.UserRepository,
	categories *repository.CategoryRepository,
	favorites *repository.ClassFavoriteRepository,
	classes *repository.ClassRepository,
	specialPrices *repository.SpecialPriceRepository,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		categoryItems, err := categories.ListActive(c.UserContext(), 500)
		if err != nil {
			return err
		}
		favoriteRows, err := favorites.ListByUser(c.UserContext(), user.ID)
		if err != nil {
			return err
		}
		favoriteIDs := make([]uint, 0, len(favoriteRows))
		favoriteClasses := make([]agentClassRow, 0, len(favoriteRows))
		for _, favorite := range favoriteRows {
			favoriteIDs = append(favoriteIDs, favorite.ClassID)
			class, err := classes.Find(c.UserContext(), favorite.ClassID)
			if err != nil {
				if repository.IsNotFound(err) {
					continue
				}
				return err
			}
			if class.Status != "online" {
				continue
			}
			special, hasSpecial, err := specialPrices.FindForUserClass(c.UserContext(), user.ID, class.ID)
			if err != nil {
				return err
			}
			favoriteClasses = append(favoriteClasses, agentClassRow{
				CourseClass: class,
				UserPrice:   agentPrice(user, class, specialPricePtr(special, hasSpecial)),
			})
		}
		return OK(c, agentOrderSubmitBootstrap{
			Profile: agentProfile{
				ID:              user.ID,
				Account:         user.Account,
				Name:            user.Name,
				Balance:         user.Balance,
				PriceRate:       user.PriceRate,
				Role:            user.Role,
				InviteCode:      user.InviteCode,
				InvitePriceRate: user.InvitePriceRate,
				Notice:          user.Notice,
			},
			Categories:  categoryItems,
			FavoriteIDs: favoriteIDs,
			Favorites:   favoriteClasses,
		})
	}
}

func AgentCourseQuery(
	users *repository.UserRepository,
	classes *repository.ClassRepository,
	connectors *repository.ConnectorRepository,
	plugins *repository.PlatformPluginRepository,
	settings *repository.SettingRepository,
	logs *repository.LogRepository,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		var req agentCourseQueryPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if req.ClassID == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "classId is required")
		}
		if strings.TrimSpace(req.Account) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "account is required")
		}
		class, err := classes.Find(c.UserContext(), req.ClassID)
		if err != nil {
			return err
		}
		if class.Status != "online" || !class.BridgeEnabled {
			return fiber.NewError(fiber.StatusBadRequest, "class is not available")
		}
		if pluginCode, ok := pluginCodeForClass(class, "query"); ok {
			plugin, err := pluginForPurpose(c.UserContext(), plugins, pluginCode, "query")
			if err != nil {
				return err
			}
			result, err := platforms.DefaultRegistry().Query(c.UserContext(), plugin.Code, platforms.CourseQueryInput{
				Class:    class,
				School:   req.School,
				Account:  req.Account,
				Password: req.AccountPassword,
				Type:     req.Type,
			})
			if err != nil {
				return err
			}
			rows := make([]agentCourseCandidateRow, 0, len(result.Candidates))
			for _, candidate := range result.Candidates {
				rows = append(rows, agentCourseCandidateRow{ID: candidate.ID, Name: candidate.Name, Raw: candidate.Raw})
			}
			auditLog(c, logs, "agent_course_query_plugin", auditText("class", class.ID, "plugin="+plugin.Code+" user="+strconv.FormatUint(uint64(user.ID), 10)+" account="+strings.TrimSpace(req.Account)), 0)
			return OK(c, agentCourseQueryResponse{
				ClassID:    class.ID,
				Raw:        result.Raw,
				Candidates: rows,
				UserInfo:   result.UserInfo,
			})
		}
		connector, err := connectorForClass(c.UserContext(), connectors, settings, class, "query")
		if err != nil {
			return err
		}
		if !connectoradapter.Is29WKKind(connector.Kind) {
			return fiber.NewError(fiber.StatusBadRequest, "class connector does not support course query")
		}
		raw, candidates, err := connectoradapter.Query29WKCourses(c.UserContext(), connector, connectoradapter.CourseQueryInput{
			Class:    class,
			School:   req.School,
			Account:  req.Account,
			Password: req.AccountPassword,
			Type:     req.Type,
		})
		if err != nil {
			return err
		}
		rows := make([]agentCourseCandidateRow, 0, len(candidates))
		for _, candidate := range candidates {
			rows = append(rows, agentCourseCandidateRow{ID: candidate.ID, Name: candidate.Name, Raw: candidate.Raw})
		}
		userInfo := strings.TrimSpace(strings.Join([]string{strings.TrimSpace(req.School), strings.TrimSpace(req.Account), strings.TrimSpace(req.AccountPassword)}, " "))
		auditLog(c, logs, "agent_course_query", auditText("class", class.ID, "user="+strconv.FormatUint(uint64(user.ID), 10)+" account="+strings.TrimSpace(req.Account)), 0)
		return OK(c, agentCourseQueryResponse{
			ClassID:    class.ID,
			Raw:        raw,
			Candidates: rows,
			UserInfo:   userInfo,
		})
	}
}

func AgentBatchCreateOrders(
	users *repository.UserRepository,
	classes *repository.ClassRepository,
	specialPrices *repository.SpecialPriceRepository,
	connectors *repository.ConnectorRepository,
	plugins *repository.PlatformPluginRepository,
	settings *repository.SettingRepository,
	orderService *service.OrderService,
	logs *repository.LogRepository,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		var req agentBatchOrderPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if len(req.Entries) == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "entries are required")
		}
		if len(req.Entries) > 100 {
			return fiber.NewError(fiber.StatusBadRequest, "too many entries")
		}
		result := agentBatchOrderResponse{Requested: len(req.Entries), Items: make([]agentBatchOrderItem, 0, len(req.Entries))}
		for index, entry := range req.Entries {
			if entry.ClassID == 0 {
				entry.ClassID = req.ClassID
			}
			order, err := createAgentOrderFromPayload(c.UserContext(), c.IP(), user, entry, users, classes, specialPrices, connectors, plugins, settings, orderService)
			if err != nil {
				result.Failed++
				result.Items = append(result.Items, agentBatchOrderItem{Index: index, Status: "failed", Error: orderSubmitErrorMessage(err)})
				continue
			}
			result.Succeeded++
			result.Items = append(result.Items, agentBatchOrderItem{Index: index, OrderID: order.ID, Status: order.Status})
			auditLog(c, logs, "agent_order_batch_create", auditText("order", order.ID, "class="+order.Platform), -order.Fee)
		}
		return OK(c, result)
	}
}

func AgentClassFavorites(favorites *repository.ClassFavoriteRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		ids, err := favorites.IDsByUser(c.UserContext(), claims.UID)
		if err != nil {
			return err
		}
		result := make([]uint, 0, len(ids))
		for id := range ids {
			result = append(result, id)
		}
		return OK(c, fiber.Map{"classIds": result})
	}
}

func AddAgentClassFavorite(favorites *repository.ClassFavoriteRepository, classes *repository.ClassRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		classID, err := pathID(c)
		if err != nil {
			return err
		}
		if _, err := classes.Find(c.UserContext(), classID); err != nil {
			return err
		}
		if err := favorites.Add(c.UserContext(), claims.UID, classID); err != nil {
			return err
		}
		auditLog(c, logs, "agent_class_favorite_add", auditText("class", classID, ""), 0)
		return OK(c, fiber.Map{"classId": classID})
	}
}

func RemoveAgentClassFavorite(favorites *repository.ClassFavoriteRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		classID, err := pathID(c)
		if err != nil {
			return err
		}
		if err := favorites.Remove(c.UserContext(), claims.UID, classID); err != nil {
			return err
		}
		auditLog(c, logs, "agent_class_favorite_remove", auditText("class", classID, ""), 0)
		return OK(c, fiber.Map{"classId": classID})
	}
}

func createAgentOrderFromPayload(
	ctx context.Context,
	sourceIP string,
	user models.User,
	req agentOrderPayload,
	users *repository.UserRepository,
	classes *repository.ClassRepository,
	specialPrices *repository.SpecialPriceRepository,
	connectors *repository.ConnectorRepository,
	plugins *repository.PlatformPluginRepository,
	settings *repository.SettingRepository,
	orderService *service.OrderService,
) (models.Order, error) {
	if req.ClassID == 0 {
		return models.Order{}, fiber.NewError(fiber.StatusBadRequest, "classId is required")
	}
	if strings.TrimSpace(req.Account) == "" || strings.TrimSpace(req.CourseName) == "" {
		return models.Order{}, fiber.NewError(fiber.StatusBadRequest, "account and courseName are required")
	}
	if req.DurationMinutes < 0 {
		return models.Order{}, fiber.NewError(fiber.StatusBadRequest, "durationMinutes must be zero or greater")
	}
	class, err := classes.Find(ctx, req.ClassID)
	if err != nil {
		return models.Order{}, err
	}
	if class.Status != "online" || !class.BridgeEnabled {
		return models.Order{}, fiber.NewError(fiber.StatusBadRequest, "class is not available")
	}
	route, err := executionRouteForClass(ctx, connectors, plugins, settings, class, "order")
	if err != nil {
		return models.Order{}, err
	}
	special, hasSpecial, err := specialPrices.FindForUserClass(ctx, user.ID, class.ID)
	if err != nil {
		return models.Order{}, err
	}
	fee := agentPrice(user, class, specialPricePtr(special, hasSpecial))
	if err := users.DeductBalanceIfEnough(ctx, user.ID, fee); err != nil {
		if repository.IsNotFound(err) {
			return models.Order{}, fiber.NewError(fiber.StatusBadRequest, "insufficient balance")
		}
		return models.Order{}, err
	}
	order := models.Order{
		UserID:          user.ID,
		ClassID:         class.ID,
		ConnectorID:     route.ConnectorID,
		ExecutionMode:   route.ExecutionMode,
		PluginCode:      route.PluginCode,
		Platform:        class.Name,
		School:          strings.TrimSpace(req.School),
		StudentName:     strings.TrimSpace(req.StudentName),
		Account:         strings.TrimSpace(req.Account),
		AccountPassword: strings.TrimSpace(req.AccountPassword),
		CourseID:        strings.TrimSpace(req.CourseID),
		CourseName:      strings.TrimSpace(req.CourseName),
		Fee:             fee,
		DockingCode:     class.DockingCode,
		FlashMode:       req.FlashMode,
		SourceIP:        sourceIP,
		DurationMinutes: req.DurationMinutes,
	}
	if err := orderService.Submit(ctx, &order); err != nil {
		_ = users.AdjustBalance(ctx, user.ID, fee)
		return models.Order{}, err
	}
	return order, nil
}

type classExecutionRoute struct {
	ConnectorID   uint
	ExecutionMode string
	PluginCode    string
}

func executionRouteForClass(ctx context.Context, connectors *repository.ConnectorRepository, plugins *repository.PlatformPluginRepository, settings *repository.SettingRepository, class models.CourseClass, purpose string) (classExecutionRoute, error) {
	if pluginCode, ok := pluginCodeForClass(class, purpose); ok {
		plugin, err := pluginForPurpose(ctx, plugins, pluginCode, purpose)
		if err != nil {
			return classExecutionRoute{}, err
		}
		return classExecutionRoute{ExecutionMode: "plugin", PluginCode: plugin.Code}, nil
	}
	connector, err := connectorForClass(ctx, connectors, settings, class, purpose)
	if err != nil {
		return classExecutionRoute{}, err
	}
	return classExecutionRoute{ConnectorID: connector.ID, ExecutionMode: "connector"}, nil
}

func pluginCodeForClass(class models.CourseClass, purpose string) (string, bool) {
	candidates := []string{class.DockingPlatform, class.QueryPlatform}
	if purpose == "query" {
		candidates = []string{class.QueryPlatform, class.DockingPlatform}
	}
	for _, candidate := range candidates {
		if code, ok := platforms.PluginCodeFromPlatform(candidate); ok {
			return code, true
		}
	}
	return "", false
}

func pluginForPurpose(ctx context.Context, plugins *repository.PlatformPluginRepository, code string, purpose string) (models.PlatformPlugin, error) {
	if plugins == nil {
		return models.PlatformPlugin{}, fiber.NewError(fiber.StatusBadRequest, "platform plugin repository is unavailable")
	}
	plugin, err := plugins.Find(ctx, code)
	if err != nil {
		if repository.IsNotFound(err) {
			return models.PlatformPlugin{}, fiber.NewError(fiber.StatusBadRequest, "platform plugin not found")
		}
		return models.PlatformPlugin{}, err
	}
	if plugin.Status != "active" {
		return models.PlatformPlugin{}, fiber.NewError(fiber.StatusBadRequest, "platform plugin is disabled")
	}
	switch purpose {
	case "query":
		if !plugin.SupportsQuery {
			return models.PlatformPlugin{}, fiber.NewError(fiber.StatusBadRequest, "platform plugin does not support course query")
		}
	case "refresh":
		if !plugin.SupportsRefresh {
			return models.PlatformPlugin{}, fiber.NewError(fiber.StatusBadRequest, "platform plugin does not support refresh")
		}
	default:
		if !plugin.SupportsSubmit {
			return models.PlatformPlugin{}, fiber.NewError(fiber.StatusBadRequest, "platform plugin does not support submit")
		}
	}
	if !platforms.DefaultRegistry().Has(plugin.Code) {
		return models.PlatformPlugin{}, fiber.NewError(fiber.StatusBadRequest, "platform plugin adapter is not installed")
	}
	return plugin, nil
}

func connectorForClass(ctx context.Context, connectors *repository.ConnectorRepository, settings *repository.SettingRepository, class models.CourseClass, purpose string) (models.Connector, error) {
	candidatePlatforms := []string{class.DockingPlatform, class.QueryPlatform}
	if purpose == "query" {
		candidatePlatforms = []string{class.QueryPlatform, class.DockingPlatform}
	}
	for _, raw := range candidatePlatforms {
		id, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
		if err != nil || id == 0 {
			continue
		}
		connector, err := connectors.Find(ctx, uint(id))
		if err != nil {
			return models.Connector{}, err
		}
		if err := validateConnectorForPurpose(connector, purpose); err != nil {
			return models.Connector{}, err
		}
		return connector, nil
	}
	connector, err := defaultConnectorFromContext(ctx, connectors, settings)
	if err != nil {
		return models.Connector{}, err
	}
	if err := validateConnectorForPurpose(connector, purpose); err != nil {
		return models.Connector{}, err
	}
	return connector, nil
}

func defaultConnectorFromContext(ctx context.Context, connectors *repository.ConnectorRepository, settings *repository.SettingRepository) (models.Connector, error) {
	values, err := settings.All(ctx)
	if err == nil {
		if raw := strings.TrimSpace(values["default_connector_id"]); raw != "" {
			id, parseErr := strconv.ParseUint(raw, 10, 64)
			if parseErr != nil || id == 0 {
				return models.Connector{}, fiber.NewError(fiber.StatusBadRequest, "default_connector_id is invalid")
			}
			return connectors.Find(ctx, uint(id))
		}
	}
	connector, err := connectors.FirstActive(ctx)
	if err != nil {
		if repository.IsNotFound(err) {
			return models.Connector{}, fiber.NewError(fiber.StatusBadRequest, "default connector is not configured")
		}
		return models.Connector{}, err
	}
	return connector, nil
}

func validateConnectorForPurpose(connector models.Connector, purpose string) error {
	if connector.Status != "active" || strings.TrimSpace(connector.BaseURL) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "connector is disabled")
	}
	if purpose == "order" && !connector.OrderSyncEnabled {
		return fiber.NewError(fiber.StatusBadRequest, "connector order sync is disabled")
	}
	return nil
}

func orderSubmitErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

package handlers

import (
	"strings"

	appmw "dw0rdwk/backend/internal/http/middleware"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type recommendationPayload struct {
	ClassID   uint   `json:"classId"`
	Title     string `json:"title"`
	Note      string `json:"note"`
	SortOrder int    `json:"sortOrder"`
	Visible   *bool  `json:"visible"`
}

type agentRecommendationRow struct {
	ID        uint          `json:"id"`
	Title     string        `json:"title"`
	Note      string        `json:"note"`
	SortOrder int           `json:"sortOrder"`
	Class     agentClassRow `json:"class"`
}

type agentRecommendationsResult struct {
	Items        []agentRecommendationRow         `json:"items"`
	PlatformRank []repository.OrderLeaderboardRow `json:"platformRank"`
	CategoryRank []repository.OrderLeaderboardRow `json:"categoryRank"`
	OrderTips    string                           `json:"orderTips"`
}

func Recommendations(repo *repository.RecommendationRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		includeHidden := strings.EqualFold(c.Query("includeHidden"), "true")
		page, err := repo.List(c.UserContext(), includeHidden, queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, page)
	}
}

func CreateRecommendation(repo *repository.RecommendationRepository, classes *repository.ClassRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req recommendationPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateRecommendationPayload(req); err != nil {
			return err
		}
		if _, err := classes.Find(c.UserContext(), req.ClassID); err != nil {
			return err
		}
		item := models.RecommendedClass{
			ClassID:   req.ClassID,
			Title:     strings.TrimSpace(req.Title),
			Note:      strings.TrimSpace(req.Note),
			SortOrder: req.SortOrder,
			Visible:   boolPointerValue(req.Visible, true),
		}
		if err := repo.Create(c.UserContext(), &item); err != nil {
			return err
		}
		auditLog(c, logs, "admin_recommendation_create", auditText("recommendation", item.ID, "class="+strings.TrimSpace(req.Title)), 0)
		return OK(c, item)
	}
}

func UpdateRecommendation(repo *repository.RecommendationRepository, classes *repository.ClassRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req recommendationPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateRecommendationPayload(req); err != nil {
			return err
		}
		if _, err := classes.Find(c.UserContext(), req.ClassID); err != nil {
			return err
		}
		values := map[string]any{
			"class_id":   req.ClassID,
			"title":      strings.TrimSpace(req.Title),
			"note":       strings.TrimSpace(req.Note),
			"sort_order": req.SortOrder,
			"visible":    boolPointerValue(req.Visible, true),
		}
		if err := repo.Update(c.UserContext(), id, values); err != nil {
			return err
		}
		auditLog(c, logs, "admin_recommendation_update", auditText("recommendation", id, auditFields(values)), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func DeleteRecommendation(repo *repository.RecommendationRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if err := repo.Delete(c.UserContext(), id); err != nil {
			return err
		}
		auditLog(c, logs, "admin_recommendation_delete", auditText("recommendation", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func AgentOrderSubmitRecommendations(
	users *repository.UserRepository,
	repo *repository.RecommendationRepository,
	specialPrices *repository.SpecialPriceRepository,
	orders *repository.OrderRepository,
	settings *repository.SettingRepository,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		user, err := users.Find(c.UserContext(), claims.UID)
		if err != nil {
			return err
		}
		recommendations, err := repo.ListVisible(c.UserContext(), 8)
		if err != nil {
			return err
		}
		items := make([]agentRecommendationRow, 0, len(recommendations))
		for _, row := range recommendations {
			class := models.CourseClass{
				ID:              row.ClassID,
				Name:            row.ClassName,
				Category:        row.ClassCategory,
				Price:           row.ClassPrice,
				Status:          row.ClassStatus,
				DockingPlatform: row.ClassDocking,
				QueryPlatform:   row.ClassQuery,
				Description:     row.ClassDesc,
				BridgeEnabled:   row.ClassBridge,
			}
			special, hasSpecial, err := specialPrices.FindForUserClass(c.UserContext(), user.ID, class.ID)
			if err != nil {
				return err
			}
			items = append(items, agentRecommendationRow{
				ID:        row.ID,
				Title:     defaultRecommendationTitle(row.Title, row.ClassName),
				Note:      row.Note,
				SortOrder: row.SortOrder,
				Class: agentClassRow{
					CourseClass: class,
					UserPrice:   agentPrice(user, class, specialPricePtr(special, hasSpecial)),
				},
			})
		}
		platformRank, err := orders.PlatformLeaderboard(c.UserContext(), 10)
		if err != nil {
			return err
		}
		categoryRank, err := orders.CategoryLeaderboard(c.UserContext(), 10)
		if err != nil {
			return err
		}
		values, _ := settings.All(c.UserContext())
		return OK(c, agentRecommendationsResult{
			Items:        items,
			PlatformRank: platformRank,
			CategoryRank: categoryRank,
			OrderTips:    strings.TrimSpace(values["order_tips"]),
		})
	}
}

func validateRecommendationPayload(req recommendationPayload) error {
	if req.ClassID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "classId is required")
	}
	if len([]rune(strings.TrimSpace(req.Title))) > 160 {
		return fiber.NewError(fiber.StatusBadRequest, "title is too long")
	}
	if len([]rune(strings.TrimSpace(req.Note))) > 5000 {
		return fiber.NewError(fiber.StatusBadRequest, "note is too long")
	}
	if req.SortOrder < 0 || req.SortOrder > 100000 {
		return fiber.NewError(fiber.StatusBadRequest, "sortOrder is invalid")
	}
	return nil
}

func boolPointerValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func defaultRecommendationTitle(title string, className string) string {
	title = strings.TrimSpace(title)
	if title != "" {
		return title
	}
	return strings.TrimSpace(className)
}

package handlers

import (
	"crypto/rand"
	"math/big"
	"strconv"
	"strings"

	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

const rechargeCardAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

type rechargeCardCreatePayload struct {
	Count  int      `json:"count"`
	Amount float64  `json:"amount"`
	Codes  []string `json:"codes"`
}

type rechargeCardCodePayload struct {
	Code string `json:"code"`
}

func RechargeCards(repo *repository.RechargeCardRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := repo.List(c.UserContext(), c.Query("q"), c.Query("status"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func CreateRechargeCards(repo *repository.RechargeCardRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req rechargeCardCreatePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateRechargeCardCreatePayload(req); err != nil {
			return err
		}
		codes := normalizeRechargeCardCodes(req.Codes)
		items := make([]models.RechargeCard, 0, rechargeCardCreateCount(req, codes))
		if len(codes) > 0 {
			for _, code := range codes {
				items = append(items, models.RechargeCard{Code: code, Amount: req.Amount, Status: "unused"})
			}
		} else {
			seen := map[string]struct{}{}
			for len(items) < req.Count {
				code, err := randomRechargeCardCode(16)
				if err != nil {
					return err
				}
				if _, ok := seen[code]; ok {
					continue
				}
				seen[code] = struct{}{}
				items = append(items, models.RechargeCard{Code: code, Amount: req.Amount, Status: "unused"})
			}
		}
		if err := repo.CreateBatch(c.UserContext(), items); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_recharge_card_create", "count="+strconv.Itoa(len(items))+" amount="+strconv.FormatFloat(req.Amount, 'f', 2, 64), 0)
		return OK(c, fiber.Map{"items": items})
	}
}

func DeleteRechargeCard(repo *repository.RechargeCardRepository, dashboard *service.DashboardService, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if err := repo.Delete(c.UserContext(), id); err != nil {
			return err
		}
		invalidateDashboard(c.UserContext(), dashboard)
		auditLog(c, logs, "admin_recharge_card_delete", auditText("recharge_card", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func AgentQueryRechargeCard(repo *repository.RechargeCardRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req rechargeCardCodePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		code := strings.TrimSpace(req.Code)
		if code == "" {
			return fiber.NewError(fiber.StatusBadRequest, "code is required")
		}
		item, err := repo.FindByCode(c.UserContext(), code)
		if err != nil {
			if repository.IsNotFound(err) {
				return fiber.NewError(fiber.StatusNotFound, "recharge card not found")
			}
			return err
		}
		return OK(c, item)
	}
}

func AgentRedeemRechargeCard(users *repository.UserRepository, repo *repository.RechargeCardRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := currentUser(c, users)
		if err != nil {
			return err
		}
		var req rechargeCardCodePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		code := strings.TrimSpace(req.Code)
		if code == "" {
			return fiber.NewError(fiber.StatusBadRequest, "code is required")
		}
		card, err := repo.Redeem(c.UserContext(), code, user.ID)
		if err != nil {
			if repository.IsRechargeCardUsed(err) {
				return fiber.NewError(fiber.StatusBadRequest, "recharge card already used")
			}
			if repository.IsNotFound(err) {
				return fiber.NewError(fiber.StatusBadRequest, "recharge card not available")
			}
			return err
		}
		auditLog(c, logs, "agent_recharge_card_redeem", auditText("recharge_card", card.ID, "code="+card.Code), card.Amount)
		return OK(c, card)
	}
}

func validateRechargeCardCreatePayload(req rechargeCardCreatePayload) error {
	if req.Amount <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "amount must be greater than zero")
	}
	codes := normalizeRechargeCardCodes(req.Codes)
	if len(codes) > 0 {
		if len(codes) > 100 {
			return fiber.NewError(fiber.StatusBadRequest, "codes must contain 100 or fewer cards")
		}
		if hasDuplicateRechargeCardCode(codes) {
			return fiber.NewError(fiber.StatusBadRequest, "codes must not contain duplicates")
		}
		return nil
	}
	if req.Count < 1 || req.Count > 100 {
		return fiber.NewError(fiber.StatusBadRequest, "count must be between 1 and 100")
	}
	return nil
}

func normalizeRechargeCardCodes(codes []string) []string {
	normalized := make([]string, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code != "" {
			normalized = append(normalized, code)
		}
	}
	return normalized
}

func hasDuplicateRechargeCardCode(codes []string) bool {
	seen := make(map[string]bool, len(codes))
	for _, code := range codes {
		key := strings.ToLower(code)
		if seen[key] {
			return true
		}
		seen[key] = true
	}
	return false
}

func rechargeCardCreateCount(req rechargeCardCreatePayload, codes []string) int {
	if len(codes) > 0 {
		return len(codes)
	}
	return req.Count
}

func randomRechargeCardCode(length int) (string, error) {
	if length < 1 {
		length = 16
	}
	limit := big.NewInt(int64(len(rechargeCardAlphabet)))
	var builder strings.Builder
	builder.Grow(length)
	for builder.Len() < length {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		builder.WriteByte(rechargeCardAlphabet[n.Int64()])
	}
	return builder.String(), nil
}

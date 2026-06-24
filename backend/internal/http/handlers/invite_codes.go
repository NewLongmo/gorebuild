package handlers

import (
	"crypto/rand"
	"strings"
	"time"

	appmw "dw0rdwk/backend/internal/http/middleware"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type inviteCodePayload struct {
	Code      string  `json:"code"`
	Note      string  `json:"note"`
	MaxUses   int     `json:"maxUses"`
	PriceRate float64 `json:"priceRate"`
	Status    string  `json:"status"`
	ExpiresAt string  `json:"expiresAt"`
}

func InviteCodes(repo *repository.InviteCodeRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		data, err := repo.List(c.UserContext(), c.Query("q"), c.Query("status"), queryInt(c, "page", 1), queryInt(c, "perPage", 20))
		if err != nil {
			return err
		}
		return OK(c, data)
	}
}

func CreateInviteCode(repo *repository.InviteCodeRepository, users *repository.UserRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req inviteCodePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if strings.TrimSpace(req.Code) == "" {
			code, err := generateInviteCode(10)
			if err != nil {
				return err
			}
			req.Code = code
		}
		if err := validateInvitePayload(req); err != nil {
			return err
		}
		code := repository.NormalizeInviteCode(req.Code)
		exists, err := users.InviteCodeExists(c.UserContext(), code, 0)
		if err != nil {
			return err
		}
		if exists {
			return fiber.NewError(fiber.StatusBadRequest, "invite code already exists")
		}
		expiresAt, err := parseInviteExpiresAt(req.ExpiresAt)
		if err != nil {
			return err
		}
		claims, _ := appmw.Claims(c)
		invite := models.InviteCode{
			Code:      code,
			Note:      strings.TrimSpace(req.Note),
			MaxUses:   normalizeInviteMaxUses(req.MaxUses),
			PriceRate: normalizeInvitePriceRate(req.PriceRate),
			Status:    normalizeInviteStatus(req.Status),
			CreatedBy: claims.UID,
			ExpiresAt: expiresAt,
		}
		if err := repo.Create(c.UserContext(), &invite); err != nil {
			return err
		}
		auditLog(c, logs, "admin_invite_create", auditText("invite", invite.ID, "code="+invite.Code+" status="+invite.Status), 0)
		return OK(c, invite)
	}
}

func UpdateInviteCode(repo *repository.InviteCodeRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		var req inviteCodePayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if err := validateInvitePayload(req); err != nil {
			return err
		}
		expiresAt, err := parseInviteExpiresAt(req.ExpiresAt)
		if err != nil {
			return err
		}
		values := map[string]any{
			"note":       strings.TrimSpace(req.Note),
			"max_uses":   normalizeInviteMaxUses(req.MaxUses),
			"price_rate": normalizeInvitePriceRate(req.PriceRate),
			"status":     normalizeInviteStatus(req.Status),
			"expires_at": expiresAt,
		}
		if err := repo.Update(c.UserContext(), id, values); err != nil {
			return err
		}
		auditLog(c, logs, "admin_invite_update", auditText("invite", id, auditFields(values)), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func DeleteInviteCode(repo *repository.InviteCodeRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		if err := repo.Delete(c.UserContext(), id); err != nil {
			return err
		}
		auditLog(c, logs, "admin_invite_delete", auditText("invite", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func validateInvitePayload(req inviteCodePayload) error {
	if code := strings.TrimSpace(req.Code); code != "" {
		normalized := repository.NormalizeInviteCode(code)
		if len(normalized) < 4 || len(normalized) > 64 {
			return fiber.NewError(fiber.StatusBadRequest, "invite code must be 4 to 64 characters")
		}
		if !isInviteCodeSafe(normalized) {
			return fiber.NewError(fiber.StatusBadRequest, "invite code may only contain letters, numbers, dash, and underscore")
		}
	}
	if req.MaxUses < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "maxUses must be zero or greater")
	}
	if req.PriceRate < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "priceRate must be zero or greater")
	}
	if status := strings.TrimSpace(req.Status); status != "" && !oneOf(status, "active", "disabled") {
		return fiber.NewError(fiber.StatusBadRequest, "status must be active or disabled")
	}
	if _, err := parseInviteExpiresAt(req.ExpiresAt); err != nil {
		return err
	}
	return nil
}

func parseInviteExpiresAt(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "expiresAt must be an RFC3339 datetime")
	}
	return &parsed, nil
}

func normalizeInviteMaxUses(value int) int {
	if value == 0 {
		return 1
	}
	return value
}

func normalizeInvitePriceRate(value float64) float64 {
	if value == 0 {
		return 1
	}
	return value
}

func normalizeInviteStatus(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "active"
	}
	return value
}

func isInviteCodeSafe(value string) bool {
	for _, r := range value {
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func generateInviteCode(length int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if length < 4 {
		length = 4
	}
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for i := range bytes {
		bytes[i] = alphabet[int(bytes[i])%len(alphabet)]
	}
	return string(bytes), nil
}

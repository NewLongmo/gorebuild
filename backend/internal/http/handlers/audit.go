package handlers

import (
	"fmt"
	"sort"
	"strings"

	appmw "dw0rdwk/backend/internal/http/middleware"
	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/repository"

	"github.com/gofiber/fiber/v2"
)

func auditLog(c *fiber.Ctx, logs *repository.LogRepository, logType string, text string, amount float64) {
	if logs == nil {
		return
	}
	claims, _ := appmw.Claims(c)
	_ = logs.Create(c.UserContext(), &models.OperationLog{
		UserID:   claims.UID,
		Type:     logType,
		Text:     strings.TrimSpace(text),
		Amount:   amount,
		SourceIP: c.IP(),
	})
}

func auditText(resource string, id uint, detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return fmt.Sprintf("%s id=%d", resource, id)
	}
	return fmt.Sprintf("%s id=%d %s", resource, id, detail)
}

func auditFields(values map[string]any) string {
	if len(values) == 0 {
		return "fields=none"
	}
	fields := make([]string, 0, len(values))
	for field := range values {
		if field == "password_hash" || field == "app_secret" {
			continue
		}
		fields = append(fields, field)
	}
	if len(fields) == 0 {
		return "fields=secret"
	}
	sort.Strings(fields)
	return "fields=" + strings.Join(fields, ",")
}

func cascadeDeleteAudit(result repository.CascadeDeleteResult) string {
	return fmt.Sprintf(
		"classes=%d favorites=%d specialPrices=%d emptyCategories=%d",
		result.DeletedClasses,
		result.DeletedFavorites,
		result.DeletedSpecialPrices,
		result.DeletedEmptyCategories,
	)
}

package middleware

import (
	"dw0rdwk/backend/internal/auth"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

const ClaimsLocalKey = "authClaims"

func RequireAuth(authService *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := authService.Validate(c.UserContext(), c.Get("Authorization"))
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		c.Locals(ClaimsLocalKey, claims)
		return c.Next()
	}
}

func RequireRole(roles ...string) fiber.Handler {
	allowed := map[string]struct{}{}
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		claims, ok := Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		if _, ok := allowed[claims.Role]; !ok {
			return fiber.NewError(fiber.StatusForbidden, "forbidden")
		}
		return c.Next()
	}
}

func Claims(c *fiber.Ctx) (auth.Claims, bool) {
	claims, ok := c.Locals(ClaimsLocalKey).(auth.Claims)
	return claims, ok
}

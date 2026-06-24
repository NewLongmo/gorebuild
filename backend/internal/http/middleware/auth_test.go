package middleware

import (
	"net/http/httptest"
	"testing"

	"dw0rdwk/backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name       string
		claims     *auth.Claims
		wantStatus int
	}{
		{name: "admin allowed", claims: &auth.Claims{Role: "admin"}, wantStatus: fiber.StatusOK},
		{name: "agent forbidden", claims: &auth.Claims{Role: "agent"}, wantStatus: fiber.StatusForbidden},
		{name: "missing claims unauthorized", wantStatus: fiber.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				if tt.claims != nil {
					c.Locals(ClaimsLocalKey, *tt.claims)
				}
				return c.Next()
			})
			app.Get("/", RequireRole("admin"), func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusOK)
			})

			resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

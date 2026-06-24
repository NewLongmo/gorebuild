package handlers

import (
	appmw "dw0rdwk/backend/internal/http/middleware"
	"dw0rdwk/backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type loginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type registerRequest struct {
	Account    string `json:"account"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	InviteCode string `json:"inviteCode"`
}

func Login(authService *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req loginRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if req.Account == "" || req.Password == "" {
			return fiber.NewError(fiber.StatusBadRequest, "account and password are required")
		}

		result, err := authService.Login(c.UserContext(), req.Account, req.Password)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid account or password")
		}
		return OK(c, result)
	}
}

func Register(authService *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req registerRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		result, err := authService.Register(c.UserContext(), service.RegisterInput{
			Account:    req.Account,
			Password:   req.Password,
			Name:       req.Name,
			InviteCode: req.InviteCode,
			SourceIP:   c.IP(),
		})
		if err != nil {
			return err
		}
		return OK(c, result)
	}
}

func Me() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		return OK(c, claims)
	}
}

func Logout(authService *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := appmw.Claims(c)
		if ok {
			_ = authService.Logout(c.UserContext(), claims)
		}
		return OK(c, fiber.Map{"ok": true})
	}
}

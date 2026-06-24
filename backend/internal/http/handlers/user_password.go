package handlers

import (
	"strings"

	"dw0rdwk/backend/internal/password"

	"github.com/gofiber/fiber/v2"
)

const defaultResetPassword = "1234567"

type resetPasswordResponse struct {
	ID       uint   `json:"id"`
	Password string `json:"password"`
}

func resetPasswordHash(raw string) (string, string, error) {
	plain := strings.TrimSpace(raw)
	if plain == "" {
		plain = defaultResetPassword
	}
	if len(plain) > 160 {
		return "", "", fiber.NewError(fiber.StatusBadRequest, "password is too long")
	}
	hash, err := password.Hash(plain)
	if err != nil {
		return "", "", err
	}
	return plain, hash, nil
}

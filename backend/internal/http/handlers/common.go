package handlers

import (
	"errors"

	"dw0rdwk/backend/internal/service"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *fiber.Ctx, data any) error {
	return c.JSON(response{Code: 0, Message: "ok", Data: data})
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	code, message := errorStatus(err)
	return c.Status(code).JSON(response{
		Code:    code,
		Message: message,
	})
}

func errorStatus(err error) (int, string) {
	code := fiber.StatusInternalServerError
	message := "internal server error"

	var fiberErr *fiber.Error
	var validationErr service.ValidationError
	var dependencyErr service.DependencyError
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &fiberErr) {
		code = fiberErr.Code
		message = fiberErr.Message
	} else if errors.As(err, &validationErr) {
		code = fiber.StatusBadRequest
		message = validationErr.Message
	} else if errors.As(err, &dependencyErr) {
		code = fiber.StatusServiceUnavailable
		message = dependencyErr.Message
	} else if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		code = fiber.StatusConflict
		message = "resource already exists"
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		code = fiber.StatusNotFound
		message = "resource not found"
	}
	return code, message
}

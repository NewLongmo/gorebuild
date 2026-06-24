package handlers

import (
	"errors"
	"testing"

	"dw0rdwk/backend/internal/service"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func TestErrorStatusMapsCommonErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantCode    int
		wantMessage string
	}{
		{
			name:        "validation",
			err:         service.ValidationError{Message: "bad input"},
			wantCode:    fiber.StatusBadRequest,
			wantMessage: "bad input",
		},
		{
			name:        "dependency",
			err:         service.DependencyError{Message: "order queue unavailable", Err: errors.New("redis disabled")},
			wantCode:    fiber.StatusServiceUnavailable,
			wantMessage: "order queue unavailable",
		},
		{
			name:        "fiber error",
			err:         fiber.NewError(fiber.StatusForbidden, "forbidden"),
			wantCode:    fiber.StatusForbidden,
			wantMessage: "forbidden",
		},
		{
			name:        "duplicate key",
			err:         &mysqlDriver.MySQLError{Number: 1062, Message: "Duplicate entry"},
			wantCode:    fiber.StatusConflict,
			wantMessage: "resource already exists",
		},
		{
			name:        "not found",
			err:         gorm.ErrRecordNotFound,
			wantCode:    fiber.StatusNotFound,
			wantMessage: "resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCode, gotMessage := errorStatus(tt.err)
			if gotCode != tt.wantCode || gotMessage != tt.wantMessage {
				t.Fatalf("errorStatus() = (%d, %q), want (%d, %q)", gotCode, gotMessage, tt.wantCode, tt.wantMessage)
			}
		})
	}
}

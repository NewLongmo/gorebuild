package handlers

import (
	"testing"
	"time"

	"dw0rdwk/backend/internal/models"
)

func TestPublicOrderRowsFromOrders(t *testing.T) {
	createdAt := time.Date(2026, 6, 24, 10, 30, 0, 0, time.UTC)
	rows := publicOrderRowsFromOrders([]models.Order{
		{
			ID:              12,
			Platform:        "平台",
			School:          "学校",
			StudentName:     "张三",
			Account:         "student-1",
			CourseName:      "课程",
			Status:          "processing",
			DockingStatus:   "sent",
			Progress:        "50%",
			Remarks:         "正常",
			Score:           "95",
			DurationMinutes: 30,
			CreatedAt:       createdAt,
			UserID:          99,
			Fee:             9.9,
			RemoteOrderID:   "remote-secret",
		},
	})
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	got := rows[0]
	if got.ID != 12 || got.Account != "student-1" || got.CourseName != "课程" {
		t.Fatalf("unexpected public row: %#v", got)
	}
	if got.CreatedAt != "2026-06-24 10:30:00" {
		t.Fatalf("CreatedAt = %q, want formatted timestamp", got.CreatedAt)
	}
}

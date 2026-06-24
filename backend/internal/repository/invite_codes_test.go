package repository

import (
	"errors"
	"testing"
	"time"

	"dw0rdwk/backend/internal/models"
)

func TestNormalizeInviteCode(t *testing.T) {
	if got := NormalizeInviteCode(" ab-c_12 "); got != "AB-C_12" {
		t.Fatalf("NormalizeInviteCode() = %q, want AB-C_12", got)
	}
}

func TestValidateInviteForUse(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	tests := []struct {
		name string
		row  models.InviteCode
		want error
	}{
		{
			name: "active with remaining uses",
			row:  models.InviteCode{Status: "active", MaxUses: 2, UsedCount: 1, ExpiresAt: &future},
		},
		{
			name: "disabled",
			row:  models.InviteCode{Status: "disabled", MaxUses: 2, UsedCount: 0},
			want: ErrInviteCodeInvalid,
		},
		{
			name: "expired",
			row:  models.InviteCode{Status: "active", MaxUses: 2, UsedCount: 0, ExpiresAt: &past},
			want: ErrInviteCodeExpired,
		},
		{
			name: "exhausted",
			row:  models.InviteCode{Status: "active", MaxUses: 1, UsedCount: 1},
			want: ErrInviteCodeExhausted,
		},
		{
			name: "invalid max uses",
			row:  models.InviteCode{Status: "active", MaxUses: -1, UsedCount: 99},
			want: ErrInviteCodeExhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInviteForUse(tt.row, now)
			if !errors.Is(err, tt.want) {
				t.Fatalf("validateInviteForUse() error = %v, want %v", err, tt.want)
			}
		})
	}
}

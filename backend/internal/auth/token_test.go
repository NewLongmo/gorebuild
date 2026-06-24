package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"dw0rdwk/backend/internal/cache"
	"dw0rdwk/backend/internal/config"
)

func TestManagerIssueValidateAndRejectTamper(t *testing.T) {
	manager := NewManager(config.AuthConfig{
		Secret:   "test-secret",
		TokenTTL: time.Hour,
	}, cache.Open(context.Background(), config.RedisConfig{Enabled: false}))

	token, claims, err := manager.Issue(context.Background(), 7, "admin", "admin")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if claims.UID != 7 || claims.Account != "admin" || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	validated, err := manager.Validate(context.Background(), "Bearer "+token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if validated.UID != claims.UID || validated.Nonce != claims.Nonce {
		t.Fatalf("validated claims mismatch: %+v vs %+v", validated, claims)
	}

	rawValidated, err := manager.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("validate raw token: %v", err)
	}
	if rawValidated.UID != claims.UID || rawValidated.Nonce != claims.Nonce {
		t.Fatalf("raw validated claims mismatch: %+v vs %+v", rawValidated, claims)
	}

	tampered := token[:len(token)-1] + "x"
	if _, err := manager.Validate(context.Background(), tampered); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token for tamper, got %v", err)
	}
}

func TestManagerRejectsExpiredToken(t *testing.T) {
	manager := NewManager(config.AuthConfig{
		Secret:   "test-secret",
		TokenTTL: -time.Second,
	}, cache.Open(context.Background(), config.RedisConfig{Enabled: false}))

	token, _, err := manager.Issue(context.Background(), 1, "expired", "agent")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	_, err = manager.Validate(context.Background(), strings.TrimSpace(token))
	if !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("expected expired token, got %v", err)
	}
}

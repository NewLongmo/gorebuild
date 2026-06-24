package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"dw0rdwk/backend/internal/cache"
	"dw0rdwk/backend/internal/config"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type Claims struct {
	UID     uint   `json:"uid"`
	Account string `json:"account"`
	Role    string `json:"role"`
	Exp     int64  `json:"exp"`
	Nonce   string `json:"nonce"`
}

type Manager struct {
	secret []byte
	ttl    time.Duration
	cache  *cache.Client
}

func NewManager(cfg config.AuthConfig, cacheClient *cache.Client) *Manager {
	if cacheClient == nil {
		cacheClient = cache.Open(context.Background(), config.RedisConfig{Enabled: false})
	}
	return &Manager{
		secret: []byte(cfg.Secret),
		ttl:    cfg.TokenTTL,
		cache:  cacheClient,
	}
}

func (m *Manager) Issue(ctx context.Context, uid uint, account string, role string) (string, Claims, error) {
	if strings.TrimSpace(role) == "" {
		role = "agent"
	}
	claims := Claims{
		UID:     uid,
		Account: account,
		Role:    role,
		Exp:     time.Now().Add(m.ttl).Unix(),
		Nonce:   randomID(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", claims, err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := m.sign(encodedPayload)
	token := "v1." + encodedPayload + "." + signature

	if err := m.cache.SetString(ctx, sessionKey(claims.Nonce), strconv.FormatUint(uint64(uid), 10), m.ttl); err != nil {
		return "", claims, err
	}
	return token, claims, nil
}

func (m *Manager) Validate(ctx context.Context, token string) (Claims, error) {
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return Claims{}, ErrInvalidToken
	}
	if !hmac.Equal([]byte(parts[2]), []byte(m.sign(parts[1]))) {
		return Claims{}, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if claims.Exp < time.Now().Unix() {
		return Claims{}, ErrExpiredToken
	}

	if m.cache.Enabled() {
		if _, ok, err := m.cache.GetString(ctx, sessionKey(claims.Nonce)); err != nil || !ok {
			return Claims{}, ErrInvalidToken
		}
	}
	return claims, nil
}

func (m *Manager) Revoke(ctx context.Context, claims Claims) error {
	return m.cache.Delete(ctx, sessionKey(claims.Nonce))
}

func (m *Manager) sign(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randomID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return base64.RawURLEncoding.EncodeToString(buf[:])
}

func sessionKey(nonce string) string {
	return "session:" + nonce
}

package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"dw0rdwk/backend/internal/config"
)

func TestPushJSONReturnsDisabledError(t *testing.T) {
	client := Open(context.Background(), config.RedisConfig{Enabled: false})

	err := client.PushJSON(context.Background(), "queue:test", map[string]any{"id": 1})
	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("PushJSON() error = %v, want ErrDisabled", err)
	}
}

func TestListLengthDisabled(t *testing.T) {
	client := Open(context.Background(), config.RedisConfig{Enabled: false})

	length, err := client.ListLength(context.Background(), "queue:test")
	if err != nil {
		t.Fatalf("ListLength() error = %v", err)
	}
	if length != 0 {
		t.Fatalf("ListLength() = %d, want 0", length)
	}
}

func TestPopJSONFromDisabled(t *testing.T) {
	client := Open(context.Background(), config.RedisConfig{Enabled: false})

	var payload map[string]any
	ok, err := client.PopJSONFrom(context.Background(), []string{"queue:flash", "queue:normal"}, time.Millisecond, &payload)
	if err != nil {
		t.Fatalf("PopJSONFrom() error = %v", err)
	}
	if ok {
		t.Fatal("PopJSONFrom() ok = true, want false")
	}
}

func TestOpenRequiredAllowsDisabledRedis(t *testing.T) {
	client, err := OpenRequired(context.Background(), config.RedisConfig{Enabled: false})
	if err != nil {
		t.Fatalf("OpenRequired() error = %v", err)
	}
	if client.Enabled() {
		t.Fatal("OpenRequired() returned enabled client for disabled redis")
	}
}

func TestOpenRequiredReturnsPingError(t *testing.T) {
	_, err := OpenRequired(context.Background(), config.RedisConfig{
		Enabled:       true,
		Addr:          "127.0.0.1:0",
		DialTimeoutMS: 1,
		PoolSize:      1,
	})
	if err == nil {
		t.Fatal("OpenRequired() accepted unreachable redis")
	}
}

func TestRedisOptionsFromConfig(t *testing.T) {
	cfg := config.RedisConfig{
		Addr:           "redis:6379",
		Password:       "secret",
		DB:             2,
		PoolSize:       40,
		MinIdleConns:   10,
		DialTimeoutMS:  1500,
		ReadTimeoutMS:  1200,
		WriteTimeoutMS: 1300,
		PoolTimeoutMS:  1600,
	}

	options := redisOptions(cfg)

	if options.Addr != cfg.Addr || options.Password != cfg.Password || options.DB != cfg.DB {
		t.Fatalf("unexpected redis address/auth/db options: %+v", options)
	}
	if options.PoolSize != cfg.PoolSize || options.MinIdleConns != cfg.MinIdleConns {
		t.Fatalf("unexpected redis pool options: %+v", options)
	}
	if options.DialTimeout != 1500*time.Millisecond {
		t.Fatalf("DialTimeout = %s", options.DialTimeout)
	}
	if options.ReadTimeout != 1200*time.Millisecond {
		t.Fatalf("ReadTimeout = %s", options.ReadTimeout)
	}
	if options.WriteTimeout != 1300*time.Millisecond {
		t.Fatalf("WriteTimeout = %s", options.WriteTimeout)
	}
	if options.PoolTimeout != 1600*time.Millisecond {
		t.Fatalf("PoolTimeout = %s", options.PoolTimeout)
	}
}

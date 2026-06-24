package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"dw0rdwk/backend/internal/config"

	"github.com/redis/go-redis/v9"
)

var ErrDisabled = errors.New("redis disabled")

type Client struct {
	client *redis.Client
	prefix string
}

func Open(ctx context.Context, cfg config.RedisConfig) *Client {
	client, _ := open(ctx, cfg, false)
	return client
}

func OpenRequired(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	return open(ctx, cfg, true)
}

func open(ctx context.Context, cfg config.RedisConfig, required bool) (*Client, error) {
	if !cfg.Enabled {
		return &Client{prefix: cfg.Prefix}, nil
	}

	client := redis.NewClient(redisOptions(cfg))

	pingCtx, cancel := context.WithTimeout(ctx, durationMS(cfg.DialTimeoutMS))
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		if required {
			return nil, err
		}
		log.Printf("redis disabled: %v", err)
		return &Client{prefix: cfg.Prefix}, nil
	}

	return &Client{client: client, prefix: cfg.Prefix}, nil
}

func redisOptions(cfg config.RedisConfig) *redis.Options {
	return &redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  durationMS(cfg.DialTimeoutMS),
		ReadTimeout:  durationMS(cfg.ReadTimeoutMS),
		WriteTimeout: durationMS(cfg.WriteTimeoutMS),
		PoolTimeout:  durationMS(cfg.PoolTimeoutMS),
	}
}

func durationMS(value int) time.Duration {
	return time.Duration(value) * time.Millisecond
}

func (c *Client) Enabled() bool {
	return c != nil && c.client != nil
}

func (c *Client) Close() error {
	if !c.Enabled() {
		return nil
	}
	return c.client.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}
	return c.client.Ping(ctx).Err()
}

func (c *Client) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	if !c.Enabled() {
		return false, nil
	}
	data, err := c.client.Get(ctx, c.prefix+key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(data, dest)
}

func (c *Client) GetString(ctx context.Context, key string) (string, bool, error) {
	if !c.Enabled() {
		return "", false, nil
	}
	value, err := c.client.Get(ctx, c.prefix+key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (c *Client) SetString(ctx context.Context, key string, value string, ttl time.Duration) error {
	if !c.Enabled() {
		return nil
	}
	return c.client.Set(ctx, c.prefix+key, value, ttl).Err()
}

func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	if !c.Enabled() {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.prefix+key, data, ttl).Err()
}

func (c *Client) PushJSON(ctx context.Context, key string, value any) error {
	if !c.Enabled() {
		return ErrDisabled
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.LPush(ctx, c.prefix+key, data).Err()
}

func (c *Client) PopJSON(ctx context.Context, key string, timeout time.Duration, dest any) (bool, error) {
	return c.PopJSONFrom(ctx, []string{key}, timeout, dest)
}

func (c *Client) PopJSONFrom(ctx context.Context, keys []string, timeout time.Duration, dest any) (bool, error) {
	if !c.Enabled() {
		return false, nil
	}
	if len(keys) == 0 {
		return false, nil
	}
	prefixed := make([]string, 0, len(keys))
	for _, key := range keys {
		prefixed = append(prefixed, c.prefix+key)
	}
	items, err := c.client.BRPop(ctx, timeout, prefixed...).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if len(items) < 2 {
		return false, nil
	}
	return true, json.Unmarshal([]byte(items[1]), dest)
}

func (c *Client) ListLength(ctx context.Context, key string) (int64, error) {
	if !c.Enabled() {
		return 0, nil
	}
	return c.client.LLen(ctx, c.prefix+key).Result()
}

func (c *Client) Delete(ctx context.Context, keys ...string) error {
	if !c.Enabled() || len(keys) == 0 {
		return nil
	}
	prefixed := make([]string, 0, len(keys))
	for _, key := range keys {
		prefixed = append(prefixed, c.prefix+key)
	}
	return c.client.Del(ctx, prefixed...).Err()
}

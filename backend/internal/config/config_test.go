package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateRejectsInvalidTokenTTL(t *testing.T) {
	cfg := minimalConfig()
	cfg.Auth.TokenTTL = 0

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() accepted zero token ttl")
	}
}

func TestValidateRejectsInvalidHTTPRuntime(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "zero read timeout",
			mutate: func(cfg *Config) {
				cfg.HTTPReadTimeout = 0
			},
		},
		{
			name: "zero write timeout",
			mutate: func(cfg *Config) {
				cfg.HTTPWriteTimeout = 0
			},
		},
		{
			name: "zero idle timeout",
			mutate: func(cfg *Config) {
				cfg.HTTPIdleTimeout = 0
			},
		},
		{
			name: "zero shutdown timeout",
			mutate: func(cfg *Config) {
				cfg.HTTPShutdownTimeout = 0
			},
		},
		{
			name: "body limit below one megabyte",
			mutate: func(cfg *Config) {
				cfg.HTTPBodyLimit = 512 * 1024
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := minimalConfig()
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() accepted invalid http runtime config")
			}
		})
	}
}

func TestValidateRejectsInvalidMySQLPool(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "zero max open",
			mutate: func(cfg *Config) {
				cfg.MySQL.MaxOpenConns = 0
			},
		},
		{
			name: "idle exceeds open",
			mutate: func(cfg *Config) {
				cfg.MySQL.MaxOpenConns = 8
				cfg.MySQL.MaxIdleConns = 9
			},
		},
		{
			name: "negative lifetime",
			mutate: func(cfg *Config) {
				cfg.MySQL.ConnMaxLifetimeMinutes = -1
			},
		},
		{
			name: "negative idle time",
			mutate: func(cfg *Config) {
				cfg.MySQL.ConnMaxIdleMinutes = -1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := minimalConfig()
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() accepted invalid mysql pool config")
			}
		})
	}
}

func TestValidateRejectsInvalidRedisPool(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "negative db",
			mutate: func(cfg *Config) {
				cfg.Redis.DB = -1
			},
		},
		{
			name: "zero pool size",
			mutate: func(cfg *Config) {
				cfg.Redis.PoolSize = 0
			},
		},
		{
			name: "idle exceeds pool",
			mutate: func(cfg *Config) {
				cfg.Redis.PoolSize = 4
				cfg.Redis.MinIdleConns = 5
			},
		},
		{
			name: "zero dial timeout",
			mutate: func(cfg *Config) {
				cfg.Redis.DialTimeoutMS = 0
			},
		},
		{
			name: "zero read timeout",
			mutate: func(cfg *Config) {
				cfg.Redis.ReadTimeoutMS = 0
			},
		},
		{
			name: "zero write timeout",
			mutate: func(cfg *Config) {
				cfg.Redis.WriteTimeoutMS = 0
			},
		},
		{
			name: "zero pool timeout",
			mutate: func(cfg *Config) {
				cfg.Redis.PoolTimeoutMS = 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := minimalConfig()
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() accepted invalid redis pool config")
			}
		})
	}
}

func TestValidateRejectsInvalidWorkerConfig(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "zero concurrency",
			mutate: func(cfg *Config) {
				cfg.Worker.Concurrency = 0
			},
		},
		{
			name: "zero max attempts",
			mutate: func(cfg *Config) {
				cfg.Worker.MaxAttempts = 0
			},
		},
		{
			name: "negative retry delay",
			mutate: func(cfg *Config) {
				cfg.Worker.RetryDelayMS = -1
			},
		},
		{
			name: "zero recovery batch size",
			mutate: func(cfg *Config) {
				cfg.Worker.RecoveryBatchSize = 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := minimalConfig()
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() accepted invalid worker config")
			}
		})
	}
}

func TestValidateRejectsInvalidConnectorCronConfig(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "zero order interval",
			mutate: func(cfg *Config) {
				cfg.ConnectorCron.Enabled = true
				cfg.ConnectorCron.OrderInterval = 0
				cfg.ConnectorCron.PriceInterval = time.Minute
				cfg.ConnectorCron.MaxPages = 20
			},
		},
		{
			name: "zero price interval",
			mutate: func(cfg *Config) {
				cfg.ConnectorCron.Enabled = true
				cfg.ConnectorCron.OrderInterval = time.Minute
				cfg.ConnectorCron.PriceInterval = 0
				cfg.ConnectorCron.MaxPages = 20
			},
		},
		{
			name: "zero max pages",
			mutate: func(cfg *Config) {
				cfg.ConnectorCron.Enabled = true
				cfg.ConnectorCron.OrderInterval = time.Minute
				cfg.ConnectorCron.PriceInterval = time.Minute
				cfg.ConnectorCron.MaxPages = 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := minimalConfig()
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() accepted invalid connector cron config")
			}
		})
	}
}

func TestProductionWarnings(t *testing.T) {
	cfg := minimalConfig()
	cfg.AppEnv = "production"
	cfg.Auth.Secret = "change-me-before-public-deploy"
	cfg.Bootstrap.SeedDefaults = true
	cfg.Bootstrap.AdminPassword = "admin123"

	warnings := strings.Join(cfg.Warnings(), "\n")
	if !strings.Contains(warnings, "AUTH_SECRET") {
		t.Fatalf("missing AUTH_SECRET warning: %q", warnings)
	}
	if !strings.Contains(warnings, "BOOTSTRAP_ADMIN_PASSWORD") {
		t.Fatalf("missing bootstrap password warning: %q", warnings)
	}
	if !strings.Contains(warnings, "MYSQL_PASSWORD") {
		t.Fatalf("missing mysql password warning: %q", warnings)
	}
}

func TestValidateRejectsWeakProductionSecrets(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "weak auth secret",
			mutate: func(cfg *Config) {
				cfg.Auth.Secret = "change-me-before-public-deploy"
				cfg.Bootstrap.AdminPassword = "strong-admin-password"
				cfg.MySQL.Password = "strong-db-password"
			},
		},
		{
			name: "placeholder auth secret",
			mutate: func(cfg *Config) {
				cfg.Auth.Secret = "replace-with-at-least-32-random-characters"
				cfg.Bootstrap.AdminPassword = "strong-admin-password"
				cfg.MySQL.Password = "strong-db-password"
			},
		},
		{
			name: "default bootstrap password",
			mutate: func(cfg *Config) {
				cfg.Auth.Secret = "0123456789abcdef0123456789abcdef"
				cfg.Bootstrap.SeedDefaults = true
				cfg.Bootstrap.AdminPassword = "admin123"
				cfg.MySQL.Password = "strong-db-password"
			},
		},
		{
			name: "placeholder bootstrap password",
			mutate: func(cfg *Config) {
				cfg.Auth.Secret = "0123456789abcdef0123456789abcdef"
				cfg.Bootstrap.SeedDefaults = true
				cfg.Bootstrap.AdminPassword = "replace-initial-admin-password"
				cfg.MySQL.Password = "strong-db-password"
			},
		},
		{
			name: "empty mysql password",
			mutate: func(cfg *Config) {
				cfg.Auth.Secret = "0123456789abcdef0123456789abcdef"
				cfg.Bootstrap.AdminPassword = "strong-admin-password"
				cfg.MySQL.Password = ""
			},
		},
		{
			name: "default mysql password",
			mutate: func(cfg *Config) {
				cfg.Auth.Secret = "0123456789abcdef0123456789abcdef"
				cfg.Bootstrap.AdminPassword = "strong-admin-password"
				cfg.MySQL.Password = "dw0rdwk"
			},
		},
		{
			name: "placeholder mysql password",
			mutate: func(cfg *Config) {
				cfg.Auth.Secret = "0123456789abcdef0123456789abcdef"
				cfg.Bootstrap.AdminPassword = "strong-admin-password"
				cfg.MySQL.Password = "replace-app-db-password"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := minimalConfig()
			cfg.AppEnv = "production"
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() accepted unsafe production config")
			}
		})
	}
}

func TestValidateAcceptsStrongProductionConfig(t *testing.T) {
	cfg := minimalConfig()
	cfg.AppEnv = "production"
	cfg.Auth.Secret = "0123456789abcdef0123456789abcdef"
	cfg.Bootstrap.SeedDefaults = true
	cfg.Bootstrap.AdminPassword = "strong-admin-password"
	cfg.MySQL.Password = "strong-db-password"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() rejected strong production config: %v", err)
	}
}

func minimalConfig() Config {
	return Config{
		AppEnv:              "development",
		HTTPAddr:            ":8080",
		HTTPReadTimeout:     15 * time.Second,
		HTTPWriteTimeout:    30 * time.Second,
		HTTPIdleTimeout:     60 * time.Second,
		HTTPShutdownTimeout: 10 * time.Second,
		HTTPBodyLimit:       4 * 1024 * 1024,
		Auth: AuthConfig{
			Secret:   "local-development-secret",
			TokenTTL: time.Hour,
		},
		MySQL: MySQLConfig{
			Host:                   "127.0.0.1",
			Port:                   3306,
			User:                   "dw0rdwk",
			Database:               "dw0rdwk",
			MaxOpenConns:           64,
			MaxIdleConns:           16,
			ConnMaxLifetimeMinutes: 30,
			ConnMaxIdleMinutes:     5,
		},
		Redis: RedisConfig{
			Enabled:        true,
			Addr:           "127.0.0.1:6379",
			PoolSize:       32,
			MinIdleConns:   8,
			DialTimeoutMS:  2000,
			ReadTimeoutMS:  1000,
			WriteTimeoutMS: 1000,
			PoolTimeoutMS:  2000,
		},
		Worker: WorkerConfig{
			Enabled:           true,
			Concurrency:       2,
			MaxAttempts:       3,
			RetryDelayMS:      1000,
			RecoverOnStart:    true,
			RecoveryBatchSize: 500,
		},
	}
}

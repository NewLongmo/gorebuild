package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

type Config struct {
	AppName             string
	AppEnv              string
	HTTPAddr            string
	HTTPReadTimeout     time.Duration
	HTTPWriteTimeout    time.Duration
	HTTPIdleTimeout     time.Duration
	HTTPShutdownTimeout time.Duration
	HTTPBodyLimit       int
	HTTPPrefork         bool
	CORSAllowOrigins    []string
	Auth                AuthConfig
	Bootstrap           BootstrapConfig
	MySQL               MySQLConfig
	Redis               RedisConfig
	Worker              WorkerConfig
	ConnectorCron       ConnectorCronConfig
}

type BootstrapConfig struct {
	SeedDefaults  bool
	AdminAccount  string
	AdminPassword string
	AdminName     string
}

type AuthConfig struct {
	Secret   string
	TokenTTL time.Duration
}

type MySQLConfig struct {
	Host                   string
	Port                   int
	User                   string
	Password               string
	Database               string
	Charset                string
	ParseTime              bool
	Location               string
	AutoMigrate            bool
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeMinutes int
	ConnMaxIdleMinutes     int
}

type RedisConfig struct {
	Enabled        bool
	Addr           string
	Password       string
	DB             int
	Prefix         string
	PoolSize       int
	MinIdleConns   int
	DialTimeoutMS  int
	ReadTimeoutMS  int
	WriteTimeoutMS int
	PoolTimeoutMS  int
}

type WorkerConfig struct {
	Enabled           bool
	Concurrency       int
	MaxAttempts       int
	RetryDelayMS      int
	RecoverOnStart    bool
	RecoveryBatchSize int
}

type ConnectorCronConfig struct {
	Enabled       bool
	OrderInterval time.Duration
	PriceInterval time.Duration
	MaxPages      int
}

func Load() Config {
	return Config{
		AppName:             env("APP_NAME", "dw0rdwk-api"),
		AppEnv:              env("APP_ENV", "development"),
		HTTPAddr:            env("HTTP_ADDR", ":8080"),
		HTTPReadTimeout:     time.Duration(envInt("HTTP_READ_TIMEOUT_SECONDS", 15)) * time.Second,
		HTTPWriteTimeout:    time.Duration(envInt("HTTP_WRITE_TIMEOUT_SECONDS", 30)) * time.Second,
		HTTPIdleTimeout:     time.Duration(envInt("HTTP_IDLE_TIMEOUT_SECONDS", 60)) * time.Second,
		HTTPShutdownTimeout: time.Duration(envInt("HTTP_SHUTDOWN_TIMEOUT_SECONDS", 10)) * time.Second,
		HTTPBodyLimit:       envInt("HTTP_BODY_LIMIT_MB", 4) * 1024 * 1024,
		HTTPPrefork:         envBool("HTTP_PREFORK", false),
		CORSAllowOrigins:    splitCSV(env("CORS_ALLOW_ORIGINS", "http://localhost:5173")),
		Auth: AuthConfig{
			Secret:   env("AUTH_SECRET", "change-me-in-production"),
			TokenTTL: time.Duration(envInt("AUTH_TOKEN_TTL_HOURS", 24)) * time.Hour,
		},
		Bootstrap: BootstrapConfig{
			SeedDefaults:  envBool("DB_SEED_DEFAULTS", true),
			AdminAccount:  env("BOOTSTRAP_ADMIN_ACCOUNT", "admin"),
			AdminPassword: env("BOOTSTRAP_ADMIN_PASSWORD", "admin123"),
			AdminName:     env("BOOTSTRAP_ADMIN_NAME", "Administrator"),
		},
		MySQL: MySQLConfig{
			Host:                   env("MYSQL_HOST", "127.0.0.1"),
			Port:                   envInt("MYSQL_PORT", 3306),
			User:                   env("MYSQL_USER", "root"),
			Password:               env("MYSQL_PASSWORD", ""),
			Database:               env("MYSQL_DATABASE", "dw0rdwk"),
			Charset:                env("MYSQL_CHARSET", "utf8mb4"),
			ParseTime:              envBool("MYSQL_PARSE_TIME", true),
			Location:               env("MYSQL_LOC", "Local"),
			AutoMigrate:            envBool("DB_AUTO_MIGRATE", false),
			MaxOpenConns:           envInt("MYSQL_MAX_OPEN_CONNS", 64),
			MaxIdleConns:           envInt("MYSQL_MAX_IDLE_CONNS", 16),
			ConnMaxLifetimeMinutes: envInt("MYSQL_CONN_MAX_LIFETIME_MINUTES", 30),
			ConnMaxIdleMinutes:     envInt("MYSQL_CONN_MAX_IDLE_MINUTES", 5),
		},
		Redis: RedisConfig{
			Enabled:        envBool("REDIS_ENABLED", true),
			Addr:           env("REDIS_ADDR", "127.0.0.1:6379"),
			Password:       env("REDIS_PASSWORD", ""),
			DB:             envInt("REDIS_DB", 0),
			Prefix:         env("REDIS_PREFIX", "dw0rdwk:"),
			PoolSize:       envInt("REDIS_POOL_SIZE", 32),
			MinIdleConns:   envInt("REDIS_MIN_IDLE_CONNS", 8),
			DialTimeoutMS:  envInt("REDIS_DIAL_TIMEOUT_MS", 2000),
			ReadTimeoutMS:  envInt("REDIS_READ_TIMEOUT_MS", 1000),
			WriteTimeoutMS: envInt("REDIS_WRITE_TIMEOUT_MS", 1000),
			PoolTimeoutMS:  envInt("REDIS_POOL_TIMEOUT_MS", 2000),
		},
		Worker: WorkerConfig{
			Enabled:           envBool("ORDER_WORKER_ENABLED", true),
			Concurrency:       envInt("ORDER_WORKER_CONCURRENCY", 2),
			MaxAttempts:       envInt("ORDER_WORKER_MAX_ATTEMPTS", 3),
			RetryDelayMS:      envInt("ORDER_WORKER_RETRY_DELAY_MS", 1000),
			RecoverOnStart:    envBool("ORDER_QUEUE_RECOVER_ON_START", true),
			RecoveryBatchSize: envInt("ORDER_QUEUE_RECOVERY_BATCH_SIZE", 500),
		},
		ConnectorCron: ConnectorCronConfig{
			Enabled:       envBool("CONNECTOR_CRON_ENABLED", true),
			OrderInterval: time.Duration(envInt("CONNECTOR_CRON_ORDER_INTERVAL_SECONDS", 300)) * time.Second,
			PriceInterval: time.Duration(envInt("CONNECTOR_CRON_PRICE_INTERVAL_SECONDS", 300)) * time.Second,
			MaxPages:      envInt("CONNECTOR_CRON_MAX_PAGES", 20),
		},
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return errors.New("HTTP_ADDR is required")
	}
	if c.HTTPReadTimeout <= 0 {
		return errors.New("HTTP_READ_TIMEOUT_SECONDS must be greater than zero")
	}
	if c.HTTPWriteTimeout <= 0 {
		return errors.New("HTTP_WRITE_TIMEOUT_SECONDS must be greater than zero")
	}
	if c.HTTPIdleTimeout <= 0 {
		return errors.New("HTTP_IDLE_TIMEOUT_SECONDS must be greater than zero")
	}
	if c.HTTPShutdownTimeout <= 0 {
		return errors.New("HTTP_SHUTDOWN_TIMEOUT_SECONDS must be greater than zero")
	}
	if c.HTTPBodyLimit < 1024*1024 {
		return errors.New("HTTP_BODY_LIMIT_MB must be at least 1")
	}
	if strings.TrimSpace(c.Auth.Secret) == "" {
		return errors.New("AUTH_SECRET is required")
	}
	if isProduction(c.AppEnv) && weakAuthSecret(c.Auth.Secret) {
		return errors.New("AUTH_SECRET must be at least 32 characters and not contain change-me in production")
	}
	if c.Auth.TokenTTL <= 0 {
		return errors.New("AUTH_TOKEN_TTL_HOURS must be greater than zero")
	}
	if isProduction(c.AppEnv) && c.Bootstrap.SeedDefaults && weakBootstrapPassword(c.Bootstrap.AdminPassword) {
		return errors.New("BOOTSTRAP_ADMIN_PASSWORD must be changed to a non-placeholder value in production")
	}
	if c.MySQL.Port <= 0 || c.MySQL.Port > 65535 {
		return errors.New("MYSQL_PORT must be between 1 and 65535")
	}
	if strings.TrimSpace(c.MySQL.Host) == "" {
		return errors.New("MYSQL_HOST is required")
	}
	if strings.TrimSpace(c.MySQL.Database) == "" {
		return errors.New("MYSQL_DATABASE is required")
	}
	if isProduction(c.AppEnv) && weakMySQLPassword(c.MySQL.Password) {
		return errors.New("MYSQL_PASSWORD must be changed in production")
	}
	if c.MySQL.MaxOpenConns < 1 {
		return errors.New("MYSQL_MAX_OPEN_CONNS must be greater than zero")
	}
	if c.MySQL.MaxIdleConns < 0 {
		return errors.New("MYSQL_MAX_IDLE_CONNS must be zero or greater")
	}
	if c.MySQL.MaxIdleConns > c.MySQL.MaxOpenConns {
		return errors.New("MYSQL_MAX_IDLE_CONNS must be less than or equal to MYSQL_MAX_OPEN_CONNS")
	}
	if c.MySQL.ConnMaxLifetimeMinutes < 0 {
		return errors.New("MYSQL_CONN_MAX_LIFETIME_MINUTES must be zero or greater")
	}
	if c.MySQL.ConnMaxIdleMinutes < 0 {
		return errors.New("MYSQL_CONN_MAX_IDLE_MINUTES must be zero or greater")
	}
	if c.Redis.Enabled && strings.TrimSpace(c.Redis.Addr) == "" {
		return errors.New("REDIS_ADDR is required when REDIS_ENABLED=true")
	}
	if c.Redis.DB < 0 {
		return errors.New("REDIS_DB must be zero or greater")
	}
	if c.Redis.PoolSize < 1 {
		return errors.New("REDIS_POOL_SIZE must be greater than zero")
	}
	if c.Redis.MinIdleConns < 0 {
		return errors.New("REDIS_MIN_IDLE_CONNS must be zero or greater")
	}
	if c.Redis.MinIdleConns > c.Redis.PoolSize {
		return errors.New("REDIS_MIN_IDLE_CONNS must be less than or equal to REDIS_POOL_SIZE")
	}
	if c.Redis.DialTimeoutMS < 1 {
		return errors.New("REDIS_DIAL_TIMEOUT_MS must be greater than zero")
	}
	if c.Redis.ReadTimeoutMS < 1 {
		return errors.New("REDIS_READ_TIMEOUT_MS must be greater than zero")
	}
	if c.Redis.WriteTimeoutMS < 1 {
		return errors.New("REDIS_WRITE_TIMEOUT_MS must be greater than zero")
	}
	if c.Redis.PoolTimeoutMS < 1 {
		return errors.New("REDIS_POOL_TIMEOUT_MS must be greater than zero")
	}
	if c.Worker.Concurrency < 1 {
		return errors.New("ORDER_WORKER_CONCURRENCY must be greater than zero")
	}
	if c.Worker.MaxAttempts < 1 {
		return errors.New("ORDER_WORKER_MAX_ATTEMPTS must be greater than zero")
	}
	if c.Worker.RetryDelayMS < 0 {
		return errors.New("ORDER_WORKER_RETRY_DELAY_MS must be zero or greater")
	}
	if c.Worker.RecoveryBatchSize < 1 {
		return errors.New("ORDER_QUEUE_RECOVERY_BATCH_SIZE must be greater than zero")
	}
	if c.ConnectorCron.Enabled {
		if c.ConnectorCron.OrderInterval <= 0 {
			return errors.New("CONNECTOR_CRON_ORDER_INTERVAL_SECONDS must be greater than zero")
		}
		if c.ConnectorCron.PriceInterval <= 0 {
			return errors.New("CONNECTOR_CRON_PRICE_INTERVAL_SECONDS must be greater than zero")
		}
		if c.ConnectorCron.MaxPages < 1 {
			return errors.New("CONNECTOR_CRON_MAX_PAGES must be greater than zero")
		}
	}
	return nil
}

func (c Config) Warnings() []string {
	warnings := []string{}
	if isProduction(c.AppEnv) {
		if weakAuthSecret(c.Auth.Secret) {
			warnings = append(warnings, "AUTH_SECRET is weak for production")
		}
		if c.Bootstrap.SeedDefaults && weakBootstrapPassword(c.Bootstrap.AdminPassword) {
			warnings = append(warnings, "BOOTSTRAP_ADMIN_PASSWORD uses the default value in production")
		}
		if weakMySQLPassword(c.MySQL.Password) {
			warnings = append(warnings, "MYSQL_PASSWORD is weak for production")
		}
	}
	return warnings
}

func isProduction(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "production")
}

func weakAuthSecret(value string) bool {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	return len(value) < 32 || strings.Contains(lower, "change-me") || strings.HasPrefix(lower, "replace-")
}

func weakBootstrapPassword(value string) bool {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	return value == "" || len(value) < 12 || lower == "admin123" || strings.Contains(lower, "change-me") || strings.HasPrefix(lower, "replace-")
}

func weakMySQLPassword(value string) bool {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	return value == "" || lower == "dw0rdwk" || strings.HasPrefix(lower, "replace-")
}

func (c MySQLConfig) DSN() string {
	driverCfg := mysqlDriver.Config{
		User:                 c.User,
		Passwd:               c.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", c.Host, c.Port),
		DBName:               c.Database,
		AllowNativePasswords: true,
		Params: map[string]string{
			"charset":   c.Charset,
			"parseTime": strconv.FormatBool(c.ParseTime),
			"loc":       c.Location,
		},
	}
	return driverCfg.FormatDSN()
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

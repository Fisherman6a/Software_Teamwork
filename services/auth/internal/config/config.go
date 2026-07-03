package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultHTTPAddr                  = ":8001"
	DefaultServiceVersion            = "0.1.0"
	DefaultEnvironment               = "local"
	DefaultShutdownTimeout           = 10 * time.Second
	DefaultReadinessTimeout          = 2 * time.Second
	DefaultRequestTimeout            = 30 * time.Second
	DefaultReadHeaderTimeout         = 5 * time.Second
	DefaultSessionTTL                = 24 * time.Hour
	DefaultTokenKeyVersion           = "v1"
	DefaultRoleCode                  = "standard"
	DefaultCredentialWorkMaxInFlight = 4
	DefaultLoginFailureLimit         = 5
	DefaultLoginFailureWindow        = 15 * time.Minute
	DefaultLoginLockDuration         = 15 * time.Minute
)

type Config struct {
	HTTPAddr                  string
	ServiceVersion            string
	Environment               string
	DatabaseURL               string
	ServiceToken              string
	GatewayAdminToken         string
	TokenHashSecret           string
	TokenKeyVersion           string
	SessionTTL                time.Duration
	DefaultRoleCode           string
	ShutdownTimeout           time.Duration
	ReadinessTimeout          time.Duration
	RequestTimeout            time.Duration
	ReadHeaderTimeout         time.Duration
	CredentialWorkMaxInFlight int
	LoginFailureLimit         int
	LoginFailureWindow        time.Duration
	LoginLockDuration         time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:                  stringValue("AUTH_HTTP_ADDR", DefaultHTTPAddr),
		ServiceVersion:            stringValue("AUTH_SERVICE_VERSION", DefaultServiceVersion),
		Environment:               stringValue("AUTH_ENV", DefaultEnvironment),
		DatabaseURL:               strings.TrimSpace(os.Getenv("AUTH_DATABASE_URL")),
		ServiceToken:              firstNonEmptyEnv("AUTH_INTERNAL_SERVICE_TOKEN", "INTERNAL_SERVICE_TOKEN"),
		GatewayAdminToken:         strings.TrimSpace(os.Getenv("AUTH_GATEWAY_ADMIN_SERVICE_TOKEN")),
		TokenHashSecret:           firstNonEmptyEnv("AUTH_TOKEN_HASH_SECRET", "TOKEN_HASH_SECRET"),
		TokenKeyVersion:           stringValue("AUTH_TOKEN_HASH_KEY_VERSION", DefaultTokenKeyVersion),
		DefaultRoleCode:           stringValue("AUTH_DEFAULT_ROLE_CODE", DefaultRoleCode),
		SessionTTL:                DefaultSessionTTL,
		ShutdownTimeout:           DefaultShutdownTimeout,
		ReadinessTimeout:          DefaultReadinessTimeout,
		RequestTimeout:            DefaultRequestTimeout,
		ReadHeaderTimeout:         DefaultReadHeaderTimeout,
		CredentialWorkMaxInFlight: DefaultCredentialWorkMaxInFlight,
		LoginFailureLimit:         DefaultLoginFailureLimit,
		LoginFailureWindow:        DefaultLoginFailureWindow,
		LoginLockDuration:         DefaultLoginLockDuration,
	}

	if raw := os.Getenv("AUTH_SHUTDOWN_TIMEOUT"); raw != "" {
		value, err := parseDurationOrSeconds(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("AUTH_SHUTDOWN_TIMEOUT must be a positive duration")
		}
		cfg.ShutdownTimeout = value
	}

	if raw := os.Getenv("AUTH_READINESS_TIMEOUT"); raw != "" {
		value, err := parseDurationOrSeconds(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("AUTH_READINESS_TIMEOUT must be a positive duration")
		}
		cfg.ReadinessTimeout = value
	}

	if raw := os.Getenv("AUTH_REQUEST_TIMEOUT"); raw != "" {
		value, err := parseDurationOrSeconds(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("AUTH_REQUEST_TIMEOUT must be a positive duration")
		}
		cfg.RequestTimeout = value
	}

	if raw := os.Getenv("AUTH_READ_HEADER_TIMEOUT"); raw != "" {
		value, err := parseDurationOrSeconds(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("AUTH_READ_HEADER_TIMEOUT must be a positive duration")
		}
		cfg.ReadHeaderTimeout = value
	}

	if raw := os.Getenv("AUTH_SESSION_TTL"); raw != "" {
		value, err := parseDurationOrSeconds(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("AUTH_SESSION_TTL must be a positive duration")
		}
		cfg.SessionTTL = value
	}

	if raw := os.Getenv("AUTH_CREDENTIAL_WORK_MAX_IN_FLIGHT"); raw != "" {
		value, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || value < 0 {
			return Config{}, fmt.Errorf("AUTH_CREDENTIAL_WORK_MAX_IN_FLIGHT must be a non-negative integer")
		}
		cfg.CredentialWorkMaxInFlight = value
	}

	if raw := os.Getenv("AUTH_LOGIN_FAILURE_LIMIT"); raw != "" {
		value, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || value < 0 {
			return Config{}, fmt.Errorf("AUTH_LOGIN_FAILURE_LIMIT must be a non-negative integer")
		}
		cfg.LoginFailureLimit = value
	}

	if raw := os.Getenv("AUTH_LOGIN_FAILURE_WINDOW"); raw != "" {
		value, err := parseDurationOrSeconds(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("AUTH_LOGIN_FAILURE_WINDOW must be a positive duration")
		}
		cfg.LoginFailureWindow = value
	}

	if raw := os.Getenv("AUTH_LOGIN_LOCK_DURATION"); raw != "" {
		value, err := parseDurationOrSeconds(raw)
		if err != nil || value < 0 {
			return Config{}, fmt.Errorf("AUTH_LOGIN_LOCK_DURATION must be a non-negative duration")
		}
		cfg.LoginLockDuration = value
	}

	if strings.TrimSpace(cfg.HTTPAddr) == "" {
		return Config{}, fmt.Errorf("AUTH_HTTP_ADDR must not be empty")
	}
	if strings.TrimSpace(cfg.ServiceVersion) == "" {
		return Config{}, fmt.Errorf("AUTH_SERVICE_VERSION must not be empty")
	}
	if strings.TrimSpace(cfg.Environment) == "" {
		return Config{}, fmt.Errorf("AUTH_ENV must not be empty")
	}
	if strings.TrimSpace(cfg.TokenKeyVersion) == "" {
		return Config{}, fmt.Errorf("AUTH_TOKEN_HASH_KEY_VERSION must not be empty")
	}
	if strings.TrimSpace(cfg.DefaultRoleCode) == "" {
		return Config{}, fmt.Errorf("AUTH_DEFAULT_ROLE_CODE must not be empty")
	}
	if cfg.DatabaseURL != "" && strings.TrimSpace(cfg.TokenHashSecret) == "" {
		return Config{}, fmt.Errorf("AUTH_TOKEN_HASH_SECRET is required when AUTH_DATABASE_URL is set")
	}
	if cfg.DatabaseURL != "" && strings.TrimSpace(cfg.ServiceToken) == "" {
		return Config{}, fmt.Errorf("AUTH_INTERNAL_SERVICE_TOKEN is required when AUTH_DATABASE_URL is set")
	}
	if cfg.DatabaseURL != "" && strings.TrimSpace(cfg.GatewayAdminToken) == "" {
		return Config{}, fmt.Errorf("AUTH_GATEWAY_ADMIN_SERVICE_TOKEN is required when AUTH_DATABASE_URL is set")
	}
	if cfg.DatabaseURL != "" && strings.TrimSpace(cfg.GatewayAdminToken) == strings.TrimSpace(cfg.ServiceToken) {
		return Config{}, fmt.Errorf("AUTH_GATEWAY_ADMIN_SERVICE_TOKEN must differ from AUTH_INTERNAL_SERVICE_TOKEN")
	}

	return cfg, nil
}

func stringValue(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func parseDurationOrSeconds(raw string) (time.Duration, error) {
	value, err := time.ParseDuration(raw)
	if err == nil {
		return value, nil
	}
	seconds, parseErr := strconv.ParseInt(raw, 10, 64)
	if parseErr != nil {
		return 0, err
	}
	return time.Duration(seconds) * time.Second, nil
}

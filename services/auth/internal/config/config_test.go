package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("AUTH_HTTP_ADDR", "")
	t.Setenv("AUTH_SERVICE_VERSION", "")
	t.Setenv("AUTH_ENV", "")
	t.Setenv("AUTH_DATABASE_URL", "")
	t.Setenv("AUTH_INTERNAL_SERVICE_TOKEN", "")
	t.Setenv("AUTH_GATEWAY_ADMIN_SERVICE_TOKEN", "")
	t.Setenv("AUTH_TOKEN_HASH_SECRET", "")
	t.Setenv("AUTH_TOKEN_HASH_KEY_VERSION", "")
	t.Setenv("AUTH_SESSION_TTL", "")
	t.Setenv("AUTH_DEFAULT_ROLE_CODE", "")
	t.Setenv("AUTH_SHUTDOWN_TIMEOUT", "")
	t.Setenv("AUTH_READINESS_TIMEOUT", "")
	t.Setenv("AUTH_REQUEST_TIMEOUT", "")
	t.Setenv("AUTH_READ_HEADER_TIMEOUT", "")
	t.Setenv("AUTH_CREDENTIAL_WORK_MAX_IN_FLIGHT", "")
	t.Setenv("AUTH_LOGIN_FAILURE_LIMIT", "")
	t.Setenv("AUTH_LOGIN_FAILURE_WINDOW", "")
	t.Setenv("AUTH_LOGIN_LOCK_DURATION", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.ServiceVersion != DefaultServiceVersion {
		t.Fatalf("ServiceVersion = %q", cfg.ServiceVersion)
	}
	if cfg.Environment != DefaultEnvironment {
		t.Fatalf("Environment = %q", cfg.Environment)
	}
	if cfg.ShutdownTimeout != DefaultShutdownTimeout {
		t.Fatalf("ShutdownTimeout = %s", cfg.ShutdownTimeout)
	}
	if cfg.ReadinessTimeout != DefaultReadinessTimeout {
		t.Fatalf("ReadinessTimeout = %s", cfg.ReadinessTimeout)
	}
	if cfg.RequestTimeout != DefaultRequestTimeout || cfg.ReadHeaderTimeout != DefaultReadHeaderTimeout {
		t.Fatalf("http timeouts = %+v", cfg)
	}
	if cfg.CredentialWorkMaxInFlight != DefaultCredentialWorkMaxInFlight {
		t.Fatalf("CredentialWorkMaxInFlight = %d", cfg.CredentialWorkMaxInFlight)
	}
	if cfg.LoginFailureLimit != DefaultLoginFailureLimit ||
		cfg.LoginFailureWindow != DefaultLoginFailureWindow ||
		cfg.LoginLockDuration != DefaultLoginLockDuration {
		t.Fatalf("login failure config = %+v", cfg)
	}
	if cfg.SessionTTL != DefaultSessionTTL {
		t.Fatalf("SessionTTL = %s", cfg.SessionTTL)
	}
	if cfg.TokenKeyVersion != DefaultTokenKeyVersion || cfg.DefaultRoleCode != DefaultRoleCode {
		t.Fatalf("token/role defaults = %+v", cfg)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("AUTH_HTTP_ADDR", ":9100")
	t.Setenv("AUTH_SERVICE_VERSION", "0.2.0")
	t.Setenv("AUTH_ENV", "test")
	t.Setenv("AUTH_DATABASE_URL", "postgres://auth:auth@localhost:5432/auth?sslmode=disable")
	t.Setenv("AUTH_INTERNAL_SERVICE_TOKEN", "test-service-token")
	t.Setenv("AUTH_GATEWAY_ADMIN_SERVICE_TOKEN", "gateway-admin-token")
	t.Setenv("AUTH_TOKEN_HASH_SECRET", "test-token-hash-secret")
	t.Setenv("AUTH_TOKEN_HASH_KEY_VERSION", "v9")
	t.Setenv("AUTH_SESSION_TTL", "2h")
	t.Setenv("AUTH_DEFAULT_ROLE_CODE", "member")
	t.Setenv("AUTH_SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("AUTH_READINESS_TIMEOUT", "3")
	t.Setenv("AUTH_REQUEST_TIMEOUT", "7s")
	t.Setenv("AUTH_READ_HEADER_TIMEOUT", "4")
	t.Setenv("AUTH_CREDENTIAL_WORK_MAX_IN_FLIGHT", "2")
	t.Setenv("AUTH_LOGIN_FAILURE_LIMIT", "3")
	t.Setenv("AUTH_LOGIN_FAILURE_WINDOW", "10m")
	t.Setenv("AUTH_LOGIN_LOCK_DURATION", "20m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":9100" || cfg.ServiceVersion != "0.2.0" || cfg.Environment != "test" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.DatabaseURL == "" {
		t.Fatalf("DatabaseURL is empty")
	}
	if cfg.ShutdownTimeout != 5*time.Second {
		t.Fatalf("ShutdownTimeout = %s", cfg.ShutdownTimeout)
	}
	if cfg.ReadinessTimeout != 3*time.Second {
		t.Fatalf("ReadinessTimeout = %s", cfg.ReadinessTimeout)
	}
	if cfg.RequestTimeout != 7*time.Second || cfg.ReadHeaderTimeout != 4*time.Second {
		t.Fatalf("http timeouts = %+v", cfg)
	}
	if cfg.CredentialWorkMaxInFlight != 2 ||
		cfg.LoginFailureLimit != 3 ||
		cfg.LoginFailureWindow != 10*time.Minute ||
		cfg.LoginLockDuration != 20*time.Minute {
		t.Fatalf("concurrency/login config = %+v", cfg)
	}
	if cfg.ServiceToken != "test-service-token" || cfg.GatewayAdminToken != "gateway-admin-token" || cfg.TokenHashSecret != "test-token-hash-secret" || cfg.TokenKeyVersion != "v9" || cfg.DefaultRoleCode != "member" {
		t.Fatalf("auth config = %+v", cfg)
	}
	if cfg.SessionTTL != 2*time.Hour {
		t.Fatalf("SessionTTL = %s", cfg.SessionTTL)
	}
}

func TestLoadUsesSharedTokenEnvironment(t *testing.T) {
	t.Setenv("AUTH_DATABASE_URL", "postgres://auth:auth@localhost:5432/auth?sslmode=disable")
	t.Setenv("INTERNAL_SERVICE_TOKEN", "shared-service-token")
	t.Setenv("AUTH_GATEWAY_ADMIN_SERVICE_TOKEN", "gateway-admin-token")
	t.Setenv("TOKEN_HASH_SECRET", "shared-token-hash-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ServiceToken != "shared-service-token" || cfg.GatewayAdminToken != "gateway-admin-token" || cfg.TokenHashSecret != "shared-token-hash-secret" {
		t.Fatalf("shared token config = %+v", cfg)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{name: "shutdown timeout", key: "AUTH_SHUTDOWN_TIMEOUT", val: "nope"},
		{name: "request timeout", key: "AUTH_REQUEST_TIMEOUT", val: "0s"},
		{name: "read header timeout", key: "AUTH_READ_HEADER_TIMEOUT", val: "-1s"},
		{name: "credential work max", key: "AUTH_CREDENTIAL_WORK_MAX_IN_FLIGHT", val: "-1"},
		{name: "login failure limit", key: "AUTH_LOGIN_FAILURE_LIMIT", val: "-1"},
		{name: "login failure window", key: "AUTH_LOGIN_FAILURE_WINDOW", val: "0s"},
		{name: "login lock duration", key: "AUTH_LOGIN_LOCK_DURATION", val: "bad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.val)
			if _, err := Load(); err == nil {
				t.Fatalf("Load() error = nil")
			}
		})
	}
}

func TestLoadRequiresTokenSecretWhenDatabaseConfigured(t *testing.T) {
	t.Setenv("AUTH_DATABASE_URL", "postgres://auth:auth@localhost:5432/auth?sslmode=disable")
	t.Setenv("AUTH_INTERNAL_SERVICE_TOKEN", "test-service-token")
	t.Setenv("AUTH_GATEWAY_ADMIN_SERVICE_TOKEN", "gateway-admin-token")
	t.Setenv("AUTH_TOKEN_HASH_SECRET", "")

	if _, err := Load(); err == nil {
		t.Fatalf("Load() error = nil")
	}
}

func TestLoadRequiresServiceTokenWhenDatabaseConfigured(t *testing.T) {
	t.Setenv("AUTH_DATABASE_URL", "postgres://auth:auth@localhost:5432/auth?sslmode=disable")
	t.Setenv("AUTH_TOKEN_HASH_SECRET", "test-token-hash-secret")
	t.Setenv("AUTH_GATEWAY_ADMIN_SERVICE_TOKEN", "gateway-admin-token")
	t.Setenv("AUTH_INTERNAL_SERVICE_TOKEN", "")

	if _, err := Load(); err == nil {
		t.Fatalf("Load() error = nil")
	}
}

func TestLoadRequiresGatewayAdminTokenWhenDatabaseConfigured(t *testing.T) {
	t.Setenv("AUTH_DATABASE_URL", "postgres://auth:auth@localhost:5432/auth?sslmode=disable")
	t.Setenv("AUTH_TOKEN_HASH_SECRET", "test-token-hash-secret")
	t.Setenv("AUTH_INTERNAL_SERVICE_TOKEN", "test-service-token")
	t.Setenv("AUTH_GATEWAY_ADMIN_SERVICE_TOKEN", "")

	if _, err := Load(); err == nil {
		t.Fatalf("Load() error = nil")
	}
}

func TestLoadRejectsSharedGatewayAdminToken(t *testing.T) {
	t.Setenv("AUTH_DATABASE_URL", "postgres://auth:auth@localhost:5432/auth?sslmode=disable")
	t.Setenv("AUTH_TOKEN_HASH_SECRET", "test-token-hash-secret")
	t.Setenv("AUTH_INTERNAL_SERVICE_TOKEN", "same-token")
	t.Setenv("AUTH_GATEWAY_ADMIN_SERVICE_TOKEN", "same-token")

	if _, err := Load(); err == nil {
		t.Fatalf("Load() error = nil")
	}
}

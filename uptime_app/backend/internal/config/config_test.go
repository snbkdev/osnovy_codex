package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func validEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32))))
	t.Setenv("MAIL_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 32))))
	for key, value := range map[string]string{"APP_ENV": "development", "PUBLIC_URL": "http://localhost:3000", "COOKIE_SECURE": "false", "SMTP_ALLOW_INSECURE": "true", "TRUSTED_PROXY_CIDR": "", "DB_MAX_OPEN": "10", "DB_MAX_IDLE": "5", "DB_CONN_LIFETIME": "30m"} {
		t.Setenv(key, value)
	}
}
func TestConfig(t *testing.T) {
	validEnv(t)
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ key, value string }{{"DATABASE_URL", ""}, {"DATABASE_URL", "not-a-dsn"}, {"APP_ENV", "prod"}, {"SMTP_ADDR", "bad"}, {"JWT_SECRET", "bad"}, {"MAIL_ENCRYPTION_KEY", "bad"}, {"DB_MAX_OPEN", "0"}, {"DB_MAX_IDLE", "11"}, {"DB_CONN_LIFETIME", "bad"}, {"PUBLIC_URL", "http://localhost:3000/path"}, {"PUBLIC_URL", "javascript:alert(1)"}, {"COOKIE_SECURE", "bad"}, {"TRUSTED_PROXY_CIDR", "bad"}, {"TRUSTED_PROXY_CIDR", "127.0.0.1/32"}, {"APP_ENV", "production"}} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatalf("accepted %s", tc.key)
			}
		})
	}
	t.Setenv("APP_ENV", "production")
	t.Setenv("PUBLIC_URL", "https://example.com")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("SMTP_ALLOW_INSECURE", "false")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}

package config

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/mail"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr, DatabaseURL                      string
	MaxOpen, MaxIdle                           int
	ConnLifetime                               time.Duration
	JWTKey, MailKey                            []byte
	Issuer, Audience, PublicURL                string
	SecureCookie                               bool
	TrustedProxy                               netip.Prefix
	ProxySecret                                string
	SMTPAddr, SMTPUser, SMTPPassword, MailFrom string
	SMTPInsecure                               bool
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func LoadDatabase() (Config, error) {
	c := Config{HTTPAddr: env("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL")}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	u, err := url.Parse(c.DatabaseURL)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Hostname() == "" || u.Path == "" || u.Path == "/" {
		return c, fmt.Errorf("DATABASE_URL must be a PostgreSQL URL with host and database")
	}
	c.MaxOpen, err = strconv.Atoi(env("DB_MAX_OPEN", "10"))
	if err != nil || c.MaxOpen < 1 {
		return c, fmt.Errorf("invalid DB_MAX_OPEN")
	}
	c.MaxIdle, err = strconv.Atoi(env("DB_MAX_IDLE", "5"))
	if err != nil || c.MaxIdle < 0 || c.MaxIdle > c.MaxOpen {
		return c, fmt.Errorf("invalid DB_MAX_IDLE")
	}
	c.ConnLifetime, err = time.ParseDuration(env("DB_CONN_LIFETIME", "30m"))
	if err != nil || c.ConnLifetime <= 0 {
		return c, fmt.Errorf("invalid DB_CONN_LIFETIME")
	}
	return c, nil
}

func Load() (Config, error) {
	c, err := LoadDatabase()
	if err != nil {
		return c, err
	}
	if v := env("APP_ENV", "development"); v != "development" && v != "production" {
		return c, fmt.Errorf("invalid APP_ENV")
	}
	c.JWTKey, err = base64.StdEncoding.DecodeString(os.Getenv("JWT_SECRET"))
	if err != nil || len(c.JWTKey) < 32 {
		return c, fmt.Errorf("JWT_SECRET must be base64 of at least 32 random bytes")
	}
	c.MailKey, err = base64.StdEncoding.DecodeString(os.Getenv("MAIL_ENCRYPTION_KEY"))
	if err != nil || len(c.MailKey) != 32 {
		return c, fmt.Errorf("MAIL_ENCRYPTION_KEY must be base64 of 32 random bytes")
	}
	c.Issuer = env("JWT_ISSUER", "uptime-api")
	c.Audience = env("JWT_AUDIENCE", "uptime-web")
	c.PublicURL = strings.TrimRight(env("PUBLIC_URL", "http://localhost:3000"), "/")
	u, err := url.Parse(c.PublicURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return c, fmt.Errorf("PUBLIC_URL must be an http(s) origin")
	}
	c.SecureCookie, err = strconv.ParseBool(env("COOKIE_SECURE", "false"))
	if err != nil {
		return c, fmt.Errorf("invalid COOKIE_SECURE")
	}
	if env("APP_ENV", "development") == "production" && (!c.SecureCookie || u.Scheme != "https") {
		return c, fmt.Errorf("production requires secure cookies and HTTPS PUBLIC_URL")
	}
	if v := os.Getenv("TRUSTED_PROXY_CIDR"); v != "" {
		c.TrustedProxy, err = netip.ParsePrefix(v)
		if err != nil {
			return c, fmt.Errorf("invalid TRUSTED_PROXY_CIDR")
		}
		c.ProxySecret = os.Getenv("PROXY_SECRET")
		if len(c.ProxySecret) < 32 {
			return c, fmt.Errorf("PROXY_SECRET must contain at least 32 characters when proxy trust is enabled")
		}
	}
	c.SMTPAddr = env("SMTP_ADDR", "127.0.0.1:1025")
	if _, _, err := net.SplitHostPort(c.SMTPAddr); err != nil {
		return c, fmt.Errorf("invalid SMTP_ADDR")
	}
	c.SMTPUser = os.Getenv("SMTP_USER")
	c.SMTPPassword = os.Getenv("SMTP_PASSWORD")
	c.MailFrom = env("MAIL_FROM", "noreply@localhost")
	if _, err := mail.ParseAddress(c.MailFrom); err != nil {
		return c, fmt.Errorf("invalid MAIL_FROM")
	}
	c.SMTPInsecure, err = strconv.ParseBool(env("SMTP_ALLOW_INSECURE", "false"))
	if err != nil {
		return c, fmt.Errorf("invalid SMTP_ALLOW_INSECURE")
	}
	if env("APP_ENV", "development") == "production" && c.SMTPInsecure {
		return c, fmt.Errorf("insecure SMTP is forbidden in production")
	}
	return c, nil
}

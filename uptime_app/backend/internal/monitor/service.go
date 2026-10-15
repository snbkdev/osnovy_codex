package monitor

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"uptime_app/backend/internal/auth"
)

const MinIntervalSeconds = 5
const MaxIntervalSeconds = 7 * 24 * 60 * 60

var ErrInvalid = errors.New("invalid_monitor")

type Monitor struct {
	ID              string    `json:"id"`
	UserID          string    `json:"-"`
	URL             string    `json:"url"`
	IntervalSeconds int       `json:"interval_seconds"`
	CreatedAt       time.Time `json:"created_at"`
}

type Repository interface {
	Create(context.Context, Monitor) (Monitor, error)
	List(context.Context, string) ([]Monitor, error)
}

type Service struct{ Repo Repository }

func normalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 2048 || strings.ContainsFunc(raw, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return "", ErrInvalid
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Opaque != "" {
		return "", ErrInvalid
	}
	host := u.Hostname()
	if net.ParseIP(host) == nil {
		if len(host) > 253 || strings.Contains(u.Host, ":") && u.Port() == "" {
			return "", ErrInvalid
		}
		for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return "", ErrInvalid
			}
			for _, c := range label {
				if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
					return "", ErrInvalid
				}
			}
		}
	}
	if strings.HasSuffix(u.Host, ":") {
		return "", ErrInvalid
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", ErrInvalid
		}
	}
	u.Host = strings.ToLower(u.Host)
	// Fragments are browser-only and are not part of an HTTP request.
	u.Fragment = ""
	u.RawFragment = ""
	if u.Path == "" {
		u.Path = "/"
	}
	normalized := u.String()
	if len(normalized) > 2048 {
		return "", ErrInvalid
	}
	return normalized, nil
}

func (s *Service) Create(ctx context.Context, userID, rawURL string, interval int) (Monitor, error) {
	if userID == "" {
		return Monitor{}, auth.ErrUnauthorized
	}
	normalized, err := normalizeURL(rawURL)
	if err != nil || interval < MinIntervalSeconds || interval > MaxIntervalSeconds {
		return Monitor{}, ErrInvalid
	}
	return s.Repo.Create(ctx, Monitor{ID: auth.RandomToken(), UserID: userID, URL: normalized, IntervalSeconds: interval, CreatedAt: time.Now().UTC()})
}

func (s *Service) List(ctx context.Context, userID string) ([]Monitor, error) {
	if userID == "" {
		return nil, auth.ErrUnauthorized
	}
	return s.Repo.List(ctx, userID)
}

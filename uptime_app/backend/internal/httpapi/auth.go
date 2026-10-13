package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"uptime_app/backend/internal/auth"
	"uptime_app/backend/internal/config"
)

type authService interface {
	Register(context.Context, string, string) error
	SendToken(context.Context, string, string) error
	Verify(context.Context, string) error
	Login(context.Context, string, string) (auth.LoginResult, error)
	Refresh(context.Context, string) (auth.LoginResult, error)
	Logout(context.Context, string) error
	Reset(context.Context, string, string) error
	Me(context.Context, string) (auth.User, error)
}

type userDTO struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func dto(u auth.User) userDTO { return userDTO{u.ID, u.Email, u.EmailVerified} }

type authInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Token    string `json:"token"`
}
type bucket struct {
	Count int
	Until time.Time
}
type limiter struct {
	mu        sync.Mutex
	entries   map[string]bucket
	nextSweep time.Time
}

func (l *limiter) allow(keys []string, max int, window time.Duration, now time.Time) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.entries == nil {
		l.entries = make(map[string]bucket)
	}
	if !now.Before(l.nextSweep) {
		for k, b := range l.entries {
			if !now.Before(b.Until) {
				delete(l.entries, k)
			}
		}
		l.nextSweep = now.Add(time.Minute)
	}
	retry := 0
	for _, key := range keys {
		b := l.entries[key]
		if now.Before(b.Until) && b.Count >= max {
			seconds := int(b.Until.Sub(now).Seconds()) + 1
			if seconds > retry {
				retry = seconds
			}
		}
	}
	if retry > 0 {
		return false, retry
	}
	for _, key := range keys {
		b := l.entries[key]
		if !now.Before(b.Until) {
			b = bucket{Until: now.Add(window)}
		}
		b.Count++
		l.entries[key] = b
	}
	return true, 0
}

type authHandler struct {
	service authService
	config  config.Config
	limits  limiter
}

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
func apiError(w http.ResponseWriter, status int, code, message string) {
	jsonResponse(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func serviceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalid):
		apiError(w, 400, "invalid_input", "Проверьте данные. Ссылка могла истечь или уже использована.")
	case errors.Is(err, auth.ErrCredentials):
		apiError(w, 401, "invalid_credentials", "Неверный email или пароль.")
	case errors.Is(err, auth.ErrUnverified):
		apiError(w, 403, "email_not_verified", "Подтвердите email перед входом.")
	case errors.Is(err, auth.ErrUnauthorized):
		apiError(w, 401, "unauthorized", "Сессия завершена. Войдите снова.")
	default:
		apiError(w, 503, "service_unavailable", "Сервис временно недоступен. Попробуйте позже.")
	}
}
func decode(w http.ResponseWriter, r *http.Request) (authInput, bool) {
	var input authInput
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		apiError(w, 400, "invalid_input", "Невалидный JSON или превышен размер запроса.")
		return input, false
	}
	if decoder.Decode(new(any)) != io.EOF {
		apiError(w, 400, "invalid_input", "Ожидается один JSON-объект.")
		return input, false
	}
	return input, true
}
func (h *authHandler) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return "unknown"
	}
	ip = ip.Unmap()
	if h.config.TrustedProxy.IsValid() && h.config.TrustedProxy.Contains(ip) && h.config.ProxySecret != "" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Uptime-Proxy-Secret")), []byte(h.config.ProxySecret)) == 1 {
		if client, err := netip.ParseAddr(r.Header.Get("X-Uptime-Client-IP")); err == nil {
			return client.Unmap().String()
		}
	}
	return ip.String()
}
func (h *authHandler) limit(w http.ResponseWriter, r *http.Request, action, email string) bool {
	ip := h.clientIP(r)
	var keys []string
	max := 5
	window := time.Hour
	switch action {
	case "login":
		max = 10
		window = 15 * time.Minute
		keys = []string{"login:" + ip + ":" + email}
	case "refresh":
		max = 60
		window = time.Minute
		keys = []string{"refresh:" + ip}
	default:
		keys = []string{"mail:ip:" + ip, "mail:email:" + email}
	}
	if ok, retry := h.limits.allow(keys, max, window, time.Now()); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		apiError(w, 429, "rate_limited", "Слишком много запросов. Попробуйте позже.")
		return false
	}
	return true
}

const cookieName = "uptime_refresh"

func (h *authHandler) cookie(w http.ResponseWriter, value string, clear bool) {
	c := &http.Cookie{Name: cookieName, Value: value, Path: "/api/v1/auth", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: h.config.SecureCookie, MaxAge: 30 * 24 * 60 * 60}
	if clear {
		c.Value = ""
		c.MaxAge = -1
		c.Expires = time.Unix(1, 0)
	}
	http.SetCookie(w, c)
}
func refreshCookie(r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return c.Value
}
func (h *authHandler) tokens(w http.ResponseWriter, result auth.LoginResult) {
	h.cookie(w, result.RefreshToken, false)
	jsonResponse(w, 200, map[string]any{"access_token": result.AccessToken, "token_type": "Bearer", "expires_in": 900, "user": dto(result.User)})
}
func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	action := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/")
	if r.Method != http.MethodPost {
		apiError(w, 405, "method_not_allowed", "Метод не поддерживается.")
		return
	}
	if action == "login" || action == "refresh" || action == "logout" {
		if r.Header.Get("Origin") != h.config.PublicURL {
			apiError(w, 403, "invalid_origin", "Недопустимый источник запроса.")
			return
		}
	}
	if action == "refresh" || action == "logout" {
		if _, err := io.Copy(io.Discard, http.MaxBytesReader(w, r.Body, 16*1024)); err != nil {
			apiError(w, 400, "invalid_input", "Превышен размер запроса.")
			return
		}
	}
	if action == "refresh" {
		if !h.limit(w, r, action, "") {
			return
		}
		result, err := h.service.Refresh(r.Context(), refreshCookie(r))
		if err != nil {
			if errors.Is(err, auth.ErrUnauthorized) {
				h.cookie(w, "", true)
			}
			serviceError(w, err)
			return
		}
		h.tokens(w, result)
		return
	}
	if action == "logout" {
		if err := h.service.Logout(r.Context(), refreshCookie(r)); err != nil {
			serviceError(w, err)
			return
		}
		h.cookie(w, "", true)
		jsonResponse(w, 200, map[string]string{"message": "Вы вышли из аккаунта."})
		return
	}
	if action != "register" && action != "verify-email" && action != "resend-verification" && action != "login" && action != "forgot-password" && action != "reset-password" {
		apiError(w, 404, "not_found", "Маршрут не найден.")
		return
	}
	input, ok := decode(w, r)
	if !ok {
		return
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if action == "register" || action == "login" || action == "resend-verification" || action == "forgot-password" {
		if !h.limit(w, r, action, email) {
			return
		}
	}
	var err error
	switch action {
	case "register":
		err = h.service.Register(r.Context(), input.Email, input.Password)
	case "verify-email":
		err = h.service.Verify(r.Context(), input.Token)
	case "resend-verification":
		err = h.service.SendToken(r.Context(), input.Email, "verify")
	case "forgot-password":
		err = h.service.SendToken(r.Context(), input.Email, "reset")
	case "reset-password":
		err = h.service.Reset(r.Context(), input.Token, input.Password)
	case "login":
		result, e := h.service.Login(r.Context(), input.Email, input.Password)
		if e != nil {
			serviceError(w, e)
			return
		}
		h.tokens(w, result)
		return
	}
	if err != nil {
		serviceError(w, err)
		return
	}
	if action == "register" || action == "resend-verification" || action == "forgot-password" {
		jsonResponse(w, 202, map[string]string{"message": "Если адрес подходит для этой операции, на него будет отправлено письмо."})
		return
	}
	jsonResponse(w, 200, map[string]string{"message": "Готово. Теперь можно войти."})
}

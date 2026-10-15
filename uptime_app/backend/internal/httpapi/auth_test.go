package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
	"uptime_app/backend/internal/auth"
	"uptime_app/backend/internal/config"
)

type stubService struct {
	err   error
	calls int
}

func (s *stubService) UpdateProfile(_ context.Context, _ auth.Identity, name string, _ *string) (auth.User, error) {
	s.calls++
	return auth.User{ID: "id", Email: "a@example.com", FullName: name, EmailVerified: true}, s.err
}

func (s *stubService) Register(context.Context, string, string) error  { s.calls++; return s.err }
func (s *stubService) SendToken(context.Context, string, string) error { s.calls++; return s.err }
func (s *stubService) Verify(context.Context, string) error            { return s.err }
func (s *stubService) Login(context.Context, string, string) (auth.LoginResult, error) {
	return auth.LoginResult{AccessToken: "access", RefreshToken: "refresh", User: auth.User{ID: "id", Email: "a@example.com", PasswordHash: "secret", EmailVerified: true}}, s.err
}
func (s *stubService) Refresh(context.Context, string, ...string) (auth.LoginResult, error) {
	return s.Login(context.Background(), "", "")
}
func (s *stubService) Logout(context.Context, string) error        { return s.err }
func (s *stubService) Reset(context.Context, string, string) error { return s.err }
func (s *stubService) Me(context.Context, string) (auth.User, error) {
	return auth.User{ID: "id", Email: "a@example.com", PasswordHash: "secret", EmailVerified: true}, s.err
}
func req(handler http.Handler, method, path, body, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "/api/v1/auth/"+path, strings.NewReader(body))
	r.Header.Set("Origin", origin)
	r.Header.Set("Authorization", "Bearer token")
	r.AddCookie(&http.Cookie{Name: cookieName, Value: "refresh"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}
func TestAuthContracts(t *testing.T) {
	s := &stubService{}
	cfg := config.Config{PublicURL: "http://localhost:3000", SecureCookie: true}
	h := NewAuthRouter(s, cfg)
	for _, path := range []string{"register", "resend-verification", "forgot-password"} {
		w := req(h, "POST", path, `{"email":"a@example.com","password":"long password"}`, cfg.PublicURL)
		if w.Code != 202 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %d", path, w.Code)
		}
	}
	w := req(h, "POST", "login", `{"email":"a@example.com","password":"long password"}`, cfg.PublicURL)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"expires_in":86400`) || strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "refresh") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("missing cookie")
	}
	c := cookies[0]
	if !c.HttpOnly || !c.Secure || c.Domain != "" || c.Path != "/api/v1/auth" || c.SameSite != http.SameSiteStrictMode {
		t.Fatalf("%+v", c)
	}
	w = req(h, "POST", "logout", `{}`, cfg.PublicURL)
	if w.Code != 200 || w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("cookie not cleared")
	}
	w = req(h, "GET", "me", "", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "secret") {
		t.Fatal("invalid me DTO")
	}
	for _, path := range []string{"login", "refresh", "logout"} {
		for _, origin := range []string{"", "http://evil.test"} {
			w = req(h, "POST", path, `{}`, origin)
			if w.Code != 403 {
				t.Fatalf("origin %s %d", path, w.Code)
			}
		}
	}
}
func TestErrorsAndJSON(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code int
		text string
	}{{auth.ErrInvalid, 400, "invalid_input"}, {auth.ErrCredentials, 401, "invalid_credentials"}, {auth.ErrUnverified, 403, "email_not_verified"}, {auth.ErrUnauthorized, 401, "unauthorized"}, {errors.New("database password"), 503, "service_unavailable"}} {
		w := httptest.NewRecorder()
		serviceError(w, tc.err)
		if w.Code != tc.code || !strings.Contains(w.Body.String(), tc.text) || strings.Contains(w.Body.String(), "database password") {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
	h := NewAuthRouter(&stubService{}, config.Config{})
	for _, body := range []string{`{`, `{} {}`, `{"unknown":1}`, `{"token":"` + strings.Repeat("x", 17*1024) + `"}`} {
		w := req(h, "POST", "verify-email", body, "")
		if w.Code != 400 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%d", w.Code)
		}
	}
}
func TestLimits(t *testing.T) {
	now := time.Now()
	var l limiter
	for i := 0; i < 10; i++ {
		if ok, _ := l.allow([]string{"login:ip:email"}, 10, 15*time.Minute, now); !ok {
			t.Fatal("early limit")
		}
	}
	if ok, retry := l.allow([]string{"login:ip:email"}, 10, 15*time.Minute, now); ok || retry < 900 {
		t.Fatal("missing limit")
	}
	if ok, _ := l.allow([]string{"login:ip:other"}, 10, 15*time.Minute, now); !ok {
		t.Fatal("wrong login key")
	}
	if ok, _ := l.allow([]string{"login:ip:email"}, 10, 15*time.Minute, now.Add(15*time.Minute)); !ok {
		t.Fatal("window did not expire")
	}
	for _, action := range []string{"register", "resend-verification", "forgot-password", "refresh", "login"} {
		h := NewAuthRouter(&stubService{}, config.Config{PublicURL: "http://localhost:3000"})
		max := 5
		if action == "refresh" {
			max = 60
		}
		if action == "login" {
			max = 10
		}
		for i := 0; i < max; i++ {
			w := req(h, "POST", action, `{"email":"a@example.com"}`, "http://localhost:3000")
			if w.Code == 429 {
				t.Fatal("early HTTP limit")
			}
		}
		w := req(h, "POST", action, `{"email":"a@example.com"}`, "http://localhost:3000")
		if w.Code != 429 || w.Header().Get("Retry-After") == "" {
			t.Fatalf("%s missing 429", action)
		}
	}
	// Email bucket is shared across source IPs, IP bucket across email addresses.
	for _, keys := range [][]string{{"mail:ip:one", "mail:email:a"}, {"mail:ip:two", "mail:email:b"}} {
		for i := 0; i < 5; i++ {
			l.allow(keys, 5, time.Hour, now)
		}
	}
	if ok, _ := l.allow([]string{"mail:ip:three", "mail:email:a"}, 5, time.Hour, now); ok {
		t.Fatal("email limit bypassed")
	}
	if ok, _ := l.allow([]string{"mail:ip:two", "mail:email:c"}, 5, time.Hour, now); ok {
		t.Fatal("IP limit bypassed")
	}
}
func TestTrustedProxyIP(t *testing.T) {
	h := authHandler{config: config.Config{TrustedProxy: netip.MustParsePrefix("127.0.0.1/32"), ProxySecret: strings.Repeat("x", 32)}}
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "127.0.0.1:4321"
	r.Header.Set("X-Forwarded-For", "8.8.8.8")
	r.Header.Set("X-Uptime-Client-IP", "1.2.3.4")
	if h.clientIP(r) != "127.0.0.1" {
		t.Fatal("untrusted header accepted")
	}
	r.Header.Set("X-Uptime-Proxy-Secret", strings.Repeat("x", 32))
	if h.clientIP(r) != "1.2.3.4" {
		t.Fatal("authenticated proxy ignored")
	}
	r.RemoteAddr = "10.0.0.1:4321"
	if h.clientIP(r) != "10.0.0.1" {
		t.Fatal("untrusted source accepted")
	}
}

func TestLoginAndResetRequestBudgets(t *testing.T) {
	for _, action := range []string{"login", "reset-password"} {
		h := &authHandler{}
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("POST", "/api/v1/auth/"+action, nil)
			req.RemoteAddr = "192.0.2.1:1234"
			if !h.limit(httptest.NewRecorder(), req, action, strings.Repeat("a", i)+"@example.com") {
				t.Fatal("early rejection", action, i)
			}
		}
		req := httptest.NewRequest("POST", "/api/v1/auth/"+action, nil)
		req.RemoteAddr = "192.0.2.1:1234"
		w := httptest.NewRecorder()
		if h.limit(w, req, action, "another@example.com") || w.Code != 429 || w.Header().Get("Retry-After") == "" {
			t.Fatal("IP budget bypass", action, w.Code)
		}
	}
	h := &authHandler{}
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		if !h.limit(httptest.NewRecorder(), req, "login", "same@example.com") {
			t.Fatal("early rejection")
		}
	}
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "192.0.2.2:1234"
	if h.limit(httptest.NewRecorder(), req, "login", "same@example.com") {
		t.Fatal("account budget bypass")
	}
}

func (s *stubService) Authenticate(ctx context.Context, bearer string) (auth.Identity, error) {
	user, err := s.Me(ctx, bearer)
	return auth.Identity{User: user, SessionID: "session"}, err
}

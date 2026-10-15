package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
	"uptime_app/backend/internal/avatar"
)

type Service struct {
	Avatars   avatar.Store
	Repo      Repository
	Signer    Signer
	Cipher    *Cipher
	PublicURL string
	dummyHash string
}

func NewService(repo Repository, signer Signer, cipher *Cipher, publicURL string) (*Service, error) {
	hash, err := HashPassword(RandomToken())
	if err != nil {
		return nil, err
	}
	return &Service{Repo: repo, Signer: signer, Cipher: cipher, PublicURL: publicURL, dummyHash: hash}, nil
}
func (s *Service) letter(email, purpose, token string) ([]byte, error) {
	path, subject := "/verify-email", "Подтвердите email"
	if purpose == "reset" {
		path, subject = "/reset-password", "Восстановление пароля"
	}
	return s.Cipher.Encrypt(Letter{email, subject, fmt.Sprintf("%s в Uptime App:\n%s%s#token=%s\nЕсли вы не запрашивали письмо, проигнорируйте его.", subject, s.PublicURL, path, token)})
}
func (s *Service) Register(ctx context.Context, email, password string) error {
	release, err := passwordSlot(ctx)
	if err != nil {
		return err
	}
	defer release()
	email, err = NormalizeEmail(email)
	if err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	token := RandomToken()
	payload, err := s.letter(email, "verify", token)
	if err != nil {
		return err
	}
	return s.Repo.Register(ctx, User{ID: RandomToken(), Email: email, PasswordHash: hash}, TokenHash(token), time.Now().Add(24*time.Hour), payload)
}
func (s *Service) SendToken(ctx context.Context, email, purpose string) error {
	email, err := NormalizeEmail(email)
	if err != nil {
		return err
	}
	if purpose != "verify" && purpose != "reset" {
		return ErrInvalid
	}
	token := RandomToken()
	payload, err := s.letter(email, purpose, token)
	if err != nil {
		return err
	}
	ttl := 24 * time.Hour
	if purpose == "reset" {
		ttl = 30 * time.Minute
	}
	return s.Repo.Issue(ctx, email, purpose, TokenHash(token), time.Now().Add(ttl), payload)
}
func (s *Service) Verify(ctx context.Context, token string) error {
	if !validToken(token) {
		return ErrInvalid
	}
	return s.Repo.Verify(ctx, TokenHash(token), time.Now())
}
func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	release, err := passwordSlot(ctx)
	if err != nil {
		return LoginResult{}, err
	}
	defer release()
	email, err = NormalizeEmail(email)
	if err != nil {
		return LoginResult{}, ErrCredentials
	}
	if ValidatePassword(password) != nil {
		return LoginResult{}, ErrCredentials
	}
	user, err := s.Repo.FindUser(ctx, email)
	if errors.Is(err, ErrCredentials) {
		CheckPassword(password, s.dummyHash)
		return LoginResult{}, ErrCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}
	if !CheckPassword(password, user.PasswordHash) {
		return LoginResult{}, ErrCredentials
	}
	if !user.EmailVerified {
		return LoginResult{}, ErrUnverified
	}
	now := time.Now()
	session := Session{RandomToken(), user.ID, now.Add(30 * 24 * time.Hour)}
	refresh := RandomToken()
	access, err := s.Signer.Issue(user.ID, session.ID, now)
	if err != nil {
		return LoginResult{}, err
	}
	if err := s.Repo.CreateSession(ctx, user, session, TokenHash(refresh), now); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{access, refresh, user}, nil
}
func (s *Service) Refresh(ctx context.Context, refresh string, attempt ...string) (LoginResult, error) {
	if refresh == "" {
		return LoginResult{}, ErrUnauthorized
	}
	next := RandomToken()
	requestHash := ""
	if len(attempt) > 1 {
		return LoginResult{}, ErrInvalid
	}
	if len(attempt) == 1 && attempt[0] != "" {
		if !validToken(attempt[0]) {
			return LoginResult{}, ErrInvalid
		}
		requestHash = TokenHash(attempt[0])
		mac := hmac.New(sha256.New, s.Signer.Key)
		mac.Write([]byte("uptime-refresh-retry-v1\x00" + refresh + "\x00" + attempt[0]))
		next = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	}
	now := time.Now()
	user, session, err := s.Repo.Rotate(ctx, TokenHash(refresh), TokenHash(next), now, requestHash)
	if err != nil {
		return LoginResult{}, err
	}
	access, err := s.Signer.Issue(user.ID, session.ID, now)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{access, next, user}, nil
}
func (s *Service) Logout(ctx context.Context, refresh string) error {
	if refresh == "" {
		return nil
	}
	return s.Repo.Revoke(ctx, TokenHash(refresh), time.Now())
}
func (s *Service) Reset(ctx context.Context, token, password string) error {
	release, err := passwordSlot(ctx)
	if err != nil {
		return err
	}
	defer release()
	if !validToken(token) {
		return ErrInvalid
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	return s.Repo.Reset(ctx, TokenHash(token), hash, time.Now())
}
func (s *Service) Authenticate(ctx context.Context, bearer string) (Identity, error) {
	now := time.Now()
	claims, err := s.Signer.Parse(bearer, now)
	if err != nil {
		return Identity{}, err
	}
	user, err := s.Repo.Current(ctx, claims.Subject, claims.SessionID, now)
	if err != nil {
		return Identity{}, err
	}
	return Identity{User: user, SessionID: claims.SessionID}, nil
}
func (s *Service) Me(ctx context.Context, bearer string) (User, error) {
	identity, err := s.Authenticate(ctx, bearer)
	return identity.User, err
}

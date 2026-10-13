package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Service struct {
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
	return &Service{repo, signer, cipher, publicURL, hash}, nil
}
func (s *Service) letter(email, purpose, token string) ([]byte, error) {
	path, subject := "/verify-email", "Подтвердите email"
	if purpose == "reset" {
		path, subject = "/reset-password", "Восстановление пароля"
	}
	return s.Cipher.Encrypt(Letter{email, subject, fmt.Sprintf("%s в Uptime App:\n%s%s#token=%s\nЕсли вы не запрашивали письмо, проигнорируйте его.", subject, s.PublicURL, path, token)})
}
func (s *Service) Register(ctx context.Context, email, password string) error {
	email, err := NormalizeEmail(email)
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
	if token == "" {
		return ErrInvalid
	}
	return s.Repo.Verify(ctx, TokenHash(token), time.Now())
}
func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email, err := NormalizeEmail(email)
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
func (s *Service) Refresh(ctx context.Context, refresh string) (LoginResult, error) {
	if refresh == "" {
		return LoginResult{}, ErrUnauthorized
	}
	next := RandomToken()
	now := time.Now()
	user, session, err := s.Repo.Rotate(ctx, TokenHash(refresh), TokenHash(next), now)
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
	if token == "" {
		return ErrInvalid
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	return s.Repo.Reset(ctx, TokenHash(token), hash, time.Now())
}
func (s *Service) Me(ctx context.Context, bearer string) (User, error) {
	now := time.Now()
	claims, err := s.Signer.Parse(bearer, now)
	if err != nil {
		return User{}, err
	}
	return s.Repo.Current(ctx, claims.Subject, claims.SessionID, now)
}

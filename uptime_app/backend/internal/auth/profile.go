package auth

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func validateProfile(name string, birthday *string, now time.Time) (string, *time.Time, error) {
	name = strings.TrimSpace(name)
	if !utf8.ValidString(name) || !strings.ContainsFunc(name, unicode.IsLetter) || utf8.RuneCountInString(name) > 200 || strings.ContainsFunc(name, unicode.IsControl) {
		return "", nil, ErrInvalid
	}
	if birthday == nil || *birthday == "" {
		return name, nil, nil
	}
	date, err := time.Parse("2006-01-02", *birthday)
	if err != nil || date.Year() < 1 || date.Format("2006-01-02") > now.UTC().Format("2006-01-02") {
		return "", nil, ErrInvalid
	}
	return name, &date, nil
}

func (s *Service) UpdateProfile(ctx context.Context, bearer, name string, birthday *string) (User, error) {
	now := time.Now()
	claims, err := s.Signer.Parse(bearer, now)
	if err != nil {
		return User{}, err
	}
	name, date, err := validateProfile(name, birthday, now)
	if err != nil {
		return User{}, err
	}
	return s.Repo.UpdateProfile(ctx, claims.Subject, claims.SessionID, name, date, now)
}

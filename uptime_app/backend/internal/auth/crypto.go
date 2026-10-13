package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 || !strings.Contains(email, "@") {
		return "", ErrInvalid
	}
	return email, nil
}
func ValidatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if !utf8.ValidString(password) || n < 12 || n > 128 {
		return ErrInvalid
	}
	return nil
}
func RandomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func TokenHash(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:])
}
func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 2, 64*1024, 2, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=2,p=2$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}
func CheckPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=65536,t=2,p=2" {
		return false
	}
	salt, e1 := base64.RawStdEncoding.DecodeString(parts[4])
	hash, e2 := base64.RawStdEncoding.DecodeString(parts[5])
	if e1 != nil || e2 != nil || len(salt) != 16 || len(hash) != 32 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, 2, 64*1024, 2, 32)
	return subtle.ConstantTimeCompare(actual, hash) == 1
}

type Claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}
type Signer struct {
	Key              []byte
	Issuer, Audience string
}

func (s Signer) Issue(userID, sessionID string, now time.Time) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{SessionID: sessionID, RegisteredClaims: jwt.RegisteredClaims{Subject: userID, Issuer: s.Issuer, Audience: jwt.ClaimStrings{s.Audience}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute))}}).SignedString(s.Key)
}
func (s Signer) Parse(value string, now time.Time) (Claims, error) {
	var c Claims
	_, err := jwt.ParseWithClaims(value, &c, func(t *jwt.Token) (any, error) { return s.Key, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(s.Issuer), jwt.WithAudience(s.Audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil || c.Subject == "" || c.SessionID == "" || c.IssuedAt == nil || c.ExpiresAt == nil || !c.ExpiresAt.After(c.IssuedAt.Time) || c.ExpiresAt.Sub(c.IssuedAt.Time) > 15*time.Minute {
		return Claims{}, ErrUnauthorized
	}
	return c, nil
}

type Cipher struct{ aead cipher.AEAD }

func NewCipher(key []byte) (*Cipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead}, nil
}
func (c *Cipher) Encrypt(letter Letter) ([]byte, error) {
	data, err := json.Marshal(letter)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, data, nil), nil
}
func (c *Cipher) Decrypt(data []byte) (Letter, error) {
	var letter Letter
	n := c.aead.NonceSize()
	if len(data) < n {
		return letter, fmt.Errorf("invalid encrypted payload")
	}
	plain, err := c.aead.Open(nil, data[:n], data[n:], nil)
	if err != nil {
		return letter, err
	}
	err = json.Unmarshal(plain, &letter)
	return letter, err
}

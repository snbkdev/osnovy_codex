package auth

import (
	"bytes"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"testing"
	"time"
)

func TestEmailAndPassword(t *testing.T) {
	email, err := NormalizeEmail("  User@Example.com \n")
	if err != nil || email != "user@example.com" {
		t.Fatalf("%q %v", email, err)
	}
	for _, value := range []string{"bad", "A <a@example.com>", "a@example.com\r\nBcc:b@example.com"} {
		if _, err := NormalizeEmail(value); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
	for _, n := range []int{11, 129} {
		if ValidatePassword(strings.Repeat("я", n)) == nil {
			t.Fatalf("accepted %d", n)
		}
	}
	for _, n := range []int{12, 128} {
		if err := ValidatePassword(strings.Repeat("я", n)); err != nil {
			t.Fatal(err)
		}
	}
	password := "  long password  "
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := HashPassword(password)
	if hash == second || !CheckPassword(password, hash) || CheckPassword(strings.TrimSpace(password), hash) || CheckPassword(password, "bad") {
		t.Fatal("salt or password verification failed")
	}
}
func TestJWT(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	signer := Signer{bytes.Repeat([]byte{42}, 32), "issuer", "audience"}
	token, err := signer.Issue("user", "session", now)
	if err != nil {
		t.Fatal(err)
	}
	c, err := signer.Parse(token, now)
	if err != nil || c.Subject != "user" || c.SessionID != "session" {
		t.Fatalf("%+v %v", c, err)
	}
	if _, err := signer.Parse(token, now.Add(15*time.Minute)); err == nil {
		t.Fatal("expired JWT accepted")
	}
	for _, bad := range []Signer{{bytes.Repeat([]byte{43}, 32), "issuer", "audience"}, {signer.Key, "wrong", "audience"}, {signer.Key, "issuer", "wrong"}} {
		if _, err := bad.Parse(token, now); err == nil {
			t.Fatal("invalid JWT accepted")
		}
	}
	wrong, _ := jwt.NewWithClaims(jwt.SigningMethodHS384, c).SignedString(signer.Key)
	if _, err := signer.Parse(wrong, now); err == nil {
		t.Fatal("HS384 accepted")
	}
	c.IssuedAt = nil
	wrong, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(signer.Key)
	if _, err := signer.Parse(wrong, now); err == nil {
		t.Fatal("missing iat accepted")
	}
	c.IssuedAt = jwt.NewNumericDate(now.Add(time.Minute))
	wrong, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(signer.Key)
	if _, err := signer.Parse(wrong, now); err == nil {
		t.Fatal("future iat accepted")
	}
}
func TestCipherAndTokens(t *testing.T) {
	c, err := NewCipher(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	letter := Letter{"a@example.com", "subject", "secret #token=value"}
	data, err := c.Encrypt(letter)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("token=value")) {
		t.Fatal("plaintext leaked")
	}
	actual, err := c.Decrypt(data)
	if err != nil || actual != letter {
		t.Fatal("round trip failed")
	}
	data[len(data)-1] ^= 1
	if _, err := c.Decrypt(data); err == nil {
		t.Fatal("tampering accepted")
	}
	if _, err := c.Decrypt(nil); err == nil {
		t.Fatal("empty accepted")
	}
	a, b := RandomToken(), RandomToken()
	if a == b || len(a) != 43 || len(TokenHash(a)) != 64 {
		t.Fatal("invalid random tokens")
	}
}

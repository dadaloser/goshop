package auth

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestSign(t *testing.T) {
	tokenString, err := Sign("kid-1", "secret-key", "issuer", "audience")
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if tokenString == "" {
		t.Fatal("Sign() returned an empty token")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("secret-key"), nil
	}, jwt.WithAudience("audience"), jwt.WithIssuer("issuer"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !token.Valid {
		t.Fatal("parsed token is not valid")
	}
	if got := token.Header["kid"]; got != "kid-1" {
		t.Fatalf("kid header = %v, want %q", got, "kid-1")
	}
}

func TestEncryptAndCompare(t *testing.T) {
	hashedPassword, err := Encrypt("plain-password")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if err := Compare(hashedPassword, "plain-password"); err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if err := Compare(hashedPassword, "wrong-password"); err == nil {
		t.Fatal("Compare() error = nil, want mismatch error")
	}
}

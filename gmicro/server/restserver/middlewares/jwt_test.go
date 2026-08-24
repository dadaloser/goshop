package middlewares

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTRejectsNonHS256TokensAcrossParserPaths(t *testing.T) {
	const key = "01234567890123456789012345678901"
	claims := CustomClaims{
		ID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	rawToken, err := token.SignedString([]byte(key))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	parser := NewJWT(key)
	if _, err := parser.ParseToken(rawToken); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("ParseToken(non-HS256) error = %v, want %v", err, ErrTokenInvalid)
	}
	if _, err := parser.RefreshToken(rawToken); err == nil {
		t.Error("RefreshToken(non-HS256) error = nil, want algorithm rejection")
	}
}

func TestJWTParseTokenWithOptionsValidatesHS256Token(t *testing.T) {
	const key = "01234567890123456789012345678901"
	parser := NewJWT(key)
	rawToken, err := parser.CreateToken(CustomClaims{
		ID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "issuer",
			Audience:  jwt.ClaimStrings{"audience"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	if err != nil {
		t.Fatalf("CreateToken() error = %v", err)
	}

	claims, err := parser.ParseTokenWithOptions(rawToken, jwt.WithIssuer("issuer"), jwt.WithAudience("audience"))
	if err != nil {
		t.Fatalf("ParseTokenWithOptions() error = %v", err)
	}
	if claims.ID != 1 {
		t.Errorf("ParseTokenWithOptions() user ID = %d, want %d", claims.ID, 1)
	}
}

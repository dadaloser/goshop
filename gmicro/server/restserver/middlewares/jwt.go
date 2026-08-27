package middlewares

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	SigningKey []byte
}

const (
	JWTTokenKey   = "JWT_TOKEN"
	JWTPayloadKey = "JWT_PAYLOAD"
)

var (
	ErrTokenExpired     = errors.New("token is expired")
	ErrTokenNotValidYet = errors.New("token not active yet")
	ErrTokenMalformed   = errors.New("that's not even a token")
	ErrTokenInvalid     = errors.New("could not handle this token")
)

func NewJWT(signKey string) *JWT {
	return NewJWTWithSigningKey([]byte(signKey))
}

// NewJWTWithSigningKey creates an HS256 JWT parser and signer from raw key bytes.
func NewJWTWithSigningKey(signingKey []byte) *JWT {
	return &JWT{
		SigningKey: append([]byte(nil), signingKey...),
	}
}

// CreateToken signs application-defined claims with HS256.
func (j *JWT) CreateToken(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}

// ParseTokenWithOptions parses an HS256 token and applies the supplied claim validation options.
func (j *JWT) ParseTokenWithOptions(tokenString string, options ...jwt.ParserOption) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	if err := j.ParseTokenWithClaims(tokenString, claims, options...); err != nil {
		return nil, err
	}
	return claims, nil
}

// ParseTokenWithClaims parses into claims owned by the caller.
func (j *JWT) ParseTokenWithClaims(tokenString string, claims jwt.Claims, options ...jwt.ParserOption) error {
	parserOptions := append([]jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	}, options...)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return j.SigningKey, nil
	}, parserOptions...)
	if err != nil {
		return err
	}
	if !token.Valid {
		return ErrTokenInvalid
	}
	return nil
}

// ParseToken parses an HS256 token.
func (j *JWT) ParseToken(tokenString string) (jwt.MapClaims, error) {
	claims, err := j.ParseTokenWithOptions(tokenString)
	if errors.Is(err, jwt.ErrTokenMalformed) {
		return nil, ErrTokenMalformed
	}
	if errors.Is(err, jwt.ErrTokenExpired) {
		return nil, ErrTokenExpired
	}
	if errors.Is(err, jwt.ErrTokenNotValidYet) {
		return nil, ErrTokenNotValidYet
	}
	// 其他所有错误
	if err != nil {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// RefreshToken refreshes an HS256 token.
func (j *JWT) RefreshToken(tokenString string) (string, error) {
	claims, err := j.ParseTokenWithOptions(tokenString, jwt.WithLeeway(0))
	if err != nil {
		return "", err
	}
	claims["exp"] = time.Now().Add(time.Hour).Unix()
	return j.CreateToken(claims)
}

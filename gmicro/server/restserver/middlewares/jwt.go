package middlewares

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	ID              uint     `json:"user_id"`
	NickName        string   `json:"nick_name,omitempty"`
	AuthorityId     uint     `json:"authority_id,omitempty"`
	Roles           []string `json:"roles,omitempty"`
	PrincipalType   string   `json:"principal_type,omitempty"`
	AccountStatus   string   `json:"status,omitempty"`
	Scope           []string `json:"scope,omitempty"`
	TokenVersion    uint64   `json:"token_version"`
	SessionID       string   `json:"session_id,omitempty"`
	ResourceDomains []string `json:"resource_domains,omitempty"`
	ResourceStores  []string `json:"resource_stores,omitempty"`
	ResourceTeams   []string `json:"resource_teams,omitempty"`
	ResourceScopes  []string `json:"resource_scopes,omitempty"`
	jwt.RegisteredClaims
}

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

// 创建一个token
func (j *JWT) CreateToken(claims CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}

// ParseTokenWithOptions parses an HS256 token and applies the supplied claim validation options.
func (j *JWT) ParseTokenWithOptions(tokenString string, options ...jwt.ParserOption) (*CustomClaims, error) {
	parserOptions := append([]jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	}, options...)
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(*jwt.Token) (any, error) {
		return j.SigningKey, nil
	}, parserOptions...)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// ParseToken parses an HS256 token.
func (j *JWT) ParseToken(tokenString string) (*CustomClaims, error) {
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
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
	return j.CreateToken(*claims)
}

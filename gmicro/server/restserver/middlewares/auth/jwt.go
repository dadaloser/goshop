package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"goshop/gmicro/server/restserver/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthzAudience defines the value of jwt audience field.
const AuthzAudience = "goshop.imooc.com"

// JWTStrategy defines jwt bearer authentication strategy.
type JWTStrategy struct {
	key              []byte
	realm            string
	audience         string
	identityKey      string
	authorizator     func(interface{}, *gin.Context) bool
	failureResponder FailureResponder
}

var _ middlewares.AuthStrategy = &JWTStrategy{}

// NewJWTStrategy creates a jwt bearer strategy backed by golang-jwt/jwt/v5.
func NewJWTStrategy(key []byte, realm, audience, identityKey string, authorizator func(interface{}, *gin.Context) bool, options ...Option) JWTStrategy {
	return JWTStrategy{
		key:              key,
		realm:            realm,
		audience:         audience,
		identityKey:      identityKey,
		authorizator:     authorizator,
		failureResponder: resolveFailureResponder(options),
	}
}

// AuthFunc defines jwt bearer strategy as the gin authentication middleware.
func (j JWTStrategy) AuthFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := GetToken(c)
		if err != nil {
			reject(c, j.failureResponder, err)
			return
		}

		claims, err := j.parseToken(tokenString)
		if err != nil {
			reject(c, j.failureResponder, ErrInvalidToken)
			return
		}

		payload := claimsToMap(*claims)
		c.Set(middlewares.JWTTokenKey, tokenString)
		c.Set(middlewares.JWTPayloadKey, payload)
		identity := payload[j.identityKey]
		if identity == nil {
			identity = payload["userid"]
		}
		if identity != nil {
			c.Set(middlewares.KeyUserID, identity)
		}

		if !j.Authorizator(identity, c) {
			reject(c, j.failureResponder, ErrUnauthorized)
			return
		}

		c.Next()
	}
}

// Authorizator evaluates the configured authorization callback.
func (j JWTStrategy) Authorizator(identity interface{}, c *gin.Context) bool {
	if j.authorizator == nil {
		return true
	}
	return j.authorizator(identity, c)
}

// GetToken extracts a bearer token from the Authorization header or cookie.
func GetToken(c *gin.Context) (string, error) {
	if c == nil || c.Request == nil {
		return "", fmt.Errorf("request is not initialized: %w", ErrMissingCredentials)
	}

	if header := strings.TrimSpace(c.GetHeader("Authorization")); header != "" {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			return "", ErrInvalidAuthorization
		}
		return strings.TrimSpace(parts[1]), nil
	}

	cookie, err := c.Cookie("jwt")
	if err == nil && strings.TrimSpace(cookie) != "" {
		return strings.TrimSpace(cookie), nil
	}
	if err != nil && err != http.ErrNoCookie {
		return "", err
	}

	return "", ErrMissingCredentials
}

// ExtractClaims returns the jwt payload map stored by the auth middleware.
func ExtractClaims(c *gin.Context) map[string]any {
	if c == nil {
		return map[string]any{}
	}
	raw, ok := c.Get(middlewares.JWTPayloadKey)
	if !ok {
		return map[string]any{}
	}
	claims, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return claims
}

func (j JWTStrategy) parseToken(tokenString string) (*middlewares.CustomClaims, error) {
	parserOptions := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	}
	if strings.TrimSpace(j.realm) != "" {
		parserOptions = append(parserOptions, jwt.WithIssuer(j.realm))
	}
	if strings.TrimSpace(j.audience) != "" {
		parserOptions = append(parserOptions, jwt.WithAudience(j.audience))
	}

	token, err := jwt.ParseWithClaims(tokenString, &middlewares.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.key, nil
	}, parserOptions...)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*middlewares.CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}
	return claims, nil
}

func claimsToMap(claims middlewares.CustomClaims) map[string]any {
	payload := make(map[string]any)
	data, err := json.Marshal(claims)
	if err != nil {
		return payload
	}
	_ = json.Unmarshal(data, &payload)
	return payload
}

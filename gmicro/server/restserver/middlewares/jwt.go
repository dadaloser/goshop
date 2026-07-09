package middlewares

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	ID           uint `json:"userid"`
	NickName     string
	AuthorityId  uint
	TokenVersion uint64 `json:"tv"`
	jwt.RegisteredClaims
}

type JWT struct {
	SigningKey []byte
}

var (
	TokenExpired     = errors.New("token is expired")
	TokenNotValidYet = errors.New("token not active yet")
	TokenMalformed   = errors.New("that's not even a token")
	TokenInvalid     = errors.New("Couldn't handle this token : ")
)

func NewJWT(signKey string) *JWT {
	return &JWT{
		[]byte(signKey), //可以设置过期时间
	}
}

// 创建一个token
func (j *JWT) CreateToken(claims CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}

// 解析 token
func (j *JWT) ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (i interface{}, e error) {
		return j.SigningKey, nil
	})
	if errors.Is(err, jwt.ErrTokenMalformed) {
		return nil, TokenMalformed
	}
	if errors.Is(err, jwt.ErrTokenExpired) {
		return nil, TokenExpired
	}
	if errors.Is(err, jwt.ErrTokenNotValidYet) {
		return nil, TokenNotValidYet
	}
	// 其他所有错误
	if err != nil {
		return nil, TokenInvalid
	}

	if token != nil {
		if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
			return claims, nil
		}
		return nil, TokenInvalid

	} else {
		return nil, TokenInvalid

	}

}

// RefreshToken 更新token
func (j *JWT) RefreshToken(tokenString string) (string, error) {

	/*jwt.TimeFunc = func() time.Time {
		return time.Unix(0, 0)
	}
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		jwt.TimeFunc = time.Now
		claims.RegisteredClaims.ExpiresAt = time.Now().Add(1 * time.Hour).Unix()
		return j.CreateToken(*claims)
	}*/

	// 1. 解析 Token
	// 使用 jwt.WithLeeway 替代 TimeFunc 来处理时间误差（如果需要）
	// 如果你之前设置 TimeFunc 是为了测试，现在可以用 jwt.WithLeeway(0) 或自定义 Validator
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	}, jwt.WithLeeway(0)) // 如果需要容错，可以设置例如 10*time.Second

	if err != nil {
		return "", err
	}

	// 2. 验证并刷新
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		// 3. 更新过期时间
		// 必须使用 jwt.NewNumericDate 来包装时间
		claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(1 * time.Hour))

		// 4. 重新生成 Token
		return j.CreateToken(*claims)
	}

	return "", TokenInvalid
}

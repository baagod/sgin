package sgin

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/xid"
)

type ClaimsValidator interface {
	ValidateClaims(*jwt.RegisteredClaims) error
}

type JWT[T any] struct {
	Key     string            // 上下文键名 (如 "user")
	Secret  []byte            // 签名密钥
	Timeout time.Duration     // 默认过期时间
	Method  jwt.SigningMethod // 签名算法
	Issuer  string            // 签发者
}

type Claims[T any] struct {
	Data T `json:"data"`
	jwt.RegisteredClaims
}

func (c *Claims[T]) Validate() error {
	// 尝试将 Data 转为 Validator 接口
	if v, ok := any(c.Data).(ClaimsValidator); ok {
		// 将外层的 RegisteredClaims 传进去，实现 “全上下文”
		return v.ValidateClaims(&c.RegisteredClaims)
	}
	return nil
}

func NewJWT[T any](key string, secret []byte, timeout time.Duration, opts ...func(*JWT[T])) *JWT[T] {
	j := &JWT[T]{
		Key:     key,
		Secret:  secret,
		Timeout: timeout,
		Method:  jwt.SigningMethodHS256,
		Issuer:  "sign",
	}

	for _, opt := range opts {
		if opt != nil {
			opt(j)
		}
	}

	return j
}

func (j *JWT[T]) IssueWith(data T, setup func(*Claims[T]), opts ...jwt.TokenOption) (string, error) {
	now := time.Now()
	claims := &Claims[T]{
		Data: data,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        xid.New().String(),      // 唯一标识
			IssuedAt:  jwt.NewNumericDate(now), // 签发时间
			NotBefore: jwt.NewNumericDate(now), // 生效时间
			Issuer:    j.Issuer,                // 签发者
		},
	}

	if j.Timeout > 0 { // 设置过期时间
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(j.Timeout))
	}

	if setup != nil {
		setup(claims)
	}

	token := jwt.NewWithClaims(j.Method, claims, opts...)
	return token.SignedString(j.Secret)
}

func (j *JWT[T]) Issue(data T, opts ...jwt.TokenOption) (string, error) {
	return j.IssueWith(data, nil, opts...)
}

func (j *JWT[T]) Parse(signed string, opts ...jwt.ParserOption) (*Claims[T], *jwt.Token, error) {
	claims := &Claims[T]{}

	token, err := jwt.ParseWithClaims(signed, claims, func(t *jwt.Token) (any, error) {
		if j.Method != nil {
			if alg := t.Method.Alg(); alg != j.Method.Alg() {
				return nil, errors.New("unexpected signing method")
			}
		}
		return j.Secret, nil
	}, opts...)

	if err != nil {
		return nil, token, err
	}

	return claims, token, nil
}

func (j *JWT[T]) Auth(failure func(*Ctx, *jwt.Token, error) error, opts ...jwt.ParserOption) Handler {
	return He(func(c *Ctx) error {
		signed := j.Token(c.Gin())
		claims, token, err := j.Parse(signed, opts...)

		if err == nil && claims != nil {
			c.Gin().Set(j.Key, claims)
			return c.Next()
		}

		if failure != nil {
			return failure(c, token, err)
		}

		return c.Send(ErrUnauthorized())
	})
}

// Token 从 Query 或 Header 中提取 Token
func (j *JWT[T]) Token(gc *gin.Context) string {
	token := gc.Query(HeaderAuthorization)
	if token == "" {
		token = gc.GetHeader(HeaderAuthorization)
	}

	if len(token) > 7 && strings.ToLower(token[0:7]) == "bearer " {
		return token[7:]
	}

	return token
}

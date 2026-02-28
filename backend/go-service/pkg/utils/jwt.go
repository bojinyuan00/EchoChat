package utils

import (
	"errors"
	"time"

	"github.com/echochat/backend/config"
	"github.com/golang-jwt/jwt/v5"
)

// Claims 自定义 JWT Claims，在标准 Claims 基础上扩展业务字段
type Claims struct {
	UserID   int64    `json:"user_id"`  // 用户 ID
	Username string   `json:"username"` // 用户名
	Roles    []string `json:"roles"`    // 用户角色代码列表
	jwt.RegisteredClaims
}

var (
	ErrTokenExpired = errors.New("token 已过期")
	ErrTokenInvalid = errors.New("token 无效")
)

// GenerateToken 生成 Access Token
// 包含 UserID、Username、Roles，有效期由配置中的 access_expire_min 决定
func GenerateToken(cfg *config.JWTConfig, userID int64, username string, roles []string) (string, error) {
	expireTime := time.Now().Add(time.Duration(cfg.AccessExpireMin) * time.Minute)

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
			Subject:   "access",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// GenerateRefreshToken 生成 Refresh Token
// 仅包含 UserID，有效期由配置中的 refresh_expire_day 决定
func GenerateRefreshToken(cfg *config.JWTConfig, userID int64) (string, error) {
	expireTime := time.Now().Add(time.Duration(cfg.RefreshExpireDay) * 24 * time.Hour)

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    cfg.Issuer,
			Subject:   "refresh",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

// ParseToken 解析并验证 JWT Token
// 验证签名、过期时间、签发者，返回解析后的 Claims
func ParseToken(cfg *config.JWTConfig, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(cfg.Secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

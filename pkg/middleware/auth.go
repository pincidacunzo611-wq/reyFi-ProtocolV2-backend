// Package middleware 提供通用 HTTP 中间件。
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/reyfi/reyfi-backend/pkg/errorx"
	"github.com/reyfi/reyfi-backend/pkg/response"
)

// 上下文键
type contextKey string

const (
	CtxKeyWalletAddress contextKey = "walletAddress"
	CtxKeyUserId        contextKey = "userId"
)

// JwtAuthConfig JWT 鉴权配置
type JwtAuthConfig struct {
	AccessSecret  string
	AccessExpire  int64 // 秒
	RefreshExpire int64 // 秒
}

// JwtClaims 自定义 JWT Claims
type JwtClaims struct {
	UserId        int64  `json:"userId"`
	WalletAddress string `json:"walletAddress"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token
func GenerateToken(cfg JwtAuthConfig, userId int64, walletAddress string) (accessToken, refreshToken string, err error) {
	now := time.Now()

	// Access Token
	accessClaims := JwtClaims{
		UserId:        userId,
		WalletAddress: walletAddress,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.AccessExpire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "reyfi",
		},
	}
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString([]byte(cfg.AccessSecret))
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	// Refresh Token (有效期更长)
	refreshClaims := JwtClaims{
		UserId:        userId,
		WalletAddress: walletAddress,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.RefreshExpire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "reyfi-refresh",
		},
	}
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshTokenObj.SignedString([]byte(cfg.AccessSecret))
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// ParseToken 解析并验证 JWT Token
func ParseToken(tokenStr string, secret string) (*JwtClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token claims")
}

// AuthMiddleware 创建 JWT 鉴权中间件
// 从 Authorization: Bearer <token> 头中提取并验证 JWT
// 验证成功后将 walletAddress 和 userId 注入 context
func AuthMiddleware(secret string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(r.Context(), w, errorx.CodeUnauthorized, "请先登录")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Error(r.Context(), w, errorx.CodeUnauthorized, "Authorization 格式错误")
				return
			}

			claims, err := ParseToken(parts[1], secret)
			if err != nil {
				response.Error(r.Context(), w, errorx.CodeTokenExpired, "Token 无效或已过期")
				return
			}

			// 将用户信息注入 context
			ctx := context.WithValue(r.Context(), CtxKeyWalletAddress, claims.WalletAddress)
			ctx = context.WithValue(ctx, CtxKeyUserId, claims.UserId)

			next(w, r.WithContext(ctx))
		}
	}
}

// OptionalAuthMiddleware 可选鉴权中间件
// 如果有 Token 则解析，没有也放行（用于公开接口但想识别用户身份的场景）
func OptionalAuthMiddleware(secret string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				claims, err := ParseToken(parts[1], secret)
				if err == nil {
					ctx := context.WithValue(r.Context(), CtxKeyWalletAddress, claims.WalletAddress)
					ctx = context.WithValue(ctx, CtxKeyUserId, claims.UserId)
					r = r.WithContext(ctx)
				}
			}

			next(w, r)
		}
	}
}

// GetWalletAddress 从 context 中获取钱包地址
func GetWalletAddress(ctx context.Context) string {
	val := ctx.Value(CtxKeyWalletAddress)
	if val == nil {
		return ""
	}
	return val.(string)
}

// GetUserId 从 context 中获取用户 ID
func GetUserId(ctx context.Context) int64 {
	val := ctx.Value(CtxKeyUserId)
	if val == nil {
		return 0
	}
	return val.(int64)
}

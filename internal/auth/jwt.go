package auth

import (
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/golang-jwt/jwt/v5"
)

// Claims はJWTのペイロード部分の構造を定義します。
type Claims struct {
	UserID        int    `json:"user_id"`
	SessionID     string `json:"session_id"`
	AccountTypeID int    `json:"account_type_id"` // 1:PLAYER, 2:EDITOR, 3:ADMIN
	jwt.RegisteredClaims
}

// GenerateToken は指定されたユーザーとセッションIDの新しいJWTを生成します。
func GenerateToken(user *entity.User, sessionID string, secret string, expirationHour int) (string, error) {
	now := time.Now()
	expirationTime := now.Add(time.Duration(expirationHour) * time.Hour)
	claims := &Claims{
		UserID:        user.ID,
		SessionID:     sessionID,
		AccountTypeID: user.AccountTypeID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	// HS256署名アルゴリズムを使用してトークンを生成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken は与えられたトークン文字列を検証します。
func ValidateToken(tokenString string, secret string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		// 署名アルゴリズムがHMACであることを検証（SEC-003対応）
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	// トークンが有効かチェック
	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}

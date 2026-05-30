package entity

import (
	"errors"
	"time"
)

// TemporaryPlayerData は未ログイン時に一時保存するプレイヤーデータです。
type TemporaryPlayerData struct {
	Token     string
	IPAddress string
	Payload   []byte
	BodyHash  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

var (
	ErrTemporaryPlayerDataTokenRequired     = errors.New("temporary player data token is required")
	ErrTemporaryPlayerDataIPAddressRequired = errors.New("temporary player data ip address is required")
	ErrTemporaryPlayerDataPayloadRequired   = errors.New("temporary player data payload is required")
	ErrTemporaryPlayerDataExpiresAtInvalid  = errors.New("temporary player data expires_at must be after created_at")
)

// NewTemporaryPlayerData は必須項目を検証した TemporaryPlayerData を生成します。
func NewTemporaryPlayerData(token string, ipAddress string, payload []byte, bodyHash string, createdAt time.Time, expiresAt time.Time) (*TemporaryPlayerData, error) {
	if token == "" {
		return nil, ErrTemporaryPlayerDataTokenRequired
	}
	if ipAddress == "" {
		return nil, ErrTemporaryPlayerDataIPAddressRequired
	}
	if len(payload) == 0 {
		return nil, ErrTemporaryPlayerDataPayloadRequired
	}
	if !expiresAt.After(createdAt) {
		return nil, ErrTemporaryPlayerDataExpiresAtInvalid
	}

	return &TemporaryPlayerData{
		Token:     token,
		IPAddress: ipAddress,
		Payload:   append([]byte(nil), payload...),
		BodyHash:  bodyHash,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}, nil
}

// IsExpired は指定時刻時点で有効期限切れかを返します。
func (t *TemporaryPlayerData) IsExpired(now time.Time) bool {
	if t == nil {
		return true
	}
	return !now.Before(t.ExpiresAt)
}

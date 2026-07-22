package entity

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/apitokenname"
)

const apiTokenHashLength = 64

var (
	ErrAPITokenIDInvalid     = errors.New("API token id is invalid")
	ErrAPITokenUserIDInvalid = errors.New("API token user id is invalid")
	ErrAPITokenHashInvalid   = errors.New("API token hash is invalid")
	ErrAPITokenPrefixInvalid = errors.New("API token prefix is invalid")
)

// APIToken は外部APIで利用する永続化トークンを表します。
type APIToken struct {
	ID          uint64
	UserID      int
	Name        apitokenname.APITokenName
	HashedToken string
	TokenPrefix *string
	LastUsedAt  *time.Time
	CreatedAt   time.Time
}

// NewAPIToken は新規発行するAPIトークンを生成します。
func NewAPIToken(userID int, name string, hashedToken string, tokenPrefix string) (*APIToken, error) {
	validatedName, err := apitokenname.NewAPITokenName(name)
	if err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, ErrAPITokenUserIDInvalid
	}
	if !isValidAPITokenHash(hashedToken) {
		return nil, ErrAPITokenHashInvalid
	}
	if len(tokenPrefix) != 5 {
		return nil, ErrAPITokenPrefixInvalid
	}
	prefix := tokenPrefix
	return &APIToken{
		UserID:      userID,
		Name:        validatedName,
		HashedToken: hashedToken,
		TokenPrefix: &prefix,
	}, nil
}

// RestoreAPIToken は永続化済みデータからAPIトークンを復元します。
// 旧仕様のトークンは表示用prefixを保持していないため、nilを許容します。
func RestoreAPIToken(id uint64, userID int, name string, hashedToken string, tokenPrefix *string, lastUsedAt *time.Time, createdAt time.Time) (*APIToken, error) {
	if id == 0 {
		return nil, ErrAPITokenIDInvalid
	}
	validatedName, err := apitokenname.NewAPITokenName(name)
	if err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, ErrAPITokenUserIDInvalid
	}
	if !isValidAPITokenHash(hashedToken) {
		return nil, ErrAPITokenHashInvalid
	}
	if tokenPrefix != nil && len(*tokenPrefix) != 5 {
		return nil, ErrAPITokenPrefixInvalid
	}
	return &APIToken{
		ID:          id,
		UserID:      userID,
		Name:        validatedName,
		HashedToken: hashedToken,
		TokenPrefix: cloneStringPointer(tokenPrefix),
		LastUsedAt:  cloneTimePointer(lastUsedAt),
		CreatedAt:   createdAt,
	}, nil
}

// Rename はAPIトークンの表示名を変更します。
func (t *APIToken) Rename(name string) error {
	validatedName, err := apitokenname.NewAPITokenName(name)
	if err != nil {
		return err
	}
	t.Name = validatedName
	return nil
}

// ShouldRecordUsage は最終利用日時を永続化する間隔を経過したか判定します。
func (t *APIToken) ShouldRecordUsage(now time.Time, interval time.Duration) bool {
	return t.LastUsedAt == nil || !now.Before(t.LastUsedAt.Add(interval))
}

// RecordUsage は最終利用日時を更新します。
func (t *APIToken) RecordUsage(usedAt time.Time) {
	t.LastUsedAt = &usedAt
}

func isValidAPITokenHash(value string) bool {
	if len(value) != apiTokenHashLength {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

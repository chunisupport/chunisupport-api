package models

import (
	"errors"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

// UserModel はデータベース用のUserモデルです。
type UserModel struct {
	ID            int       `db:"id"`
	Username      string    `db:"username"`
	FirebaseUID   *string   `db:"firebase_uid"`
	PasswordHash  string    `db:"password_hash"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
	PlayerID      *int      `db:"player_id"`
	AccountTypeID int       `db:"account_type_id"`
	IsSuspicious  bool      `db:"is_suspicious"`
	IsPrivate     bool      `db:"is_private"`
}

func (m *UserModel) ToEntity() (*entity.User, error) {
	uname, err := username.NewUserName(m.Username)
	if err != nil {
		return nil, err
	}

	// 空のpassword_hashはFirebase認証専用ユーザーを表します。
	var phash passwordhash.PasswordHash
	if m.PasswordHash == "" {
		if m.FirebaseUID == nil || strings.TrimSpace(*m.FirebaseUID) == "" {
			return nil, errors.New("password hash cannot be empty")
		}
		phash = passwordhash.NewEmptyPasswordHash()
	} else {
		phash, err = passwordhash.NewPasswordHash(m.PasswordHash)
		if err != nil {
			return nil, err
		}
	}

	return &entity.User{
		ID:            m.ID,
		Username:      uname,
		FirebaseUID:   m.FirebaseUID,
		PasswordHash:  phash,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		PlayerID:      m.PlayerID,
		AccountTypeID: m.AccountTypeID,
		IsSuspicious:  m.IsSuspicious,
		IsPrivate:     m.IsPrivate,
	}, nil
}

// FromEntity はentity.UserをUserModelに変換します。
func FromUserEntity(e *entity.User) *UserModel {
	return &UserModel{
		ID:            e.ID,
		Username:      e.Username.String(),
		FirebaseUID:   e.FirebaseUID,
		PasswordHash:  e.PasswordHash.String(),
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
		PlayerID:      e.PlayerID,
		AccountTypeID: e.AccountTypeID,
		IsSuspicious:  e.IsSuspicious,
		IsPrivate:     e.IsPrivate,
	}
}

// UserWithPlayerRow はユーザーとプレイヤー情報のJOIN結果を格納するモデルです。
// StructScanでLEFT JOIN結果を取得するために使用します。
type UserWithPlayerRow struct {
	// ユーザー情報
	UserID       int    `db:"user_id"`
	Username     string `db:"username"`
	UserPlayerID *int   `db:"user_player_id"`

	// プレイヤー情報（LEFT JOINなのでnull許容）
	PlayerID             *int     `db:"player_id"`
	PlayerName           *string  `db:"player_name"`
	PlayerOfficialRating *float64 `db:"player_official_rating"`
	PlayerOverpowerValue *float64 `db:"player_overpower_value"`
}
